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
var ErrTranscriptOutputDirectoryRequired = errors.New("transcript output directory is required")

type PluginScope string

const (
	PluginScopeUser    PluginScope = "user"
	PluginScopeBundled PluginScope = "bundled"
	PluginScopeProject PluginScope = "project"
	PluginScopeManaged PluginScope = "managed"
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
	Args    []string `json:"args,omitempty"`
}

type ManifestContributes map[string]json.RawMessage

type ScanSource struct {
	Scope PluginScope
	Dir   string
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

type TranscriptSource struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Kind         string   `json:"kind"`
	Enabled      bool     `json:"enabled"`
	SessionCount int      `json:"session_count"`
	Paths        []string `json:"paths"`
}

type TranscriptSyncInput struct {
	PluginID            string `json:"plugin_id,omitempty"`
	OutputDirectory     string `json:"output_directory,omitempty"`
	IncludeClaudeCode   *bool  `json:"include_claude_code,omitempty"`
	IncludeCodexCLI     *bool  `json:"include_codex_cli,omitempty"`
	IncludeSystemEvents *bool  `json:"include_system_events,omitempty"`
	RedactSecrets       *bool  `json:"redact_secrets,omitempty"`
	OverwriteExisting   *bool  `json:"overwrite_existing,omitempty"`
}

type TranscriptSyncResult struct {
	PluginID        string                       `json:"plugin_id"`
	Status          string                       `json:"status"`
	OutputDirectory string                       `json:"output_directory"`
	ExportedCount   int                          `json:"exported_count"`
	SkippedCount    int                          `json:"skipped_count"`
	FailedCount     int                          `json:"failed_count"`
	StartedAt       string                       `json:"started_at"`
	FinishedAt      string                       `json:"finished_at"`
	AuditLogID      string                       `json:"audit_log_id"`
	Sources         []TranscriptSourceSyncResult `json:"sources"`
	Failures        []TranscriptSyncFailure      `json:"failures"`
}

type TranscriptSourceSyncResult struct {
	SourceID      string `json:"source_id"`
	Discovered    int    `json:"discovered"`
	ExportedCount int    `json:"exported_count"`
	SkippedCount  int    `json:"skipped_count"`
	FailedCount   int    `json:"failed_count"`
}

type TranscriptSyncFailure struct {
	SourceID  string `json:"source_id"`
	SessionID string `json:"session_id,omitempty"`
	Path      string `json:"path,omitempty"`
	Error     string `json:"error"`
}

type TranscriptExportState struct {
	Source      string `json:"source"`
	SessionID   string `json:"session_id"`
	RawPath     string `json:"raw_path"`
	RawMTime    int64  `json:"raw_mtime"`
	RawSize     int64  `json:"raw_size"`
	OutputPath  string `json:"output_path"`
	ExportedAt  string `json:"exported_at"`
	ContentHash string `json:"content_hash"`
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
