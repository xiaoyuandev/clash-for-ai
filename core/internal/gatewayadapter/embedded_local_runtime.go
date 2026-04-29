package gatewayadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/settings"
)

type EmbeddedLocalRuntimeAdapter struct {
	baseURL      string
	listenAddr   string
	executable   string
	dataDir      string
	client       *http.Client
	capabilities RuntimeCapabilities
	admin        runtimeAdminClient

	mu        sync.Mutex
	process   *exec.Cmd
	startedAt time.Time
}

func NewEmbeddedLocalRuntimeAdapter(
	localSettings settings.LocalGatewaySettings,
	executable string,
	dataDir string,
) *EmbeddedLocalRuntimeAdapter {
	listenHost := resolveListenHost(localSettings.ListenHost)
	listenPort := localSettings.ListenPort
	if listenPort <= 0 {
		listenPort = 8788
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return &EmbeddedLocalRuntimeAdapter{
		baseURL:    fmt.Sprintf("http://%s:%d", normalizeBaseURLHost(listenHost), listenPort),
		listenAddr: fmt.Sprintf("%s:%d", listenHost, listenPort),
		executable: executable,
		dataDir:    dataDir,
		client:     client,
		capabilities: RuntimeCapabilities{
			SupportsOpenAICompatible:    true,
			SupportsAnthropicCompatible: true,
			SupportsModelsAPI:           true,
			SupportsStream:              true,
			SupportsAdminAPI:            true,
			SupportsModelSourceAdmin:    true,
			SupportsSelectedModelAdmin:  true,
		},
		admin: runtimeAdminClient{
			baseURL: fmt.Sprintf("http://%s:%d", normalizeBaseURLHost(listenHost), listenPort),
			client:  client,
		},
	}
}

func (a *EmbeddedLocalRuntimeAdapter) Start(ctx context.Context) error {
	a.mu.Lock()
	running := a.process != nil && a.process.Process != nil && a.process.ProcessState == nil
	a.mu.Unlock()

	if running {
		if err := a.waitForHealth(ctx, 2, 150*time.Millisecond); err == nil {
			return nil
		}
	}

	if err := a.startProcess(); err != nil {
		return err
	}

	return a.waitForHealth(ctx, 20, 250*time.Millisecond)
}

func (a *EmbeddedLocalRuntimeAdapter) Stop(_ context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.process == nil || a.process.Process == nil {
		return nil
	}

	err := a.process.Process.Kill()
	a.process = nil
	return err
}

func (a *EmbeddedLocalRuntimeAdapter) Discover(ctx context.Context) (RuntimeInfo, error) {
	if err := a.EnsureReady(ctx); err != nil {
		return RuntimeInfo{}, err
	}

	return RuntimeInfo{
		BaseURL:    a.baseURL,
		ListenAddr: a.listenAddr,
		Mode:       "embedded",
		Embedded:   true,
	}, nil
}

func (a *EmbeddedLocalRuntimeAdapter) EnsureReady(ctx context.Context) error {
	return a.Start(ctx)
}

func (a *EmbeddedLocalRuntimeAdapter) CheckHealth(ctx context.Context) (RuntimeHealth, error) {
	if err := a.EnsureReady(ctx); err != nil {
		return RuntimeHealth{}, err
	}

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
	return a.capabilities, nil
}

func (a *EmbeddedLocalRuntimeAdapter) ListModelSources(ctx context.Context) ([]modelsource.Source, error) {
	if err := a.EnsureReady(ctx); err != nil {
		return nil, err
	}
	return a.admin.ListModelSources(ctx)
}

func (a *EmbeddedLocalRuntimeAdapter) CreateModelSource(ctx context.Context, input modelsource.CreateInput) (modelsource.Source, error) {
	if err := a.EnsureReady(ctx); err != nil {
		return modelsource.Source{}, err
	}
	return a.admin.CreateModelSource(ctx, input)
}

func (a *EmbeddedLocalRuntimeAdapter) UpdateModelSource(ctx context.Context, id string, input modelsource.UpdateInput) (modelsource.Source, error) {
	if err := a.EnsureReady(ctx); err != nil {
		return modelsource.Source{}, err
	}
	return a.admin.UpdateModelSource(ctx, id, input)
}

func (a *EmbeddedLocalRuntimeAdapter) DeleteModelSource(ctx context.Context, id string) error {
	if err := a.EnsureReady(ctx); err != nil {
		return err
	}
	return a.admin.DeleteModelSource(ctx, id)
}

func (a *EmbeddedLocalRuntimeAdapter) ReplaceModelSourceOrder(ctx context.Context, items []modelsource.Source) ([]modelsource.Source, error) {
	if err := a.EnsureReady(ctx); err != nil {
		return nil, err
	}
	return a.admin.ReplaceModelSourceOrder(ctx, items)
}

func (a *EmbeddedLocalRuntimeAdapter) ListSelectedModels(ctx context.Context) ([]provider.SelectedModel, error) {
	if err := a.EnsureReady(ctx); err != nil {
		return nil, err
	}
	return a.admin.ListSelectedModels(ctx)
}

func (a *EmbeddedLocalRuntimeAdapter) ReplaceSelectedModels(ctx context.Context, items []provider.SelectedModel) ([]provider.SelectedModel, error) {
	if err := a.EnsureReady(ctx); err != nil {
		return nil, err
	}
	return a.admin.ReplaceSelectedModels(ctx, items)
}

func (a *EmbeddedLocalRuntimeAdapter) startProcess() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.process != nil && a.process.Process != nil && a.process.ProcessState == nil {
		return nil
	}

	cmd := exec.Command(a.executable)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"CLASH_FOR_AI_MODE=local-gateway-runtime",
		"CORE_DATA_DIR="+a.dataDir,
		"LOCAL_GATEWAY_RUNTIME_HOST="+runtimeHostFromListenAddr(a.listenAddr),
		"LOCAL_GATEWAY_RUNTIME_PORT="+runtimePortFromListenAddr(a.listenAddr),
	)

	if err := cmd.Start(); err != nil {
		return err
	}

	a.process = cmd
	a.startedAt = time.Now()

	go func() {
		_ = cmd.Wait()
		a.mu.Lock()
		if a.process == cmd {
			a.process = nil
		}
		a.mu.Unlock()
	}()

	return nil
}

func (a *EmbeddedLocalRuntimeAdapter) waitForHealth(ctx context.Context, attempts int, interval time.Duration) error {
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/health", nil)
		if err != nil {
			return err
		}

		resp, err := a.client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 400 {
				return nil
			}
			lastErr = fmt.Errorf("runtime health returned HTTP %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("runtime healthcheck failed")
	}
	return lastErr
}

func runtimeHostFromListenAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "127.0.0.1"
	}
	return strings.TrimSpace(host)
}

func runtimePortFromListenAddr(addr string) string {
	_, port, found := strings.Cut(addr, ":")
	if !found {
		return strconv.Itoa(8788)
	}
	return strings.TrimSpace(port)
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
