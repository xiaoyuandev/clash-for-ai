package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
)

type CreateInput struct {
	Name         string            `json:"name"`
	BaseURL      string            `json:"base_url"`
	AuthMode     AuthMode          `json:"auth_mode"`
	ExtraHeaders map[string]string `json:"extra_headers"`
	APIKey       string            `json:"api_key"`
}

type UpdateInput struct {
	Name         string            `json:"name"`
	BaseURL      string            `json:"base_url"`
	AuthMode     AuthMode          `json:"auth_mode"`
	ExtraHeaders map[string]string `json:"extra_headers"`
	APIKey       string            `json:"api_key"`
}

type Service struct {
	repository  Repository
	credentials credential.Store
}

func NewService(repository Repository, credentials credential.Store) *Service {
	return &Service{
		repository:  repository,
		credentials: credentials,
	}
}

func (s *Service) List(ctx context.Context) ([]Provider, error) {
	return s.repository.List(ctx)
}

func (s *Service) GetActive(ctx context.Context) (*Provider, error) {
	return s.repository.GetActive(ctx)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Provider, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	id := fmt.Sprintf("provider-%d", time.Now().UnixNano())
	apiKeyRef, err := s.credentials.Save(ctx, fmt.Sprintf("provider/%s/api-key", id), input.APIKey)
	if err != nil {
		return Provider{}, err
	}

	item := Provider{
		ID:           id,
		Name:         input.Name,
		BaseURL:      input.BaseURL,
		APIKeyRef:    apiKeyRef,
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
			LastHealthStatus: "pending",
		},
		APIKeyMasked: maskAPIKey(input.APIKey),
	}

	item.Status.LastHealthcheckAt = now

	return s.repository.Create(ctx, item)
}

func (s *Service) Activate(ctx context.Context, id string) (*Provider, error) {
	return s.repository.Activate(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Provider, error) {
	item, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return Provider{}, err
	}

	item.Name = strings.TrimSpace(input.Name)
	item.BaseURL = strings.TrimSpace(input.BaseURL)
	item.AuthMode = input.AuthMode
	item.ExtraHeaders = input.ExtraHeaders
	if item.ExtraHeaders == nil {
		item.ExtraHeaders = map[string]string{}
	}

	if strings.TrimSpace(input.APIKey) != "" {
		if err := s.credentials.Delete(ctx, item.APIKeyRef); err != nil {
			return Provider{}, err
		}

		apiKeyRef, err := s.credentials.Save(ctx, fmt.Sprintf("provider/%s/api-key", id), input.APIKey)
		if err != nil {
			return Provider{}, err
		}

		item.APIKeyRef = apiKeyRef
		item.APIKeyMasked = maskAPIKey(input.APIKey)
	}

	return s.repository.Update(ctx, *item)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	item, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.credentials.Delete(ctx, item.APIKeyRef); err != nil {
		return err
	}

	return s.repository.Delete(ctx, id)
}

func maskAPIKey(value string) string {
	if len(value) <= 4 {
		return "****"
	}

	return fmt.Sprintf("****%s", value[len(value)-4:])
}
