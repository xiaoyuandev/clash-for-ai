package extension

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xiaoyuandev/relay-switch/core/internal/storage"
)

func TestServiceScanLoadsValidManifestAndPersistsEnableState(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	writeTestManifest(t, service.sources[0].Dir, "relay-switch.markdown-archive", `{
		"manifestVersion": 1,
		"id": "relay-switch.markdown-archive",
		"name": "Markdown Archive",
		"version": "0.1.0",
		"description": "Exports local transcripts.",
		"publisher": "relay-switch",
		"engines": {"relaySwitch": ">=1.0.0"},
		"entry": {"type": "none"},
		"contributes": {"commands": [{"id": "markdownArchive.syncNow", "title": "Sync"}]},
		"permissions": []
	}`)

	items, err := service.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan extensions: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one plugin, got %+v", items)
	}
	if items[0].Status != PluginStatusInstalled || items[0].Enabled {
		t.Fatalf("unexpected initial plugin state: %+v", items[0])
	}
	if items[0].Permissions == nil {
		t.Fatalf("expected permissions to serialize as an empty array, got nil")
	}
	if items[0].Contributes["commands"] != 1 {
		t.Fatalf("expected command contribution count, got %+v", items[0].Contributes)
	}

	enabled, err := service.Enable(context.Background(), "relay-switch.markdown-archive")
	if err != nil {
		t.Fatalf("enable plugin: %v", err)
	}
	if !enabled.Enabled || enabled.Status != PluginStatusEnabled {
		t.Fatalf("unexpected enabled plugin state: %+v", enabled)
	}

	items, err = service.Scan(context.Background())
	if err != nil {
		t.Fatalf("rescan extensions: %v", err)
	}
	if len(items) != 1 || !items[0].Enabled || items[0].Status != PluginStatusEnabled {
		t.Fatalf("expected rescan to preserve enabled state, got %+v", items)
	}

	disabled, err := service.Disable(context.Background(), "relay-switch.markdown-archive")
	if err != nil {
		t.Fatalf("disable plugin: %v", err)
	}
	if disabled.Enabled || disabled.Status != PluginStatusDisabled {
		t.Fatalf("unexpected disabled plugin state: %+v", disabled)
	}
}

func TestServiceScanMarksUnknownPermissionInvalid(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	writeTestManifest(t, service.sources[0].Dir, "relay-switch.bad-permission", `{
		"manifestVersion": 1,
		"id": "relay-switch.bad-permission",
		"name": "Bad Permission",
		"version": "0.1.0",
		"entry": {"type": "none"},
		"contributes": {},
		"permissions": ["provider.secret.read"]
	}`)

	items, err := service.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan extensions: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one invalid plugin, got %+v", items)
	}
	if items[0].Status != PluginStatusInvalid {
		t.Fatalf("expected invalid status, got %+v", items[0])
	}
	if !strings.Contains(items[0].LastError, "unknown permission") {
		t.Fatalf("expected validation error, got %+v", items[0])
	}
}

func TestServiceScanDuplicatePluginIDFailsClosed(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	manifest := `{
		"manifestVersion": 1,
		"id": "relay-switch.duplicate",
		"name": "Duplicate",
		"version": "0.1.0",
		"entry": {"type": "none"},
		"contributes": {},
		"permissions": []
	}`
	writeTestManifest(t, service.sources[0].Dir, "first", manifest)
	writeTestManifest(t, service.sources[0].Dir, "second", manifest)

	items, err := service.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan extensions: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one stored plugin id, got %+v", items)
	}
	if items[0].Status != PluginStatusInvalid {
		t.Fatalf("expected duplicate plugin to be invalid, got %+v", items[0])
	}
	if !strings.Contains(items[0].LastError, "duplicate plugin id") {
		t.Fatalf("expected duplicate error, got %+v", items[0])
	}
}

func TestManifestUnknownContributionWarnsButLoads(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	writeTestManifest(t, service.sources[0].Dir, "relay-switch.future", `{
		"manifestVersion": 1,
		"id": "relay-switch.future",
		"name": "Future",
		"version": "0.1.0",
		"entry": {"type": "none"},
		"contributes": {"futureThings": [{"id": "one"}]},
		"permissions": []
	}`)

	items, err := service.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan extensions: %v", err)
	}
	if len(items) != 1 || items[0].Status != PluginStatusInstalled {
		t.Fatalf("expected future contribution to load, got %+v", items)
	}
	if len(items[0].Warnings) != 1 || !strings.Contains(items[0].Warnings[0], "futureThings") {
		t.Fatalf("expected unsupported contribution warning, got %+v", items[0].Warnings)
	}
}

func newTestExtensionService(t *testing.T) *Service {
	t.Helper()

	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "extensions.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = sqliteStore.Close() })

	root := filepath.Join(t.TempDir(), "extensions")
	return NewService(NewSQLiteRepository(sqliteStore.DB), []ScanSource{
		{
			Scope: PluginScopeUser,
			Dir:   root,
		},
	})
}

func writeTestManifest(t *testing.T, root string, pluginDir string, content string) {
	t.Helper()

	dir := filepath.Join(root, pluginDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(content), 0o644); err != nil {
		t.Fatalf("write plugin manifest: %v", err)
	}
}
