import type { Provider } from "../types/provider";
import type { RuntimeConfig, RuntimeHealth } from "../types/runtime";

export const LOCAL_GATEWAY_PROVIDER_ID = "system-local-gateway";

export function buildLocalGatewayProvider(
  runtimeConfig: RuntimeConfig,
  runtimeHealth: RuntimeHealth | null
): Provider | null {
  if (runtimeConfig.mode !== "external-portkey") {
    return null;
  }

  const baseURL = runtimeConfig.base_url.trim();
  if (!baseURL) {
    return null;
  }

  return {
    id: LOCAL_GATEWAY_PROVIDER_ID,
    name: "Clash Local Gateway",
    base_url: baseURL,
    api_key: "",
    auth_mode: "bearer",
    extra_headers: {},
    capabilities: {
      supports_openai_compatible: true,
      supports_anthropic_compatible: true,
      supports_models_api: true,
      supports_balance_api: false,
      supports_stream: true
    },
    status: {
      is_active: true,
      last_health_status: runtimeHealth?.status ?? "pending",
      last_healthcheck_at: runtimeHealth?.checked_at
    },
    api_key_masked: "system-managed",
    claude_code_model_map: {
      opus: "",
      sonnet: "",
      haiku: ""
    },
    is_system: true,
    system_kind: "local-gateway"
  };
}

