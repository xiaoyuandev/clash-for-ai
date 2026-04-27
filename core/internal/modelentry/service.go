package modelentry

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context) ([]Entry, error) {
	return s.repository.List(ctx)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Entry, error) {
	entry := normalizeEntry(Entry{
		ID:           fmt.Sprintf("gateway-model-%d", time.Now().UnixNano()),
		Name:         input.Name,
		ModelID:      input.ModelID,
		BaseURL:      input.BaseURL,
		APIKey:       input.APIKey,
		ProviderType: input.ProviderType,
		Protocol:     input.Protocol,
		Enabled:      input.Enabled,
	})

	items, err := s.repository.List(ctx)
	if err != nil {
		return Entry{}, err
	}
	entry.Position = len(items)

	return s.repository.Create(ctx, entry)
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Entry, error) {
	current, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return Entry{}, err
	}

	entry := normalizeEntry(Entry{
		ID:           current.ID,
		Name:         input.Name,
		ModelID:      input.ModelID,
		BaseURL:      input.BaseURL,
		APIKey:       input.APIKey,
		ProviderType: input.ProviderType,
		Protocol:     input.Protocol,
		Enabled:      input.Enabled,
		Position:     current.Position,
	})

	return s.repository.Update(ctx, entry)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repository.Delete(ctx, id)
}

func (s *Service) ReplaceOrder(ctx context.Context, items []Entry) ([]Entry, error) {
	currentItems, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	currentByID := make(map[string]Entry, len(currentItems))
	for _, item := range currentItems {
		currentByID[item.ID] = item
	}

	normalized := make([]Entry, 0, len(items))
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

func normalizeEntry(entry Entry) Entry {
	entry.Name = strings.TrimSpace(entry.Name)
	entry.ModelID = strings.TrimSpace(entry.ModelID)
	entry.BaseURL = strings.TrimSpace(entry.BaseURL)
	entry.APIKey = strings.TrimSpace(entry.APIKey)
	entry.ProviderType = strings.TrimSpace(entry.ProviderType)
	entry.Protocol = strings.TrimSpace(entry.Protocol)
	return entry
}
