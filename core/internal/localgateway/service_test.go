package localgateway

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/storage"
)

func TestServiceCreateAndUpdateSource(t *testing.T) {
	t.Parallel()

	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "phase1.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	defer sqliteStore.Close()

	service := NewService(
		NewSQLiteRepository(sqliteStore.DB),
		credential.NewInMemoryStore(),
	)

	ctx := context.Background()
	created, err := service.CreateSource(ctx, CreateModelSourceInput{
		Name:            "OpenAI Direct",
		BaseURL:         "https://api.openai.com/v1",
		APIKey:          "sk-test-openai",
		ProviderType:    "openai-compatible",
		DefaultModelID:  "gpt-4.1",
		ExposedModelIDs: []string{"gpt-4.1-mini", "gpt-4.1-mini"},
		Enabled:         true,
		Position:        0,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	if created.ID == "" {
		t.Fatal("expected source id")
	}
	if created.APIKeyRef == "" {
		t.Fatal("expected api key ref")
	}
	if created.APIKeyMasked == "" {
		t.Fatal("expected api key masked")
	}
	if len(created.ExposedModelIDs) != 1 || created.ExposedModelIDs[0] != "gpt-4.1-mini" {
		t.Fatalf("unexpected exposed model ids: %+v", created.ExposedModelIDs)
	}

	updated, err := service.UpdateSource(ctx, created.ID, UpdateModelSourceInput{
		Name:            "Anthropic Direct",
		BaseURL:         "https://api.anthropic.com",
		APIKey:          "sk-test-anthropic",
		ProviderType:    "anthropic-compatible",
		DefaultModelID:  "claude-sonnet-4-0",
		ExposedModelIDs: []string{"claude-haiku-4-0"},
		Enabled:         false,
		Position:        1,
	})
	if err != nil {
		t.Fatalf("update source: %v", err)
	}

	if updated.ProviderType != "anthropic-compatible" {
		t.Fatalf("unexpected provider type: %s", updated.ProviderType)
	}
	if updated.DefaultModelID != "claude-sonnet-4-0" {
		t.Fatalf("unexpected default model id: %s", updated.DefaultModelID)
	}
	if updated.Enabled {
		t.Fatal("expected source disabled")
	}
	if updated.Position != 1 {
		t.Fatalf("unexpected position: %d", updated.Position)
	}
}

func TestServiceReplaceSelectedModels(t *testing.T) {
	t.Parallel()

	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "phase1.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	defer sqliteStore.Close()

	service := NewService(
		NewSQLiteRepository(sqliteStore.DB),
		credential.NewInMemoryStore(),
	)

	ctx := context.Background()
	items, err := service.ReplaceSelectedModels(ctx, []SelectedModel{
		{ModelID: "gpt-4.1", Position: 8},
		{ModelID: " ", Position: 9},
		{ModelID: "claude-sonnet-4-0", Position: 10},
	})
	if err != nil {
		t.Fatalf("replace selected models: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("unexpected selected models length: %d", len(items))
	}
	if items[0].ModelID != "gpt-4.1" || items[0].Position != 0 {
		t.Fatalf("unexpected first selected model: %+v", items[0])
	}
	if items[1].ModelID != "claude-sonnet-4-0" || items[1].Position != 1 {
		t.Fatalf("unexpected second selected model: %+v", items[1])
	}
}

func TestServiceBuildSyncInput(t *testing.T) {
	t.Parallel()

	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "phase1.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	defer sqliteStore.Close()

	service := NewService(
		NewSQLiteRepository(sqliteStore.DB),
		credential.NewInMemoryStore(),
	)

	ctx := context.Background()
	source, err := service.CreateSource(ctx, CreateModelSourceInput{
		Name:            "OpenAI Direct",
		BaseURL:         "https://api.openai.com/v1",
		APIKey:          "sk-test-openai",
		ProviderType:    "openai-compatible",
		DefaultModelID:  "gpt-4.1",
		ExposedModelIDs: []string{"gpt-4.1-mini"},
		Enabled:         true,
		Position:        0,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	if _, err := service.ReplaceSelectedModels(ctx, []SelectedModel{
		{ModelID: "gpt-4.1", Position: 0},
	}); err != nil {
		t.Fatalf("replace selected models: %v", err)
	}

	input, err := service.BuildSyncInput(ctx)
	if err != nil {
		t.Fatalf("build sync input: %v", err)
	}

	if len(input.Sources) != 1 {
		t.Fatalf("unexpected sync source count: %d", len(input.Sources))
	}
	if input.Sources[0].ID != source.ID {
		t.Fatalf("unexpected sync source id: %s", input.Sources[0].ID)
	}
	if input.Sources[0].APIKey != "sk-test-openai" {
		t.Fatalf("unexpected sync api key: %s", input.Sources[0].APIKey)
	}
	if len(input.SelectedModels) != 1 || input.SelectedModels[0].ModelID != "gpt-4.1" {
		t.Fatalf("unexpected sync selected models: %+v", input.SelectedModels)
	}
}
