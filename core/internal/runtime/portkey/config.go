package portkey

import (
	"context"
	"sort"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelentry"
	runtimecfg "github.com/xiaoyuandev/clash-for-ai/core/internal/runtime"
)

type Template struct {
	RuntimeMode   string          `json:"runtime_mode"`
	RuntimeURL    string          `json:"runtime_url"`
	GeneratedAt   time.Time       `json:"generated_at"`
	TotalEntries  int             `json:"total_entries"`
	EnabledCount  int             `json:"enabled_count"`
	DisabledCount int             `json:"disabled_count"`
	Entries       []TemplateEntry `json:"entries"`
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

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Enabled != entries[j].Enabled {
			return entries[i].Enabled && !entries[j].Enabled
		}
		if entries[i].Position != entries[j].Position {
			return entries[i].Position < entries[j].Position
		}
		return entries[i].Name < entries[j].Name
	})

	items := make([]TemplateEntry, 0, len(entries))
	enabledCount := 0
	for _, entry := range entries {
		if entry.Enabled {
			enabledCount += 1
		}
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
		RuntimeMode:   string(runtimeConfig.Mode),
		RuntimeURL:    runtimeConfig.BaseURL,
		GeneratedAt:   time.Now().UTC(),
		TotalEntries:  len(entries),
		EnabledCount:  enabledCount,
		DisabledCount: len(entries) - enabledCount,
		Entries:       items,
	}, nil
}
