export interface ModelPreset {
  id: string;
  label: string;
  description?: string;
  aliases?: string[];
  providers: ModelPresetProvider[];
  tags?: string[];
  deprecated?: boolean;
  disabled?: boolean;
}

export interface ModelPresetProvider {
  id?: string;
  label?: string;
  provider_type: "openai-compatible" | "anthropic-compatible";
  base_url: string;
  models_api?: "auto" | "supported" | "unsupported";
  model_ids: string[];
}

export interface ModelPresetCatalog {
  schema_version: number;
  updated_at?: string;
  presets: ModelPreset[];
  source_url?: string;
  cached_at?: string;
  last_refresh_error?: string;
}
