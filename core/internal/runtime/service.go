package runtime

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Service struct {
	repository Repository
	client     *http.Client
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func NormalizeConfig(cfg Config) Config {
	cfg.Mode = normalizeMode(cfg.Mode)
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	return cfg
}

func normalizeMode(mode Mode) Mode {
	switch mode {
	case ModeExternalPortkey:
		return ModeExternalPortkey
	default:
		return ModeLegacy
	}
}

func (s *Service) GetConfig(ctx context.Context) (Config, error) {
	cfg, err := s.repository.GetConfig(ctx)
	if err != nil {
		return Config{}, err
	}
	return NormalizeConfig(cfg), nil
}

func (s *Service) UpdateConfig(ctx context.Context, cfg Config) (Config, error) {
	cfg = NormalizeConfig(cfg)

	if cfg.Mode == ModeExternalPortkey {
		if cfg.BaseURL == "" {
			return Config{}, fmt.Errorf("runtime base_url is required for external-portkey mode")
		}
		if _, err := url.ParseRequestURI(cfg.BaseURL); err != nil {
			return Config{}, fmt.Errorf("invalid runtime base_url: %w", err)
		}
	}

	return s.repository.SaveConfig(ctx, cfg)
}

func (s *Service) Health(ctx context.Context) (Health, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return Health{}, err
	}

	checkedAt := time.Now().UTC()
	if cfg.Mode == ModeLegacy {
		return Health{
			Mode:      cfg.Mode,
			BaseURL:   cfg.BaseURL,
			Status:    "ok",
			Message:   "legacy built-in runtime",
			CheckedAt: checkedAt,
		}, nil
	}

	if cfg.BaseURL == "" {
		return Health{
			Mode:      cfg.Mode,
			BaseURL:   cfg.BaseURL,
			Status:    "error",
			Message:   "runtime base_url is empty",
			CheckedAt: checkedAt,
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.BaseURL, nil)
	if err != nil {
		return Health{}, fmt.Errorf("build runtime health request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return Health{
			Mode:      cfg.Mode,
			BaseURL:   cfg.BaseURL,
			Status:    "error",
			Message:   err.Error(),
			CheckedAt: checkedAt,
		}, nil
	}
	defer resp.Body.Close()

	return Health{
		Mode:      cfg.Mode,
		BaseURL:   cfg.BaseURL,
		Status:    "ok",
		Message:   fmt.Sprintf("runtime reachable: HTTP %d", resp.StatusCode),
		CheckedAt: checkedAt,
	}, nil
}

func normalizeClaudeCodeModelMap(input ClaudeCodeModelMap) ClaudeCodeModelMap {
	return ClaudeCodeModelMap{
		Opus:   strings.TrimSpace(input.Opus),
		Sonnet: strings.TrimSpace(input.Sonnet),
		Haiku:  strings.TrimSpace(input.Haiku),
	}
}

func (s *Service) GetLocalGatewayClaudeMap(ctx context.Context) (ClaudeCodeModelMap, error) {
	cfg, err := s.repository.GetLocalGatewayClaudeMap(ctx)
	if err != nil {
		return ClaudeCodeModelMap{}, err
	}
	return normalizeClaudeCodeModelMap(cfg), nil
}

func (s *Service) UpdateLocalGatewayClaudeMap(ctx context.Context, cfg ClaudeCodeModelMap) (ClaudeCodeModelMap, error) {
	return s.repository.SaveLocalGatewayClaudeMap(ctx, normalizeClaudeCodeModelMap(cfg))
}

func (s *Service) GetLocalGatewaySelectedModels(ctx context.Context) ([]SelectedModel, error) {
	items, err := s.repository.GetLocalGatewaySelectedModels(ctx)
	if err != nil {
		return nil, err
	}

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

	return normalized, nil
}

func (s *Service) UpdateLocalGatewaySelectedModels(ctx context.Context, items []SelectedModel) ([]SelectedModel, error) {
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

	return s.repository.SaveLocalGatewaySelectedModels(ctx, normalized)
}
