import type { Provider } from "../types/provider";
import type { RequestLog } from "../types/request-log";
import type { ProviderModel } from "../types/provider-model";

function getApiBase(apiBase?: string) {
  return apiBase ?? "http://127.0.0.1:3456";
}

async function readErrorMessage(response: Response, fallback: string) {
  const text = (await response.text()).trim();
  if (!text) {
    return fallback;
  }

  try {
    const payload = JSON.parse(text) as { error?: string; message?: string };
    const details = [payload.error, payload.message].filter(Boolean).join(": ");
    return details || `${fallback}: ${text}`;
  } catch {
    return `${fallback}: ${text}`;
  }
}

async function fetchJson<T>(input: string, init: RequestInit, fallback: string): Promise<T> {
  let response: Response;

  try {
    response = await fetch(input, init);
  } catch (error) {
    throw new Error(
      `${fallback} to ${new URL(input).origin}: ${
        error instanceof Error ? error.message : "network error"
      }`
    );
  }

  if (!response.ok) {
    throw new Error(await readErrorMessage(response, `${fallback} with ${response.status}`));
  }

  return response.json() as Promise<T>;
}

async function fetchVoid(input: string, init: RequestInit, fallback: string): Promise<void> {
  let response: Response;

  try {
    response = await fetch(input, init);
  } catch (error) {
    throw new Error(
      `${fallback} to ${new URL(input).origin}: ${
        error instanceof Error ? error.message : "network error"
      }`
    );
  }

  if (!response.ok) {
    throw new Error(await readErrorMessage(response, `${fallback} with ${response.status}`));
  }
}

export interface HealthResponse {
  status: string;
  version: string;
}

export async function getHealth(apiBase?: string): Promise<HealthResponse> {
  return fetchJson<HealthResponse>(
    `${getApiBase(apiBase)}/health`,
    {},
    "Health request failed"
  );
}

export async function getProviders(apiBase?: string): Promise<Provider[]> {
  return fetchJson<Provider[]>(
    `${getApiBase(apiBase)}/api/providers`,
    {},
    "Provider request failed"
  );
}

export interface CreateProviderInput {
  name: string;
  base_url: string;
  api_key: string;
  auth_mode?: "bearer" | "x-api-key" | "both";
  extra_headers: Record<string, string>;
}

export async function createProvider(
  input: CreateProviderInput,
  apiBase?: string
): Promise<Provider> {
  return fetchJson<Provider>(
    `${getApiBase(apiBase)}/api/providers`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify(input)
    },
    "Create provider failed"
  );
}

export async function activateProvider(id: string, apiBase?: string): Promise<Provider> {
  return fetchJson<Provider>(
    `${getApiBase(apiBase)}/api/providers/${id}/activate`,
    {
      method: "POST"
    },
    "Activate provider failed"
  );
}

export async function updateProvider(
  id: string,
  input: CreateProviderInput,
  apiBase?: string
): Promise<Provider> {
  return fetchJson<Provider>(
    `${getApiBase(apiBase)}/api/providers/${id}`,
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify(input)
    },
    "Update provider failed"
  );
}

export async function deleteProvider(id: string, apiBase?: string): Promise<void> {
  return fetchVoid(
    `${getApiBase(apiBase)}/api/providers/${id}`,
    {
      method: "DELETE"
    },
    "Delete provider failed"
  );
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
  return fetchJson<ProviderHealthcheck>(
    `${getApiBase(apiBase)}/api/providers/${id}/healthcheck`,
    {
      method: "POST"
    },
    "Healthcheck failed"
  );
}

export async function getLogs(limit = 100, apiBase?: string): Promise<RequestLog[]> {
  return fetchJson<RequestLog[]>(
    `${getApiBase(apiBase)}/api/logs?limit=${limit}`,
    {},
    "Log request failed"
  );
}

export async function getProviderModels(
  id: string,
  apiBase?: string
): Promise<ProviderModel[]> {
  return fetchJson<ProviderModel[]>(
    `${getApiBase(apiBase)}/api/providers/${id}/models`,
    {},
    "Models request failed"
  );
}
