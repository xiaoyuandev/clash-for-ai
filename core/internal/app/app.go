package app

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/api"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/config"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/gateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/health"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/logging"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/storage"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/tooling"
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
	localGatewayRepository := localgateway.NewSQLiteRepository(sqliteStore.DB)
	logRepository := logging.NewSQLiteRepository(sqliteStore.DB)
	logService := logging.NewService(logRepository, cfg.LogRetentionDays, cfg.LogMaxRecords)
	providerService := provider.NewService(providerRepository, credentialStore)
	localGatewayService := localgateway.NewService(localGatewayRepository, credentialStore)
	localGatewayAdapter := localgateway.NewAdapter(cfg.LocalGatewayRuntimeKind, nil)
	localGatewayManager := localgateway.NewManager(localGatewayService, localGatewayAdapter, localgateway.RuntimeConfig{
		Executable: cfg.LocalGatewayRuntimeExecutable,
		Host:       cfg.LocalGatewayRuntimeHost,
		Port:       cfg.LocalGatewayRuntimePort,
		DataDir:    cfg.LocalGatewayRuntimeDataDir,
	})
	healthService := health.NewService(providerService, credentialStore)
	toolingService := tooling.NewService(providerService)
	gatewayHandler := gateway.NewHandler(providerService, credentialStore, logService)

	if _, err := providerService.EnsureManagedLocalGateway(
		context.Background(),
		"Local Gateway",
		localGatewayProviderBaseURL(cfg.LocalGatewayRuntimeHost, cfg.LocalGatewayRuntimePort),
		"dummy",
	); err != nil {
		log.Printf("ensure managed local gateway provider: %v", err)
	}

	if err := localGatewayManager.Bootstrap(context.Background()); err != nil {
		log.Printf("bootstrap local gateway runtime: %v", err)
	}

	handler := api.NewRouter(
		providerService,
		healthService,
		logService,
		localGatewayManager,
		toolingService,
		cfg.HTTPPort,
		gatewayHandler,
	)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.GatewayBind, cfg.HTTPPort),
		Handler: handler,
		BaseContext: func(net.Listener) context.Context {
			return context.Background()
		},
	}

	return server.ListenAndServe()
}

func localGatewayProviderBaseURL(host string, port int) string {
	clientHost := host
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		clientHost = "127.0.0.1"
	}

	return fmt.Sprintf("http://%s:%d/v1", clientHost, port)
}
