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

export interface Provider {
  id: string;
  name: string;
  base_url: string;
  auth_mode: AuthMode;
  extra_headers: Record<string, string>;
  capabilities: ProviderCapabilities;
  status: ProviderStatus;
  api_key_masked: string;
}
