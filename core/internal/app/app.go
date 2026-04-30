package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/api"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/config"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/gateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/gatewayadapter"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/health"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgatewaycontrol"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgatewayruntime"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/logging"
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
	settingsRepository := settings.NewSQLiteRepository(sqliteStore.DB)
	logRepository := logging.NewSQLiteRepository(sqliteStore.DB)
	logService := logging.NewService(logRepository, cfg.LogRetentionDays, cfg.LogMaxRecords)
	providerService := provider.NewService(providerRepository, credentialStore)
	settingsService := settings.NewService(settingsRepository)
	currentSettings, err := settingsService.Get(context.Background())
	if err != nil {
		return err
	}

	runtimeDataDir := localgatewayruntime.RuntimeDataDir(cfg.DataDir)
	if err := localgatewayruntime.PrepareStateIfNeeded(
		context.Background(),
		runtimeDataDir,
		sqliteStore.DB,
		credentialStore,
		settingsService,
		providerService,
		currentSettings,
	); err != nil {
		return err
	}

	localRuntimeAdapter, err := resolveLocalRuntimeAdapter(runtimeDataDir, currentSettings.LocalGateway)
	if err != nil {
		return err
	}
	if currentSettings.LocalGateway.Enabled {
		if err := localRuntimeAdapter.Start(context.Background()); err != nil {
			return err
		}
	}
	runtimeInfo, err := localRuntimeAdapter.Discover(context.Background())
	if err != nil {
		return err
	}
	runtimeCapabilities, err := localRuntimeAdapter.Capabilities(context.Background())
	if err != nil {
		return err
	}
	if missing := runtimeCapabilities.MissingRequiredCapabilities(); len(missing) > 0 {
		return fmt.Errorf("local gateway runtime missing required capabilities: %s", strings.Join(missing, ", "))
	}
	healthService := health.NewService(providerService, credentialStore, localRuntimeAdapter)

	if err := initializeLocalGatewayProvider(
		context.Background(),
		providerService,
		settingsService,
		currentSettings,
		runtimeInfo.BaseURL,
		buildLocalGatewayProviderCapabilities(runtimeCapabilities),
	); err != nil {
		return err
	}
	providerService.BindLocalRuntimeAdapter(localRuntimeAdapter)

	gatewayHandler := gateway.NewHandler(
		providerService,
		credentialStore,
		localRuntimeAdapter,
		logService,
	)
	localGatewayAdmin := localgatewaycontrol.NewService(providerService, localRuntimeAdapter)

	handler := api.NewRouter(providerService, healthService, logService, gatewayHandler, localGatewayAdmin)

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
	capabilities provider.Capabilities,
) error {
	localProvider, err := providers.EnsureSystemLocalGateway(ctx, baseURL, capabilities)
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

func buildLocalGatewayProviderCapabilities(input gatewayadapter.RuntimeCapabilities) provider.Capabilities {
	return provider.Capabilities{
		SupportsOpenAICompatible:    input.SupportsOpenAICompatible,
		SupportsAnthropicCompatible: input.SupportsAnthropicCompatible,
		SupportsModelsAPI:           input.SupportsModelsAPI,
		SupportsBalanceAPI:          false,
		SupportsStream:              input.SupportsStream,
	}
}

func resolveLocalRuntimeAdapter(
	runtimeDataDir string,
	localSettings settings.LocalGatewaySettings,
) (gatewayadapter.LocalRuntimeAdapter, error) {
	if externalBaseURL := strings.TrimSpace(os.Getenv("LOCAL_GATEWAY_RUNTIME_BASE_URL")); externalBaseURL != "" {
		return gatewayadapter.NewExternalRuntimeAdapter(externalBaseURL), nil
	}

	currentExecutablePath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	executablePath := resolveEmbeddedRuntimeExecutablePath(currentExecutablePath)
	return gatewayadapter.NewEmbeddedLocalRuntimeAdapter(
		localSettings,
		executablePath,
		runtimeDataDir,
		executablePath == currentExecutablePath,
	), nil
}

func resolveEmbeddedRuntimeExecutablePath(currentExecutablePath string) string {
	if runtimeExecutable := strings.TrimSpace(os.Getenv("LOCAL_GATEWAY_RUNTIME_EXECUTABLE")); runtimeExecutable != "" {
		return runtimeExecutable
	}

	return currentExecutablePath
}
