package extension

import (
	"encoding/json"
	"errors"
)

var ErrPluginNotFound = errors.New("plugin not found")

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

func pluginStatusForEnabled(enabled bool, fallback PluginStatus) PluginStatus {
	if enabled {
		return PluginStatusEnabled
	}
	if fallback == PluginStatusDisabled {
		return PluginStatusDisabled
	}
	return PluginStatusInstalled
}
