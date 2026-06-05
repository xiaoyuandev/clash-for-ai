package extension

import (
	"encoding/json"
	"errors"
)

var ErrPluginNotFound = errors.New("plugin not found")
var ErrCommandNotFound = errors.New("extension command not found")
var ErrPluginNotEnabled = errors.New("plugin is not enabled")
var ErrPluginInvalid = errors.New("plugin is invalid")
var ErrInvalidSettings = errors.New("invalid plugin settings")

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
