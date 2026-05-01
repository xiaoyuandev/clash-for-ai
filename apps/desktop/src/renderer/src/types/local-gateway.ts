import type { SelectedModel } from "./selected-model";

export interface LocalGatewayModelSource {
  id: string;
  name: string;
  base_url: string;
  api_key: string;
  provider_type: "openai-compatible" | "anthropic-compatible";
  default_model_id: string;
  exposed_model_ids: string[];
  enabled: boolean;
  position: number;
  api_key_masked: string;
  last_sync_status: string;
  last_sync_error?: string;
  created_at: string;
  updated_at: string;
}

export interface LocalGatewayRuntimeStatus {
  runtime_kind: string;
  state: string;
  managed: boolean;
  running: boolean;
  healthy: boolean;
  api_base: string;
  host: string;
  port: number;
  pid?: number;
  version?: string;
  commit?: string;
  last_error?: string;
}

export interface LocalGatewayRuntimeResponse {
  runtime: LocalGatewayRuntimeStatus;
  last_sync: {
    applied_sources: number;
    applied_selected_models: number;
    last_synced_at: string;
  };
  last_sync_error?: string;
}

export interface LocalGatewayCapabilities {
  supports_openai_compatible: boolean;
  supports_anthropic_compatible: boolean;
  supports_models_api: boolean;
  supports_stream: boolean;
  supports_admin_api: boolean;
  supports_model_source_admin: boolean;
  supports_selected_model_admin: boolean;
  supports_source_capabilities: boolean;
  supports_atomic_source_sync: boolean;
  supports_runtime_version: boolean;
  supports_explicit_source_health: boolean;
}

export interface CreateLocalGatewayModelSourceInput {
  name: string;
  base_url: string;
  api_key: string;
  provider_type: "openai-compatible" | "anthropic-compatible";
  default_model_id: string;
  exposed_model_ids: string[];
  enabled: boolean;
  position: number;
}

export interface SyncLocalGatewayResponse {
  applied_sources: number;
  applied_selected_models: number;
  last_synced_at: string;
}

export type LocalGatewaySelectedModel = SelectedModel;
