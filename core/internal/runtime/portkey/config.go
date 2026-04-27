package portkey

import (
	"context"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelentry"
	runtimecfg "github.com/xiaoyuandev/clash-for-ai/core/internal/runtime"
)

type Template struct {
	RuntimeMode string          `json:"runtime_mode"`
	RuntimeURL  string          `json:"runtime_url"`
	GeneratedAt time.Time       `json:"generated_at"`
	Entries     []TemplateEntry `json:"entries"`
}

type TemplateEntry struct {
	Name         string `json:"name"`
	ModelID      string `json:"model_id"`
	BaseURL      string `json:"base_url"`
	ProviderType string `json:"provider_type"`
	Protocol     string `json:"protocol"`
	Enabled      bool   `json:"enabled"`
	Position     int    `json:"position"`
}

type ConfigBuilder struct {
	runtimeService *runtimecfg.Service
	modelService   *modelentry.Service
}

func NewConfigBuilder(runtimeService *runtimecfg.Service, modelService *modelentry.Service) *ConfigBuilder {
	return &ConfigBuilder{
		runtimeService: runtimeService,
		modelService:   modelService,
	}
}

func (b *ConfigBuilder) BuildTemplate(ctx context.Context) (Template, error) {
	runtimeConfig, err := b.runtimeService.GetConfig(ctx)
	if err != nil {
		return Template{}, err
	}

	entries, err := b.modelService.List(ctx)
	if err != nil {
		return Template{}, err
	}

	items := make([]TemplateEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, TemplateEntry{
			Name:         entry.Name,
			ModelID:      entry.ModelID,
			BaseURL:      entry.BaseURL,
			ProviderType: entry.ProviderType,
			Protocol:     entry.Protocol,
			Enabled:      entry.Enabled,
			Position:     entry.Position,
		})
	}

	return Template{
		RuntimeMode: string(runtimeConfig.Mode),
		RuntimeURL:  runtimeConfig.BaseURL,
		GeneratedAt: time.Now().UTC(),
		Entries:     items,
	}, nil
}
