package gatewayadapter

import (
	"context"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type RuntimeInfo struct {
	BaseURL    string
	ListenAddr string
	Mode       string
	Embedded   bool
}

type RuntimeHealth struct {
	Status    string
	Summary   string
	CheckedAt time.Time
}

type RuntimeCapabilities struct {
	SupportsOpenAICompatible    bool
	SupportsAnthropicCompatible bool
	SupportsModelsAPI           bool
	SupportsStream              bool
	SupportsAdminAPI            bool
	SupportsModelSourceAdmin    bool
	SupportsSelectedModelAdmin  bool
}

func (c RuntimeCapabilities) MissingRequiredCapabilities() []string {
	missing := make([]string, 0, 4)
	if !c.SupportsOpenAICompatible {
		missing = append(missing, "openai-compatible")
	}
	if !c.SupportsAnthropicCompatible {
		missing = append(missing, "anthropic-compatible")
	}
	if !c.SupportsModelsAPI {
		missing = append(missing, "models-api")
	}
	if !c.SupportsStream {
		missing = append(missing, "stream")
	}
	return missing
}

func (c RuntimeCapabilities) MissingOptionalCapabilities() []string {
	missing := make([]string, 0, 3)
	if !c.SupportsAdminAPI {
		missing = append(missing, "admin-api")
	}
	if !c.SupportsModelSourceAdmin {
		missing = append(missing, "model-source-admin")
	}
	if !c.SupportsSelectedModelAdmin {
		missing = append(missing, "selected-model-admin")
	}
	return missing
}

type RuntimeAdapter interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Discover(ctx context.Context) (RuntimeInfo, error)
	EnsureReady(ctx context.Context) error
	CheckHealth(ctx context.Context) (RuntimeHealth, error)
	Capabilities(ctx context.Context) (RuntimeCapabilities, error)
}

type AdminAdapter interface {
	ListModelSources(ctx context.Context) ([]modelsource.Source, error)
	CreateModelSource(ctx context.Context, input modelsource.CreateInput) (modelsource.Source, error)
	UpdateModelSource(ctx context.Context, id string, input modelsource.UpdateInput) (modelsource.Source, error)
	DeleteModelSource(ctx context.Context, id string) error
	ReplaceModelSourceOrder(ctx context.Context, items []modelsource.Source) ([]modelsource.Source, error)
}

type LocalRuntimeAdapter interface {
	RuntimeAdapter
	AdminAdapter
	ListSelectedModels(ctx context.Context) ([]provider.SelectedModel, error)
	ReplaceSelectedModels(ctx context.Context, items []provider.SelectedModel) ([]provider.SelectedModel, error)
}
