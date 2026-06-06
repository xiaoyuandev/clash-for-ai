package extension

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
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

func TestServiceSettingsSaveLoadValidateAndAudit(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	writeTestManifest(t, service.sources[0].Dir, "relay-switch.settings", `{
		"manifestVersion": 1,
		"id": "relay-switch.settings",
		"name": "Settings",
		"version": "0.1.0",
		"entry": {"type": "none"},
		"contributes": {
			"settings": {
				"type": "object",
				"properties": {
					"outputDirectory": {
						"type": "string",
						"title": "Output Directory",
						"default": ""
					},
					"enabled": {
						"type": "boolean",
						"default": true
					},
					"tags": {
						"type": "array",
						"items": {"type": "string"},
						"default": []
					}
				},
				"required": ["outputDirectory"]
			}
		},
		"permissions": []
	}`)

	if _, err := service.Scan(context.Background()); err != nil {
		t.Fatalf("scan extensions: %v", err)
	}

	initial, err := service.GetSettings(context.Background(), "relay-switch.settings")
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if initial.EffectiveValues["enabled"] != true {
		t.Fatalf("expected default effective value, got %+v", initial.EffectiveValues)
	}

	updated, err := service.UpdateSettings(context.Background(), "relay-switch.settings", UpdateSettingsInput{
		Values: map[string]json.RawMessage{
			"outputDirectory": json.RawMessage(`"/tmp/archive"`),
			"enabled":         json.RawMessage(`false`),
			"tags":            json.RawMessage(`["codex","claude"]`),
		},
	})
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if updated.Values["outputDirectory"] != "/tmp/archive" || updated.EffectiveValues["enabled"] != false {
		t.Fatalf("unexpected updated settings: %+v", updated)
	}

	if _, err := service.UpdateSettings(context.Background(), "relay-switch.settings", UpdateSettingsInput{
		Values: map[string]json.RawMessage{
			"outputDirectory": json.RawMessage(`123`),
		},
	}); !errors.Is(err, ErrInvalidSettings) {
		t.Fatalf("expected invalid settings error, got %v", err)
	}

	audit, err := service.ListAudit(context.Background(), "relay-switch.settings", 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(audit) != 1 || audit[0].Capability != "settings.write" || audit[0].Status != "success" {
		t.Fatalf("unexpected settings audit entries: %+v", audit)
	}
}

func TestServiceCommandExecutionNoopAndAudit(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	writeTestManifest(t, service.sources[0].Dir, "relay-switch.commands", `{
		"manifestVersion": 1,
		"id": "relay-switch.commands",
		"name": "Commands",
		"version": "0.1.0",
		"entry": {"type": "none"},
		"contributes": {
			"commands": [
				{"id": "commands.sync", "title": "Sync Now", "category": "Archive"}
			]
		},
		"permissions": []
	}`)

	if _, err := service.Scan(context.Background()); err != nil {
		t.Fatalf("scan extensions: %v", err)
	}
	if _, err := service.Enable(context.Background(), "relay-switch.commands"); err != nil {
		t.Fatalf("enable plugin: %v", err)
	}

	commands, err := service.ListCommands(context.Background())
	if err != nil {
		t.Fatalf("list commands: %v", err)
	}
	if len(commands) != 1 || commands[0].ID != "commands.sync" || !commands[0].Enabled {
		t.Fatalf("unexpected commands: %+v", commands)
	}

	result, err := service.ExecuteCommand(context.Background(), "commands.sync")
	if err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if result.Status != "skipped" || result.AuditLogID == "" {
		t.Fatalf("unexpected command result: %+v", result)
	}

	audit, err := service.ListAudit(context.Background(), "relay-switch.commands", 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(audit) != 1 || audit[0].Capability != "commands.execute" || audit[0].Status != "skipped" {
		t.Fatalf("unexpected command audit entries: %+v", audit)
	}
}

func TestServiceCommandExecutionDisabledPluginAuditsFailure(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	writeTestManifest(t, service.sources[0].Dir, "relay-switch.disabled-command", `{
		"manifestVersion": 1,
		"id": "relay-switch.disabled-command",
		"name": "Disabled Command",
		"version": "0.1.0",
		"entry": {"type": "none"},
		"contributes": {
			"commands": [
				{"id": "disabled.sync", "title": "Sync Now"}
			]
		},
		"permissions": []
	}`)

	if _, err := service.Scan(context.Background()); err != nil {
		t.Fatalf("scan extensions: %v", err)
	}

	result, err := service.ExecuteCommand(context.Background(), "disabled.sync")
	if !errors.Is(err, ErrPluginNotEnabled) {
		t.Fatalf("expected disabled plugin error, got %v", err)
	}
	if result.Status != "failed" || result.AuditLogID == "" {
		t.Fatalf("unexpected disabled command result: %+v", result)
	}

	audit, err := service.ListAudit(context.Background(), "relay-switch.disabled-command", 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(audit) != 1 || audit[0].Status != "failed" || audit[0].ErrorMessage == "" {
		t.Fatalf("unexpected disabled command audit entries: %+v", audit)
	}
}

func TestServiceToolIntegrationsDeclaredProcessesAndAudit(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	writeTestManifest(t, service.sources[0].Dir, "relay-switch.rtk", `{
		"manifestVersion": 1,
		"id": "relay-switch.rtk",
		"name": "RTK",
		"version": "0.1.0",
		"entry": {"type": "none"},
		"contributes": {
			"toolIntegrations": [
				{
					"id": "rtk",
					"title": "RTK",
					"supportsDetect": true,
					"supportsConfigure": true,
					"supportsRestore": false
				}
			],
			"declaredProcesses": [
				{"id": "rtk.version", "command": "rtk", "args": ["--version"], "timeoutMs": 3000},
				{"id": "rtk.init.global.codex", "command": "rtk", "args": ["init", "-g", "--codex"], "timeoutMs": 15000}
			]
		},
		"permissions": ["process.exec.declared", "tool.detect", "tool.config.write"]
	}`)

	if _, err := service.Scan(context.Background()); err != nil {
		t.Fatalf("scan extensions: %v", err)
	}
	if _, err := service.Enable(context.Background(), "relay-switch.rtk"); err != nil {
		t.Fatalf("enable plugin: %v", err)
	}

	integrations, err := service.ListToolIntegrations(context.Background())
	if err != nil {
		t.Fatalf("list tool integrations: %v", err)
	}
	if len(integrations) != 1 || integrations[0].ID != "rtk" || !integrations[0].SupportsConfigure {
		t.Fatalf("unexpected tool integrations: %+v", integrations)
	}

	processes, err := service.ListDeclaredProcesses(context.Background())
	if err != nil {
		t.Fatalf("list declared processes: %v", err)
	}
	if len(processes) != 2 || processes[0].ID != "rtk.init.global.codex" || len(processes[0].Args) != 3 {
		t.Fatalf("unexpected declared processes: %+v", processes)
	}

	result, err := service.ExecuteToolIntegrationAction(context.Background(), "rtk", "configure")
	if err != nil {
		t.Fatalf("execute tool integration action: %v", err)
	}
	if result.Status != "skipped" || result.AuditLogID == "" {
		t.Fatalf("unexpected tool integration result: %+v", result)
	}

	if _, err := service.ExecuteToolIntegrationAction(context.Background(), "rtk", "restore"); !errors.Is(err, ErrToolIntegrationActionUnsupported) {
		t.Fatalf("expected unsupported action error, got %v", err)
	}

	audit, err := service.ListAudit(context.Background(), "relay-switch.rtk", 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(audit) != 1 || audit[0].Capability != "tool.integration.configure" || audit[0].Status != "skipped" {
		t.Fatalf("unexpected tool integration audit entries: %+v", audit)
	}
}

func TestServiceDeclaredProcessesRequireDeclaredExecPermission(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	writeTestManifest(t, service.sources[0].Dir, "relay-switch.missing-declared-permission", `{
		"manifestVersion": 1,
		"id": "relay-switch.missing-declared-permission",
		"name": "Missing Declared Permission",
		"version": "0.1.0",
		"entry": {"type": "none"},
		"contributes": {
			"declaredProcesses": [
				{"id": "rtk.version", "command": "rtk", "args": ["--version"], "timeoutMs": 3000}
			]
		},
		"permissions": []
	}`)

	items, err := service.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan extensions: %v", err)
	}
	if len(items) != 1 || items[0].Status != PluginStatusInvalid {
		t.Fatalf("expected invalid declared process plugin, got %+v", items)
	}
	if !strings.Contains(items[0].LastError, "process.exec.declared") {
		t.Fatalf("expected declared permission error, got %+v", items[0])
	}
}

func TestServiceBundledMarkdownArchiveRegistersContributions(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	service.bundled = []BundledPlugin{MarkdownArchiveBundledPlugin()}

	items, err := service.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan extensions: %v", err)
	}
	if len(items) != 1 || items[0].ID != MarkdownArchivePluginID || items[0].Scope != PluginScopeBundled {
		t.Fatalf("unexpected bundled plugin: %+v", items)
	}
	if items[0].Contributes["transcriptSources"] != 2 || items[0].Contributes["backgroundTasks"] != 1 {
		t.Fatalf("unexpected bundled contributions: %+v", items[0].Contributes)
	}

	tasks, err := service.ListBackgroundTasks(context.Background())
	if err != nil {
		t.Fatalf("list background tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != "markdownArchive.autoSync" {
		t.Fatalf("unexpected background tasks: %+v", tasks)
	}
}

func TestServiceTranscriptArchiveSyncExportsMarkdownAndSkipsUnchanged(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	service.bundled = []BundledPlugin{MarkdownArchiveBundledPlugin()}
	homeDir := t.TempDir()
	service.homeDir = homeDir
	outputDir := filepath.Join(t.TempDir(), "archive")

	claudePath := filepath.Join(homeDir, ".claude", "projects", "relay-switch", "claude-session.jsonl")
	writeTextFile(t, claudePath, strings.Join([]string{
		`{"sessionId":"claude-session","type":"user","cwd":"/workspace/relay-switch","timestamp":"2026-06-05T06:32:10Z","message":{"role":"user","content":"hello sk-abcdefghijklmnop"}}`,
		`{"sessionId":"claude-session","type":"assistant","timestamp":"2026-06-05T06:33:10Z","message":{"role":"assistant","content":"done"}}`,
	}, "\n")+"\n")

	codexPath := filepath.Join(homeDir, ".codex", "sessions", "2026", "codex-session.jsonl")
	writeTextFile(t, codexPath, strings.Join([]string{
		`{"session_id":"codex-session","role":"user","cwd":"/workspace/relay-switch","timestamp":"2026-06-05T07:32:10Z","content":"codex hello"}`,
		`{"session_id":"codex-session","role":"assistant","timestamp":"2026-06-05T07:33:10Z","content":"codex done"}`,
	}, "\n")+"\n")

	if _, err := service.Scan(context.Background()); err != nil {
		t.Fatalf("scan extensions: %v", err)
	}
	if _, err := service.Enable(context.Background(), MarkdownArchivePluginID); err != nil {
		t.Fatalf("enable markdown archive: %v", err)
	}

	sources, err := service.ListTranscriptSources(context.Background())
	if err != nil {
		t.Fatalf("list transcript sources: %v", err)
	}
	if len(sources) != 2 || sources[0].SessionCount+sources[1].SessionCount != 2 {
		t.Fatalf("unexpected transcript sources: %+v", sources)
	}

	result, err := service.SyncTranscriptArchive(context.Background(), TranscriptSyncInput{
		OutputDirectory: outputDir,
	})
	if err != nil {
		t.Fatalf("sync transcript archive: %v", err)
	}
	if result.ExportedCount != 2 || result.SkippedCount != 0 || result.FailedCount != 0 || result.AuditLogID == "" {
		t.Fatalf("unexpected sync result: %+v", result)
	}

	markdownFiles := listMarkdownFiles(t, outputDir)
	if len(markdownFiles) != 2 {
		t.Fatalf("expected two markdown files, got %+v", markdownFiles)
	}
	claudeMarkdown := readTextFile(t, markdownFiles[0]) + readTextFile(t, markdownFiles[1])
	if !strings.Contains(claudeMarkdown, "[redacted]") || strings.Contains(claudeMarkdown, "sk-abcdefghijklmnop") {
		t.Fatalf("expected redacted Claude secret, got %s", claudeMarkdown)
	}

	second, err := service.SyncTranscriptArchive(context.Background(), TranscriptSyncInput{
		OutputDirectory: outputDir,
	})
	if err != nil {
		t.Fatalf("sync transcript archive second run: %v", err)
	}
	if second.ExportedCount != 0 || second.SkippedCount != 2 {
		t.Fatalf("expected unchanged sessions to be skipped, got %+v", second)
	}

	audit, err := service.ListAudit(context.Background(), MarkdownArchivePluginID, 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(audit) != 2 || audit[0].Capability != "tool.transcripts.read" {
		t.Fatalf("unexpected transcript audit entries: %+v", audit)
	}
}

func TestServiceTranscriptArchiveSyncRequiresOutputDirectory(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	service.bundled = []BundledPlugin{MarkdownArchiveBundledPlugin()}
	service.homeDir = t.TempDir()

	if _, err := service.Scan(context.Background()); err != nil {
		t.Fatalf("scan extensions: %v", err)
	}
	if _, err := service.Enable(context.Background(), MarkdownArchivePluginID); err != nil {
		t.Fatalf("enable markdown archive: %v", err)
	}

	result, err := service.SyncTranscriptArchive(context.Background(), TranscriptSyncInput{})
	if !errors.Is(err, ErrTranscriptOutputDirectoryRequired) {
		t.Fatalf("expected output directory error, got %v", err)
	}
	if result.Status != "failed" || result.AuditLogID == "" {
		t.Fatalf("expected failed sync audit result, got %+v", result)
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

func writeTextFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create file dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	return string(content)
}

func listMarkdownFiles(t *testing.T, root string) []string {
	t.Helper()
	paths := []string{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk markdown files: %v", err)
	}
	sort.Strings(paths)
	return paths
}
