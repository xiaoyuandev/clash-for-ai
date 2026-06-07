package extension

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xiaoyuandev/relay-switch/core/internal/storage"
)

func TestServiceScanLoadsValidManifestAndPersistsEnableState(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	writeTestManifest(t, service.sources[0].Dir, "acme.archive", `{
		"manifestVersion": 1,
		"id": "acme.archive",
		"name": "Archive",
		"version": "0.1.0",
		"description": "External archive plugin.",
		"publisher": "acme",
		"engines": {"relaySwitch": ">=1.0.0"},
		"entry": {"type": "none"},
		"contributes": {"commands": [{"id": "archive.syncNow", "title": "Sync"}]},
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

	enabled, err := service.Enable(context.Background(), "acme.archive")
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

	disabled, err := service.Disable(context.Background(), "acme.archive")
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
	writeTestManifest(t, service.sources[0].Dir, "acme.tool", `{
		"manifestVersion": 1,
		"id": "acme.tool",
		"name": "External Tool",
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
				{"id": "external-tool.configure", "command": "external-tool", "args": ["configure", "--check"], "timeoutMs": 3000},
				{"id": "external-tool.version", "command": "external-tool", "args": ["--version"], "timeoutMs": 3000}
			]
		},
		"permissions": ["process.exec.declared", "tool.detect", "tool.config.write"]
	}`)

	if _, err := service.Scan(context.Background()); err != nil {
		t.Fatalf("scan extensions: %v", err)
	}
	if _, err := service.Enable(context.Background(), "acme.tool"); err != nil {
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
	if len(processes) != 2 || processes[0].ID != "external-tool.configure" || len(processes[0].Args) != 2 {
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

	audit, err := service.ListAudit(context.Background(), "acme.tool", 10)
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

func TestManifestNodePackageEntryBuildsNpxRuntimePlan(t *testing.T) {
	t.Parallel()

	manifest := Manifest{
		ManifestVersion: 1,
		ID:              "acme.node-plugin",
		Name:            "Node Plugin",
		Version:         "1.0.0",
		Entry: ManifestEntry{
			Type:    "nodePackage",
			Package: "@acme/relay-switch-plugin",
			Version: "1.2.3",
			Bin:     "relay-switch-plugin",
			Args:    []string{"serve"},
		},
		Contributes: ManifestContributes{},
		Permissions: []string{},
	}
	if _, err := ValidateManifest(manifest); err != nil {
		t.Fatalf("validate manifest: %v", err)
	}
	plan, err := RuntimePlanForPlugin(Plugin{
		ID:           manifest.ID,
		ManifestPath: filepath.Join(t.TempDir(), ManifestFileName),
		Manifest:     manifest,
	})
	if err != nil {
		t.Fatalf("build runtime plan: %v", err)
	}
	if plan.Command != "npx" {
		t.Fatalf("expected npx command, got %+v", plan)
	}
	expected := []string{"--yes", "--package", "@acme/relay-switch-plugin@1.2.3", "relay-switch-plugin", "serve"}
	if strings.Join(plan.Args, "\x00") != strings.Join(expected, "\x00") {
		t.Fatalf("unexpected npx args: %+v", plan.Args)
	}
}

func TestManifestNodePackageRejectsNonExactVersion(t *testing.T) {
	t.Parallel()

	_, err := ValidateManifest(Manifest{
		ManifestVersion: 1,
		ID:              "acme.node-plugin",
		Name:            "Node Plugin",
		Version:         "1.0.0",
		Entry: ManifestEntry{
			Type:    "nodePackage",
			Package: "@acme/relay-switch-plugin",
			Version: "^1.2.3",
			Bin:     "relay-switch-plugin",
		},
		Contributes: ManifestContributes{},
		Permissions: []string{},
	})
	if err == nil || !strings.Contains(err.Error(), "exact semver") {
		t.Fatalf("expected exact semver validation error, got %v", err)
	}
}

func TestServiceInstallGitHubPluginAutoEnablesAndUninstalls(t *testing.T) {
	t.Parallel()

	fixture := t.TempDir()
	writeTestManifest(t, fixture, ".", `{
		"manifestVersion": 1,
		"id": "acme.external-plugin",
		"name": "External Plugin",
		"version": "1.0.0",
		"entry": {"type": "none"},
		"contributes": {
			"backgroundTasks": [
				{"id": "external.refresh", "title": "Refresh", "minimumIntervalSeconds": 30}
			]
		},
		"permissions": []
	}`)

	managedDir := filepath.Join(t.TempDir(), "managed")
	dataDir := filepath.Join(t.TempDir(), "plugin-data")
	service := newTestExtensionServiceWithOptions(t, ServiceOptions{
		ManagedInstallDir: managedDir,
		PluginDataDir:     dataDir,
		GitExecutable:     writeFakeGit(t, fixture),
	})

	installed, err := service.Install(context.Background(), InstallPluginInput{
		Source: "github",
		URL:    "https://github.com/acme/external-plugin",
	})
	if err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	if !installed.Enabled || installed.Status != PluginStatusEnabled || installed.Scope != PluginScopeManaged {
		t.Fatalf("expected installed plugin to auto-enable, got %+v", installed)
	}
	if installed.Install == nil || installed.Install.SourceURL != "https://github.com/acme/external-plugin" || installed.Install.GitCommit != "abcdef123456" {
		t.Fatalf("unexpected install metadata: %+v", installed.Install)
	}

	tasks, err := service.ListBackgroundTasks(context.Background())
	if err != nil {
		t.Fatalf("list background tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != "external.refresh" || !tasks[0].Enabled {
		t.Fatalf("unexpected background tasks: %+v", tasks)
	}
	if _, err := service.Install(context.Background(), InstallPluginInput{
		Source: "github",
		URL:    "https://github.com/acme/external-plugin.git",
	}); !errors.Is(err, ErrPluginAlreadyInstalled) {
		t.Fatalf("expected duplicate install error, got %v", err)
	}

	installDir := installed.Install.InstallDir
	if err := service.Uninstall(context.Background(), installed.ID); err != nil {
		t.Fatalf("uninstall plugin: %v", err)
	}
	if _, err := service.GetByID(context.Background(), installed.ID); !errors.Is(err, ErrPluginNotFound) {
		t.Fatalf("expected uninstalled plugin to be removed, got %v", err)
	}
	if _, err := os.Stat(installDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected install directory to be removed, got %v", err)
	}
}

func newTestExtensionService(t *testing.T) *Service {
	t.Helper()
	return newTestExtensionServiceWithOptions(t, ServiceOptions{})
}

func newTestExtensionServiceWithOptions(t *testing.T, options ServiceOptions) *Service {
	t.Helper()

	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "extensions.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = sqliteStore.Close() })

	root := filepath.Join(t.TempDir(), "extensions")
	sources := []ScanSource{
		{
			Scope: PluginScopeUser,
			Dir:   root,
		},
	}
	if options.ManagedInstallDir != "" {
		sources = append(sources, ScanSource{
			Scope: PluginScopeManaged,
			Dir:   options.ManagedInstallDir,
		})
	}
	return NewServiceWithOptions(NewSQLiteRepository(sqliteStore.DB), sources, options)
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

func writeFakeGit(t *testing.T, fixtureDir string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "git")
	script := "#!/bin/sh\n" +
		"set -eu\n" +
		"case \"$1\" in\n" +
		"  clone)\n" +
		"    target=\"$5\"\n" +
		"    mkdir -p \"$target\"\n" +
		"    cp -R " + shellQuote(fixtureDir) + "/. \"$target\"\n" +
		"    ;;\n" +
		"  rev-parse)\n" +
		"    echo abcdef123456\n" +
		"    ;;\n" +
		"  pull)\n" +
		"    exit 0\n" +
		"    ;;\n" +
		"  *)\n" +
		"    echo \"unexpected git args: $*\" >&2\n" +
		"    exit 2\n" +
		"    ;;\n" +
		"esac\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}
	return path
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
