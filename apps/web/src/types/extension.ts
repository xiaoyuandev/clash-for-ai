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
