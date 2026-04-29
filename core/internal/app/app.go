package app

import (
	"context"
	"fmt"
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

	if err := initializeLocalGatewayProvider(context.Background(), providerService, settingsService, cfg.GatewayBind, cfg.HTTPPort); err != nil {
		return err
	}

	gatewayHandler := gateway.NewHandler(
		providerService,
		modelSourceService,
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
	bindHost string,
	port int,
) error {
	localProvider, err := providers.EnsureSystemLocalGateway(ctx, buildLocalGatewayBaseURL(bindHost, port))
	if err != nil {
		return err
	}

	currentSettings, err := settingsService.Get(ctx)
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
