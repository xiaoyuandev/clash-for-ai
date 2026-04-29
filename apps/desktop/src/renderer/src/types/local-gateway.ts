import type { ModelSource } from "./model-source";
import type { SelectedModel } from "./selected-model";

export interface LocalGatewayRuntimeCapabilities {
  supports_openai_compatible: boolean;
  supports_anthropic_compatible: boolean;
  supports_models_api: boolean;
  supports_stream: boolean;
  supports_admin_api: boolean;
  supports_model_source_admin: boolean;
  supports_selected_model_admin: boolean;
}

export interface LocalGatewayRuntimeInfo {
  base_url: string;
  listen_addr?: string;
  mode: string;
  embedded: boolean;
}

export interface LocalGatewayRuntimeHealth {
  status: string;
  summary: string;
  checked_at: string;
}

export interface LocalGatewayRuntimeStatus {
  provider_id: string;
  base_url: string;
  runtime: LocalGatewayRuntimeInfo;
  health: LocalGatewayRuntimeHealth;
  capabilities: LocalGatewayRuntimeCapabilities;
  missing_optional_capabilities: string[];
}

export type { ModelSource, SelectedModel };
