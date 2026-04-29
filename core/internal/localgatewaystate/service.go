package localgatewaystate

import (
	"context"
	"strings"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) ListSelectedModels(ctx context.Context) ([]SelectedModel, error) {
	items, err := s.repository.ListSelectedModels(ctx)
	if err != nil {
		return nil, err
	}
	return normalizeSelectedModels(items), nil
}

func (s *Service) ReplaceSelectedModels(ctx context.Context, items []SelectedModel) ([]SelectedModel, error) {
	normalized := normalizeSelectedModels(items)
	if err := s.repository.ReplaceSelectedModels(ctx, normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

func normalizeSelectedModels(items []SelectedModel) []SelectedModel {
	normalized := make([]SelectedModel, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		modelID := strings.TrimSpace(item.ModelID)
		if modelID == "" {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}
		seen[modelID] = struct{}{}
		normalized = append(normalized, SelectedModel{
			ModelID:  modelID,
			Position: len(normalized),
		})
	}
	return normalized
}
