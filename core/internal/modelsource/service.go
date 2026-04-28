package modelsource

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
)

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

func (s *Service) List(ctx context.Context) ([]Source, error) {
	items, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	for index := range items {
		items[index] = s.refreshMaskedKey(ctx, items[index])
	}

	return items, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Source, error) {
	id := fmt.Sprintf("model-source-%d", time.Now().UnixNano())
	apiKeyRef, err := s.credentials.Save(ctx, fmt.Sprintf("model-source/%s/api-key", id), strings.TrimSpace(input.APIKey))
	if err != nil {
		return Source{}, err
	}

	items, err := s.repository.List(ctx)
	if err != nil {
		return Source{}, err
	}

	source := normalizeSource(Source{
		ID:             id,
		Name:           input.Name,
		BaseURL:        input.BaseURL,
		ProviderType:   input.ProviderType,
		DefaultModelID: input.DefaultModelID,
		Enabled:        input.Enabled,
		Position:       len(items),
		APIKeyRef:      apiKeyRef,
		APIKey:         input.APIKey,
		APIKeyMasked:   maskAPIKey(input.APIKey),
	})

	return s.repository.Create(ctx, source)
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Source, error) {
	current, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return Source{}, err
	}

	current.Name = strings.TrimSpace(input.Name)
	current.BaseURL = strings.TrimSpace(input.BaseURL)
	current.ProviderType = strings.TrimSpace(input.ProviderType)
	current.DefaultModelID = strings.TrimSpace(input.DefaultModelID)
	current.Enabled = input.Enabled

	if strings.TrimSpace(input.APIKey) != "" {
		if err := s.credentials.Delete(ctx, current.APIKeyRef); err != nil {
			return Source{}, err
		}
		apiKeyRef, err := s.credentials.Save(ctx, fmt.Sprintf("model-source/%s/api-key", id), strings.TrimSpace(input.APIKey))
		if err != nil {
			return Source{}, err
		}
		current.APIKeyRef = apiKeyRef
		current.APIKey = input.APIKey
		current.APIKeyMasked = maskAPIKey(input.APIKey)
	}

	return s.repository.Update(ctx, normalizeSource(*current))
}

func (s *Service) Delete(ctx context.Context, id string) error {
	current, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.credentials.Delete(ctx, current.APIKeyRef); err != nil {
		return err
	}

	return s.repository.Delete(ctx, id)
}

func (s *Service) ReplaceOrder(ctx context.Context, items []Source) ([]Source, error) {
	currentItems, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	currentByID := make(map[string]Source, len(currentItems))
	for _, item := range currentItems {
		currentByID[item.ID] = item
	}

	normalized := make([]Source, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		current, ok := currentByID[id]
		if !ok {
			continue
		}
		current.Position = len(normalized)
		normalized = append(normalized, current)
		seen[id] = struct{}{}
	}

	if err := s.repository.ReplaceOrder(ctx, normalized); err != nil {
		return nil, err
	}

	return normalized, nil
}

func normalizeSource(source Source) Source {
	source.Name = strings.TrimSpace(source.Name)
	source.BaseURL = strings.TrimSpace(source.BaseURL)
	source.ProviderType = strings.TrimSpace(source.ProviderType)
	source.DefaultModelID = strings.TrimSpace(source.DefaultModelID)
	return source
}

func maskAPIKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 4 {
		return "****"
	}

	if len(trimmed) <= 12 {
		return fmt.Sprintf("%s****", trimmed[:len(trimmed)-4])
	}

	return fmt.Sprintf("%s••••%s", trimmed[:8], trimmed[len(trimmed)-4:])
}

func (s *Service) refreshMaskedKey(ctx context.Context, item Source) Source {
	if strings.TrimSpace(item.APIKeyRef) == "" {
		return item
	}

	value, err := s.credentials.Get(ctx, item.APIKeyRef)
	if err != nil {
		item.APIKey = ""
		return item
	}

	item.APIKey = value
	item.APIKeyMasked = maskAPIKey(value)
	return item
}
