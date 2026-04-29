package gatewayadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

var ErrRuntimeAdminUnsupported = errors.New("runtime admin unsupported")

type ExternalRuntimeAdapter struct {
	baseURL string
	client  *http.Client
}

func NewExternalRuntimeAdapter(baseURL string) *ExternalRuntimeAdapter {
	return &ExternalRuntimeAdapter{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (a *ExternalRuntimeAdapter) Start(context.Context) error {
	return nil
}

func (a *ExternalRuntimeAdapter) Stop(context.Context) error {
	return nil
}

func (a *ExternalRuntimeAdapter) Discover(context.Context) (RuntimeInfo, error) {
	return RuntimeInfo{
		BaseURL:  a.baseURL,
		Mode:     "external",
		Embedded: false,
	}, nil
}

func (a *ExternalRuntimeAdapter) EnsureReady(ctx context.Context) error {
	health, err := a.CheckHealth(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(health.Status) != "" && !strings.EqualFold(strings.TrimSpace(health.Status), "ok") {
		return fmt.Errorf("external runtime unavailable: %s", health.Summary)
	}
	return nil
}

func (a *ExternalRuntimeAdapter) CheckHealth(ctx context.Context) (RuntimeHealth, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/health", nil)
	if err != nil {
		return RuntimeHealth{}, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return RuntimeHealth{
			Status:    "error",
			Summary:   err.Error(),
			CheckedAt: time.Now().UTC(),
		}, nil
	}
	defer resp.Body.Close()

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return RuntimeHealth{}, err
	}

	status := payload.Status
	if status == "" {
		status = "ok"
	}

	return RuntimeHealth{
		Status:    status,
		Summary:   fmt.Sprintf("HTTP %d", resp.StatusCode),
		CheckedAt: time.Now().UTC(),
	}, nil
}

func (a *ExternalRuntimeAdapter) Capabilities(context.Context) (RuntimeCapabilities, error) {
	return RuntimeCapabilities{
		SupportsOpenAICompatible:    true,
		SupportsAnthropicCompatible: true,
		SupportsModelsAPI:           true,
		SupportsStream:              true,
		SupportsAdminAPI:            false,
		SupportsModelSourceAdmin:    false,
		SupportsSelectedModelAdmin:  false,
	}, nil
}

func (a *ExternalRuntimeAdapter) ListModelSources(context.Context) ([]modelsource.Source, error) {
	return nil, ErrRuntimeAdminUnsupported
}

func (a *ExternalRuntimeAdapter) CreateModelSource(context.Context, modelsource.CreateInput) (modelsource.Source, error) {
	return modelsource.Source{}, ErrRuntimeAdminUnsupported
}

func (a *ExternalRuntimeAdapter) UpdateModelSource(context.Context, string, modelsource.UpdateInput) (modelsource.Source, error) {
	return modelsource.Source{}, ErrRuntimeAdminUnsupported
}

func (a *ExternalRuntimeAdapter) DeleteModelSource(context.Context, string) error {
	return ErrRuntimeAdminUnsupported
}

func (a *ExternalRuntimeAdapter) ReplaceModelSourceOrder(context.Context, []modelsource.Source) ([]modelsource.Source, error) {
	return nil, ErrRuntimeAdminUnsupported
}

func (a *ExternalRuntimeAdapter) ListSelectedModels(context.Context) ([]provider.SelectedModel, error) {
	return nil, ErrRuntimeAdminUnsupported
}

func (a *ExternalRuntimeAdapter) ReplaceSelectedModels(context.Context, []provider.SelectedModel) ([]provider.SelectedModel, error) {
	return nil, ErrRuntimeAdminUnsupported
}
