export interface ProviderPreset {
  name: string;
  base_url: string;
  models_path?: string;
}

export interface ProviderPresetCatalog {
  schema_version: number;
  updated_at?: string;
  presets: ProviderPreset[];
  source_url?: string;
  cached_at?: string;
  last_refresh_error?: string;
}

export function normalizeProviderPresetCatalog(value: unknown, sourceURL?: string): ProviderPresetCatalog;
export function validateProviderPresetCatalog(value: unknown): void;
