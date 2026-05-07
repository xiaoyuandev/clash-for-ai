package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/gateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/health"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/storage"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/tooling"
)

func TestLocalGatewayRuntimeEndpointWithoutExecutable(t *testing.T) {
	t.Parallel()

	handler := newTestRouter(t, nil, localgateway.RuntimeConfig{
		Host:    "127.0.0.1",
		Port:    3457,
		DataDir: filepath.Join(t.TempDir(), "runtime"),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/local-gateway/runtime", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Runtime localgateway.RuntimeStatus `json:"runtime"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime payload: %v", err)
	}

	if payload.Runtime.State != localgateway.RuntimeStateStopped {
		t.Fatalf("unexpected runtime state: %+v", payload.Runtime)
	}
	if payload.Runtime.LastError == "" {
		t.Fatalf("expected runtime last error: %+v", payload.Runtime)
	}
}

func TestLocalGatewaySourceAndSyncEndpoints(t *testing.T) {
	t.Parallel()

	adapter := &localgatewaySpyAdapter{
		mockGatewayAdapter: mockGatewayAdapter{},
		runtimeStatus: localgateway.RuntimeStatus{
			RuntimeKind: localgateway.RuntimeKindAIMiniGateway,
			State:       localgateway.RuntimeStateRunning,
			Running:     true,
			Healthy:     true,
			APIBase:     "http://127.0.0.1:3457",
		},
		syncResult: localgateway.SyncResult{
			AppliedSources:        1,
			AppliedSelectedModels: 0,
			LastSyncedAt:          "2026-05-01T00:00:00Z",
		},
		sourceCapabilities: []localgateway.ModelSourceCapability{
			{
				SourceID:                      "source-1",
				Name:                          "OpenAI Direct",
				ProviderType:                  "openai-compatible",
				SupportsModelsAPI:             true,
				ModelsAPIStatus:               "supported",
				SupportsOpenAIChatCompletions: true,
				OpenAIChatCompletionsStatus:   "supported",
				SupportsOpenAIResponses:       true,
				OpenAIResponsesStatus:         "supported",
				SupportsAnthropicMessages:     false,
				AnthropicMessagesStatus:       "unsupported",
				SupportsAnthropicCountTokens:  false,
				AnthropicCountTokensStatus:    "unsupported",
				SupportsStream:                true,
				StreamStatus:                  "supported",
			},
		},
		sourceHealthcheck: localgateway.ModelSourceHealthcheck{
			Status:     "ok",
			StatusCode: http.StatusOK,
			LatencyMS:  123,
			Summary:    "healthcheck ok",
			CheckedAt:  "2026-05-01T00:00:00Z",
		},
	}
	handler := newTestRouter(t, adapter, localgateway.RuntimeConfig{
		Executable: "/tmp/ai-mini-gateway",
		Host:       "127.0.0.1",
		Port:       3457,
		DataDir:    filepath.Join(t.TempDir(), "runtime"),
	})

	createBody := bytes.NewBufferString(`{
		"name":"OpenAI Direct",
		"base_url":"https://api.openai.com/v1",
		"api_key":"sk-test-openai",
		"provider_type":"openai-compatible",
		"default_model_id":"gpt-4.1",
		"enabled":true,
		"position":0
	}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/local-gateway/sources", createBody)
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: %d body=%s", createRec.Code, createRec.Body.String())
	}
	assertLocalGatewaySourceResponseIncludesAPIKey(t, createRec.Body.Bytes())

	listReq := httptest.NewRequest(http.MethodGet, "/api/local-gateway/sources", nil)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d body=%s", listRec.Code, listRec.Body.String())
	}
	assertLocalGatewaySourceResponseIncludesAPIKey(t, listRec.Body.Bytes())

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created source: %v", err)
	}

	updateBody := bytes.NewBufferString(`{
		"name":"OpenAI Direct Updated",
		"base_url":"https://api.openai.com/v1",
		"api_key":"",
		"provider_type":"openai-compatible",
		"default_model_id":"gpt-4.1",
		"enabled":true,
		"position":99
	}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/local-gateway/sources/"+created.ID, updateBody)
	updateRec := httptest.NewRecorder()
	handler.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("unexpected update status: %d body=%s", updateRec.Code, updateRec.Body.String())
	}
	assertLocalGatewaySourceResponseIncludesAPIKey(t, updateRec.Body.Bytes())

	selectedBody := bytes.NewBufferString(`[{"model_id":"gpt-4.1","position":0}]`)
	selectedReq := httptest.NewRequest(http.MethodPut, "/api/local-gateway/selected-models", selectedBody)
	selectedRec := httptest.NewRecorder()
	handler.ServeHTTP(selectedRec, selectedReq)
	if selectedRec.Code != http.StatusOK {
		t.Fatalf("unexpected selected status: %d body=%s", selectedRec.Code, selectedRec.Body.String())
	}

	syncReq := httptest.NewRequest(http.MethodPost, "/api/local-gateway/sync", nil)
	syncRec := httptest.NewRecorder()
	handler.ServeHTTP(syncRec, syncReq)
	if syncRec.Code != http.StatusOK {
		t.Fatalf("unexpected sync status: %d body=%s", syncRec.Code, syncRec.Body.String())
	}

	if len(adapter.syncInputs) != 1 {
		t.Fatalf("unexpected sync count: %d", len(adapter.syncInputs))
	}
	if len(adapter.syncInputs[0].Sources) != 1 {
		t.Fatalf("unexpected synced sources: %+v", adapter.syncInputs[0].Sources)
	}
	if adapter.syncInputs[0].Sources[0].APIKey != "sk-test-openai" {
		t.Fatalf("unexpected synced api key: %s", adapter.syncInputs[0].Sources[0].APIKey)
	}
	if len(adapter.syncInputs[0].SelectedModels) != 0 {
		t.Fatalf("expected selected models to be omitted from runtime sync, got %+v", adapter.syncInputs[0].SelectedModels)
	}

	capabilityReq := httptest.NewRequest(http.MethodGet, "/api/local-gateway/source-capabilities", nil)
	capabilityRec := httptest.NewRecorder()
	handler.ServeHTTP(capabilityRec, capabilityReq)
	if capabilityRec.Code != http.StatusOK {
		t.Fatalf("unexpected source capabilities status: %d body=%s", capabilityRec.Code, capabilityRec.Body.String())
	}

	healthReq := httptest.NewRequest(http.MethodPost, "/api/local-gateway/sources/"+created.ID+"/healthcheck", nil)
	healthRec := httptest.NewRecorder()
	handler.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("unexpected source healthcheck status: %d body=%s", healthRec.Code, healthRec.Body.String())
	}
}

func TestLocalGatewaySourceModelsPreviewEndpoint(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test-openai" {
			t.Fatalf("unexpected upstream auth header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1","object":"model"}]}`))
	}))
	defer upstream.Close()

	handler := newTestRouter(t, nil, localgateway.RuntimeConfig{
		Host:    "127.0.0.1",
		Port:    3457,
		DataDir: filepath.Join(t.TempDir(), "runtime"),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/local-gateway/source-models/preview", bytes.NewBufferString(`{
		"base_url":"`+upstream.URL+`/v1",
		"api_key":"sk-test-openai",
		"provider_type":"openai-compatible"
	}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected preview status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload []localgateway.SourceModelInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode preview payload: %v", err)
	}
	if len(payload) != 1 || payload[0].ID != "gpt-4.1" {
		t.Fatalf("unexpected preview payload: %+v", payload)
	}
}

func TestManagedLocalGatewayProviderActivationRequiresHealthyRuntime(t *testing.T) {
	t.Parallel()

	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "router.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	defer sqliteStore.Close()

	credentialStore := credential.NewInMemoryStore()
	providerService := provider.NewService(provider.NewInMemoryRepository(), credentialStore)
	healthService := health.NewService(providerService, credentialStore)
	gatewayHandler := gateway.NewHandler(providerService, credentialStore, nil)
	localService := localgateway.NewService(localgateway.NewSQLiteRepository(sqliteStore.DB), credentialStore)
	adapter := &localgatewaySpyAdapter{
		mockGatewayAdapter: mockGatewayAdapter{},
		runtimeStatus: localgateway.RuntimeStatus{
			RuntimeKind: localgateway.RuntimeKindAIMiniGateway,
			State:       localgateway.RuntimeStateDegraded,
			Running:     true,
			Healthy:     false,
			APIBase:     "http://127.0.0.1:3457",
			LastError:   "runtime healthcheck returned non-200",
		},
	}
	manager := localgateway.NewManager(localService, adapter, localgateway.RuntimeConfig{
		Executable: "/tmp/ai-mini-gateway",
		Host:       "127.0.0.1",
		Port:       3457,
		DataDir:    filepath.Join(t.TempDir(), "runtime"),
	})

	if _, err := providerService.EnsureManagedLocalGateway(
		context.Background(),
		"Local Gateway",
		"http://127.0.0.1:3457/v1",
		"dummy",
	); err != nil {
		t.Fatalf("ensure managed local gateway: %v", err)
	}

	handler := NewRouter(providerService, healthService, nil, manager, tooling.NewService(providerService), 3456, gatewayHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/providers/provider-local-gateway/activate", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("unexpected activate status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode activation error payload: %v", err)
	}
	if payload["error"] != string(localgateway.AdapterErrorConflict) {
		t.Fatalf("unexpected activation error payload: %+v", payload)
	}
}

func assertLocalGatewaySourceResponseIncludesAPIKey(t *testing.T, body []byte) {
	t.Helper()

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode local gateway source payload: %v body=%s", err, string(body))
	}

	switch typed := payload.(type) {
	case map[string]any:
		if _, ok := typed["api_key"]; !ok {
			t.Fatalf("expected source response to include api_key, got %s", string(body))
		}
		if _, ok := typed["api_key_masked"]; !ok {
			t.Fatalf("expected source response to include api_key_masked, got %s", string(body))
		}
	case []any:
		for _, item := range typed {
			object, ok := item.(map[string]any)
			if !ok {
				t.Fatalf("unexpected source list item: %T", item)
			}
			if _, ok := object["api_key"]; !ok {
				t.Fatalf("expected source list item to include api_key, got %s", string(body))
			}
			if _, ok := object["api_key_masked"]; !ok {
				t.Fatalf("expected source list item to include api_key_masked, got %s", string(body))
			}
		}
	default:
		t.Fatalf("unexpected source payload shape: %T", payload)
	}
}

func newTestRouter(t *testing.T, adapter localgateway.GatewayAdapter, runtime localgateway.RuntimeConfig) http.Handler {
	t.Helper()

	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "router.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = sqliteStore.Close() })

	credentialStore := credential.NewInMemoryStore()
	providerService := provider.NewService(provider.NewInMemoryRepository(), credentialStore)
	healthService := health.NewService(providerService, credentialStore)
	gatewayHandler := gateway.NewHandler(providerService, credentialStore, nil)
	localService := localgateway.NewService(localgateway.NewSQLiteRepository(sqliteStore.DB), credentialStore)
	if adapter == nil {
		adapter = &localgatewaySpyAdapter{
			mockGatewayAdapter: mockGatewayAdapter{},
		}
	}
	manager := localgateway.NewManager(localService, adapter, runtime)

	return NewRouter(providerService, healthService, nil, manager, tooling.NewService(providerService), 3456, gatewayHandler)
}

type localgatewaySpyAdapter struct {
	mockGatewayAdapter
	runtimeStatus      localgateway.RuntimeStatus
	syncResult         localgateway.SyncResult
	syncInputs         []localgateway.SyncInput
	sourceCapabilities []localgateway.ModelSourceCapability
	sourceHealthcheck  localgateway.ModelSourceHealthcheck
}

func (s *localgatewaySpyAdapter) GetRuntimeStatus(context.Context) (localgateway.RuntimeStatus, error) {
	return s.runtimeStatus, nil
}

func (s *localgatewaySpyAdapter) StartRuntime(context.Context, localgateway.StartRuntimeInput) (localgateway.RuntimeStatus, error) {
	return s.runtimeStatus, nil
}

func (s *localgatewaySpyAdapter) SyncFromProductState(_ context.Context, input localgateway.SyncInput) (localgateway.SyncResult, error) {
	s.syncInputs = append(s.syncInputs, input)
	return s.syncResult, nil
}

func (s *localgatewaySpyAdapter) ListModelSourceCapabilities(context.Context) ([]localgateway.ModelSourceCapability, error) {
	return append([]localgateway.ModelSourceCapability(nil), s.sourceCapabilities...), nil
}

func (s *localgatewaySpyAdapter) CheckModelSourceHealth(context.Context, string) (localgateway.ModelSourceHealthcheck, error) {
	return s.sourceHealthcheck, nil
}

type mockGatewayAdapter struct{}

func (m mockGatewayAdapter) RuntimeKind() string {
	return localgateway.RuntimeKindAIMiniGateway
}

func (m mockGatewayAdapter) StartRuntime(context.Context, localgateway.StartRuntimeInput) (localgateway.RuntimeStatus, error) {
	return localgateway.RuntimeStatus{RuntimeKind: localgateway.RuntimeKindAIMiniGateway}, nil
}

func (m mockGatewayAdapter) StopRuntime(context.Context) error {
	return nil
}

func (m mockGatewayAdapter) GetRuntimeStatus(context.Context) (localgateway.RuntimeStatus, error) {
	return localgateway.RuntimeStatus{RuntimeKind: localgateway.RuntimeKindAIMiniGateway}, nil
}

func (m mockGatewayAdapter) GetCapabilities(context.Context) (localgateway.RuntimeCapabilities, error) {
	return localgateway.RuntimeCapabilities{}, nil
}

func (m mockGatewayAdapter) ListModelSources(context.Context) ([]localgateway.RuntimeModelSource, error) {
	return nil, nil
}

func (m mockGatewayAdapter) ListModelSourceCapabilities(context.Context) ([]localgateway.ModelSourceCapability, error) {
	return nil, nil
}

func (m mockGatewayAdapter) CheckModelSourceHealth(context.Context, string) (localgateway.ModelSourceHealthcheck, error) {
	return localgateway.ModelSourceHealthcheck{}, nil
}

func (m mockGatewayAdapter) CreateModelSource(context.Context, localgateway.RuntimeModelSourceInput) (localgateway.RuntimeModelSource, error) {
	return localgateway.RuntimeModelSource{}, nil
}

func (m mockGatewayAdapter) UpdateModelSource(context.Context, string, localgateway.RuntimeModelSourceInput) (localgateway.RuntimeModelSource, error) {
	return localgateway.RuntimeModelSource{}, nil
}

func (m mockGatewayAdapter) DeleteModelSource(context.Context, string) error {
	return nil
}

func (m mockGatewayAdapter) ListSelectedModels(context.Context) ([]localgateway.SelectedModel, error) {
	return nil, nil
}

func (m mockGatewayAdapter) ReplaceSelectedModels(context.Context, []localgateway.SelectedModel) ([]localgateway.SelectedModel, error) {
	return nil, nil
}

func (m mockGatewayAdapter) SyncFromProductState(context.Context, localgateway.SyncInput) (localgateway.SyncResult, error) {
	return localgateway.SyncResult{}, nil
}
