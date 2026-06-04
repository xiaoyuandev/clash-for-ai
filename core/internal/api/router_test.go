package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xiaoyuandev/relay-switch/core/internal/credential"
	"github.com/xiaoyuandev/relay-switch/core/internal/gateway"
	"github.com/xiaoyuandev/relay-switch/core/internal/health"
	"github.com/xiaoyuandev/relay-switch/core/internal/localgateway"
	"github.com/xiaoyuandev/relay-switch/core/internal/logging"
	"github.com/xiaoyuandev/relay-switch/core/internal/provider"
	"github.com/xiaoyuandev/relay-switch/core/internal/storage"
	"github.com/xiaoyuandev/relay-switch/core/internal/tooling"
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

func TestReleaseEndpointWithoutMetadata(t *testing.T) {
	t.Parallel()

	handler := newTestRouter(t, nil, localgateway.RuntimeConfig{
		Host:    "127.0.0.1",
		Port:    3457,
		DataDir: filepath.Join(t.TempDir(), "runtime"),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/release", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Available bool `json:"available"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode release payload: %v", err)
	}

	if payload.Available {
		t.Fatalf("expected unavailable release metadata")
	}
}

func TestLogsEndpointCanClearLocalLogs(t *testing.T) {
	t.Parallel()

	handler, logs := newTestRouterWithLogs(t, nil, localgateway.RuntimeConfig{
		Host:    "127.0.0.1",
		Port:    3457,
		DataDir: filepath.Join(t.TempDir(), "runtime"),
	})

	if err := logs.Record(context.Background(), logging.Entry{
		ProviderID:   "provider-1",
		ProviderName: "OpenAI",
		Method:       http.MethodPost,
		Path:         "/v1/chat/completions",
		LatencyMs:    120,
	}); err != nil {
		t.Fatalf("record log: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/logs?limit=10", nil)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d body=%s", listRec.Code, listRec.Body.String())
	}

	var before []logging.RequestLog
	if err := json.Unmarshal(listRec.Body.Bytes(), &before); err != nil {
		t.Fatalf("decode logs: %v", err)
	}
	if len(before) != 1 {
		t.Fatalf("expected one log before clear, got %d", len(before))
	}

	clearReq := httptest.NewRequest(http.MethodDelete, "/api/logs", nil)
	clearRec := httptest.NewRecorder()
	handler.ServeHTTP(clearRec, clearReq)
	if clearRec.Code != http.StatusNoContent {
		t.Fatalf("unexpected clear status: %d body=%s", clearRec.Code, clearRec.Body.String())
	}

	afterReq := httptest.NewRequest(http.MethodGet, "/api/logs?limit=10", nil)
	afterRec := httptest.NewRecorder()
	handler.ServeHTTP(afterRec, afterReq)
	if afterRec.Code != http.StatusOK {
		t.Fatalf("unexpected list-after-clear status: %d body=%s", afterRec.Code, afterRec.Body.String())
	}

	var after []logging.RequestLog
	if err := json.Unmarshal(afterRec.Body.Bytes(), &after); err != nil {
		t.Fatalf("decode logs after clear: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("expected no logs after clear, got %d", len(after))
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

func TestProviderModelTestEndpoint(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1"}]}`))
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"chatcmpl-test","choices":[{"message":{"role":"assistant","content":"o"}}]}`))
		default:
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	handler := newTestRouter(t, nil, localgateway.RuntimeConfig{
		Host:    "127.0.0.1",
		Port:    3457,
		DataDir: filepath.Join(t.TempDir(), "runtime"),
	})

	createReq := httptest.NewRequest(http.MethodPost, "/api/providers", bytes.NewBufferString(`{
		"name":"OpenAI Relay",
		"base_url":"`+upstream.URL+`/v1",
		"api_key":"sk-test",
		"auth_mode":"bearer",
		"extra_headers":{},
		"claude_code_model_map":{"opus":"","sonnet":"","haiku":""}
	}`))
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("unexpected provider create status: %d body=%s", createRec.Code, createRec.Body.String())
	}

	var created provider.Provider
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created provider: %v", err)
	}

	testReq := httptest.NewRequest(http.MethodPost, "/api/providers/"+created.ID+"/model-tests", bytes.NewBufferString(`{"model_id":"gpt-4.1"}`))
	testRec := httptest.NewRecorder()
	handler.ServeHTTP(testRec, testReq)
	if testRec.Code != http.StatusOK {
		t.Fatalf("unexpected model test status: %d body=%s", testRec.Code, testRec.Body.String())
	}

	var result provider.ModelTestResult
	if err := json.Unmarshal(testRec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode model test result: %v", err)
	}
	if result.Status != "ok" || result.ModelID != "gpt-4.1" {
		t.Fatalf("unexpected model test result: %+v", result)
	}
}

func TestProviderCodexModelsSyncsActiveProviderCatalog(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	handler := newTestRouter(t, nil, localgateway.RuntimeConfig{
		Host:    "127.0.0.1",
		Port:    3457,
		DataDir: filepath.Join(t.TempDir(), "runtime"),
	})

	first := createProviderViaAPI(t, handler, "First Provider")
	second := createProviderViaAPI(t, handler, "Second Provider")

	activateReq := httptest.NewRequest(http.MethodPost, "/api/providers/"+first.ID+"/activate", nil)
	activateRec := httptest.NewRecorder()
	handler.ServeHTTP(activateRec, activateReq)
	if activateRec.Code != http.StatusOK {
		t.Fatalf("unexpected first activate status: %d body=%s", activateRec.Code, activateRec.Body.String())
	}

	nonActiveReq := httptest.NewRequest(http.MethodPut, "/api/providers/"+second.ID+"/codex-models", bytes.NewBufferString(`[{"model_id":"second-model","display_name":"Second Model","enabled":true,"position":0}]`))
	nonActiveRec := httptest.NewRecorder()
	handler.ServeHTTP(nonActiveRec, nonActiveReq)
	if nonActiveRec.Code != http.StatusOK {
		t.Fatalf("unexpected non-active codex models status: %d body=%s", nonActiveRec.Code, nonActiveRec.Body.String())
	}
	if models := readRouterCodexCatalogModels(t, filepath.Join(home, ".codex", "relay-switch-models.json")); len(models) != 0 {
		t.Fatalf("non-active provider should not sync relay models: %+v", models)
	}

	activeReq := httptest.NewRequest(http.MethodPut, "/api/providers/"+first.ID+"/codex-models", bytes.NewBufferString(`[{"model_id":"first-model","display_name":"First Model","enabled":true,"position":0}]`))
	activeRec := httptest.NewRecorder()
	handler.ServeHTTP(activeRec, activeReq)
	if activeRec.Code != http.StatusOK {
		t.Fatalf("unexpected active codex models status: %d body=%s", activeRec.Code, activeRec.Body.String())
	}
	models := readRouterCodexCatalogModels(t, filepath.Join(home, ".codex", "relay-switch-models.json"))
	if len(models) != 1 || models[0]["slug"] != "first-model" {
		t.Fatalf("active provider should sync relay models: %+v", models)
	}

	switchReq := httptest.NewRequest(http.MethodPost, "/api/providers/"+second.ID+"/activate", nil)
	switchRec := httptest.NewRecorder()
	handler.ServeHTTP(switchRec, switchReq)
	if switchRec.Code != http.StatusOK {
		t.Fatalf("unexpected second activate status: %d body=%s", switchRec.Code, switchRec.Body.String())
	}
	models = readRouterCodexCatalogModels(t, filepath.Join(home, ".codex", "relay-switch-models.json"))
	if len(models) != 1 || models[0]["slug"] != "second-model" {
		t.Fatalf("provider activation should sync new active provider models: %+v", models)
	}
}

func TestCodexModelCatalogAPIReadsAndWritesConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	handler := newTestRouter(t, nil, localgateway.RuntimeConfig{
		Host:    "127.0.0.1",
		Port:    3457,
		DataDir: filepath.Join(t.TempDir(), "runtime"),
	})

	getReq := httptest.NewRequest(http.MethodGet, "/api/tools/codex-model-catalog", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("unexpected initial get status: %d body=%s", getRec.Code, getRec.Body.String())
	}
	var state struct {
		Enabled            bool   `json:"enabled"`
		CatalogPath        string `json:"catalog_path"`
		HideOfficialModels bool   `json:"hide_official_models"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode initial state: %v", err)
	}
	if state.Enabled {
		t.Fatalf("initial state should be disabled: %+v", state)
	}
	if state.HideOfficialModels {
		t.Fatalf("initial state should not hide official models: %+v", state)
	}

	hideReq := httptest.NewRequest(http.MethodPut, "/api/tools/codex-model-catalog", bytes.NewBufferString(`{"hide_official_models":true}`))
	hideRec := httptest.NewRecorder()
	handler.ServeHTTP(hideRec, hideReq)
	if hideRec.Code != http.StatusOK {
		t.Fatalf("unexpected hide status: %d body=%s", hideRec.Code, hideRec.Body.String())
	}
	if err := json.Unmarshal(hideRec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode hide state: %v", err)
	}
	if state.Enabled {
		t.Fatalf("hide-only update should not enable model catalog: %+v", state)
	}
	if !state.HideOfficialModels {
		t.Fatalf("hide official models should be enabled: %+v", state)
	}
	if !strings.Contains(readRouterText(t, filepath.Join(home, ".codex", "config.toml")), `relay_switch_hide_official_models = true`) {
		t.Fatalf("config should contain hide official models")
	}
	if models := readRouterCodexCatalogModels(t, filepath.Join(home, ".codex", "relay-switch-model-catalog.json")); len(models) == 0 {
		t.Fatal("hide official update should write relay catalog")
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/tools/codex-model-catalog", bytes.NewBufferString(`{"enabled":true}`))
	putRec := httptest.NewRecorder()
	handler.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("unexpected enable status: %d body=%s", putRec.Code, putRec.Body.String())
	}
	if !strings.Contains(readRouterText(t, filepath.Join(home, ".codex", "config.toml")), `model_catalog_json = "`) {
		t.Fatalf("config should contain model_catalog_json after enable")
	}
	if !strings.Contains(readRouterText(t, filepath.Join(home, ".codex", "config.toml")), `relay_switch_hide_official_models = true`) {
		t.Fatalf("enabled-only update should keep hide official models")
	}
	if models := readRouterCodexCatalogModels(t, filepath.Join(home, ".codex", "relay-switch-models.json")); len(models) != 0 {
		t.Fatalf("enable should write empty relay models without active codex models: %+v", models)
	}
	if models := readRouterCodexCatalogModels(t, filepath.Join(home, ".codex", "relay-switch-model-catalog.json")); len(models) == 0 {
		t.Fatal("enable should write relay catalog")
	}

	deleteReq := httptest.NewRequest(http.MethodPut, "/api/tools/codex-model-catalog", bytes.NewBufferString(`{"enabled":false}`))
	deleteRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("unexpected disable status: %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
	if strings.Contains(readRouterText(t, filepath.Join(home, ".codex", "config.toml")), `model_catalog_json = `) {
		t.Fatalf("config should remove model_catalog_json after disable")
	}
	if !strings.Contains(readRouterText(t, filepath.Join(home, ".codex", "config.toml")), `relay_switch_hide_official_models = true`) {
		t.Fatalf("disable enabled should not clear hide official models")
	}
	if !routerFileExists(filepath.Join(home, ".codex", "relay-switch-models.json")) {
		t.Fatal("disable should keep relay models")
	}
	if !routerFileExists(filepath.Join(home, ".codex", "relay-switch-model-catalog.json")) {
		t.Fatal("disable should keep relay catalog")
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

	handler := NewRouter(providerService, healthService, nil, manager, tooling.NewService(providerService, t.TempDir()), 3456, "", gatewayHandler)

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

func createProviderViaAPI(t *testing.T, handler http.Handler, name string) provider.Provider {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/providers", bytes.NewBufferString(`{
		"name":"`+name+`",
		"base_url":"https://api.example.com/v1",
		"api_key":"sk-test",
		"auth_mode":"bearer",
		"extra_headers":{},
		"claude_code_model_map":{"opus":"","sonnet":"","haiku":""}
	}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected provider create status: %d body=%s", rec.Code, rec.Body.String())
	}

	var created provider.Provider
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created provider: %v", err)
	}
	return created
}

func readRouterCodexCatalogModels(t *testing.T, path string) []map[string]any {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read catalog %s: %v", path, err)
	}
	var payload struct {
		Models []map[string]any `json:"models"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("decode catalog %s: %v", path, err)
	}
	return payload.Models
}

func readRouterText(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func routerFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
	handler, _ := newTestRouterWithLogs(t, adapter, runtime)
	return handler
}

func newTestRouterWithLogs(t *testing.T, adapter localgateway.GatewayAdapter, runtime localgateway.RuntimeConfig) (http.Handler, *logging.Service) {
	t.Helper()

	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "router.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = sqliteStore.Close() })

	credentialStore := credential.NewInMemoryStore()
	providerService := provider.NewService(provider.NewInMemoryRepository(), credentialStore)
	healthService := health.NewService(providerService, credentialStore)
	loggingService := logging.NewService(logging.NewSQLiteRepository(sqliteStore.DB), 7, 1000)
	gatewayHandler := gateway.NewHandler(providerService, credentialStore, loggingService)
	localService := localgateway.NewService(localgateway.NewSQLiteRepository(sqliteStore.DB), credentialStore)
	if adapter == nil {
		adapter = &localgatewaySpyAdapter{
			mockGatewayAdapter: mockGatewayAdapter{},
		}
	}
	manager := localgateway.NewManager(localService, adapter, runtime)

	return NewRouter(providerService, healthService, loggingService, manager, tooling.NewService(providerService, t.TempDir()), 3456, "", gatewayHandler), loggingService
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
