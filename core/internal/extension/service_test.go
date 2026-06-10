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

func TestServiceRuntimeInitializesWithPersistedSettings(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	pluginDir := filepath.Join(service.sources[0].Dir, "relay-switch.runtime-settings")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("create plugin dir: %v", err)
	}

	capturePath := filepath.Join(pluginDir, "initialize.jsonl")
	runtimeScript := "#!/bin/sh\n" +
		"capture=" + shellQuote(capturePath) + "\n" +
		"while IFS= read -r line; do\n" +
		"  id=$(printf '%s\\n' \"$line\" | sed -n 's/.*\"id\":\\([0-9][0-9]*\\).*/\\1/p')\n" +
		"  [ -n \"$id\" ] || continue\n" +
		"  case \"$line\" in\n" +
		"    *'\"method\":\"initialize\"'*)\n" +
		"      printf '%s\\n' \"$line\" >> \"$capture\"\n" +
		"      printf '{\"jsonrpc\":\"2.0\",\"id\":%s,\"result\":{\"status\":\"ok\"}}\\n' \"$id\"\n" +
		"      ;;\n" +
		"    *'\"method\":\"executeCommand\"'*)\n" +
		"      printf '{\"jsonrpc\":\"2.0\",\"id\":%s,\"result\":{\"status\":\"success\",\"message\":\"runtime command ok\"}}\\n' \"$id\"\n" +
		"      ;;\n" +
		"    *'\"method\":\"shutdown\"'*)\n" +
		"      printf '{\"jsonrpc\":\"2.0\",\"id\":%s,\"result\":{\"status\":\"success\"}}\\n' \"$id\"\n" +
		"      exit 0\n" +
		"      ;;\n" +
		"    *)\n" +
		"      printf '{\"jsonrpc\":\"2.0\",\"id\":%s,\"result\":{}}\\n' \"$id\"\n" +
		"      ;;\n" +
		"  esac\n" +
		"done\n"
	if err := os.WriteFile(filepath.Join(pluginDir, "runtime-fixture"), []byte(runtimeScript), 0o755); err != nil {
		t.Fatalf("write runtime fixture: %v", err)
	}

	writeTestManifest(t, service.sources[0].Dir, "relay-switch.runtime-settings", `{
		"manifestVersion": 1,
		"id": "relay-switch.runtime-settings",
		"name": "Runtime Settings",
		"version": "0.1.0",
		"entry": {"type": "process", "command": "runtime-fixture"},
		"contributes": {
			"commands": [
				{"id": "runtime-settings.sync", "title": "Sync Now"}
			],
			"settings": {
				"type": "object",
				"properties": {
					"outputDirectory": {"type": "string", "default": ""},
					"archiveEnabled": {"type": "boolean", "default": false}
				}
			}
		},
		"permissions": []
	}`)

	if _, err := service.Scan(context.Background()); err != nil {
		t.Fatalf("scan extensions: %v", err)
	}
	outputDirectory := filepath.Join(t.TempDir(), "archive")
	if _, err := service.UpdateSettings(context.Background(), "relay-switch.runtime-settings", UpdateSettingsInput{
		Values: map[string]json.RawMessage{
			"outputDirectory": mustRawJSONString(outputDirectory),
			"archiveEnabled":  json.RawMessage(`true`),
		},
	}); err != nil {
		t.Fatalf("update settings: %v", err)
	}

	if _, err := service.Enable(context.Background(), "relay-switch.runtime-settings"); err != nil {
		t.Fatalf("enable plugin: %v", err)
	}
	settings := readCapturedInitializeSettings(t, capturePath)
	if settings["outputDirectory"] != outputDirectory || settings["archiveEnabled"] != true {
		t.Fatalf("runtime initialized with unexpected settings on enable: %+v", settings)
	}

	if err := service.runtimeHost.Stop(context.Background(), "relay-switch.runtime-settings"); err != nil {
		t.Fatalf("stop runtime: %v", err)
	}
	if err := os.Remove(capturePath); err != nil {
		t.Fatalf("reset capture file: %v", err)
	}

	result, err := service.ExecuteCommand(context.Background(), "runtime-settings.sync")
	if err != nil {
		t.Fatalf("execute runtime command: %v", err)
	}
	if result.Status != "success" || result.Message != "runtime command ok" {
		t.Fatalf("unexpected runtime command result: %+v", result)
	}
	settings = readCapturedInitializeSettings(t, capturePath)
	if settings["outputDirectory"] != outputDirectory || settings["archiveEnabled"] != true {
		t.Fatalf("runtime initialized with unexpected settings on command restart: %+v", settings)
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

func TestLocalNodePackageEntryUsesBuiltLocalBin(t *testing.T) {
	t.Parallel()

	pluginDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pluginDir, "package.json"), []byte(`{
		"name": "@acme/relay-switch-plugin",
		"bin": {
			"relay-switch-plugin": "./dist/main.js"
		}
	}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pluginDir, "dist"), 0o755); err != nil {
		t.Fatalf("create dist dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "dist", "main.js"), []byte(`console.log("ok");`), 0o644); err != nil {
		t.Fatalf("write local bin: %v", err)
	}

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
	plan, err := RuntimePlanForPlugin(Plugin{
		ID:           manifest.ID,
		ManifestPath: filepath.Join(pluginDir, ManifestFileName),
		Manifest:     manifest,
		Install: &PluginInstall{
			SourceType: string(PluginSourceLocalDirectory),
			InstallDir: pluginDir,
			SourceURL:  pluginDir,
		},
	})
	if err != nil {
		t.Fatalf("build local runtime plan: %v", err)
	}
	if plan.Command != "node" {
		t.Fatalf("expected local node command, got %+v", plan)
	}
	expected := []string{filepath.Join(pluginDir, "dist", "main.js"), "serve"}
	if strings.Join(plan.Args, "\x00") != strings.Join(expected, "\x00") {
		t.Fatalf("unexpected local runtime args: %+v", plan.Args)
	}
}

func TestManagedNodePackageEntryUsesBuiltLocalBin(t *testing.T) {
	t.Parallel()

	pluginDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pluginDir, "package.json"), []byte(`{
		"name": "@acme/relay-switch-plugin",
		"bin": {
			"relay-switch-plugin": "./dist/main.js"
		}
	}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pluginDir, "dist"), 0o755); err != nil {
		t.Fatalf("create dist dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "dist", "main.js"), []byte(`console.log("ok");`), 0o644); err != nil {
		t.Fatalf("write local bin: %v", err)
	}

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
	plan, err := RuntimePlanForPlugin(Plugin{
		ID:           manifest.ID,
		ManifestPath: filepath.Join(pluginDir, ManifestFileName),
		Manifest:     manifest,
		Install: &PluginInstall{
			SourceType: string(PluginSourceGitHub),
			InstallDir: pluginDir,
			SourceURL:  "https://github.com/acme/relay-switch-plugin",
		},
	})
	if err != nil {
		t.Fatalf("build managed runtime plan: %v", err)
	}
	if plan.Command != "node" {
		t.Fatalf("expected local node command, got %+v", plan)
	}
	expected := []string{filepath.Join(pluginDir, "dist", "main.js"), "serve"}
	if strings.Join(plan.Args, "\x00") != strings.Join(expected, "\x00") {
		t.Fatalf("unexpected managed runtime args: %+v", plan.Args)
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

func TestServiceInstallGitHubNodePackagePreparesBeforeInstall(t *testing.T) {
	t.Parallel()

	fixture := t.TempDir()
	writeTestManifest(t, fixture, ".", `{
		"manifestVersion": 1,
		"id": "acme.external-node-plugin",
		"name": "External Node Plugin",
		"version": "1.0.0",
		"entry": {
			"type": "nodePackage",
			"package": "@acme/relay-switch-plugin",
			"version": "1.2.3",
			"bin": "relay-switch-plugin",
			"args": ["serve"]
		},
		"contributes": {},
		"permissions": []
	}`)
	if err := os.WriteFile(filepath.Join(fixture, "package.json"), []byte(`{
		"name": "@acme/relay-switch-plugin",
		"bin": {"relay-switch-plugin": "./dist/main.js"},
		"scripts": {"build": "node build.js"}
	}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "pnpm-lock.yaml"), []byte("lockfileVersion: '9.0'\n"), 0o644); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}

	managedDir := filepath.Join(t.TempDir(), "managed")
	prepareLog := filepath.Join(t.TempDir(), "prepare.log")
	service := newTestExtensionServiceWithOptions(t, ServiceOptions{
		ManagedInstallDir: managedDir,
		GitExecutable:     writeFakeGit(t, fixture),
		PrepareExecutable: writeFakePrepare(t, prepareLog, true),
		PrepareArgs:       []string{"prepare"},
	})

	installed, err := service.Install(context.Background(), InstallPluginInput{
		Source: "github",
		URL:    "https://github.com/acme/external-node-plugin",
	})
	if err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	if installed.Install == nil || installed.Install.InstallDir == "" {
		t.Fatalf("expected install metadata, got %+v", installed.Install)
	}
	content, err := os.ReadFile(prepareLog)
	if err != nil {
		t.Fatalf("read prepare log: %v", err)
	}
	if !strings.Contains(string(content), "prepare") {
		t.Fatalf("expected prepare command to run, got %q", string(content))
	}

	plan, err := RuntimePlanForPlugin(*installed)
	if err != nil {
		t.Fatalf("build runtime plan: %v", err)
	}
	if plan.Command != "node" || len(plan.Args) == 0 || !strings.HasSuffix(plan.Args[0], filepath.Join("dist", "main.js")) {
		t.Fatalf("expected prepared local runtime plan, got %+v", plan)
	}
}

func TestServiceInstallGitHubNodePackagePrepareFailureCleansClone(t *testing.T) {
	t.Parallel()

	fixture := t.TempDir()
	writeTestManifest(t, fixture, ".", `{
		"manifestVersion": 1,
		"id": "acme.broken-node-plugin",
		"name": "Broken Node Plugin",
		"version": "1.0.0",
		"entry": {
			"type": "nodePackage",
			"package": "@acme/relay-switch-plugin",
			"version": "1.2.3",
			"bin": "relay-switch-plugin"
		},
		"contributes": {},
		"permissions": []
	}`)
	if err := os.WriteFile(filepath.Join(fixture, "package.json"), []byte(`{
		"name": "@acme/relay-switch-plugin",
		"bin": {"relay-switch-plugin": "./dist/main.js"},
		"scripts": {"build": "node build.js"}
	}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	managedDir := filepath.Join(t.TempDir(), "managed")
	service := newTestExtensionServiceWithOptions(t, ServiceOptions{
		ManagedInstallDir: managedDir,
		GitExecutable:     writeFakeGit(t, fixture),
		PrepareExecutable: writeFakePrepare(t, filepath.Join(t.TempDir(), "prepare.log"), false),
		PrepareArgs:       []string{"prepare"},
	})

	if _, err := service.Install(context.Background(), InstallPluginInput{
		Source: "github",
		URL:    "https://github.com/acme/broken-node-plugin",
	}); err == nil || !errors.Is(err, ErrInvalidPluginSource) {
		t.Fatalf("expected prepare failure, got %v", err)
	}
	if _, err := service.GetByID(context.Background(), "acme.broken-node-plugin"); !errors.Is(err, ErrPluginNotFound) {
		t.Fatalf("expected failed install not to be persisted, got %v", err)
	}
	entries, err := os.ReadDir(managedDir)
	if err != nil {
		t.Fatalf("read managed dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected failed clone to be cleaned, got %d entries", len(entries))
	}
}

func TestServiceInstallGitHubNodePackageRequiresLocalPackageBin(t *testing.T) {
	t.Parallel()

	fixture := t.TempDir()
	writeTestManifest(t, fixture, ".", `{
		"manifestVersion": 1,
		"id": "acme.mismatched-node-plugin",
		"name": "Mismatched Node Plugin",
		"version": "1.0.0",
		"entry": {
			"type": "nodePackage",
			"package": "@acme/relay-switch-plugin",
			"version": "1.2.3",
			"bin": "relay-switch-plugin"
		},
		"contributes": {},
		"permissions": []
	}`)
	if err := os.WriteFile(filepath.Join(fixture, "package.json"), []byte(`{
		"name": "@acme/different-plugin",
		"bin": {"relay-switch-plugin": "./dist/main.js"}
	}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	managedDir := filepath.Join(t.TempDir(), "managed")
	service := newTestExtensionServiceWithOptions(t, ServiceOptions{
		ManagedInstallDir: managedDir,
		GitExecutable:     writeFakeGit(t, fixture),
		PrepareExecutable: writeFakePrepare(t, filepath.Join(t.TempDir(), "prepare.log"), true),
		PrepareArgs:       []string{"prepare"},
	})

	if _, err := service.Install(context.Background(), InstallPluginInput{
		Source: "github",
		URL:    "https://github.com/acme/mismatched-node-plugin",
	}); err == nil || !errors.Is(err, ErrInvalidPluginSource) {
		t.Fatalf("expected local package bin failure, got %v", err)
	}
	entries, err := os.ReadDir(managedDir)
	if err != nil {
		t.Fatalf("read managed dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected failed clone to be cleaned, got %d entries", len(entries))
	}
}

func TestServiceLocalInstallRequiresDeveloperMode(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	localDir := t.TempDir()
	writeTestManifest(t, localDir, ".", `{
		"manifestVersion": 1,
		"id": "acme.local-disabled",
		"name": "Local Disabled",
		"version": "0.1.0",
		"entry": {"type": "none"},
		"contributes": {},
		"permissions": []
	}`)

	if _, err := service.LocalInstall(context.Background(), LocalInstallPluginInput{Path: localDir}); !errors.Is(err, ErrDeveloperModeDisabled) {
		t.Fatalf("expected developer mode error, got %v", err)
	}
}

func TestServiceLocalInstallReloadsAndDoesNotDeleteSource(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	if _, err := service.UpdateDeveloperMode(context.Background(), true); err != nil {
		t.Fatalf("enable developer mode: %v", err)
	}

	localDir := t.TempDir()
	writeTestManifest(t, localDir, ".", `{
		"manifestVersion": 1,
		"id": "acme.local-plugin",
		"name": "Local Plugin",
		"version": "0.1.0",
		"entry": {"type": "none"},
		"contributes": {"commands": [{"id": "local.sync", "title": "Sync"}]},
		"permissions": []
	}`)

	installed, err := service.LocalInstall(context.Background(), LocalInstallPluginInput{Path: localDir})
	if err != nil {
		t.Fatalf("local install plugin: %v", err)
	}
	normalizedLocalDir, err := filepath.EvalSymlinks(localDir)
	if err != nil {
		t.Fatalf("normalize local dir: %v", err)
	}
	if !installed.Enabled || installed.Status != PluginStatusEnabled || installed.Scope != PluginScopeDevelopment {
		t.Fatalf("unexpected local plugin state: %+v", installed)
	}
	if installed.Install == nil || installed.Install.SourceType != string(PluginSourceLocalDirectory) || installed.Install.InstallDir != normalizedLocalDir {
		t.Fatalf("unexpected local install metadata: %+v", installed.Install)
	}

	writeTestManifest(t, localDir, ".", `{
		"manifestVersion": 1,
		"id": "acme.local-plugin",
		"name": "Local Plugin Reloaded",
		"version": "0.2.0",
		"entry": {"type": "none"},
		"contributes": {"commands": [{"id": "local.sync", "title": "Sync"}]},
		"permissions": []
	}`)
	reloaded, err := service.LocalInstall(context.Background(), LocalInstallPluginInput{Path: localDir})
	if err != nil {
		t.Fatalf("reload local plugin: %v", err)
	}
	if reloaded.Version != "0.2.0" || reloaded.Name != "Local Plugin Reloaded" {
		t.Fatalf("expected local install to reload manifest, got %+v", reloaded)
	}

	writeTestManifest(t, localDir, ".", `{
		"manifestVersion": 1,
		"id": "acme.local-plugin",
		"name": "Local Plugin Updated",
		"version": "0.3.0",
		"entry": {"type": "none"},
		"contributes": {"commands": [{"id": "local.sync", "title": "Sync"}]},
		"permissions": []
	}`)
	updated, err := service.Update(context.Background(), installed.ID)
	if err != nil {
		t.Fatalf("update local plugin: %v", err)
	}
	if updated.Version != "0.3.0" {
		t.Fatalf("expected local update to reread manifest, got %+v", updated)
	}

	if err := service.Uninstall(context.Background(), installed.ID); err != nil {
		t.Fatalf("uninstall local plugin: %v", err)
	}
	if _, err := os.Stat(filepath.Join(localDir, ManifestFileName)); err != nil {
		t.Fatalf("local source directory should remain after uninstall: %v", err)
	}
}

func TestServiceDeveloperModeDisableStopsAndDisablesLocalPlugins(t *testing.T) {
	t.Parallel()

	service := newTestExtensionService(t)
	if _, err := service.UpdateDeveloperMode(context.Background(), true); err != nil {
		t.Fatalf("enable developer mode: %v", err)
	}

	localDir := t.TempDir()
	writeTestManifest(t, localDir, ".", `{
		"manifestVersion": 1,
		"id": "acme.local-toggle",
		"name": "Local Toggle",
		"version": "0.1.0",
		"entry": {"type": "none"},
		"contributes": {},
		"permissions": []
	}`)
	if _, err := service.LocalInstall(context.Background(), LocalInstallPluginInput{Path: localDir}); err != nil {
		t.Fatalf("local install plugin: %v", err)
	}

	if _, err := service.UpdateDeveloperMode(context.Background(), false); err != nil {
		t.Fatalf("disable developer mode: %v", err)
	}
	item, err := service.GetByID(context.Background(), "acme.local-toggle")
	if err != nil {
		t.Fatalf("get local plugin: %v", err)
	}
	if item.Enabled || item.Status != PluginStatusDisabled {
		t.Fatalf("expected local plugin disabled after developer mode off, got %+v", item)
	}
	if _, err := service.Enable(context.Background(), "acme.local-toggle"); !errors.Is(err, ErrDeveloperModeDisabled) {
		t.Fatalf("expected enable to be blocked while developer mode is off, got %v", err)
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

func writeFakePrepare(t *testing.T, logPath string, success bool) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "prepare")
	exitBlock := "mkdir -p dist\nprintf '%s\\n' 'console.log(\"ok\");' > dist/main.js\n"
	if !success {
		exitBlock = "echo 'build failed' >&2\nexit 9\n"
	}
	script := "#!/bin/sh\n" +
		"set -eu\n" +
		"printf '%s\\n' \"$*\" >> " + shellQuote(logPath) + "\n" +
		exitBlock
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake prepare: %v", err)
	}
	return path
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func mustRawJSONString(value string) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func readCapturedInitializeSettings(t *testing.T, capturePath string) map[string]any {
	t.Helper()

	content, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("read runtime initialize capture: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[len(lines)-1]) == "" {
		t.Fatalf("runtime initialize capture is empty")
	}

	var request struct {
		Params struct {
			Settings map[string]any `json:"settings"`
		} `json:"params"`
	}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &request); err != nil {
		t.Fatalf("decode runtime initialize capture: %v", err)
	}
	return request.Params.Settings
}
