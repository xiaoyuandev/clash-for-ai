package provider

import (
	"context"
	"fmt"
	"time"
)

type CreateInput struct {
	Name         string            `json:"name"`
	BaseURL      string            `json:"base_url"`
	AuthMode     AuthMode          `json:"auth_mode"`
	ExtraHeaders map[string]string `json:"extra_headers"`
	APIKey       string            `json:"api_key"`
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context) ([]Provider, error) {
	return s.repository.List(ctx)
}

func (s *Service) GetActive(ctx context.Context) (*Provider, error) {
	return s.repository.GetActive(ctx)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Provider, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	item := Provider{
		ID:           fmt.Sprintf("provider-%d", time.Now().UnixNano()),
		Name:         input.Name,
		BaseURL:      input.BaseURL,
		APIKeyRef:    "memory://pending",
		AuthMode:     input.AuthMode,
		ExtraHeaders: input.ExtraHeaders,
		Capabilities: Capabilities{
			SupportsOpenAICompatible:    true,
			SupportsAnthropicCompatible: true,
			SupportsModelsAPI:           true,
			SupportsBalanceAPI:          false,
			SupportsStream:              true,
		},
		Status: Status{
			IsActive:         false,
			LastHealthStatus: "unknown",
		},
		APIKeyMasked: maskAPIKey(input.APIKey),
	}

	item.Status.LastHealthcheckAt = now

	return s.repository.Create(ctx, item)
}

func (s *Service) Activate(ctx context.Context, id string) (*Provider, error) {
	return s.repository.Activate(ctx, id)
}

func maskAPIKey(value string) string {
	if len(value) <= 4 {
		return "****"
	}

	return fmt.Sprintf("****%s", value[len(value)-4:])
}
