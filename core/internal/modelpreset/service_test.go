package modelpreset

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestDecodeCatalogValidatesAndNormalizes(t *testing.T) {
	catalog, err := DecodeCatalog([]byte(`{
		"schema_version": 1,
		"updated_at": "2026-05-29",
		"presets": [
			{
				"id": "openai-gpt-4o",
				"label": "OpenAI GPT-4o",
				"aliases": ["4o", "4o", " "],
				"providers": [
					{
						"provider_type": "openai-compatible",
						"base_url": "https://api.openai.com/v1",
						"models_api": "supported",
						"model_ids": ["gpt-4o", "gpt-4o", " "]
					}
				],
				"tags": ["openai", "popular", "openai"]
			}
		]
	}`))
	if err != nil {
		t.Fatalf("DecodeCatalog returned error: %v", err)
	}

	if catalog.SchemaVersion != 1 {
		t.Fatalf("schema version = %d, want 1", catalog.SchemaVersion)
	}
	if got := catalog.Presets[0].Providers[0].ModelIDs; len(got) != 1 || got[0] != "gpt-4o" {
		t.Fatalf("model ids = %#v, want [gpt-4o]", got)
	}
	if got := catalog.Presets[0].Aliases; len(got) != 1 || got[0] != "4o" {
		t.Fatalf("aliases = %#v, want [4o]", got)
	}
	if got := catalog.Presets[0].Tags; len(got) != 2 {
		t.Fatalf("tags = %#v, want two normalized tags", got)
	}
}

func TestDecodeCatalogRejectsDuplicateIDs(t *testing.T) {
	_, err := DecodeCatalog([]byte(`{
		"schema_version": 1,
		"presets": [
			{
				"id": "duplicate",
				"label": "One",
				"providers": [
					{
						"provider_type": "openai-compatible",
						"base_url": "https://api.openai.com/v1",
						"model_ids": ["gpt-4o"]
					}
				]
			},
			{
				"id": "duplicate",
				"label": "Two",
				"providers": [
					{
						"provider_type": "openai-compatible",
						"base_url": "https://api.openai.com/v1",
						"model_ids": ["gpt-4o-mini"]
					}
				]
			}
		]
	}`))
	if err == nil {
		t.Fatal("DecodeCatalog accepted duplicate ids")
	}
}

func TestServiceRefreshWritesCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"schema_version": 1,
			"presets": [
				{
					"id": "deepseek-chat",
					"label": "DeepSeek Chat",
					"providers": [
						{
							"provider_type": "openai-compatible",
							"base_url": "https://api.deepseek.com/v1",
							"models_api": "unsupported",
							"model_ids": ["deepseek-chat"]
						}
					]
				}
			]
		}`))
	}))
	defer server.Close()

	service := NewService(filepath.Join(t.TempDir(), "model-presets.json"), server.URL)
	response, err := service.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if len(response.Presets) != 1 {
		t.Fatalf("presets length = %d, want 1", len(response.Presets))
	}

	cached := service.Get(context.Background())
	if len(cached.Presets) != 1 || cached.Presets[0].ID != "deepseek-chat" {
		t.Fatalf("cached presets = %#v", cached.Presets)
	}
}
