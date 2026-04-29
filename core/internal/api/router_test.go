package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/gatewayadapter"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/health"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgatewaycontrol"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type routerRuntimeAdapterStub struct{}

func (s *routerRuntimeAdapterStub) Start(context.Context) error { return nil }
func (s *routerRuntimeAdapterStub) Stop(context.Context) error  { return nil }
func (s *routerRuntimeAdapterStub) Discover(context.Context) (gatewayadapter.RuntimeInfo, error) {
	return gatewayadapter.RuntimeInfo{
		BaseURL:  "http://127.0.0.1:8788",
		Mode:     "external",
		Embedded: false,
	}, nil
}
func (s *routerRuntimeAdapterStub) EnsureReady(context.Context) error { return nil }
func (s *routerRuntimeAdapterStub) CheckHealth(context.Context) (gatewayadapter.RuntimeHealth, error) {
	return gatewayadapter.RuntimeHealth{Status: "ok", Summary: "HTTP 200"}, nil
}
func (s *routerRuntimeAdapterStub) Capabilities(context.Context) (gatewayadapter.RuntimeCapabilities, error) {
	return gatewayadapter.RuntimeCapabilities{
		SupportsOpenAICompatible:    true,
		SupportsAnthropicCompatible: true,
		SupportsModelsAPI:           true,
		SupportsStream:              true,
	}, nil
}
func (s *routerRuntimeAdapterStub) ListModelSources(context.Context) ([]modelsource.Source, error) {
	return nil, gatewayadapter.ErrRuntimeAdminUnsupported
}
func (s *routerRuntimeAdapterStub) CreateModelSource(context.Context, modelsource.CreateInput) (modelsource.Source, error) {
	return modelsource.Source{}, gatewayadapter.ErrRuntimeAdminUnsupported
}
func (s *routerRuntimeAdapterStub) UpdateModelSource(context.Context, string, modelsource.UpdateInput) (modelsource.Source, error) {
	return modelsource.Source{}, gatewayadapter.ErrRuntimeAdminUnsupported
}
func (s *routerRuntimeAdapterStub) DeleteModelSource(context.Context, string) error {
	return gatewayadapter.ErrRuntimeAdminUnsupported
}
func (s *routerRuntimeAdapterStub) ReplaceModelSourceOrder(context.Context, []modelsource.Source) ([]modelsource.Source, error) {
	return nil, gatewayadapter.ErrRuntimeAdminUnsupported
}
func (s *routerRuntimeAdapterStub) ListSelectedModels(context.Context) ([]provider.SelectedModel, error) {
	return nil, gatewayadapter.ErrRuntimeAdminUnsupported
}
func (s *routerRuntimeAdapterStub) ReplaceSelectedModels(context.Context, []provider.SelectedModel) ([]provider.SelectedModel, error) {
	return nil, gatewayadapter.ErrRuntimeAdminUnsupported
}

func TestLocalGatewayRuntimeEndpoint(t *testing.T) {
	providers := provider.NewService(provider.NewInMemoryRepository(), nil)
	if _, err := providers.EnsureSystemLocalGateway(context.Background(), "http://127.0.0.1:8788", provider.DefaultLocalGatewayCapabilities()); err != nil {
		t.Fatalf("EnsureSystemLocalGateway returned error: %v", err)
	}

	localGatewayAdmin := localgatewaycontrol.NewService(providers, &routerRuntimeAdapterStub{})
	handler := NewRouter(providers, health.NewService(providers, nil, &routerRuntimeAdapterStub{}), nil, http.NotFoundHandler(), localGatewayAdmin)

	req := httptest.NewRequest(http.MethodGet, "/api/local-gateway/runtime", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestLocalGatewaySelectedModelsUnsupportedReturnsNotImplemented(t *testing.T) {
	providers := provider.NewService(provider.NewInMemoryRepository(), nil)
	if _, err := providers.EnsureSystemLocalGateway(context.Background(), "http://127.0.0.1:8788", provider.DefaultLocalGatewayCapabilities()); err != nil {
		t.Fatalf("EnsureSystemLocalGateway returned error: %v", err)
	}

	localGatewayAdmin := localgatewaycontrol.NewService(providers, &routerRuntimeAdapterStub{})
	handler := NewRouter(providers, health.NewService(providers, nil, &routerRuntimeAdapterStub{}), nil, http.NotFoundHandler(), localGatewayAdmin)

	req := httptest.NewRequest(http.MethodGet, "/api/local-gateway/selected-models", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("unexpected status code: %d body=%s", recorder.Code, recorder.Body.String())
	}
}
