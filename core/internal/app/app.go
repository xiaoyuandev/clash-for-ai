package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/api"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/config"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/gateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/gatewayadapter"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/health"
	localgatewayexecutor "github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway/executor"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgatewaycontrol"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgatewaystate"
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
	settingsRepository := settings.NewSQLiteRepository(sqliteStore.DB)
	logRepository := logging.NewSQLiteRepository(sqliteStore.DB)
	logService := logging.NewService(logRepository, cfg.LogRetentionDays, cfg.LogMaxRecords)
	providerService := provider.NewService(providerRepository, credentialStore)
	settingsService := settings.NewService(settingsRepository)
	localGatewayExecutor := localgatewayexecutor.New(nil)
	modelSourceRepository := modelsource.NewSQLiteRepository(sqliteStore.DB)
	localGatewayStateRepository := localgatewaystate.NewSQLiteRepository(sqliteStore.DB)
	modelSourceService := modelsource.NewService(modelSourceRepository, credentialStore)
	localGatewayStateService := localgatewaystate.NewService(localGatewayStateRepository)
	currentSettings, err := settingsService.Get(context.Background())
	if err != nil {
		return err
	}

	runtimeDataDir := resolveLocalRuntimeDataDir(cfg.DataDir)
	if err := prepareLocalRuntimeState(
		context.Background(),
		runtimeDataDir,
		modelSourceService,
		localGatewayStateService,
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
		localGatewayExecutor,
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

func RunLocalGatewayRuntime() error {
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

	modelSourceRepository := modelsource.NewSQLiteRepository(sqliteStore.DB)
	localGatewayStateRepository := localgatewaystate.NewSQLiteRepository(sqliteStore.DB)
	settingsRepository := settings.NewSQLiteRepository(sqliteStore.DB)
	modelSourceService := modelsource.NewService(modelSourceRepository, credentialStore)
	localGatewayStateService := localgatewaystate.NewService(localGatewayStateRepository)
	settingsService := settings.NewService(settingsRepository)
	localGatewayExecutor := localgatewayexecutor.New(nil)

	currentSettings, err := settingsService.Get(context.Background())
	if err != nil {
		return err
	}
	localSettings := resolveRuntimeLocalGatewaySettings(currentSettings.LocalGateway)

	handler := gateway.NewLocalRuntimeHandler(
		modelSourceService,
		localGatewayStateService,
		localGatewayExecutor,
	)

	server := &http.Server{
		Addr:    localGatewayListenAddr(localSettings),
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

	executablePath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	return gatewayadapter.NewEmbeddedLocalRuntimeAdapter(localSettings, executablePath, runtimeDataDir), nil
}

func resolveRuntimeLocalGatewaySettings(input settings.LocalGatewaySettings) settings.LocalGatewaySettings {
	settingsValue := input

	if value := strings.TrimSpace(os.Getenv("LOCAL_GATEWAY_RUNTIME_HOST")); value != "" {
		settingsValue.ListenHost = value
	}

	if value := strings.TrimSpace(os.Getenv("LOCAL_GATEWAY_RUNTIME_PORT")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			settingsValue.ListenPort = parsed
		}
	}

	if strings.TrimSpace(settingsValue.ListenHost) == "" {
		settingsValue.ListenHost = "127.0.0.1"
	}
	if settingsValue.ListenPort <= 0 {
		settingsValue.ListenPort = 8788
	}

	return settingsValue
}

func localGatewayListenAddr(input settings.LocalGatewaySettings) string {
	return fmt.Sprintf("%s:%d", strings.TrimSpace(input.ListenHost), input.ListenPort)
}

func resolveLocalRuntimeDataDir(coreDataDir string) string {
	return filepath.Join(coreDataDir, "local-gateway-runtime")
}

func prepareLocalRuntimeState(
	ctx context.Context,
	runtimeDataDir string,
	coreModelSources *modelsource.Service,
	coreLocalGatewayState *localgatewaystate.Service,
	coreSettings *settings.Service,
	providers *provider.Service,
	currentSettings settings.AppSettings,
) error {
	runtimeStore, err := storage.NewSQLite(filepath.Join(runtimeDataDir, "clash-for-ai.db"))
	if err != nil {
		return err
	}
	defer runtimeStore.Close()

	runtimeCredentialStore, err := credential.NewFileStore(filepath.Join(runtimeDataDir, "credentials.json"))
	if err != nil {
		return err
	}

	runtimeModelSourceRepository := modelsource.NewSQLiteRepository(runtimeStore.DB)
	runtimeLocalGatewayStateRepository := localgatewaystate.NewSQLiteRepository(runtimeStore.DB)
	runtimeModelSources := modelsource.NewService(runtimeModelSourceRepository, runtimeCredentialStore)
	runtimeLocalGatewayState := localgatewaystate.NewService(runtimeLocalGatewayStateRepository)

	runtimeSources, err := runtimeModelSources.List(ctx)
	if err != nil {
		return err
	}
	migratedSources := false
	if len(runtimeSources) == 0 {
		coreSources, listErr := coreModelSources.List(ctx)
		if listErr != nil {
			return listErr
		}
		for _, item := range coreSources {
			if _, createErr := runtimeModelSources.Create(ctx, modelsource.CreateInput{
				Name:           item.Name,
				BaseURL:        item.BaseURL,
				ProviderType:   item.ProviderType,
				DefaultModelID: item.DefaultModelID,
				Enabled:        item.Enabled,
				APIKey:         item.APIKey,
			}); createErr != nil {
				return createErr
			}
		}
		migratedSources = len(coreSources) > 0
	}

	runtimeSelected, err := runtimeLocalGatewayState.ListSelectedModels(ctx)
	if err != nil {
		return err
	}
	migratedSelected := false
	if len(runtimeSelected) == 0 {
		legacySelected := make([]localgatewaystate.SelectedModel, 0, len(currentSettings.LocalGatewaySelected))
		for _, item := range currentSettings.LocalGatewaySelected {
			legacySelected = append(legacySelected, localgatewaystate.SelectedModel{
				ModelID:  item.ModelID,
				Position: item.Position,
			})
		}
		if len(legacySelected) == 0 {
			providerSelected, listErr := providers.ListSelectedModels(ctx, provider.LocalGatewayProviderID)
			if listErr != nil {
				return listErr
			}
			legacySelected = make([]localgatewaystate.SelectedModel, 0, len(providerSelected))
			for _, item := range providerSelected {
				legacySelected = append(legacySelected, localgatewaystate.SelectedModel{
					ModelID:  item.ModelID,
					Position: item.Position,
				})
			}
		}

		if len(legacySelected) > 0 {
			if _, updateErr := runtimeLocalGatewayState.ReplaceSelectedModels(ctx, legacySelected); updateErr != nil {
				return updateErr
			}
			migratedSelected = true
		}
	}

	if migratedSources {
		coreSources, listErr := coreModelSources.List(ctx)
		if listErr != nil {
			return listErr
		}
		for _, item := range coreSources {
			if deleteErr := coreModelSources.Delete(ctx, item.ID); deleteErr != nil {
				return deleteErr
			}
		}
	}

	if migratedSelected && len(currentSettings.LocalGatewaySelected) > 0 {
		currentSettings.LocalGatewaySelected = []settings.SelectedModel{}
		if _, saveErr := coreSettings.Save(ctx, currentSettings); saveErr != nil {
			return saveErr
		}
	}

	if migratedSelected {
		if _, replaceErr := coreLocalGatewayState.ReplaceSelectedModels(ctx, nil); replaceErr != nil {
			return replaceErr
		}
	}

	return nil
}
