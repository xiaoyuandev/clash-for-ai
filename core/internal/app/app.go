package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/api"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/config"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/gateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/health"
	localgatewayexecutor "github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway/executor"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/logging"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/settings"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/storage"
)

func Run() error {
	cfg := config.Load()

	sqliteStore, err := storage.NewSQLite(filepath.Join(cfg.DataDir, "clash-for-ai.db"))
	if err != nil {
		return err
	}
	defer sqliteStore.Close()

	credentialStore, err := credential.NewFileStore(filepath.Join(cfg.DataDir, "credentials.json"))
	if err != nil {
		return err
	}

	providerRepository := provider.NewSQLiteRepository(sqliteStore.DB)
	modelSourceRepository := modelsource.NewSQLiteRepository(sqliteStore.DB)
	settingsRepository := settings.NewSQLiteRepository(sqliteStore.DB)
	logRepository := logging.NewSQLiteRepository(sqliteStore.DB)
	logService := logging.NewService(logRepository, cfg.LogRetentionDays, cfg.LogMaxRecords)
	modelSourceService := modelsource.NewService(modelSourceRepository, credentialStore)
	providerService := provider.NewService(providerRepository, credentialStore, modelSourceService)
	settingsService := settings.NewService(settingsRepository)
	healthService := health.NewService(providerService, credentialStore)
	localGatewayExecutor := localgatewayexecutor.New(nil)
	currentSettings, err := settingsService.Get(context.Background())
	if err != nil {
		return err
	}

	localGatewayBaseURL := buildLocalGatewayBaseURL(
		currentSettings.LocalGateway.ListenHost,
		currentSettings.LocalGateway.ListenPort,
	)
	if currentSettings.LocalGateway.Enabled {
		if err := startLocalGatewayRuntime(
			currentSettings.LocalGateway,
			gateway.NewLocalRuntimeHandler(providerService, modelSourceService, localGatewayExecutor),
		); err != nil {
			return err
		}
	}

	if err := initializeLocalGatewayProvider(
		context.Background(),
		providerService,
		settingsService,
		currentSettings,
		localGatewayBaseURL,
	); err != nil {
		return err
	}

	gatewayHandler := gateway.NewHandler(
		providerService,
		localGatewayExecutor,
		credentialStore,
		logService,
	)

	handler := api.NewRouter(providerService, modelSourceService, healthService, logService, gatewayHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.GatewayBind, cfg.HTTPPort),
		Handler: handler,
		BaseContext: func(net.Listener) context.Context {
			return context.Background()
		},
	}

	return server.ListenAndServe()
}

func initializeLocalGatewayProvider(
	ctx context.Context,
	providers *provider.Service,
	settingsService *settings.Service,
	currentSettings settings.AppSettings,
	baseURL string,
) error {
	localProvider, err := providers.EnsureSystemLocalGateway(ctx, baseURL)
	if err != nil {
		return err
	}

	needsSettingsCleanup := false

	if isEmptyClaudeCodeModelMap(localProvider.ClaudeCodeModelMap) && hasLegacyClaudeCodeModelMap(currentSettings.LocalGatewayClaude) {
		if _, err := providers.UpdateClaudeCodeModelMap(ctx, provider.LocalGatewayProviderID, provider.ClaudeCodeModelMap{
			Opus:   currentSettings.LocalGatewayClaude.Opus,
			Sonnet: currentSettings.LocalGatewayClaude.Sonnet,
			Haiku:  currentSettings.LocalGatewayClaude.Haiku,
		}); err != nil {
			return err
		}
		currentSettings.LocalGatewayClaude = settings.ClaudeCodeModelMap{}
		needsSettingsCleanup = true
	}

	selectedModels, err := providers.ListSelectedModels(ctx, provider.LocalGatewayProviderID)
	if err != nil {
		return err
	}

	if len(selectedModels) == 0 && len(currentSettings.LocalGatewaySelected) > 0 {
		items := make([]provider.SelectedModel, 0, len(currentSettings.LocalGatewaySelected))
		for _, item := range currentSettings.LocalGatewaySelected {
			modelID := strings.TrimSpace(item.ModelID)
			if modelID == "" {
				continue
			}
			items = append(items, provider.SelectedModel{
				ModelID:  modelID,
				Position: item.Position,
			})
		}

		if _, err := providers.ReplaceSelectedModels(ctx, provider.LocalGatewayProviderID, items); err != nil {
			return err
		}

		currentSettings.LocalGatewaySelected = []settings.SelectedModel{}
		needsSettingsCleanup = true
	}

	if needsSettingsCleanup {
		if _, err := settingsService.Save(ctx, currentSettings); err != nil {
			return err
		}
	}

	return nil
}

func startLocalGatewayRuntime(
	localSettings settings.LocalGatewaySettings,
	handler http.Handler,
) error {
	listenHost := resolveLocalGatewayListenHost(localSettings.ListenHost)
	addr := fmt.Sprintf("%s:%d", listenHost, localSettings.ListenPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("start local gateway runtime listener: %w", err)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
		BaseContext: func(net.Listener) context.Context {
			return context.Background()
		},
	}

	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Printf("local gateway runtime exited: %v", serveErr)
		}
	}()

	return nil
}

func resolveLocalGatewayListenHost(host string) string {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" {
		return "127.0.0.1"
	}
	return trimmed
}

func buildLocalGatewayBaseURL(bindHost string, port int) string {
	host := strings.TrimSpace(bindHost)
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}

	return fmt.Sprintf("http://%s:%d", host, port)
}

func isEmptyClaudeCodeModelMap(input provider.ClaudeCodeModelMap) bool {
	return strings.TrimSpace(input.Opus) == "" &&
		strings.TrimSpace(input.Sonnet) == "" &&
		strings.TrimSpace(input.Haiku) == ""
}

func hasLegacyClaudeCodeModelMap(input settings.ClaudeCodeModelMap) bool {
	return strings.TrimSpace(input.Opus) != "" ||
		strings.TrimSpace(input.Sonnet) != "" ||
		strings.TrimSpace(input.Haiku) != ""
}
