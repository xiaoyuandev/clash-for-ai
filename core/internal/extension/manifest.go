package extension

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const ManifestFileName = "relay-switch-plugin.json"

var (
	pluginIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*(\.[a-z0-9][a-z0-9-]*)*$`)
	semverPattern   = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
)

var knownPermissions = map[string]struct{}{
	"filesystem.homeConfig":                  {},
	"filesystem.pluginData":                  {},
	"filesystem.toolTranscriptStore":         {},
	"filesystem.userSelectedDirectory.write": {},
	"gateway.request.body.read":              {},
	"gateway.request.body.write":             {},
	"gateway.response.body.read":             {},
	"gateway.response.body.write":            {},
	"network.external":                       {},
	"process.exec":                           {},
	"process.exec.declared":                  {},
	"process.readVersion":                    {},
	"provider.request.proxy.streaming":       {},
	"tool.config.backup":                     {},
	"tool.config.read":                       {},
	"tool.config.write":                      {},
	"tool.transcripts.read":                  {},
}

var knownContributionPoints = map[string]struct{}{
	"backgroundTasks":          {},
	"commands":                 {},
	"conversationExporters":    {},
	"gatewayHooks":             {},
	"gatewayProtocolBridges":   {},
	"managedToolBinaries":      {},
	"pages":                    {},
	"providerRequestTemplates": {},
	"runtimeAdapters":          {},
	"settings":                 {},
	"toolIntegrations":         {},
	"transcriptSources":        {},
}

func LoadManifest(path string) (Manifest, []string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return Manifest{}, nil, fmt.Errorf("decode manifest: %w", err)
	}

	warnings, err := ValidateManifest(manifest)
	if err != nil {
		return manifest, warnings, err
	}

	return manifest, warnings, nil
}

func ValidateManifest(manifest Manifest) ([]string, error) {
	var problems []string

	if manifest.ManifestVersion != 1 {
		problems = append(problems, "manifestVersion must be 1")
	}
	if strings.TrimSpace(manifest.ID) == "" {
		problems = append(problems, "id is required")
	} else if !pluginIDPattern.MatchString(manifest.ID) {
		problems = append(problems, "id must be a slug or reverse-domain identifier")
	}
	if strings.TrimSpace(manifest.Name) == "" {
		problems = append(problems, "name is required")
	}
	if strings.TrimSpace(manifest.Version) == "" {
		problems = append(problems, "version is required")
	} else if !semverPattern.MatchString(manifest.Version) {
		problems = append(problems, "version must be valid semver")
	}

	entryType := strings.TrimSpace(manifest.Entry.Type)
	switch entryType {
	case "none":
	case "process":
		if strings.TrimSpace(manifest.Entry.Command) == "" {
			problems = append(problems, "entry.command is required for process entries")
		} else if isDangerousEntryCommand(manifest.Entry.Command) {
			problems = append(problems, "entry.command points to a disallowed executable")
		}
	default:
		problems = append(problems, "entry.type must be process or none")
	}

	for _, permission := range manifest.Permissions {
		if _, ok := knownPermissions[permission]; !ok {
			problems = append(problems, fmt.Sprintf("unknown permission: %s", permission))
		}
	}

	if len(problems) > 0 {
		return ManifestWarnings(manifest), errors.New(strings.Join(problems, "; "))
	}

	return ManifestWarnings(manifest), nil
}

func ManifestWarnings(manifest Manifest) []string {
	warnings := []string{}
	for name := range manifest.Contributes {
		if _, ok := knownContributionPoints[name]; !ok {
			warnings = append(warnings, fmt.Sprintf("unsupported contribution point: %s", name))
		}
	}
	sort.Strings(warnings)
	return warnings
}

func ContributionSummary(contributes ManifestContributes) map[string]int {
	summary := map[string]int{}
	for name, raw := range contributes {
		var list []json.RawMessage
		if err := json.Unmarshal(raw, &list); err == nil {
			summary[name] = len(list)
			continue
		}

		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err == nil {
			summary[name] = len(object)
			continue
		}

		if strings.TrimSpace(string(raw)) != "" && string(raw) != "null" {
			summary[name] = 1
		}
	}
	return summary
}

func manifestToPlugin(manifest Manifest, scope PluginScope, manifestPath string, warnings []string) Plugin {
	return Plugin{
		ID:           manifest.ID,
		Name:         manifest.Name,
		Version:      manifest.Version,
		Description:  manifest.Description,
		Publisher:    manifest.Publisher,
		Scope:        scope,
		ManifestPath: manifestPath,
		Status:       PluginStatusInstalled,
		Manifest:     manifest,
		Permissions:  cloneStringSlice(manifest.Permissions),
		Contributes:  ContributionSummary(manifest.Contributes),
		Warnings:     cloneStringSlice(warnings),
	}
}

func normalizePluginFromManifest(item Plugin) Plugin {
	item.Name = item.Manifest.Name
	item.Version = item.Manifest.Version
	item.Description = item.Manifest.Description
	item.Publisher = item.Manifest.Publisher
	item.Permissions = cloneStringSlice(item.Manifest.Permissions)
	item.Contributes = ContributionSummary(item.Manifest.Contributes)
	item.Warnings = ManifestWarnings(item.Manifest)
	return item
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func isDangerousEntryCommand(command string) bool {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return false
	}

	base := strings.ToLower(filepath.Base(trimmed))
	if strings.HasSuffix(base, ".exe") {
		base = strings.TrimSuffix(base, ".exe")
	}

	dangerousNames := map[string]struct{}{
		"bash":       {},
		"cmd":        {},
		"powershell": {},
		"pwsh":       {},
		"rm":         {},
		"sh":         {},
		"su":         {},
		"sudo":       {},
		"zsh":        {},
	}

	if !filepath.IsAbs(trimmed) {
		return false
	}

	_, dangerous := dangerousNames[base]
	return dangerous
}
