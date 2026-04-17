package app

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/api"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/config"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/gateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

func Run() error {
	cfg := config.Load()

	providerRepository := provider.NewInMemoryRepository()
	credentialStore := credential.NewInMemoryStore()
	providerService := provider.NewService(providerRepository, credentialStore)
	gatewayHandler := gateway.NewHandler(providerService, credentialStore)

	handler := api.NewRouter(providerService, gatewayHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.GatewayBind, cfg.HTTPPort),
		Handler: handler,
		BaseContext: func(net.Listener) context.Context {
			return context.Background()
		},
	}

	return server.ListenAndServe()
}
