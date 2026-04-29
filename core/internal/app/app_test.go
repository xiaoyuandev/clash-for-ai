package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgatewaystate"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/settings"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/storage"
)

func TestPrepareLocalRuntimeStateMigratesAndCleansLegacyState(t *testing.T) {
	root := t.TempDir()
	coreDataDir := filepath.Join(root, "core-data")
	runtimeDataDir := resolveLocalRuntimeDataDir(coreDataDir)

	coreStore, err := storage.NewSQLite(filepath.Join(coreDataDir, "clash-for-ai.db"))
	if err != nil {
		t.Fatalf("NewSQLite core store returned error: %v", err)
	}
	defer coreStore.Close()

	coreCredentials, err := credential.NewFileStore(filepath.Join(coreDataDir, "credentials.json"))
	if err != nil {
		t.Fatalf("NewFileStore core credentials returned error: %v", err)
	}

	coreModelSources := modelsource.NewService(modelsource.NewSQLiteRepository(coreStore.DB), coreCredentials)
	coreLocalGatewayState := localgatewaystate.NewService(localgatewaystate.NewSQLiteRepository(coreStore.DB))
	coreSettings := settings.NewService(settings.NewSQLiteRepository(coreStore.DB))
	providers := provider.NewService(provider.NewSQLiteRepository(coreStore.DB), coreCredentials)

	if _, err := providers.EnsureSystemLocalGateway(context.Background(), "http://127.0.0.1:8788", provider.DefaultLocalGatewayCapabilities()); err != nil {
		t.Fatalf("EnsureSystemLocalGateway returned error: %v", err)
	}

	if _, err := coreModelSources.Create(context.Background(), modelsource.CreateInput{
		Name:           "Source A",
		BaseURL:        "https://api.example.com",
		ProviderType:   "openai-compatible",
		DefaultModelID: "gpt-4.1",
		Enabled:        true,
		APIKey:         "secret-key",
	}); err != nil {
		t.Fatalf("Create core model source returned error: %v", err)
	}

	initialSettings := settings.DefaultSettings()
	initialSettings.LocalGatewaySelected = []settings.SelectedModel{{ModelID: "gpt-4.1", Position: 0}}
	if _, err := coreSettings.Save(context.Background(), initialSettings); err != nil {
		t.Fatalf("Save core settings returned error: %v", err)
	}

	if err := prepareLocalRuntimeState(
		context.Background(),
		runtimeDataDir,
		coreModelSources,
		coreLocalGatewayState,
		coreSettings,
		providers,
		initialSettings,
	); err != nil {
		t.Fatalf("prepareLocalRuntimeState returned error: %v", err)
	}

	runtimeStore, err := storage.NewSQLite(filepath.Join(runtimeDataDir, "clash-for-ai.db"))
	if err != nil {
		t.Fatalf("NewSQLite runtime store returned error: %v", err)
	}
	defer runtimeStore.Close()

	runtimeCredentials, err := credential.NewFileStore(filepath.Join(runtimeDataDir, "credentials.json"))
	if err != nil {
		t.Fatalf("NewFileStore runtime credentials returned error: %v", err)
	}

	runtimeModelSources := modelsource.NewService(modelsource.NewSQLiteRepository(runtimeStore.DB), runtimeCredentials)
	runtimeLocalGatewayState := localgatewaystate.NewService(localgatewaystate.NewSQLiteRepository(runtimeStore.DB))

	runtimeSources, err := runtimeModelSources.List(context.Background())
	if err != nil {
		t.Fatalf("List runtime model sources returned error: %v", err)
	}
	if len(runtimeSources) != 1 || runtimeSources[0].DefaultModelID != "gpt-4.1" {
		t.Fatalf("unexpected runtime model sources: %+v", runtimeSources)
	}

	selected, err := runtimeLocalGatewayState.ListSelectedModels(context.Background())
	if err != nil {
		t.Fatalf("ListSelectedModels returned error: %v", err)
	}
	if len(selected) != 1 || selected[0].ModelID != "gpt-4.1" {
		t.Fatalf("unexpected runtime selected models: %+v", selected)
	}

	coreSourcesAfter, err := coreModelSources.List(context.Background())
	if err != nil {
		t.Fatalf("List core model sources returned error: %v", err)
	}
	if len(coreSourcesAfter) != 0 {
		t.Fatalf("expected core model sources to be cleaned up, got %+v", coreSourcesAfter)
	}

	coreSettingsAfter, err := coreSettings.Get(context.Background())
	if err != nil {
		t.Fatalf("Get core settings returned error: %v", err)
	}
	if len(coreSettingsAfter.LocalGatewaySelected) != 0 {
		t.Fatalf("expected core selected models to be cleared, got %+v", coreSettingsAfter.LocalGatewaySelected)
	}

	if _, err := os.Stat(localRuntimeStateMarkerPath(runtimeDataDir)); err != nil {
		t.Fatalf("expected runtime migration marker to exist: %v", err)
	}
}

func TestPrepareLocalRuntimeStateRunsOnlyOnceAfterMarker(t *testing.T) {
	root := t.TempDir()
	coreDataDir := filepath.Join(root, "core-data")
	runtimeDataDir := resolveLocalRuntimeDataDir(coreDataDir)

	coreStore, err := storage.NewSQLite(filepath.Join(coreDataDir, "clash-for-ai.db"))
	if err != nil {
		t.Fatalf("NewSQLite core store returned error: %v", err)
	}
	defer coreStore.Close()

	coreCredentials, err := credential.NewFileStore(filepath.Join(coreDataDir, "credentials.json"))
	if err != nil {
		t.Fatalf("NewFileStore core credentials returned error: %v", err)
	}

	coreModelSources := modelsource.NewService(modelsource.NewSQLiteRepository(coreStore.DB), coreCredentials)
	coreLocalGatewayState := localgatewaystate.NewService(localgatewaystate.NewSQLiteRepository(coreStore.DB))
	coreSettings := settings.NewService(settings.NewSQLiteRepository(coreStore.DB))
	providers := provider.NewService(provider.NewSQLiteRepository(coreStore.DB), coreCredentials)

	if _, err := providers.EnsureSystemLocalGateway(context.Background(), "http://127.0.0.1:8788", provider.DefaultLocalGatewayCapabilities()); err != nil {
		t.Fatalf("EnsureSystemLocalGateway returned error: %v", err)
	}

	initialSettings := settings.DefaultSettings()
	if err := prepareLocalRuntimeState(
		context.Background(),
		runtimeDataDir,
		coreModelSources,
		coreLocalGatewayState,
		coreSettings,
		providers,
		initialSettings,
	); err != nil {
		t.Fatalf("first prepareLocalRuntimeState returned error: %v", err)
	}

	if _, err := coreModelSources.Create(context.Background(), modelsource.CreateInput{
		Name:           "Late Source",
		BaseURL:        "https://late.example.com",
		ProviderType:   "openai-compatible",
		DefaultModelID: "gpt-4.1-mini",
		Enabled:        true,
		APIKey:         "late-key",
	}); err != nil {
		t.Fatalf("Create late core model source returned error: %v", err)
	}

	if err := prepareLocalRuntimeState(
		context.Background(),
		runtimeDataDir,
		coreModelSources,
		coreLocalGatewayState,
		coreSettings,
		providers,
		initialSettings,
	); err != nil {
		t.Fatalf("second prepareLocalRuntimeState returned error: %v", err)
	}

	runtimeStore, err := storage.NewSQLite(filepath.Join(runtimeDataDir, "clash-for-ai.db"))
	if err != nil {
		t.Fatalf("NewSQLite runtime store returned error: %v", err)
	}
	defer runtimeStore.Close()

	runtimeCredentials, err := credential.NewFileStore(filepath.Join(runtimeDataDir, "credentials.json"))
	if err != nil {
		t.Fatalf("NewFileStore runtime credentials returned error: %v", err)
	}

	runtimeModelSources := modelsource.NewService(modelsource.NewSQLiteRepository(runtimeStore.DB), runtimeCredentials)
	runtimeSources, err := runtimeModelSources.List(context.Background())
	if err != nil {
		t.Fatalf("List runtime model sources returned error: %v", err)
	}
	if len(runtimeSources) != 0 {
		t.Fatalf("expected no runtime sources to be migrated after marker, got %+v", runtimeSources)
	}

	coreSourcesAfter, err := coreModelSources.List(context.Background())
	if err != nil {
		t.Fatalf("List core model sources returned error: %v", err)
	}
	if len(coreSourcesAfter) != 1 || coreSourcesAfter[0].Name != "Late Source" {
		t.Fatalf("expected core sources to remain untouched after marker, got %+v", coreSourcesAfter)
	}
}
