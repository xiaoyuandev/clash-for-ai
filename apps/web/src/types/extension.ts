export type PluginScope = "user" | "bundled" | "project" | "managed";

export type PluginStatus =
  | "installed"
  | "enabled"
  | "disabled"
  | "incompatible"
  | "invalid";

export interface ExtensionManifestEntry {
  type: "process" | "none" | string;
  command?: string;
  args?: string[];
}

export interface ExtensionManifest {
  manifestVersion: number;
  id: string;
  name: string;
  version: string;
  description?: string;
  publisher?: string;
  engines?: {
    relaySwitch?: string;
  };
  entry: ExtensionManifestEntry;
  contributes: Record<string, unknown>;
  permissions: string[];
}

export interface ExtensionPlugin {
  id: string;
  name: string;
  version: string;
  description: string;
  publisher: string;
  scope: PluginScope;
  manifest_path: string;
  enabled: boolean;
  status: PluginStatus;
  last_error: string;
  manifest: ExtensionManifest;
  permissions: string[];
  contributes: Record<string, number>;
  warnings: string[];
  created_at: string;
  updated_at: string;
}

export interface ExtensionSettingsProperty {
  type: "string" | "boolean" | "integer" | "number" | "array" | string;
  title?: string;
  description?: string;
  default?: unknown;
  enum?: unknown[];
  items?: ExtensionSettingsProperty;
}

export interface ExtensionSettingsSchema {
  type: "object" | string;
  title?: string;
  properties: Record<string, ExtensionSettingsProperty>;
  required?: string[];
}

export interface ExtensionSettings {
  plugin_id: string;
  schema: ExtensionSettingsSchema;
  values: Record<string, unknown>;
  effective_values: Record<string, unknown>;
  updated_at: string;
}

export interface ExtensionCommand {
  id: string;
  title: string;
  category: string;
  plugin_id: string;
  plugin_name: string;
  enabled: boolean;
  status: PluginStatus;
}

export interface ExtensionCommandResult {
  command_id: string;
  plugin_id: string;
  status: "skipped" | "failed" | string;
  message: string;
  audit_log_id: string;
  executed_at: string;
}

export interface ExtensionToolIntegration {
  id: string;
  title: string;
  plugin_id: string;
  plugin_name: string;
  enabled: boolean;
  status: PluginStatus;
  supports_detect: boolean;
  supports_configure: boolean;
  supports_restore: boolean;
}

export type ExtensionToolIntegrationAction = "detect" | "configure" | "restore";

export interface ExtensionToolIntegrationResult {
  integration_id: string;
  plugin_id: string;
  action: ExtensionToolIntegrationAction | string;
  status: "skipped" | "failed" | string;
  message: string;
  audit_log_id: string;
  executed_at: string;
}

export interface ExtensionDeclaredProcess {
  id: string;
  command: string;
  args: string[];
  timeout_ms: number;
  plugin_id: string;
  plugin_name: string;
  enabled: boolean;
  status: PluginStatus;
}

export interface ExtensionAuditLog {
  id: string;
  timestamp: string;
  plugin_id: string;
  plugin_version: string;
  capability: string;
  action: string;
  resource_type?: string;
  resource_id?: string;
  status: string;
  latency_ms?: number;
  approval_source?: string;
  error_message?: string;
  metadata?: Record<string, unknown>;
}
