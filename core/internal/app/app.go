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
	"github.com/xiaoyuandev/clash-for-ai/core/internal/gatewayadapter"
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

	localRuntimeAdapter := gatewayadapter.NewEmbeddedLocalRuntimeAdapter(
		currentSettings.LocalGateway,
		gateway.NewLocalRuntimeHandler(providerService, modelSourceService, settingsService, localGatewayExecutor),
	)
	if currentSettings.LocalGateway.Enabled {
		if err := localRuntimeAdapter.Start(); err != nil {
			return err
		}
	}

	if err := initializeLocalGatewayProvider(
		context.Background(),
		providerService,
		settingsService,
		currentSettings,
		localRuntimeAdapter.BaseURL(),
	); err != nil {
		return err
	}
	providerService.BindLocalRuntimeState(localRuntimeAdapter)

	gatewayHandler := gateway.NewHandler(
		providerService,
		localGatewayExecutor,
		credentialStore,
		logService,
	)

	handler := api.NewRouter(providerService, localRuntimeAdapter, healthService, logService, gatewayHandler)

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

	if len(currentSettings.LocalGatewaySelected) == 0 && len(selectedModels) > 0 {
		items := make([]settings.SelectedModel, 0, len(selectedModels))
		for _, item := range selectedModels {
			modelID := strings.TrimSpace(item.ModelID)
			if modelID == "" {
				continue
			}
			items = append(items, settings.SelectedModel{
				ModelID:  modelID,
				Position: item.Position,
			})
		}

		if _, err := settingsService.UpdateLocalGatewaySelectedModels(ctx, items); err != nil {
			return err
		}

		if _, err := providers.ReplaceSelectedModels(ctx, provider.LocalGatewayProviderID, nil); err != nil {
			return err
		}
	}

	if needsSettingsCleanup {
		if _, err := settingsService.Save(ctx, currentSettings); err != nil {
			return err
		}
	}

	return nil
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
