package gatewayadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/settings"
)

type EmbeddedLocalRuntimeAdapter struct {
	baseURL    string
	listenAddr string
	handler    http.Handler
	client     *http.Client

	mu        sync.Mutex
	startOnce sync.Once
	startErr  error
	server    *http.Server
	listener  net.Listener
}

func NewEmbeddedLocalRuntimeAdapter(
	localSettings settings.LocalGatewaySettings,
	handler http.Handler,
) *EmbeddedLocalRuntimeAdapter {
	listenHost := resolveListenHost(localSettings.ListenHost)
	listenAddr := fmt.Sprintf("%s:%d", listenHost, localSettings.ListenPort)

	return &EmbeddedLocalRuntimeAdapter{
		baseURL:    fmt.Sprintf("http://%s:%d", normalizeBaseURLHost(listenHost), localSettings.ListenPort),
		listenAddr: listenAddr,
		handler:    handler,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (a *EmbeddedLocalRuntimeAdapter) Start(_ context.Context) error {
	a.startOnce.Do(func() {
		listener, err := net.Listen("tcp", a.listenAddr)
		if err != nil {
			a.startErr = fmt.Errorf("start local gateway runtime listener: %w", err)
			return
		}

		server := &http.Server{
			Addr:    a.listenAddr,
			Handler: a.handler,
			BaseContext: func(net.Listener) context.Context {
				return context.Background()
			},
		}
		a.mu.Lock()
		a.server = server
		a.listener = listener
		a.mu.Unlock()

		go func() {
			_ = server.Serve(listener)
		}()
	})

	return a.startErr
}

func (a *EmbeddedLocalRuntimeAdapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	server := a.server
	listener := a.listener
	a.mu.Unlock()

	if server == nil {
		return nil
	}

	err := server.Shutdown(ctx)
	if listener != nil {
		_ = listener.Close()
	}
	return err
}

func (a *EmbeddedLocalRuntimeAdapter) Discover(_ context.Context) (RuntimeInfo, error) {
	return RuntimeInfo{
		BaseURL:    a.baseURL,
		ListenAddr: a.listenAddr,
		Mode:       "embedded",
		Embedded:   true,
	}, nil
}

func (a *EmbeddedLocalRuntimeAdapter) CheckHealth(ctx context.Context) (RuntimeHealth, error) {
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

func (a *EmbeddedLocalRuntimeAdapter) Capabilities(context.Context) (RuntimeCapabilities, error) {
	return RuntimeCapabilities{
		SupportsOpenAICompatible:    true,
		SupportsAnthropicCompatible: true,
		SupportsModelsAPI:           true,
		SupportsStream:              true,
		SupportsAdminAPI:            true,
		SupportsModelSourceAdmin:    true,
		SupportsSelectedModelAdmin:  true,
	}, nil
}

func (a *EmbeddedLocalRuntimeAdapter) ListModelSources(ctx context.Context) ([]modelsource.Source, error) {
	var items []modelsource.Source
	if err := a.fetchJSON(ctx, http.MethodGet, "/admin/model-sources", nil, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (a *EmbeddedLocalRuntimeAdapter) CreateModelSource(ctx context.Context, input modelsource.CreateInput) (modelsource.Source, error) {
	var item modelsource.Source
	if err := a.fetchJSON(ctx, http.MethodPost, "/admin/model-sources", input, &item); err != nil {
		return modelsource.Source{}, err
	}
	return item, nil
}

func (a *EmbeddedLocalRuntimeAdapter) UpdateModelSource(ctx context.Context, id string, input modelsource.UpdateInput) (modelsource.Source, error) {
	var item modelsource.Source
	if err := a.fetchJSON(ctx, http.MethodPut, "/admin/model-sources/"+id, input, &item); err != nil {
		return modelsource.Source{}, err
	}
	return item, nil
}

func (a *EmbeddedLocalRuntimeAdapter) DeleteModelSource(ctx context.Context, id string) error {
	return a.fetchJSON(ctx, http.MethodDelete, "/admin/model-sources/"+id, nil, nil)
}

func (a *EmbeddedLocalRuntimeAdapter) ReplaceModelSourceOrder(ctx context.Context, items []modelsource.Source) ([]modelsource.Source, error) {
	var saved []modelsource.Source
	if err := a.fetchJSON(ctx, http.MethodPut, "/admin/model-sources/order", items, &saved); err != nil {
		return nil, err
	}
	return saved, nil
}

func (a *EmbeddedLocalRuntimeAdapter) ListSelectedModels(ctx context.Context) ([]provider.SelectedModel, error) {
	var items []provider.SelectedModel
	if err := a.fetchJSON(ctx, http.MethodGet, "/admin/selected-models", nil, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (a *EmbeddedLocalRuntimeAdapter) ReplaceSelectedModels(ctx context.Context, items []provider.SelectedModel) ([]provider.SelectedModel, error) {
	var saved []provider.SelectedModel
	if err := a.fetchJSON(ctx, http.MethodPut, "/admin/selected-models", items, &saved); err != nil {
		return nil, err
	}
	return saved, nil
}

func (a *EmbeddedLocalRuntimeAdapter) fetchJSON(
	ctx context.Context,
	method string,
	path string,
	payload any,
	target any,
) error {
	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusNotFound:
			return modelsource.ErrSourceNotFound
		default:
			return fmt.Errorf("runtime admin request failed: %s %s -> HTTP %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(body)))
		}
	}
	if target == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func resolveListenHost(host string) string {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" {
		return "127.0.0.1"
	}
	return trimmed
}

func normalizeBaseURLHost(host string) string {
	switch strings.TrimSpace(host) {
	case "", "0.0.0.0", "::", "[::]":
		return "127.0.0.1"
	default:
		return strings.TrimSpace(host)
	}
}
