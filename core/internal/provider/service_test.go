package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/storage"
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

func newTestService(t *testing.T) *Service {
	t.Helper()

	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "provider.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = sqliteStore.Close() })

	return NewService(NewSQLiteRepository(sqliteStore.DB), credential.NewInMemoryStore())
}
