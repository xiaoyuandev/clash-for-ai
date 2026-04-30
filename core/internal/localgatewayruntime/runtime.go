package localgatewayruntime

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/config"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/gateway"
	localgatewayexecutor "github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway/executor"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgatewaystate"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
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
