package provider

import (
	"context"
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

func newTestService(t *testing.T) *Service {
	t.Helper()

	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "provider.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = sqliteStore.Close() })

	return NewService(NewSQLiteRepository(sqliteStore.DB), credential.NewInMemoryStore())
}
