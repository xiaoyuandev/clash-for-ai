package localgatewaycontrol

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/gatewayadapter"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type stubProviderReader struct {
	item *provider.Provider
	err  error
}

func (s *stubProviderReader) GetByID(context.Context, string) (*provider.Provider, error) {
	return s.item, s.err
}

type stubLocalRuntimeAdapter struct {
	discover        gatewayadapter.RuntimeInfo
	discoverErr     error
	health          gatewayadapter.RuntimeHealth
	healthErr       error
	capabilities    gatewayadapter.RuntimeCapabilities
	capabilitiesErr error

	modelSources     []modelsource.Source
	modelSourceErr   error
	selectedModels   []provider.SelectedModel
	selectedModelErr error
}

func (s *stubLocalRuntimeAdapter) Start(context.Context) error { return nil }
func (s *stubLocalRuntimeAdapter) Stop(context.Context) error  { return nil }
func (s *stubLocalRuntimeAdapter) Discover(context.Context) (gatewayadapter.RuntimeInfo, error) {
	return s.discover, s.discoverErr
}
func (s *stubLocalRuntimeAdapter) EnsureReady(context.Context) error { return nil }
func (s *stubLocalRuntimeAdapter) CheckHealth(context.Context) (gatewayadapter.RuntimeHealth, error) {
	return s.health, s.healthErr
}
func (s *stubLocalRuntimeAdapter) Capabilities(context.Context) (gatewayadapter.RuntimeCapabilities, error) {
	return s.capabilities, s.capabilitiesErr
}
func (s *stubLocalRuntimeAdapter) ListModelSources(context.Context) ([]modelsource.Source, error) {
	return s.modelSources, s.modelSourceErr
}
func (s *stubLocalRuntimeAdapter) CreateModelSource(context.Context, modelsource.CreateInput) (modelsource.Source, error) {
	return modelsource.Source{}, s.modelSourceErr
}
func (s *stubLocalRuntimeAdapter) UpdateModelSource(context.Context, string, modelsource.UpdateInput) (modelsource.Source, error) {
	return modelsource.Source{}, s.modelSourceErr
}
func (s *stubLocalRuntimeAdapter) DeleteModelSource(context.Context, string) error {
	return s.modelSourceErr
}
func (s *stubLocalRuntimeAdapter) ReplaceModelSourceOrder(context.Context, []modelsource.Source) ([]modelsource.Source, error) {
	return s.modelSources, s.modelSourceErr
}
func (s *stubLocalRuntimeAdapter) ListSelectedModels(context.Context) ([]provider.SelectedModel, error) {
	return s.selectedModels, s.selectedModelErr
}
func (s *stubLocalRuntimeAdapter) ReplaceSelectedModels(context.Context, []provider.SelectedModel) ([]provider.SelectedModel, error) {
	return s.selectedModels, s.selectedModelErr
}

func TestGetRuntimeStatus(t *testing.T) {
	now := time.Now().UTC()
	service := NewService(
		&stubProviderReader{
			item: &provider.Provider{
				ID:      provider.LocalGatewayProviderID,
				BaseURL: "http://127.0.0.1:8788",
			},
		},
		&stubLocalRuntimeAdapter{
			discover: gatewayadapter.RuntimeInfo{
				BaseURL:  "http://127.0.0.1:8788",
				Mode:     "external",
				Embedded: false,
			},
			health: gatewayadapter.RuntimeHealth{
				Status:    "ok",
				Summary:   "HTTP 200",
				CheckedAt: now,
			},
			capabilities: gatewayadapter.RuntimeCapabilities{
				SupportsOpenAICompatible:    true,
				SupportsAnthropicCompatible: true,
				SupportsModelsAPI:           true,
				SupportsStream:              true,
			},
		},
	)

	status, err := service.GetRuntimeStatus(context.Background())
	if err != nil {
		t.Fatalf("GetRuntimeStatus returned error: %v", err)
	}
	if status.ProviderID != provider.LocalGatewayProviderID {
		t.Fatalf("unexpected provider id: %s", status.ProviderID)
	}
	if status.Runtime.Mode != "external" {
		t.Fatalf("unexpected runtime mode: %+v", status.Runtime)
	}
	if status.Health.CheckedAt != now {
		t.Fatalf("unexpected health timestamp: %+v", status.Health)
	}
}

func TestListModelSourcesReturnsUnsupportedError(t *testing.T) {
	service := NewService(
		&stubProviderReader{},
		&stubLocalRuntimeAdapter{modelSourceErr: gatewayadapter.ErrRuntimeAdminUnsupported},
	)

	_, err := service.ListModelSources(context.Background())
	if !errors.Is(err, gatewayadapter.ErrRuntimeAdminUnsupported) {
		t.Fatalf("expected ErrRuntimeAdminUnsupported, got %v", err)
	}
}
