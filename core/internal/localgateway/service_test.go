package localgateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/xiaoyuandev/relay-switch/core/internal/credential"
	"github.com/xiaoyuandev/relay-switch/core/internal/storage"
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
	if created.Position != 0 {
		t.Fatalf("unexpected created position: %d", created.Position)
	}

	second, err := service.CreateSource(ctx, CreateModelSourceInput{
		Name:           "Another Source",
		BaseURL:        "https://example.com/v1",
		APIKey:         "sk-test-second",
		ProviderType:   "openai-compatible",
		DefaultModelID: "gpt-4o-mini",
		Enabled:        true,
		Position:       99,
	})
	if err != nil {
		t.Fatalf("create second source: %v", err)
	}
	if second.Position != 1 {
		t.Fatalf("unexpected second source position: %d", second.Position)
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
	if updated.Position != 0 {
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
	if _, err := service.CreateSource(ctx, CreateModelSourceInput{
		Name:            "OpenAI Direct",
		BaseURL:         "https://api.openai.com/v1",
		APIKey:          "sk-test-openai",
		ProviderType:    "openai-compatible",
		DefaultModelID:  "gpt-4.1",
		ExposedModelIDs: []string{"claude-sonnet-4-0"},
		Enabled:         true,
	}); err != nil {
		t.Fatalf("create source: %v", err)
	}

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

	if _, err := service.ReplaceSelectedModels(ctx, []SelectedModel{
		{ModelID: "not-available", Position: 0},
	}); err == nil {
		t.Fatal("expected invalid selected model error")
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
	if input.Sources[0].ExternalID != source.ID {
		t.Fatalf("unexpected sync source external id: %s", input.Sources[0].ExternalID)
	}
	if input.Sources[0].APIKey != "sk-test-openai" {
		t.Fatalf("unexpected sync api key: %s", input.Sources[0].APIKey)
	}
	if len(input.SelectedModels) != 0 {
		t.Fatalf("unexpected sync selected models: %+v", input.SelectedModels)
	}
}

func TestServicePreviewSourceModels(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test-openai" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1","object":"model"},{"id":"gpt-4.1-mini","object":"model"}]}`))
	}))
	defer server.Close()

	service := NewService(nil, nil)

	items, err := service.PreviewSourceModels(context.Background(), PreviewModelSourceInput{
		BaseURL:      server.URL + "/v1",
		APIKey:       "sk-test-openai",
		ProviderType: "openai-compatible",
	})
	if err != nil {
		t.Fatalf("preview source models: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("unexpected source models length: %d", len(items))
	}
	if items[0].ID != "gpt-4.1" || items[1].ID != "gpt-4.1-mini" {
		t.Fatalf("unexpected source models: %+v", items)
	}
}

func TestServicePreviewSourceModelsUsesModelsPathOverride(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"doubao-seed-code","object":"model"}]}`))
	}))
	defer server.Close()

	service := NewService(nil, nil)

	items, err := service.PreviewSourceModels(context.Background(), PreviewModelSourceInput{
		BaseURL:      server.URL + "/api/v3",
		ModelsPath:   "/models",
		APIKey:       "sk-test-openai",
		ProviderType: "openai-compatible",
	})
	if err != nil {
		t.Fatalf("preview source models: %v", err)
	}

	if len(items) != 1 || items[0].ID != "doubao-seed-code" {
		t.Fatalf("unexpected source models: %+v", items)
	}
}

func TestServicePreviewSourceModelsRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil)

	if _, err := service.PreviewSourceModels(context.Background(), PreviewModelSourceInput{
		BaseURL:      "https://example.com/v1",
		APIKey:       "",
		ProviderType: "openai-compatible",
	}); err == nil {
		t.Fatal("expected preview source models validation error")
	}
}

func TestServiceValidateSyncInput(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil)

	if err := service.ValidateSyncInput(SyncInput{
		Sources: []SyncModelSource{
			{
				ID:             "source-a",
				ExternalID:     "source-a",
				Name:           "OpenAI",
				BaseURL:        "https://api.openai.com/v1",
				ProviderType:   "openai-compatible",
				DefaultModelID: "gpt-4.1",
				Enabled:        true,
			},
		},
		SelectedModels: []SelectedModel{{ModelID: "gpt-4.1"}},
	}); err != nil {
		t.Fatalf("validate sync input: %v", err)
	}

	if err := service.ValidateSyncInput(SyncInput{
		Sources: []SyncModelSource{
			{
				ID:             "source-a",
				ExternalID:     "source-a",
				Name:           "OpenAI",
				BaseURL:        "not-a-url",
				ProviderType:   "openai-compatible",
				DefaultModelID: "gpt-4.1",
				Enabled:        true,
			},
		},
		SelectedModels: []SelectedModel{{ModelID: "gpt-4.1"}},
	}); err == nil {
		t.Fatal("expected invalid source url error")
	}
}

func TestServiceCreateSourceRejectsInvalidFields(t *testing.T) {
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

	if _, err := service.CreateSource(context.Background(), CreateModelSourceInput{
		Name:           "Invalid Source",
		BaseURL:        "not-a-url",
		APIKey:         "sk-test-openai",
		ProviderType:   "openai-compatible",
		DefaultModelID: "gpt-4.1",
		Enabled:        true,
	}); err == nil {
		t.Fatal("expected invalid base url error")
	}

	if _, err := service.CreateSource(context.Background(), CreateModelSourceInput{
		Name:           "Invalid Source",
		BaseURL:        "https://api.openai.com/v1",
		APIKey:         "sk-test-openai",
		ProviderType:   "unsupported",
		DefaultModelID: "gpt-4.1",
		Enabled:        true,
	}); err == nil {
		t.Fatal("expected invalid provider type error")
	}
}

func TestServiceDeleteSourceNormalizesPositionsAndRemovesInvalidSelectedModels(t *testing.T) {
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
	first, err := service.CreateSource(ctx, CreateModelSourceInput{
		Name:            "OpenAI Direct",
		BaseURL:         "https://api.openai.com/v1",
		APIKey:          "sk-test-openai",
		ProviderType:    "openai-compatible",
		DefaultModelID:  "gpt-4.1",
		ExposedModelIDs: []string{"gpt-4.1-mini"},
		Enabled:         true,
	})
	if err != nil {
		t.Fatalf("create first source: %v", err)
	}
	second, err := service.CreateSource(ctx, CreateModelSourceInput{
		Name:            "Anthropic Direct",
		BaseURL:         "https://api.anthropic.com/v1",
		APIKey:          "sk-test-anthropic",
		ProviderType:    "anthropic-compatible",
		DefaultModelID:  "claude-sonnet-4-0",
		ExposedModelIDs: []string{"claude-haiku-4-0"},
		Enabled:         true,
	})
	if err != nil {
		t.Fatalf("create second source: %v", err)
	}

	if _, err := service.ReplaceSelectedModels(ctx, []SelectedModel{
		{ModelID: "gpt-4.1", Position: 0},
		{ModelID: "claude-sonnet-4-0", Position: 1},
	}); err != nil {
		t.Fatalf("replace selected models: %v", err)
	}

	if err := service.DeleteSource(ctx, first.ID); err != nil {
		t.Fatalf("delete source: %v", err)
	}

	sources, err := service.ListSources(ctx)
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("unexpected sources length: %d", len(sources))
	}
	if sources[0].ID != second.ID || sources[0].Position != 0 {
		t.Fatalf("unexpected normalized source state: %+v", sources[0])
	}

	selected, err := service.ListSelectedModels(ctx)
	if err != nil {
		t.Fatalf("list selected models: %v", err)
	}
	if len(selected) != 1 || selected[0].ModelID != "claude-sonnet-4-0" || selected[0].Position != 0 {
		t.Fatalf("unexpected selected models after delete: %+v", selected)
	}
}

func TestServiceUpdateSourceRemovesInvalidSelectedModels(t *testing.T) {
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
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	if _, err := service.ReplaceSelectedModels(ctx, []SelectedModel{
		{ModelID: "gpt-4.1-mini", Position: 0},
	}); err != nil {
		t.Fatalf("replace selected models: %v", err)
	}

	if _, err := service.UpdateSource(ctx, source.ID, UpdateModelSourceInput{
		Name:            source.Name,
		BaseURL:         source.BaseURL,
		ProviderType:    source.ProviderType,
		DefaultModelID:  source.DefaultModelID,
		ExposedModelIDs: []string{},
		Enabled:         false,
	}); err != nil {
		t.Fatalf("update source: %v", err)
	}

	selected, err := service.ListSelectedModels(ctx)
	if err != nil {
		t.Fatalf("list selected models: %v", err)
	}
	if len(selected) != 0 {
		t.Fatalf("expected selected models to be cleaned up, got %+v", selected)
	}
}
