package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/api"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/config"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/gateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/health"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/logging"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelentry"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/runtime"
	portkeyruntime "github.com/xiaoyuandev/clash-for-ai/core/internal/runtime/portkey"
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
	runtimeRepository := runtime.NewSQLiteRepository(sqliteStore.DB)
	modelEntryRepository := modelentry.NewSQLiteRepository(sqliteStore.DB)
	logRepository := logging.NewSQLiteRepository(sqliteStore.DB)
	logService := logging.NewService(logRepository, cfg.LogRetentionDays, cfg.LogMaxRecords)
	providerService := provider.NewService(providerRepository, credentialStore)
	runtimeService := runtime.NewService(runtimeRepository)
	modelEntryService := modelentry.NewService(modelEntryRepository)
	portkeyConfigBuilder := portkeyruntime.NewConfigBuilder(runtimeService, modelEntryService)
	healthService := health.NewService(providerService, credentialStore)
	gatewayHandler := gateway.NewHandler(providerService, runtimeService, credentialStore, logService)

	handler := api.NewRouter(providerService, runtimeService, modelEntryService, portkeyConfigBuilder, healthService, logService, gatewayHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.GatewayBind, cfg.HTTPPort),
		Handler: handler,
		BaseContext: func(net.Listener) context.Context {
			return context.Background()
		},
	}

	return server.ListenAndServe()
}
