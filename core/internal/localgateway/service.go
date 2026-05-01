package localgateway

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
)

type CreateModelSourceInput struct {
	Name            string   `json:"name"`
	BaseURL         string   `json:"base_url"`
	APIKey          string   `json:"api_key"`
	ProviderType    string   `json:"provider_type"`
	DefaultModelID  string   `json:"default_model_id"`
	ExposedModelIDs []string `json:"exposed_model_ids"`
	Enabled         bool     `json:"enabled"`
	Position        int      `json:"position"`
}

type UpdateModelSourceInput struct {
	Name            string   `json:"name"`
	BaseURL         string   `json:"base_url"`
	APIKey          string   `json:"api_key"`
	ProviderType    string   `json:"provider_type"`
	DefaultModelID  string   `json:"default_model_id"`
	ExposedModelIDs []string `json:"exposed_model_ids"`
	Enabled         bool     `json:"enabled"`
	Position        int      `json:"position"`
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

func (s *Service) ListSources(ctx context.Context) ([]ModelSource, error) {
	items, err := s.repository.ListSources(ctx)
	if err != nil {
		return nil, err
	}

	for index := range items {
		items[index] = s.refreshMaskedKey(ctx, items[index])
	}

	return items, nil
}

func (s *Service) GetSourceByID(ctx context.Context, id string) (*ModelSource, error) {
	item, err := s.repository.GetSourceByID(ctx, id)
	if err != nil || item == nil {
		return item, err
	}

	refreshed := s.refreshMaskedKey(ctx, *item)
	return &refreshed, nil
}

func (s *Service) CreateSource(ctx context.Context, input CreateModelSourceInput) (ModelSource, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	id := fmt.Sprintf("local-source-%d", time.Now().UnixNano())
	apiKeyRef, err := s.credentials.Save(ctx, fmt.Sprintf("local-gateway/%s/api-key", id), input.APIKey)
	if err != nil {
		return ModelSource{}, err
	}

	item := ModelSource{
		ID:              id,
		Name:            strings.TrimSpace(input.Name),
		BaseURL:         strings.TrimSpace(input.BaseURL),
		APIKeyRef:       apiKeyRef,
		APIKey:          input.APIKey,
		ProviderType:    strings.TrimSpace(input.ProviderType),
		DefaultModelID:  strings.TrimSpace(input.DefaultModelID),
		ExposedModelIDs: normalizeModelIDs(input.ExposedModelIDs),
		Enabled:         input.Enabled,
		Position:        input.Position,
		APIKeyMasked:    maskAPIKey(input.APIKey),
		LastSyncStatus:  "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	return s.repository.CreateSource(ctx, item)
}

func (s *Service) UpdateSource(ctx context.Context, id string, input UpdateModelSourceInput) (ModelSource, error) {
	item, err := s.repository.GetSourceByID(ctx, id)
	if err != nil {
		return ModelSource{}, err
	}

	item.Name = strings.TrimSpace(input.Name)
	item.BaseURL = strings.TrimSpace(input.BaseURL)
	item.ProviderType = strings.TrimSpace(input.ProviderType)
	item.DefaultModelID = strings.TrimSpace(input.DefaultModelID)
	item.ExposedModelIDs = normalizeModelIDs(input.ExposedModelIDs)
	item.Enabled = input.Enabled
	item.Position = input.Position
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if strings.TrimSpace(input.APIKey) != "" {
		if err := s.credentials.Delete(ctx, item.APIKeyRef); err != nil {
			return ModelSource{}, err
		}

		apiKeyRef, err := s.credentials.Save(ctx, fmt.Sprintf("local-gateway/%s/api-key", id), input.APIKey)
		if err != nil {
			return ModelSource{}, err
		}

		item.APIKeyRef = apiKeyRef
		item.APIKey = input.APIKey
		item.APIKeyMasked = maskAPIKey(input.APIKey)
	}

	return s.repository.UpdateSource(ctx, *item)
}

func (s *Service) DeleteSource(ctx context.Context, id string) error {
	item, err := s.repository.GetSourceByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.credentials.Delete(ctx, item.APIKeyRef); err != nil {
		return err
	}

	return s.repository.DeleteSource(ctx, id)
}

func (s *Service) ListSelectedModels(ctx context.Context) ([]SelectedModel, error) {
	return s.repository.ListSelectedModels(ctx)
}

func (s *Service) BuildSyncInput(ctx context.Context) (SyncInput, error) {
	sources, err := s.repository.ListSources(ctx)
	if err != nil {
		return SyncInput{}, err
	}

	resolvedSources := make([]SyncModelSource, 0, len(sources))
	for _, source := range sources {
		apiKey, err := s.credentials.Get(ctx, source.APIKeyRef)
		if err != nil {
			return SyncInput{}, err
		}

		resolvedSources = append(resolvedSources, SyncModelSource{
			ID:              source.ID,
			Name:            source.Name,
			BaseURL:         source.BaseURL,
			APIKey:          apiKey,
			ProviderType:    source.ProviderType,
			DefaultModelID:  source.DefaultModelID,
			ExposedModelIDs: append([]string(nil), source.ExposedModelIDs...),
			Enabled:         source.Enabled,
			Position:        source.Position,
		})
	}

	selectedModels, err := s.repository.ListSelectedModels(ctx)
	if err != nil {
		return SyncInput{}, err
	}

	return SyncInput{
		Sources:        resolvedSources,
		SelectedModels: selectedModels,
	}, nil
}

func (s *Service) ReplaceSelectedModels(ctx context.Context, items []SelectedModel) ([]SelectedModel, error) {
	normalized := make([]SelectedModel, 0, len(items))
	for index, item := range items {
		modelID := strings.TrimSpace(item.ModelID)
		if modelID == "" {
			continue
		}
		normalized = append(normalized, SelectedModel{
			ModelID:  modelID,
			Position: index,
		})
	}

	if err := s.repository.ReplaceSelectedModels(ctx, normalized); err != nil {
		return nil, err
	}

	return s.repository.ListSelectedModels(ctx)
}

func (s *Service) refreshMaskedKey(ctx context.Context, item ModelSource) ModelSource {
	if item.APIKeyMasked != "" {
		return item
	}

	if item.APIKeyRef == "" {
		return item
	}

	value, err := s.credentials.Get(ctx, item.APIKeyRef)
	if err != nil {
		return item
	}

	item.APIKeyMasked = maskAPIKey(value)
	return item
}

func normalizeModelIDs(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}

	normalized := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	return normalized
}

func maskAPIKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 4 {
		return "****"
	}
	if len(trimmed) <= 12 {
		return trimmed[:len(trimmed)-4] + "****"
	}

	return trimmed[:8] + "****" + trimmed[len(trimmed)-4:]
}
