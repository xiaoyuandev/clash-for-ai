import type { Provider } from "../types/provider";
import type { RequestLog } from "../types/request-log";
import type { ProviderModel } from "../types/provider-model";

function getApiBase(apiBase?: string) {
  return apiBase ?? "http://127.0.0.1:3456";
}

export interface HealthResponse {
  status: string;
  version: string;
}

export async function getHealth(apiBase?: string): Promise<HealthResponse> {
  const response = await fetch(`${getApiBase(apiBase)}/health`);
  if (!response.ok) {
    throw new Error(`Health request failed with ${response.status}`);
  }
  return response.json() as Promise<HealthResponse>;
}

export async function getProviders(apiBase?: string): Promise<Provider[]> {
  const response = await fetch(`${getApiBase(apiBase)}/api/providers`);
  if (!response.ok) {
    throw new Error(`Provider request failed with ${response.status}`);
  }
  return response.json() as Promise<Provider[]>;
}

export interface CreateProviderInput {
  name: string;
  base_url: string;
  api_key: string;
  auth_mode: "bearer" | "x-api-key" | "both";
  extra_headers: Record<string, string>;
}

export async function createProvider(
  input: CreateProviderInput,
  apiBase?: string
): Promise<Provider> {
  const response = await fetch(`${getApiBase(apiBase)}/api/providers`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(input)
  });

  if (!response.ok) {
    throw new Error(`Create provider failed with ${response.status}`);
  }

  return response.json() as Promise<Provider>;
}

export async function activateProvider(id: string, apiBase?: string): Promise<Provider> {
  const response = await fetch(`${getApiBase(apiBase)}/api/providers/${id}/activate`, {
    method: "POST"
  });

  if (!response.ok) {
    throw new Error(`Activate provider failed with ${response.status}`);
  }

  return response.json() as Promise<Provider>;
}

export async function updateProvider(
  id: string,
  input: CreateProviderInput,
  apiBase?: string
): Promise<Provider> {
  const response = await fetch(`${getApiBase(apiBase)}/api/providers/${id}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(input)
  });

  if (!response.ok) {
    throw new Error(`Update provider failed with ${response.status}`);
  }

  return response.json() as Promise<Provider>;
}

export async function deleteProvider(id: string, apiBase?: string): Promise<void> {
  const response = await fetch(`${getApiBase(apiBase)}/api/providers/${id}`, {
    method: "DELETE"
  });

  if (!response.ok) {
    throw new Error(`Delete provider failed with ${response.status}`);
  }
}

export interface ProviderHealthcheck {
  status: string;
  status_code: number;
  latency_ms: number;
  summary: string;
  checked_at: string;
  provider_id: string;
  provider_url: string;
}

export async function runProviderHealthcheck(
  id: string,
  apiBase?: string
): Promise<ProviderHealthcheck> {
  const response = await fetch(`${getApiBase(apiBase)}/api/providers/${id}/healthcheck`, {
    method: "POST"
  });

  if (!response.ok) {
    throw new Error(`Healthcheck failed with ${response.status}`);
  }

  return response.json() as Promise<ProviderHealthcheck>;
}

export async function getLogs(limit = 100, apiBase?: string): Promise<RequestLog[]> {
  const response = await fetch(`${getApiBase(apiBase)}/api/logs?limit=${limit}`);
  if (!response.ok) {
    throw new Error(`Log request failed with ${response.status}`);
  }
  return response.json() as Promise<RequestLog[]>;
}

export async function getProviderModels(
  id: string,
  apiBase?: string
): Promise<ProviderModel[]> {
  const response = await fetch(`${getApiBase(apiBase)}/api/providers/${id}/models`);
  if (!response.ok) {
    throw new Error(`Models request failed with ${response.status}`);
  }
  return response.json() as Promise<ProviderModel[]>;
}
