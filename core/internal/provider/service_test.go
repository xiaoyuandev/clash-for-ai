package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/xiaoyuandev/relay-switch/core/internal/credential"
	"github.com/xiaoyuandev/relay-switch/core/internal/storage"
)

func TestEnsureManagedLocalGatewayCreatesSystemProvider(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	ctx := context.Background()

	item, err := service.EnsureManagedLocalGateway(ctx, "Local Gateway", "http://127.0.0.1:3457/v1", "dummy")
	if err != nil {
		t.Fatalf("ensure managed local gateway: %v", err)
	}

	if !item.IsSystemManaged || item.IsEditable || item.IsDeletable {
		t.Fatalf("unexpected provider management flags: %+v", item)
	}
	if item.RuntimeKind != RuntimeKindManagedLocalGate {
		t.Fatalf("unexpected runtime kind: %s", item.RuntimeKind)
	}

	list, err := service.List(ctx)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("unexpected provider count: %d", len(list))
	}
}

func TestManagedLocalGatewayCannotBeUpdatedOrDeleted(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	ctx := context.Background()

	item, err := service.EnsureManagedLocalGateway(ctx, "Local Gateway", "http://127.0.0.1:3457/v1", "dummy")
	if err != nil {
		t.Fatalf("ensure managed local gateway: %v", err)
	}

	if _, err := service.Update(ctx, item.ID, UpdateInput{
		Name:    "Changed",
		BaseURL: "http://127.0.0.1:9999/v1",
		APIKey:  "changed",
	}); err != ErrProviderNotEditable {
		t.Fatalf("unexpected update error: %v", err)
	}

	if err := service.Delete(ctx, item.ID); err != ErrProviderNotDeletable {
		t.Fatalf("unexpected delete error: %v", err)
	}
}

func TestManagedLocalGatewayAllowsClaudeCodeModelMapUpdate(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	ctx := context.Background()

	item, err := service.EnsureManagedLocalGateway(ctx, "Local Gateway", "http://127.0.0.1:3457/v1", "dummy")
	if err != nil {
		t.Fatalf("ensure managed local gateway: %v", err)
	}

	updated, err := service.Update(ctx, item.ID, UpdateInput{
		Name:         item.Name,
		BaseURL:      item.BaseURL,
		APIKey:       "dummy",
		AuthMode:     item.AuthMode,
		ExtraHeaders: map[string]string{},
		ClaudeCodeModelMap: ClaudeCodeModelMap{
			Opus:   "gpt-5",
			Sonnet: "gpt-5-mini",
			Haiku:  "gpt-5-nano",
		},
	})
	if err != nil {
		t.Fatalf("update managed local gateway claude slots: %v", err)
	}

	if updated.ClaudeCodeModelMap.Opus != "gpt-5" ||
		updated.ClaudeCodeModelMap.Sonnet != "gpt-5-mini" ||
		updated.ClaudeCodeModelMap.Haiku != "gpt-5-nano" {
		t.Fatalf("unexpected claude code model map: %+v", updated.ClaudeCodeModelMap)
	}
	if updated.BaseURL != item.BaseURL {
		t.Fatalf("unexpected base_url change: %s", updated.BaseURL)
	}
}

func TestFetchModelsAcceptsEmptyOpenAIModelsResponse(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/models" {
			t.Fatalf("unexpected models path: %s", req.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer upstream.Close()

	service := newTestService(t)
	ctx := context.Background()
	item, err := service.Create(ctx, CreateInput{
		Name:    "Empty Models",
		BaseURL: upstream.URL + "/v1",
		APIKey:  "sk-test",
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	models, err := service.FetchModels(ctx, item.ID)
	if err != nil {
		t.Fatalf("fetch models: %v", err)
	}
	if len(models) != 0 {
		t.Fatalf("expected empty models, got %+v", models)
	}
}

func TestFetchModelsUsesModelsPathOverride(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v3/models" {
			t.Fatalf("unexpected models path: %s", req.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"doubao-seed-code"}]}`))
	}))
	defer upstream.Close()

	service := newTestService(t)
	ctx := context.Background()
	item, err := service.Create(ctx, CreateInput{
		Name:       "Volcengine Coding Plan",
		BaseURL:    upstream.URL + "/api/v3",
		ModelsPath: "/models",
		APIKey:     "sk-test",
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	models, err := service.FetchModels(ctx, item.ID)
	if err != nil {
		t.Fatalf("fetch models: %v", err)
	}
	if len(models) != 1 || models[0].ID != "doubao-seed-code" {
		t.Fatalf("unexpected models: %+v", models)
	}
}

func TestResolveModelsPathWithOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		basePath   string
		modelsPath string
		want       string
	}{
		{
			name:       "default path does not duplicate v1 base path",
			basePath:   "/v1",
			modelsPath: "/v1/models",
			want:       "/v1/models",
		},
		{
			name:       "models path appends to custom base path",
			basePath:   "/api/v3",
			modelsPath: "/models",
			want:       "/api/v3/models",
		},
		{
			name:       "explicit full custom path is preserved",
			basePath:   "/api/v3",
			modelsPath: "/api/v3/models",
			want:       "/api/v3/models",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveModelsPath(test.basePath, test.modelsPath); got != test.want {
				t.Fatalf("unexpected models path: got %q want %q", got, test.want)
			}
		})
	}
}

func TestTestModelAvailabilityUsesOpenAIChatCompletions(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected model test path: %s", req.URL.Path)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		var payload struct {
			Model     string `json:"model"`
			MaxTokens int    `json:"max_tokens"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode model test payload: %v", err)
		}
		if payload.Model != "gpt-4.1" || payload.MaxTokens != 1 {
			t.Fatalf("unexpected model test payload: %+v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-test","choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer upstream.Close()

	service := newTestService(t)
	ctx := context.Background()
	item, err := service.Create(ctx, CreateInput{
		Name:     "OpenAI Relay",
		BaseURL:  upstream.URL + "/v1",
		APIKey:   "sk-test",
		AuthMode: AuthModeBearer,
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	result, err := service.TestModelAvailability(ctx, item.ID, "gpt-4.1")
	if err != nil {
		t.Fatalf("test model availability: %v", err)
	}
	if result.Status != "ok" || result.StatusCode != http.StatusOK {
		t.Fatalf("unexpected model test result: %+v", result)
	}
	if result.Protocol != "openai-compatible" || result.RequestPath != "/v1/chat/completions" {
		t.Fatalf("unexpected model test protocol/path: %+v", result)
	}
}

func TestTestModelAvailabilityUsesAnthropicMessages(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected model test path: %s", req.URL.Path)
		}
		if got := req.Header.Get("x-api-key"); got != "sk-ant-test" {
			t.Fatalf("unexpected x-api-key header: %s", got)
		}
		if got := req.Header.Get("anthropic-version"); got == "" {
			t.Fatalf("expected anthropic-version header")
		}
		var payload struct {
			Model     string `json:"model"`
			MaxTokens int    `json:"max_tokens"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode model test payload: %v", err)
		}
		if payload.Model != "claude-sonnet-4-20250514" || payload.MaxTokens != 1 {
			t.Fatalf("unexpected model test payload: %+v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg-test","type":"message","content":[{"type":"text","text":"o"}]}`))
	}))
	defer upstream.Close()

	service := newTestService(t)
	ctx := context.Background()
	item, err := service.Create(ctx, CreateInput{
		Name:     "Anthropic Relay",
		BaseURL:  upstream.URL + "/v1",
		APIKey:   "sk-ant-test",
		AuthMode: AuthModeAPIKey,
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	result, err := service.TestModelAvailability(ctx, item.ID, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("test model availability: %v", err)
	}
	if result.Status != "ok" || result.StatusCode != http.StatusOK {
		t.Fatalf("unexpected model test result: %+v", result)
	}
	if result.Protocol != "anthropic-compatible" || result.RequestPath != "/v1/messages" {
		t.Fatalf("unexpected model test protocol/path: %+v", result)
	}
}

func TestReplaceCodexModelsNormalizesAndPersists(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	ctx := context.Background()
	item, err := service.Create(ctx, CreateInput{
		Name:    "Codex Models",
		BaseURL: "https://api.example.com/v1",
		APIKey:  "sk-test",
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	contextWindow := 128000
	saved, err := service.ReplaceCodexModels(ctx, item.ID, []CodexModel{
		{ModelID: " qwen-plus ", DisplayName: "", Enabled: true, ContextWindow: &contextWindow},
		{ModelID: "", DisplayName: "skip", Enabled: true},
		{ModelID: "qwen-plus", DisplayName: "duplicate", Enabled: true},
		{ModelID: " deepseek-chat ", DisplayName: " DeepSeek Chat ", Enabled: true},
	})
	if err != nil {
		t.Fatalf("replace codex models: %v", err)
	}
	if len(saved) != 2 {
		t.Fatalf("unexpected saved models: %+v", saved)
	}
	if saved[0].ProviderID != item.ID || saved[0].ModelID != "qwen-plus" || saved[0].DisplayName != "qwen-plus" || saved[0].Position != 0 {
		t.Fatalf("unexpected first model: %+v", saved[0])
	}
	if saved[0].ContextWindow == nil || *saved[0].ContextWindow != contextWindow {
		t.Fatalf("unexpected context window: %+v", saved[0].ContextWindow)
	}
	if saved[1].ModelID != "deepseek-chat" || saved[1].DisplayName != "DeepSeek Chat" || saved[1].Position != 1 {
		t.Fatalf("unexpected second model: %+v", saved[1])
	}

	list, err := service.ListCodexModels(ctx, item.ID)
	if err != nil {
		t.Fatalf("list codex models: %v", err)
	}
	if len(list) != 2 || list[0].ModelID != "qwen-plus" || list[1].ModelID != "deepseek-chat" {
		t.Fatalf("unexpected persisted models: %+v", list)
	}
}

func TestReplaceCodexModelsOverwritesAndDeleteProviderClearsModels(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	ctx := context.Background()
	item, err := service.Create(ctx, CreateInput{
		Name:    "Codex Models",
		BaseURL: "https://api.example.com/v1",
		APIKey:  "sk-test",
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	if _, err := service.ReplaceCodexModels(ctx, item.ID, []CodexModel{
		{ModelID: "first", DisplayName: "First", Enabled: true},
		{ModelID: "second", DisplayName: "Second", Enabled: true},
	}); err != nil {
		t.Fatalf("first replace codex models: %v", err)
	}
	if _, err := service.ReplaceCodexModels(ctx, item.ID, []CodexModel{
		{ModelID: "third", DisplayName: "Third", Enabled: true},
	}); err != nil {
		t.Fatalf("second replace codex models: %v", err)
	}

	list, err := service.ListCodexModels(ctx, item.ID)
	if err != nil {
		t.Fatalf("list codex models: %v", err)
	}
	if len(list) != 1 || list[0].ModelID != "third" {
		t.Fatalf("replace should overwrite old models: %+v", list)
	}

	if err := service.Delete(ctx, item.ID); err != nil {
		t.Fatalf("delete provider: %v", err)
	}
	if _, err := service.ListCodexModels(ctx, item.ID); err != ErrProviderNotFound {
		t.Fatalf("expected provider not found after delete, got %v", err)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()

	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "provider.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = sqliteStore.Close() })

	return NewService(NewSQLiteRepository(sqliteStore.DB), credential.NewInMemoryStore())
}
