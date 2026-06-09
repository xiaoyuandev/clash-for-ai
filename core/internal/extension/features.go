package extension

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"
)

type manifestCommandContribution struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
}

type manifestToolIntegrationContribution struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	SupportsDetect    bool   `json:"supportsDetect"`
	SupportsConfigure bool   `json:"supportsConfigure"`
	SupportsRestore   bool   `json:"supportsRestore"`
}

type manifestDeclaredProcessContribution struct {
	ID        string   `json:"id"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	TimeoutMs int      `json:"timeoutMs"`
}

type manifestBackgroundTaskContribution struct {
	ID                     string `json:"id"`
	Title                  string `json:"title"`
	MinimumIntervalSeconds int    `json:"minimumIntervalSeconds"`
}

type manifestSettingContribution struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Default     any    `json:"default"`
	Enum        []any  `json:"enum"`
}

func (s *Service) ListCommands(ctx context.Context) ([]CommandContribution, error) {
	plugins, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	items := []CommandContribution{}
	for _, plugin := range plugins {
		if plugin.Status == PluginStatusInvalid || plugin.Status == PluginStatusIncompatible {
			continue
		}
		for _, command := range commandsFromPlugin(plugin) {
			items = append(items, command)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].PluginID == items[j].PluginID {
			return items[i].ID < items[j].ID
		}
		return items[i].PluginID < items[j].PluginID
	})
	return items, nil
}

func (s *Service) ExecuteCommand(ctx context.Context, commandID string) (CommandExecutionResult, error) {
	command, plugin, err := s.findCommand(ctx, commandID)
	if err != nil {
		return CommandExecutionResult{}, err
	}

	executedAt := time.Now().UTC().Format(time.RFC3339)
	if !plugin.Enabled || plugin.Status != PluginStatusEnabled {
		entry, auditErr := s.repository.RecordAudit(ctx, AuditLogEntry{
			PluginID:      plugin.ID,
			PluginVersion: plugin.Version,
			Capability:    "commands.execute",
			Action:        command.ID,
			ResourceType:  "command",
			ResourceID:    command.ID,
			Status:        "failed",
			ErrorMessage:  ErrPluginNotEnabled.Error(),
			Metadata: map[string]any{
				"mode": "noop",
			},
		})
		if auditErr != nil {
			return CommandExecutionResult{}, auditErr
		}
		return CommandExecutionResult{
			CommandID:  command.ID,
			PluginID:   plugin.ID,
			Status:     "failed",
			Message:    ErrPluginNotEnabled.Error(),
			AuditLogID: entry.ID,
			ExecutedAt: executedAt,
		}, ErrPluginNotEnabled
	}

	if plan, planErr := RuntimePlanForPlugin(plugin); s.runtimeHost != nil && planErr == nil && plan.EntryType != "none" {
		result, runtimeErr := s.executeCommandThroughRuntime(ctx, plugin, command, executedAt)
		if runtimeErr != nil {
			return result, runtimeErr
		}
		return result, nil
	}

	entry, err := s.repository.RecordAudit(ctx, AuditLogEntry{
		PluginID:      plugin.ID,
		PluginVersion: plugin.Version,
		Capability:    "commands.execute",
		Action:        command.ID,
		ResourceType:  "command",
		ResourceID:    command.ID,
		Status:        "skipped",
		Metadata: map[string]any{
			"mode":   "noop",
			"reason": "extension host is not available in milestone 2",
		},
	})
	if err != nil {
		return CommandExecutionResult{}, err
	}

	return CommandExecutionResult{
		CommandID:  command.ID,
		PluginID:   plugin.ID,
		Status:     "skipped",
		Message:    "Extension host is not available; command execution was recorded as a no-op.",
		AuditLogID: entry.ID,
		ExecutedAt: executedAt,
	}, nil
}

func (s *Service) executeCommandThroughRuntime(ctx context.Context, plugin Plugin, command CommandContribution, executedAt string) (CommandExecutionResult, error) {
	type runtimeCommandResult struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	if err := s.guardDevelopmentRuntime(ctx, plugin); err != nil {
		entry, auditErr := s.repository.RecordAudit(ctx, AuditLogEntry{
			PluginID:      plugin.ID,
			PluginVersion: plugin.Version,
			Capability:    "commands.execute",
			Action:        command.ID,
			ResourceType:  "command",
			ResourceID:    command.ID,
			Status:        "failed",
			ErrorMessage:  err.Error(),
			Metadata: map[string]any{
				"mode": "runtime",
			},
		})
		if auditErr != nil {
			return CommandExecutionResult{}, auditErr
		}
		return CommandExecutionResult{
			CommandID:  command.ID,
			PluginID:   plugin.ID,
			Status:     "failed",
			Message:    err.Error(),
			AuditLogID: entry.ID,
			ExecutedAt: executedAt,
		}, err
	}

	initialSettings := s.effectiveSettingsForRuntime(ctx, plugin.ID)
	payload, err := s.runtimeHost.Call(ctx, plugin, "executeCommand", map[string]any{
		"commandId": command.ID,
	}, initialSettings)
	if err != nil {
		entry, auditErr := s.repository.RecordAudit(ctx, AuditLogEntry{
			PluginID:      plugin.ID,
			PluginVersion: plugin.Version,
			Capability:    "commands.execute",
			Action:        command.ID,
			ResourceType:  "command",
			ResourceID:    command.ID,
			Status:        "failed",
			ErrorMessage:  err.Error(),
			Metadata: map[string]any{
				"mode": "runtime",
			},
		})
		if auditErr != nil {
			return CommandExecutionResult{}, auditErr
		}
		return CommandExecutionResult{
			CommandID:  command.ID,
			PluginID:   plugin.ID,
			Status:     "failed",
			Message:    err.Error(),
			AuditLogID: entry.ID,
			ExecutedAt: executedAt,
		}, err
	}

	runtimeResult := runtimeCommandResult{
		Status:  "success",
		Message: "Command executed by plugin runtime.",
	}
	if len(payload) > 0 && string(payload) != "null" {
		_ = json.Unmarshal(payload, &runtimeResult)
	}
	if strings.TrimSpace(runtimeResult.Status) == "" {
		runtimeResult.Status = "success"
	}

	entry, err := s.repository.RecordAudit(ctx, AuditLogEntry{
		PluginID:      plugin.ID,
		PluginVersion: plugin.Version,
		Capability:    "commands.execute",
		Action:        command.ID,
		ResourceType:  "command",
		ResourceID:    command.ID,
		Status:        runtimeResult.Status,
		Metadata: map[string]any{
			"mode": "runtime",
		},
	})
	if err != nil {
		return CommandExecutionResult{}, err
	}

	return CommandExecutionResult{
		CommandID:  command.ID,
		PluginID:   plugin.ID,
		Status:     runtimeResult.Status,
		Message:    runtimeResult.Message,
		AuditLogID: entry.ID,
		ExecutedAt: executedAt,
	}, nil
}

func (s *Service) GetSettings(ctx context.Context, pluginID string) (PluginSettings, error) {
	plugin, err := s.repository.GetByID(ctx, pluginID)
	if err != nil {
		return PluginSettings{}, err
	}
	if plugin.Status == PluginStatusInvalid || plugin.Status == PluginStatusIncompatible {
		return PluginSettings{}, ErrPluginInvalid
	}

	schema, err := settingsSchemaFromPlugin(*plugin)
	if err != nil {
		return PluginSettings{}, err
	}

	rawValues, updatedAt, err := s.repository.GetSettings(ctx, plugin.ID)
	if err != nil {
		return PluginSettings{}, err
	}

	values, err := decodeSettingsValues(schema, rawValues)
	if err != nil {
		return PluginSettings{}, err
	}

	return PluginSettings{
		PluginID:        plugin.ID,
		Schema:          schema,
		Values:          values,
		EffectiveValues: effectiveSettingsValues(schema, values),
		UpdatedAt:       updatedAt,
	}, nil
}

func (s *Service) UpdateSettings(ctx context.Context, pluginID string, input UpdateSettingsInput) (PluginSettings, error) {
	plugin, err := s.repository.GetByID(ctx, pluginID)
	if err != nil {
		return PluginSettings{}, err
	}
	if plugin.Status == PluginStatusInvalid || plugin.Status == PluginStatusIncompatible {
		return PluginSettings{}, ErrPluginInvalid
	}

	schema, err := settingsSchemaFromPlugin(*plugin)
	if err != nil {
		return PluginSettings{}, err
	}

	normalized, _, err := validateSettingsValues(schema, input.Values)
	if err != nil {
		return PluginSettings{}, err
	}

	if _, err := s.repository.ReplaceSettings(ctx, plugin.ID, normalized); err != nil {
		return PluginSettings{}, err
	}

	if _, err := s.repository.RecordAudit(ctx, AuditLogEntry{
		PluginID:      plugin.ID,
		PluginVersion: plugin.Version,
		Capability:    "settings.write",
		Action:        "settings.update",
		ResourceType:  "settings",
		ResourceID:    plugin.ID,
		Status:        "success",
		Metadata: map[string]any{
			"keys": sortedRawKeys(normalized),
		},
	}); err != nil {
		return PluginSettings{}, err
	}

	settings, err := s.GetSettings(ctx, plugin.ID)
	if err != nil {
		return PluginSettings{}, err
	}
	if plugin.Enabled && plugin.Status == PluginStatusEnabled && s.runtimeHost != nil {
		if plan, planErr := RuntimePlanForPlugin(*plugin); planErr == nil && plan.EntryType != "none" {
			if err := s.guardDevelopmentRuntime(ctx, *plugin); err == nil {
				_ = s.runtimeHost.Notify(ctx, *plugin, "settingsChanged", map[string]any{
					"values": settings.EffectiveValues,
				}, settings.EffectiveValues)
			}
		}
	}
	return settings, nil
}

func (s *Service) ListAudit(ctx context.Context, pluginID string, limit int) ([]AuditLogEntry, error) {
	if strings.TrimSpace(pluginID) != "" {
		if _, err := s.repository.GetByID(ctx, pluginID); err != nil {
			return nil, err
		}
	}
	return s.repository.ListAudit(ctx, pluginID, normalizeAuditLimit(limit))
}

func (s *Service) findCommand(ctx context.Context, commandID string) (CommandContribution, Plugin, error) {
	plugins, err := s.repository.List(ctx)
	if err != nil {
		return CommandContribution{}, Plugin{}, err
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].ID < plugins[j].ID
	})
	for _, plugin := range plugins {
		if plugin.Status == PluginStatusInvalid || plugin.Status == PluginStatusIncompatible {
			continue
		}
		for _, command := range commandsFromPlugin(plugin) {
			if command.ID == commandID {
				return command, plugin, nil
			}
		}
	}

	return CommandContribution{}, Plugin{}, ErrCommandNotFound
}

func (s *Service) ListToolIntegrations(ctx context.Context) ([]ToolIntegrationContribution, error) {
	plugins, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	items := []ToolIntegrationContribution{}
	for _, plugin := range plugins {
		if plugin.Status == PluginStatusInvalid || plugin.Status == PluginStatusIncompatible {
			continue
		}
		for _, integration := range toolIntegrationsFromPlugin(plugin) {
			items = append(items, integration)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].PluginID == items[j].PluginID {
			return items[i].ID < items[j].ID
		}
		return items[i].PluginID < items[j].PluginID
	})
	return items, nil
}

func (s *Service) ListDeclaredProcesses(ctx context.Context) ([]DeclaredProcessContribution, error) {
	plugins, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	items := []DeclaredProcessContribution{}
	for _, plugin := range plugins {
		if plugin.Status == PluginStatusInvalid || plugin.Status == PluginStatusIncompatible {
			continue
		}
		for _, process := range declaredProcessesFromPlugin(plugin) {
			items = append(items, process)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].PluginID == items[j].PluginID {
			return items[i].ID < items[j].ID
		}
		return items[i].PluginID < items[j].PluginID
	})
	return items, nil
}

func (s *Service) ListBackgroundTasks(ctx context.Context) ([]BackgroundTaskContribution, error) {
	plugins, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	items := []BackgroundTaskContribution{}
	for _, plugin := range plugins {
		if plugin.Status == PluginStatusInvalid || plugin.Status == PluginStatusIncompatible {
			continue
		}
		for _, task := range backgroundTasksFromPlugin(plugin) {
			items = append(items, task)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].PluginID == items[j].PluginID {
			return items[i].ID < items[j].ID
		}
		return items[i].PluginID < items[j].PluginID
	})
	return items, nil
}

func (s *Service) ExecuteToolIntegrationAction(ctx context.Context, integrationID string, action string) (ToolIntegrationActionResult, error) {
	integration, plugin, err := s.findToolIntegration(ctx, integrationID)
	if err != nil {
		return ToolIntegrationActionResult{}, err
	}
	if !toolIntegrationSupportsAction(integration, action) {
		return ToolIntegrationActionResult{}, ErrToolIntegrationActionUnsupported
	}

	executedAt := time.Now().UTC().Format(time.RFC3339)
	metadata := map[string]any{
		"mode":               "noop",
		"declared_processes": declaredProcessIDs(declaredProcessesFromPlugin(plugin)),
	}
	capability := "tool.integration." + action

	if !plugin.Enabled || plugin.Status != PluginStatusEnabled {
		entry, auditErr := s.repository.RecordAudit(ctx, AuditLogEntry{
			PluginID:      plugin.ID,
			PluginVersion: plugin.Version,
			Capability:    capability,
			Action:        action,
			ResourceType:  "toolIntegration",
			ResourceID:    integration.ID,
			Status:        "failed",
			ErrorMessage:  ErrPluginNotEnabled.Error(),
			Metadata:      metadata,
		})
		if auditErr != nil {
			return ToolIntegrationActionResult{}, auditErr
		}
		return ToolIntegrationActionResult{
			IntegrationID: integration.ID,
			PluginID:      plugin.ID,
			Action:        action,
			Status:        "failed",
			Message:       ErrPluginNotEnabled.Error(),
			AuditLogID:    entry.ID,
			ExecutedAt:    executedAt,
		}, ErrPluginNotEnabled
	}

	entry, err := s.repository.RecordAudit(ctx, AuditLogEntry{
		PluginID:      plugin.ID,
		PluginVersion: plugin.Version,
		Capability:    capability,
		Action:        action,
		ResourceType:  "toolIntegration",
		ResourceID:    integration.ID,
		Status:        "skipped",
		Metadata: mergeMetadata(metadata, map[string]any{
			"reason": "process broker is not available in milestone 3",
		}),
	})
	if err != nil {
		return ToolIntegrationActionResult{}, err
	}

	return ToolIntegrationActionResult{
		IntegrationID: integration.ID,
		PluginID:      plugin.ID,
		Action:        action,
		Status:        "skipped",
		Message:       "Process broker is not available; tool integration action was recorded as a no-op.",
		AuditLogID:    entry.ID,
		ExecutedAt:    executedAt,
	}, nil
}

func commandsFromPlugin(plugin Plugin) []CommandContribution {
	raw, ok := plugin.Manifest.Contributes["commands"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return []CommandContribution{}
	}

	var manifests []manifestCommandContribution
	if err := json.Unmarshal(raw, &manifests); err != nil {
		return []CommandContribution{}
	}

	commands := make([]CommandContribution, 0, len(manifests))
	for _, manifest := range manifests {
		id := strings.TrimSpace(manifest.ID)
		title := strings.TrimSpace(manifest.Title)
		if id == "" || title == "" {
			continue
		}
		commands = append(commands, CommandContribution{
			ID:         id,
			Title:      title,
			Category:   strings.TrimSpace(manifest.Category),
			PluginID:   plugin.ID,
			PluginName: plugin.Name,
			Enabled:    plugin.Enabled,
			Status:     plugin.Status,
		})
	}
	return commands
}

func (s *Service) findToolIntegration(ctx context.Context, integrationID string) (ToolIntegrationContribution, Plugin, error) {
	plugins, err := s.repository.List(ctx)
	if err != nil {
		return ToolIntegrationContribution{}, Plugin{}, err
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].ID < plugins[j].ID
	})
	for _, plugin := range plugins {
		if plugin.Status == PluginStatusInvalid || plugin.Status == PluginStatusIncompatible {
			continue
		}
		for _, integration := range toolIntegrationsFromPlugin(plugin) {
			if integration.ID == integrationID {
				return integration, plugin, nil
			}
		}
	}

	return ToolIntegrationContribution{}, Plugin{}, ErrToolIntegrationNotFound
}

func toolIntegrationsFromPlugin(plugin Plugin) []ToolIntegrationContribution {
	raw, ok := plugin.Manifest.Contributes["toolIntegrations"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return []ToolIntegrationContribution{}
	}

	var manifests []manifestToolIntegrationContribution
	if err := json.Unmarshal(raw, &manifests); err != nil {
		return []ToolIntegrationContribution{}
	}

	items := make([]ToolIntegrationContribution, 0, len(manifests))
	for _, manifest := range manifests {
		id := strings.TrimSpace(manifest.ID)
		title := strings.TrimSpace(manifest.Title)
		if id == "" || title == "" {
			continue
		}
		items = append(items, ToolIntegrationContribution{
			ID:                id,
			Title:             title,
			PluginID:          plugin.ID,
			PluginName:        plugin.Name,
			Enabled:           plugin.Enabled,
			Status:            plugin.Status,
			SupportsDetect:    manifest.SupportsDetect,
			SupportsConfigure: manifest.SupportsConfigure,
			SupportsRestore:   manifest.SupportsRestore,
		})
	}
	return items
}

func declaredProcessesFromPlugin(plugin Plugin) []DeclaredProcessContribution {
	raw, ok := plugin.Manifest.Contributes["declaredProcesses"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return []DeclaredProcessContribution{}
	}

	var manifests []manifestDeclaredProcessContribution
	if err := json.Unmarshal(raw, &manifests); err != nil {
		return []DeclaredProcessContribution{}
	}

	items := make([]DeclaredProcessContribution, 0, len(manifests))
	for _, manifest := range manifests {
		id := strings.TrimSpace(manifest.ID)
		command := strings.TrimSpace(manifest.Command)
		if id == "" || command == "" {
			continue
		}
		args := append([]string(nil), manifest.Args...)
		if args == nil {
			args = []string{}
		}
		items = append(items, DeclaredProcessContribution{
			ID:         id,
			Command:    command,
			Args:       args,
			TimeoutMs:  manifest.TimeoutMs,
			PluginID:   plugin.ID,
			PluginName: plugin.Name,
			Enabled:    plugin.Enabled,
			Status:     plugin.Status,
		})
	}
	return items
}

func backgroundTasksFromPlugin(plugin Plugin) []BackgroundTaskContribution {
	raw, ok := plugin.Manifest.Contributes["backgroundTasks"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return []BackgroundTaskContribution{}
	}

	var manifests []manifestBackgroundTaskContribution
	if err := json.Unmarshal(raw, &manifests); err != nil {
		return []BackgroundTaskContribution{}
	}

	items := make([]BackgroundTaskContribution, 0, len(manifests))
	for _, manifest := range manifests {
		id := strings.TrimSpace(manifest.ID)
		title := strings.TrimSpace(manifest.Title)
		if id == "" || title == "" {
			continue
		}
		items = append(items, BackgroundTaskContribution{
			ID:                     id,
			Title:                  title,
			MinimumIntervalSeconds: manifest.MinimumIntervalSeconds,
			PluginID:               plugin.ID,
			PluginName:             plugin.Name,
			Enabled:                plugin.Enabled,
			Status:                 plugin.Status,
		})
	}
	return items
}

func toolIntegrationSupportsAction(integration ToolIntegrationContribution, action string) bool {
	switch action {
	case "detect":
		return integration.SupportsDetect
	case "configure":
		return integration.SupportsConfigure
	case "restore":
		return integration.SupportsRestore
	default:
		return false
	}
}

func declaredProcessIDs(processes []DeclaredProcessContribution) []string {
	ids := make([]string, 0, len(processes))
	for _, process := range processes {
		ids = append(ids, process.ID)
	}
	sort.Strings(ids)
	return ids
}

func mergeMetadata(base map[string]any, extra map[string]any) map[string]any {
	merged := map[string]any{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func settingsSchemaFromPlugin(plugin Plugin) (SettingsSchema, error) {
	empty := SettingsSchema{
		Type:       "object",
		Properties: map[string]SettingsProperty{},
		Required:   []string{},
	}

	raw, ok := plugin.Manifest.Contributes["settings"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return empty, nil
	}

	var schema SettingsSchema
	if err := json.Unmarshal(raw, &schema); err == nil && (schema.Type != "" || schema.Properties != nil) {
		if schema.Type == "" {
			schema.Type = "object"
		}
		if schema.Properties == nil {
			schema.Properties = map[string]SettingsProperty{}
		}
		if schema.Required == nil {
			schema.Required = []string{}
		}
		return validateSettingsSchema(schema)
	}

	var legacy []manifestSettingContribution
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return SettingsSchema{}, fmt.Errorf("%w: settings contribution must be a JSON Schema object", ErrInvalidSettings)
	}

	schema = empty
	for _, item := range legacy {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		settingType := strings.TrimSpace(item.Type)
		if settingType == "" {
			settingType = "string"
		}
		schema.Properties[id] = SettingsProperty{
			Type:        settingType,
			Title:       strings.TrimSpace(item.Title),
			Description: strings.TrimSpace(item.Description),
			Default:     item.Default,
			Enum:        item.Enum,
		}
	}
	return validateSettingsSchema(schema)
}

func validateSettingsSchema(schema SettingsSchema) (SettingsSchema, error) {
	if schema.Type != "object" {
		return SettingsSchema{}, fmt.Errorf("%w: settings schema root must be an object", ErrInvalidSettings)
	}

	for name, property := range schema.Properties {
		if strings.TrimSpace(name) == "" {
			return SettingsSchema{}, fmt.Errorf("%w: settings property name is required", ErrInvalidSettings)
		}
		if err := validateSettingsPropertySchema(property); err != nil {
			return SettingsSchema{}, fmt.Errorf("%w: %s", err, name)
		}
	}

	return schema, nil
}

func validateSettingsPropertySchema(property SettingsProperty) error {
	switch property.Type {
	case "string", "boolean", "integer", "number":
		return nil
	case "array":
		if property.Items != nil && property.Items.Type != "" && property.Items.Type != "string" {
			return fmt.Errorf("%w: only string array items are supported", ErrInvalidSettings)
		}
		return nil
	default:
		return fmt.Errorf("%w: unsupported settings type %q", ErrInvalidSettings, property.Type)
	}
}

func validateSettingsValues(schema SettingsSchema, values map[string]json.RawMessage) (map[string]json.RawMessage, map[string]any, error) {
	if values == nil {
		values = map[string]json.RawMessage{}
	}

	normalized := map[string]json.RawMessage{}
	decoded := map[string]any{}
	for key, raw := range values {
		property, ok := schema.Properties[key]
		if !ok {
			return nil, nil, fmt.Errorf("%w: unknown settings key %q", ErrInvalidSettings, key)
		}

		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, nil, fmt.Errorf("%w: decode %s: %v", ErrInvalidSettings, key, err)
		}
		if err := validateSettingsValue(property, value); err != nil {
			return nil, nil, fmt.Errorf("%w: %s", err, key)
		}

		canonical, err := json.Marshal(value)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: encode %s: %v", ErrInvalidSettings, key, err)
		}
		normalized[key] = canonical
		decoded[key] = value
	}

	for _, key := range schema.Required {
		if _, ok := decoded[key]; ok {
			continue
		}
		property, ok := schema.Properties[key]
		if ok && property.Default != nil {
			continue
		}
		return nil, nil, fmt.Errorf("%w: required settings key %q is missing", ErrInvalidSettings, key)
	}

	return normalized, decoded, nil
}

func decodeSettingsValues(schema SettingsSchema, values map[string]json.RawMessage) (map[string]any, error) {
	_, decoded, err := validateSettingsValuesWithoutRequired(schema, values)
	return decoded, err
}

func validateSettingsValuesWithoutRequired(schema SettingsSchema, values map[string]json.RawMessage) (map[string]json.RawMessage, map[string]any, error) {
	if values == nil {
		values = map[string]json.RawMessage{}
	}

	normalized := map[string]json.RawMessage{}
	decoded := map[string]any{}
	for key, raw := range values {
		property, ok := schema.Properties[key]
		if !ok {
			continue
		}
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, nil, fmt.Errorf("%w: decode %s: %v", ErrInvalidSettings, key, err)
		}
		if err := validateSettingsValue(property, value); err != nil {
			return nil, nil, fmt.Errorf("%w: %s", err, key)
		}
		canonical, err := json.Marshal(value)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: encode %s: %v", ErrInvalidSettings, key, err)
		}
		normalized[key] = canonical
		decoded[key] = value
	}
	return normalized, decoded, nil
}

func validateSettingsValue(property SettingsProperty, value any) error {
	if len(property.Enum) > 0 && !enumContains(property.Enum, value) {
		return fmt.Errorf("%w: value is not in enum", ErrInvalidSettings)
	}

	switch property.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%w: expected string", ErrInvalidSettings)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%w: expected boolean", ErrInvalidSettings)
		}
	case "integer":
		number, ok := value.(float64)
		if !ok || math.Trunc(number) != number {
			return fmt.Errorf("%w: expected integer", ErrInvalidSettings)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("%w: expected number", ErrInvalidSettings)
		}
	case "array":
		items, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%w: expected array", ErrInvalidSettings)
		}
		if property.Items != nil && property.Items.Type == "string" {
			for _, item := range items {
				if _, ok := item.(string); !ok {
					return fmt.Errorf("%w: expected string array", ErrInvalidSettings)
				}
			}
		}
	default:
		return fmt.Errorf("%w: unsupported settings type %q", ErrInvalidSettings, property.Type)
	}

	return nil
}

func effectiveSettingsValues(schema SettingsSchema, values map[string]any) map[string]any {
	effective := map[string]any{}
	for key, property := range schema.Properties {
		if property.Default != nil {
			effective[key] = property.Default
		}
	}
	for key, value := range values {
		effective[key] = value
	}
	return effective
}

func enumContains(enum []any, value any) bool {
	for _, item := range enum {
		if reflect.DeepEqual(item, value) {
			return true
		}
	}
	return false
}

func sortedRawKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func normalizeAuditLimit(limit int) int {
	if limit <= 0 {
		return 100
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func isPluginStateError(err error) bool {
	return errors.Is(err, ErrPluginInvalid) || errors.Is(err, ErrPluginNotEnabled)
}
