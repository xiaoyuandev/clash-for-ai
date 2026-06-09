package extension

import (
	"encoding/json"
	"errors"
)

var ErrPluginNotFound = errors.New("plugin not found")
var ErrCommandNotFound = errors.New("extension command not found")
var ErrToolIntegrationNotFound = errors.New("extension tool integration not found")
var ErrToolIntegrationActionUnsupported = errors.New("extension tool integration action is not supported")
var ErrPluginNotEnabled = errors.New("plugin is not enabled")
var ErrPluginInvalid = errors.New("plugin is invalid")
var ErrInvalidSettings = errors.New("invalid plugin settings")
var ErrPluginAlreadyInstalled = errors.New("plugin is already installed")
var ErrPluginNotManaged = errors.New("plugin is not managed by Relay Switch")
var ErrInvalidPluginSource = errors.New("invalid plugin source")
var ErrDeveloperModeDisabled = errors.New("developer mode is disabled")

type PluginScope string

const (
	PluginScopeUser        PluginScope = "user"
	PluginScopeProject     PluginScope = "project"
	PluginScopeManaged     PluginScope = "managed"
	PluginScopeDevelopment PluginScope = "development"
)

type PluginStatus string

const (
	PluginStatusInstalled    PluginStatus = "installed"
	PluginStatusEnabled      PluginStatus = "enabled"
	PluginStatusDisabled     PluginStatus = "disabled"
	PluginStatusIncompatible PluginStatus = "incompatible"
	PluginStatusInvalid      PluginStatus = "invalid"
)

type Plugin struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Version      string         `json:"version"`
	Description  string         `json:"description"`
	Publisher    string         `json:"publisher"`
	Scope        PluginScope    `json:"scope"`
	ManifestPath string         `json:"manifest_path"`
	Enabled      bool           `json:"enabled"`
	Status       PluginStatus   `json:"status"`
	LastError    string         `json:"last_error"`
	Manifest     Manifest       `json:"manifest"`
	Install      *PluginInstall `json:"install,omitempty"`
	Runtime      PluginRuntime  `json:"runtime"`
	Permissions  []string       `json:"permissions"`
	Contributes  map[string]int `json:"contributes"`
	Warnings     []string       `json:"warnings"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
}

type Manifest struct {
	ManifestVersion int                 `json:"manifestVersion"`
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Version         string              `json:"version"`
	Description     string              `json:"description"`
	Publisher       string              `json:"publisher"`
	Engines         ManifestEngines     `json:"engines"`
	Entry           ManifestEntry       `json:"entry"`
	Contributes     ManifestContributes `json:"contributes"`
	Permissions     []string            `json:"permissions"`
}

type ManifestEngines struct {
	RelaySwitch string `json:"relaySwitch"`
}

type ManifestEntry struct {
	Type    string   `json:"type"`
	Command string   `json:"command,omitempty"`
	Package string   `json:"package,omitempty"`
	Version string   `json:"version,omitempty"`
	Bin     string   `json:"bin,omitempty"`
	Args    []string `json:"args,omitempty"`
}

type ManifestContributes map[string]json.RawMessage

type ScanSource struct {
	Scope PluginScope
	Dir   string
}

type PluginSourceType string

const (
	PluginSourceGitHub         PluginSourceType = "github"
	PluginSourceLocalDirectory PluginSourceType = "localDirectory"
)

type PluginInstall struct {
	PluginID    string `json:"plugin_id"`
	SourceType  string `json:"source_type"`
	SourceURL   string `json:"source_url"`
	InstallDir  string `json:"install_dir"`
	GitCommit   string `json:"git_commit"`
	InstalledAt string `json:"installed_at"`
	UpdatedAt   string `json:"updated_at"`
}

type PluginRuntimeState string

const (
	PluginRuntimeStateNone     PluginRuntimeState = "none"
	PluginRuntimeStateStopped  PluginRuntimeState = "stopped"
	PluginRuntimeStateStarting PluginRuntimeState = "starting"
	PluginRuntimeStateRunning  PluginRuntimeState = "running"
	PluginRuntimeStateDegraded PluginRuntimeState = "degraded"
)

type PluginRuntime struct {
	State     PluginRuntimeState `json:"state"`
	EntryType string             `json:"entry_type"`
	Command   string             `json:"command,omitempty"`
	Args      []string           `json:"args,omitempty"`
	Cwd       string             `json:"cwd,omitempty"`
	LastError string             `json:"last_error,omitempty"`
	UpdatedAt string             `json:"updated_at,omitempty"`
}

type PluginRuntimePlan struct {
	EntryType string
	Command   string
	Args      []string
	Cwd       string
}

type InstallPluginInput struct {
	Source string `json:"source"`
	URL    string `json:"url"`
}

type LocalInstallPluginInput struct {
	Path string `json:"path"`
}

type DeveloperModeState struct {
	Enabled bool `json:"enabled"`
}

type CommandContribution struct {
	ID         string       `json:"id"`
	Title      string       `json:"title"`
	Category   string       `json:"category"`
	PluginID   string       `json:"plugin_id"`
	PluginName string       `json:"plugin_name"`
	Enabled    bool         `json:"enabled"`
	Status     PluginStatus `json:"status"`
}

type CommandExecutionResult struct {
	CommandID  string `json:"command_id"`
	PluginID   string `json:"plugin_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	AuditLogID string `json:"audit_log_id"`
	ExecutedAt string `json:"executed_at"`
}

type ToolIntegrationContribution struct {
	ID                string       `json:"id"`
	Title             string       `json:"title"`
	PluginID          string       `json:"plugin_id"`
	PluginName        string       `json:"plugin_name"`
	Enabled           bool         `json:"enabled"`
	Status            PluginStatus `json:"status"`
	SupportsDetect    bool         `json:"supports_detect"`
	SupportsConfigure bool         `json:"supports_configure"`
	SupportsRestore   bool         `json:"supports_restore"`
}

type ToolIntegrationActionResult struct {
	IntegrationID string `json:"integration_id"`
	PluginID      string `json:"plugin_id"`
	Action        string `json:"action"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	AuditLogID    string `json:"audit_log_id"`
	ExecutedAt    string `json:"executed_at"`
}

type DeclaredProcessContribution struct {
	ID         string       `json:"id"`
	Command    string       `json:"command"`
	Args       []string     `json:"args"`
	TimeoutMs  int          `json:"timeout_ms"`
	PluginID   string       `json:"plugin_id"`
	PluginName string       `json:"plugin_name"`
	Enabled    bool         `json:"enabled"`
	Status     PluginStatus `json:"status"`
}

type BackgroundTaskContribution struct {
	ID                     string       `json:"id"`
	Title                  string       `json:"title"`
	MinimumIntervalSeconds int          `json:"minimum_interval_seconds"`
	PluginID               string       `json:"plugin_id"`
	PluginName             string       `json:"plugin_name"`
	Enabled                bool         `json:"enabled"`
	Status                 PluginStatus `json:"status"`
}

type SettingsSchema struct {
	Type       string                      `json:"type"`
	Title      string                      `json:"title,omitempty"`
	Properties map[string]SettingsProperty `json:"properties"`
	Required   []string                    `json:"required,omitempty"`
}

type SettingsProperty struct {
	Type        string            `json:"type"`
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Default     any               `json:"default,omitempty"`
	Enum        []any             `json:"enum,omitempty"`
	Items       *SettingsProperty `json:"items,omitempty"`
}

type PluginSettings struct {
	PluginID        string         `json:"plugin_id"`
	Schema          SettingsSchema `json:"schema"`
	Values          map[string]any `json:"values"`
	EffectiveValues map[string]any `json:"effective_values"`
	UpdatedAt       string         `json:"updated_at"`
}

type UpdateSettingsInput struct {
	Values map[string]json.RawMessage `json:"values"`
}

type AuditLogEntry struct {
	ID             string         `json:"id"`
	Timestamp      string         `json:"timestamp"`
	PluginID       string         `json:"plugin_id"`
	PluginVersion  string         `json:"plugin_version"`
	Capability     string         `json:"capability"`
	Action         string         `json:"action"`
	ResourceType   string         `json:"resource_type,omitempty"`
	ResourceID     string         `json:"resource_id,omitempty"`
	Status         string         `json:"status"`
	LatencyMs      *int64         `json:"latency_ms,omitempty"`
	ApprovalSource string         `json:"approval_source,omitempty"`
	ErrorMessage   string         `json:"error_message,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

func pluginStatusForEnabled(enabled bool, fallback PluginStatus) PluginStatus {
	if enabled {
		return PluginStatusEnabled
	}
	if fallback == PluginStatusDisabled {
		return PluginStatusDisabled
	}
	return PluginStatusInstalled
}
