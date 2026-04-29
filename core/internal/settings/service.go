package settings

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

func DefaultSettings() AppSettings {
	return AppSettings{
		LocalGateway: LocalGatewaySettings{
			Enabled:    true,
			ListenHost: "127.0.0.1",
			ListenPort: 8788,
		},
		LocalGatewaySelected: []SelectedModel{},
		LocalGatewayClaude:   ClaudeCodeModelMap{},
	}
}

func NormalizeSettings(input AppSettings) AppSettings {
	settings := input
	defaults := DefaultSettings()

	if !settings.LocalGateway.Enabled &&
		strings.TrimSpace(settings.LocalGateway.ListenHost) == "" &&
		settings.LocalGateway.ListenPort == 0 {
		settings.LocalGateway.Enabled = defaults.LocalGateway.Enabled
	}
	if strings.TrimSpace(settings.LocalGateway.ListenHost) == "" {
		settings.LocalGateway.ListenHost = defaults.LocalGateway.ListenHost
	}
	if settings.LocalGateway.ListenPort <= 0 {
		settings.LocalGateway.ListenPort = defaults.LocalGateway.ListenPort
	}

	if settings.LocalGatewaySelected == nil {
		settings.LocalGatewaySelected = defaults.LocalGatewaySelected
	}

	return settings
}

func (s *Service) Get(ctx context.Context) (AppSettings, error) {
	settings, err := s.repository.Get(ctx)
	if err != nil {
		return AppSettings{}, err
	}
	return NormalizeSettings(settings), nil
}

func (s *Service) Save(ctx context.Context, settings AppSettings) (AppSettings, error) {
	return s.repository.Save(ctx, NormalizeSettings(settings))
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

func (s *Service) GetLocalGatewaySelectedModels(ctx context.Context) ([]SelectedModel, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}
	return normalizeSelectedModels(settings.LocalGatewaySelected), nil
}

func (s *Service) UpdateLocalGatewaySelectedModels(ctx context.Context, items []SelectedModel) ([]SelectedModel, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}
	settings.LocalGatewaySelected = normalizeSelectedModels(items)
	saved, err := s.Save(ctx, settings)
	if err != nil {
		return nil, err
	}
	return saved.LocalGatewaySelected, nil
}

func normalizeClaudeCodeModelMap(input ClaudeCodeModelMap) ClaudeCodeModelMap {
	return ClaudeCodeModelMap{
		Opus:   strings.TrimSpace(input.Opus),
		Sonnet: strings.TrimSpace(input.Sonnet),
		Haiku:  strings.TrimSpace(input.Haiku),
	}
}

func (s *Service) GetLocalGatewayClaudeMap(ctx context.Context) (ClaudeCodeModelMap, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return ClaudeCodeModelMap{}, err
	}
	return normalizeClaudeCodeModelMap(settings.LocalGatewayClaude), nil
}

func (s *Service) UpdateLocalGatewayClaudeMap(ctx context.Context, input ClaudeCodeModelMap) (ClaudeCodeModelMap, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return ClaudeCodeModelMap{}, err
	}
	settings.LocalGatewayClaude = normalizeClaudeCodeModelMap(input)
	saved, err := s.Save(ctx, settings)
	if err != nil {
		return ClaudeCodeModelMap{}, err
	}
	return saved.LocalGatewayClaude, nil
}
