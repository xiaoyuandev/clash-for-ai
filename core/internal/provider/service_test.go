package provider

import (
	"context"
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

func newTestService(t *testing.T) *Service {
	t.Helper()

	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "provider.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = sqliteStore.Close() })

	return NewService(NewSQLiteRepository(sqliteStore.DB), credential.NewInMemoryStore())
}
