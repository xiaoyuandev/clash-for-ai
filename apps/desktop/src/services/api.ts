import type { Provider } from "../types/provider";

const API_BASE = "http://127.0.0.1:3456";

export interface HealthResponse {
  status: string;
  version: string;
}

export async function getHealth(): Promise<HealthResponse> {
  const response = await fetch(`${API_BASE}/health`);
  if (!response.ok) {
    throw new Error(`Health request failed with ${response.status}`);
  }
  return response.json() as Promise<HealthResponse>;
}

export async function getProviders(): Promise<Provider[]> {
  const response = await fetch(`${API_BASE}/api/providers`);
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

export async function createProvider(input: CreateProviderInput): Promise<Provider> {
  const response = await fetch(`${API_BASE}/api/providers`, {
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

export async function activateProvider(id: string): Promise<Provider> {
  const response = await fetch(`${API_BASE}/api/providers/${id}/activate`, {
    method: "POST"
  });

  if (!response.ok) {
    throw new Error(`Activate provider failed with ${response.status}`);
  }

  return response.json() as Promise<Provider>;
}
