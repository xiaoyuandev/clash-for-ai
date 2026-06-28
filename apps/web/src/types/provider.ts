export type AuthMode = "bearer" | "x-api-key" | "both";

export interface ProviderCapabilities {
  supports_openai_compatible: boolean;
  supports_anthropic_compatible: boolean;
  supports_models_api: boolean;
  supports_balance_api: boolean;
  supports_stream: boolean;
}

export interface ProviderStatus {
  is_active: boolean;
  last_health_status: string;
  last_healthcheck_at?: string;
}

export interface ClaudeCodeModelMap {
  opus: string;
  sonnet: string;
  haiku: string;
}

export interface CodexModelEntry {
  provider_id?: string;
  model_id: string;
  display_name: string;
  enabled: boolean;
  position: number;
  context_window?: number;
}

export interface CodexModelCatalogState {
  enabled: boolean;
  catalog_path: string;
  hide_official_models: boolean;
}

export interface Provider {
  id: string;
  name: string;
  base_url: string;
  models_path: string;
  api_key: string;
  auth_mode: AuthMode;
  extra_headers: Record<string, string>;
  capabilities: ProviderCapabilities;
  status: ProviderStatus;
  api_key_masked: string;
  claude_code_model_map: ClaudeCodeModelMap;
  is_system_managed: boolean;
  is_editable: boolean;
  is_deletable: boolean;
  runtime_kind: string;
}
