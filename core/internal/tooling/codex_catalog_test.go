package tooling

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xiaoyuandev/relay-switch/core/internal/credential"
	"github.com/xiaoyuandev/relay-switch/core/internal/provider"
	"github.com/xiaoyuandev/relay-switch/core/internal/storage"
)

type roundTripClient func(*http.Request) (*http.Response, error)

func (f roundTripClient) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSyncCodexModelCatalogWritesRelaySwitchFiles(t *testing.T) {
	service, providerService := newCodexCatalogTestService(t)
	ctx := context.Background()

	item, err := providerService.Create(ctx, provider.CreateInput{
		Name:    "Third Party",
		BaseURL: "https://api.example.com/v1",
		APIKey:  "sk-test",
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if _, err := providerService.Activate(ctx, item.ID); err != nil {
		t.Fatalf("activate provider: %v", err)
	}
	contextWindow := 64000
	if _, err := providerService.ReplaceCodexModels(ctx, item.ID, []provider.CodexModel{
		{ModelID: " qwen-plus ", DisplayName: " Qwen Plus ", Enabled: true, ContextWindow: &contextWindow},
	}); err != nil {
		t.Fatalf("replace codex models: %v", err)
	}

	if err := service.SyncCodexModelCatalog(ctx); err != nil {
		t.Fatalf("sync codex model catalog: %v", err)
	}

	relayModels := readCatalogFile(t, codexRelaySwitchModelsPath())
	if len(relayModels.Models) != 1 {
		t.Fatalf("unexpected relay model count: %+v", relayModels.Models)
	}
	if relayModels.Models[0]["slug"] != "qwen-plus" ||
		relayModels.Models[0]["display_name"] != "Qwen Plus" ||
		relayModels.Models[0]["visibility"] != "list" ||
		relayModels.Models[0]["supported_in_api"] != true {
		t.Fatalf("unexpected relay model: %+v", relayModels.Models[0])
	}
	if relayModels.Models[0]["context_window"] != float64(contextWindow) {
		t.Fatalf("unexpected context window: %+v", relayModels.Models[0]["context_window"])
	}

	fullCatalog := readCatalogFile(t, codexRelaySwitchCatalogPath())
	if len(fullCatalog.Models) <= len(relayModels.Models) {
		t.Fatalf("full catalog should include default models and relay models: %d", len(fullCatalog.Models))
	}
	if fullCatalog.Models[len(fullCatalog.Models)-1]["slug"] != "qwen-plus" {
		t.Fatalf("relay model should be appended to catalog: %+v", fullCatalog.Models[len(fullCatalog.Models)-1])
	}

	if fileExists(codexConfigPath()) {
		t.Fatalf("sync should not write codex config: %s", codexConfigPath())
	}
}

func TestSyncCodexModelCatalogUsesDefaultCatalogWhenNoActiveModels(t *testing.T) {
	service, providerService := newCodexCatalogTestService(t)
	ctx := context.Background()

	item, err := providerService.Create(ctx, provider.CreateInput{
		Name:    "Empty",
		BaseURL: "https://api.example.com/v1",
		APIKey:  "sk-test",
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if _, err := providerService.Activate(ctx, item.ID); err != nil {
		t.Fatalf("activate provider: %v", err)
	}

	if err := service.SyncCodexModelCatalog(ctx); err != nil {
		t.Fatalf("sync codex model catalog: %v", err)
	}

	relayModels := readCatalogFile(t, codexRelaySwitchModelsPath())
	if len(relayModels.Models) != 0 {
		t.Fatalf("relay models should be empty: %+v", relayModels.Models)
	}
	fullCatalog := readCatalogFile(t, codexRelaySwitchCatalogPath())
	if len(fullCatalog.Models) == 0 {
		t.Fatal("default catalog should not be empty")
	}
	if fileExists(codexConfigPath()) {
		t.Fatalf("sync should not write codex config: %s", codexConfigPath())
	}
}

func TestRefreshCodexDefaultModelsCatalogWritesCache(t *testing.T) {
	service, _ := newCodexCatalogTestService(t)
	service.catalogURL = "https://example.test/models.json"
	service.catalogClient = roundTripClient(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"models":[{"slug":"fresh","display_name":"Fresh","visibility":"list","supported_in_api":true}]}`)),
		}, nil
	})

	if err := service.RefreshCodexDefaultModelsCatalog(context.Background()); err != nil {
		t.Fatalf("refresh default models catalog: %v", err)
	}

	catalog := readCatalogFile(t, service.codexDefaultModelsCachePath())
	if len(catalog.Models) != 1 || catalog.Models[0]["slug"] != "fresh" {
		t.Fatalf("unexpected cached catalog: %+v", catalog.Models)
	}
}

func TestBootstrapCodexModelCatalogDoesNotSyncRelayFilesOrConfig(t *testing.T) {
	service, _ := newCodexCatalogTestService(t)
	service.catalogURL = ""

	service.BootstrapCodexModelCatalog(context.Background())

	if fileExists(codexRelaySwitchModelsPath()) {
		t.Fatalf("bootstrap should not write relay models: %s", codexRelaySwitchModelsPath())
	}
	if fileExists(codexRelaySwitchCatalogPath()) {
		t.Fatalf("bootstrap should not write relay catalog: %s", codexRelaySwitchCatalogPath())
	}
	if fileExists(codexConfigPath()) {
		t.Fatalf("bootstrap should not write codex config: %s", codexConfigPath())
	}
}

func TestSyncCodexModelCatalogPrefersDefaultCache(t *testing.T) {
	service, _ := newCodexCatalogTestService(t)
	writeTestText(t, service.codexDefaultModelsCachePath(), `{"models":[{"slug":"cached","display_name":"Cached","visibility":"list","supported_in_api":true}]}`+"\n")

	if err := service.SyncCodexModelCatalog(context.Background()); err != nil {
		t.Fatalf("sync codex model catalog: %v", err)
	}

	catalog := readCatalogFile(t, codexRelaySwitchCatalogPath())
	if len(catalog.Models) != 1 || catalog.Models[0]["slug"] != "cached" {
		t.Fatalf("sync should use cached default catalog: %+v", catalog.Models)
	}
}

func TestCodexModelCatalogStateFollowsConfig(t *testing.T) {
	service, _ := newCodexCatalogTestService(t)

	state, err := service.GetCodexModelCatalogState()
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if state.Enabled {
		t.Fatalf("missing config should be disabled: %+v", state)
	}
	if state.CatalogPath != codexRelaySwitchCatalogPath() {
		t.Fatalf("unexpected catalog path: %+v", state)
	}

	writeTestText(t, codexConfigPath(), `model_catalog_json = "/tmp/other.json"`+"\n")
	state, err = service.GetCodexModelCatalogState()
	if err != nil {
		t.Fatalf("get state with other catalog: %v", err)
	}
	if state.Enabled {
		t.Fatalf("non-relay catalog should be disabled: %+v", state)
	}

	state, err = service.SetCodexModelCatalogEnabled(true)
	if err != nil {
		t.Fatalf("enable catalog: %v", err)
	}
	if !state.Enabled {
		t.Fatalf("enabled state expected: %+v", state)
	}
	if !strings.Contains(readTestText(t, codexConfigPath()), `model_catalog_json = "`+codexRelaySwitchCatalogPath()+`"`) {
		t.Fatalf("config missing relay catalog:\n%s", readTestText(t, codexConfigPath()))
	}

	state, err = service.SetCodexModelCatalogEnabled(false)
	if err != nil {
		t.Fatalf("disable catalog: %v", err)
	}
	if state.Enabled {
		t.Fatalf("disabled state expected: %+v", state)
	}
	if strings.Contains(readTestText(t, codexConfigPath()), `model_catalog_json = `) {
		t.Fatalf("config should remove model_catalog_json:\n%s", readTestText(t, codexConfigPath()))
	}
}

func TestRemoveTopLevelTomlKeyDoesNotRemoveSimilarOrNestedKey(t *testing.T) {
	content := strings.Join([]string{
		`model_catalog_json = "/tmp/catalog.json"`,
		`model_catalog_json_extra = "keep"`,
		"",
		"[model_providers.OpenAI]",
		`model_catalog_json = "nested"`,
	}, "\n") + "\n"

	updated := removeTopLevelTomlKey(content, "model_catalog_json")
	if strings.Contains(updated, `model_catalog_json = "/tmp/catalog.json"`) {
		t.Fatalf("top-level key should be removed:\n%s", updated)
	}
	if !strings.Contains(updated, `model_catalog_json_extra = "keep"`) {
		t.Fatalf("similar key should be preserved:\n%s", updated)
	}
	if !strings.Contains(updated, `model_catalog_json = "nested"`) {
		t.Fatalf("nested key should be preserved:\n%s", updated)
	}
}

func TestSetTopLevelTomlStringDoesNotReplaceSimilarKey(t *testing.T) {
	content := strings.Join([]string{
		`model_catalog_json_extra = "keep"`,
		"",
		"[model_providers.OpenAI]",
		`base_url = "https://api.openai.com/v1"`,
	}, "\n") + "\n"

	updated := setTopLevelTomlString(content, "model_catalog_json", "/tmp/catalog.json")
	if !strings.Contains(updated, `model_catalog_json_extra = "keep"`) {
		t.Fatalf("similar key should be preserved:\n%s", updated)
	}
	if !strings.Contains(updated, `model_catalog_json = "/tmp/catalog.json"`) {
		t.Fatalf("target key should be inserted:\n%s", updated)
	}
}

func TestSetTopLevelTomlStringReplacesExistingKey(t *testing.T) {
	content := strings.Join([]string{
		`model_catalog_json = "/tmp/old.json"`,
		`model_catalog_json_extra = "keep"`,
	}, "\n") + "\n"

	updated := setTopLevelTomlString(content, "model_catalog_json", "/tmp/new.json")
	if !strings.Contains(updated, `model_catalog_json = "/tmp/new.json"`) {
		t.Fatalf("target key should be replaced:\n%s", updated)
	}
	if strings.Contains(updated, `model_catalog_json = "/tmp/old.json"`) {
		t.Fatalf("old target value should be removed:\n%s", updated)
	}
	if !strings.Contains(updated, `model_catalog_json_extra = "keep"`) {
		t.Fatalf("similar key should be preserved:\n%s", updated)
	}
}

func newCodexCatalogTestService(t *testing.T) (*Service, *provider.Service) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "provider.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = sqliteStore.Close() })

	providerService := provider.NewService(provider.NewSQLiteRepository(sqliteStore.DB), credential.NewInMemoryStore())
	service := NewService(providerService, t.TempDir())
	service.catalogURL = ""
	return service, providerService
}

func readCatalogFile(t *testing.T, path string) codexModelsCatalog {
	t.Helper()
	var catalog codexModelsCatalog
	if err := json.Unmarshal([]byte(readTestText(t, path)), &catalog); err != nil {
		t.Fatalf("decode catalog %s: %v", path, err)
	}
	return catalog
}

func readTestText(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func writeTestText(t *testing.T, path string, content string) {
	t.Helper()
	if err := ensureDir(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
