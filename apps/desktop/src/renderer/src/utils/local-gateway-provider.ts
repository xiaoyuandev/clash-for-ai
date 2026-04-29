import type { Provider } from "../types/provider";

export const LOCAL_GATEWAY_PROVIDER_ID = "system-local-gateway";

export function buildLocalGatewayProvider(
  apiBase?: string,
  options?: {
    isActive?: boolean;
    claudeCodeModelMap?: Provider["claude_code_model_map"];
  }
): Provider | null {
  if (!apiBase) {
    return null;
  }

  return {
    id: LOCAL_GATEWAY_PROVIDER_ID,
    name: "Clash Local Gateway",
    base_url: apiBase,
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
      is_active: options?.isActive ?? false,
      last_health_status: "ok"
    },
    api_key_masked: "system-managed",
    claude_code_model_map: options?.claudeCodeModelMap ?? {
      opus: "",
      sonnet: "",
      haiku: ""
    },
    is_system: true
  };
}

export function decorateProvidersWithLocalGateway(items: Provider[], apiBase?: string): Provider[] {
  const hasActiveRemoteProvider = items.some((provider) => provider.status.is_active);
  const localProvider = buildLocalGatewayProvider(apiBase, {
    isActive: !hasActiveRemoteProvider
  });
  if (!localProvider) {
    return items;
  }

  return [localProvider, ...items];
}
