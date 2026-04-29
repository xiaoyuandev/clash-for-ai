package localgatewaycontrol

import (
	"context"
	"errors"
	"fmt"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/gatewayadapter"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type ProviderReader interface {
	GetByID(ctx context.Context, id string) (*provider.Provider, error)
}

type RuntimeStatus struct {
	ProviderID   string                             `json:"provider_id"`
	BaseURL      string                             `json:"base_url"`
	Runtime      gatewayadapter.RuntimeInfo         `json:"runtime"`
	Health       gatewayadapter.RuntimeHealth       `json:"health"`
	Capabilities gatewayadapter.RuntimeCapabilities `json:"capabilities"`
	Missing      []string                           `json:"missing_optional_capabilities"`
}

type Service struct {
	providers ProviderReader
	runtime   gatewayadapter.LocalRuntimeAdapter
}

func NewService(providers ProviderReader, runtime gatewayadapter.LocalRuntimeAdapter) *Service {
	return &Service{
		providers: providers,
		runtime:   runtime,
	}
}

func (s *Service) GetRuntimeStatus(ctx context.Context) (RuntimeStatus, error) {
	item, err := s.providers.GetByID(ctx, provider.LocalGatewayProviderID)
	if err != nil {
		return RuntimeStatus{}, err
	}

	info, err := s.runtime.Discover(ctx)
	if err != nil {
		return RuntimeStatus{}, err
	}
	health, err := s.runtime.CheckHealth(ctx)
	if err != nil {
		return RuntimeStatus{}, err
	}
	capabilities, err := s.runtime.Capabilities(ctx)
	if err != nil {
		return RuntimeStatus{}, err
	}

	return RuntimeStatus{
		ProviderID:   item.ID,
		BaseURL:      item.BaseURL,
		Runtime:      info,
		Health:       health,
		Capabilities: capabilities,
		Missing:      capabilities.MissingOptionalCapabilities(),
	}, nil
}

func (s *Service) ListModelSources(ctx context.Context) ([]modelsource.Source, error) {
	items, err := s.runtime.ListModelSources(ctx)
	if err != nil {
		return nil, normalizeAdminError(err)
	}
	return items, nil
}

func (s *Service) CreateModelSource(ctx context.Context, input modelsource.CreateInput) (modelsource.Source, error) {
	item, err := s.runtime.CreateModelSource(ctx, input)
	if err != nil {
		return modelsource.Source{}, normalizeAdminError(err)
	}
	return item, nil
}

func (s *Service) UpdateModelSource(ctx context.Context, id string, input modelsource.UpdateInput) (modelsource.Source, error) {
	item, err := s.runtime.UpdateModelSource(ctx, id, input)
	if err != nil {
		return modelsource.Source{}, normalizeAdminError(err)
	}
	return item, nil
}

func (s *Service) DeleteModelSource(ctx context.Context, id string) error {
	return normalizeAdminError(s.runtime.DeleteModelSource(ctx, id))
}

func (s *Service) ReplaceModelSourceOrder(ctx context.Context, items []modelsource.Source) ([]modelsource.Source, error) {
	updated, err := s.runtime.ReplaceModelSourceOrder(ctx, items)
	if err != nil {
		return nil, normalizeAdminError(err)
	}
	return updated, nil
}

func (s *Service) ListSelectedModels(ctx context.Context) ([]provider.SelectedModel, error) {
	items, err := s.runtime.ListSelectedModels(ctx)
	if err != nil {
		return nil, normalizeAdminError(err)
	}
	return items, nil
}

func (s *Service) ReplaceSelectedModels(ctx context.Context, items []provider.SelectedModel) ([]provider.SelectedModel, error) {
	updated, err := s.runtime.ReplaceSelectedModels(ctx, items)
	if err != nil {
		return nil, normalizeAdminError(err)
	}
	return updated, nil
}

func normalizeAdminError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gatewayadapter.ErrRuntimeAdminUnsupported) {
		return err
	}
	return fmt.Errorf("local gateway runtime admin request failed: %w", err)
}
