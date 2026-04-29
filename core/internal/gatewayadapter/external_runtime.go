package gatewayadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

var ErrRuntimeAdminUnsupported = errors.New("runtime admin unsupported")

type ExternalRuntimeAdapter struct {
	baseURL string
	client  *http.Client

	mu          sync.Mutex
	cachedProbe RuntimeCapabilities
	hasProbe    bool
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

	capabilities, err := a.Capabilities(ctx)
	if err != nil {
		return err
	}
	if missing := capabilities.MissingRequiredCapabilities(); len(missing) > 0 {
		return fmt.Errorf("external runtime missing required capabilities: %s", strings.Join(missing, ", "))
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
		// Fallback to the models endpoint for runtimes that do not implement /health.
		if capabilities, probeErr := a.probeCapabilities(ctx); probeErr == nil && capabilities.SupportsModelsAPI {
			return RuntimeHealth{
				Status:    "ok",
				Summary:   "fallback health via models endpoint",
				CheckedAt: time.Now().UTC(),
			}, nil
		}
		return RuntimeHealth{
			Status:    "error",
			Summary:   err.Error(),
			CheckedAt: time.Now().UTC(),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		if capabilities, probeErr := a.probeCapabilities(ctx); probeErr == nil && capabilities.SupportsModelsAPI {
			return RuntimeHealth{
				Status:    "ok",
				Summary:   "fallback health via models endpoint",
				CheckedAt: time.Now().UTC(),
			}, nil
		}
	}

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

func (a *ExternalRuntimeAdapter) Capabilities(ctx context.Context) (RuntimeCapabilities, error) {
	a.mu.Lock()
	if a.hasProbe {
		cached := a.cachedProbe
		a.mu.Unlock()
		return cached, nil
	}
	a.mu.Unlock()

	return a.probeCapabilities(ctx)
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

func (a *ExternalRuntimeAdapter) probeCapabilities(ctx context.Context) (RuntimeCapabilities, error) {
	openAI, err := a.supportsEndpoint(ctx, http.MethodPost, "/v1/chat/completions", []byte(`{"model":"probe","messages":[]}`))
	if err != nil {
		return RuntimeCapabilities{}, err
	}
	anthropic, err := a.supportsEndpoint(ctx, http.MethodPost, "/v1/messages", []byte(`{"model":"probe","max_tokens":1,"messages":[]}`))
	if err != nil {
		return RuntimeCapabilities{}, err
	}
	models, err := a.supportsEndpoint(ctx, http.MethodGet, "/v1/models", nil)
	if err != nil {
		return RuntimeCapabilities{}, err
	}
	modelSourceAdmin, err := a.supportsEndpoint(ctx, http.MethodGet, "/admin/model-sources", nil)
	if err != nil {
		return RuntimeCapabilities{}, err
	}
	selectedModelAdmin, err := a.supportsEndpoint(ctx, http.MethodGet, "/admin/selected-models", nil)
	if err != nil {
		return RuntimeCapabilities{}, err
	}

	capabilities := RuntimeCapabilities{
		SupportsOpenAICompatible:    openAI,
		SupportsAnthropicCompatible: anthropic,
		SupportsModelsAPI:           models,
		SupportsStream:              openAI || anthropic,
		SupportsAdminAPI:            modelSourceAdmin || selectedModelAdmin,
		SupportsModelSourceAdmin:    modelSourceAdmin,
		SupportsSelectedModelAdmin:  selectedModelAdmin,
	}

	a.mu.Lock()
	a.cachedProbe = capabilities
	a.hasProbe = true
	a.mu.Unlock()

	return capabilities, nil
}

func (a *ExternalRuntimeAdapter) supportsEndpoint(
	ctx context.Context,
	method string,
	path string,
	body []byte,
) (bool, error) {
	var reqBody *strings.Reader
	if len(body) == 0 {
		reqBody = strings.NewReader("")
	} else {
		reqBody = strings.NewReader(string(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, reqBody)
	if err != nil {
		return false, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return false, nil
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent,
		http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
		http.StatusMethodNotAllowed, http.StatusTooManyRequests,
		http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return true, nil
	default:
		return resp.StatusCode < 500, nil
	}
}
