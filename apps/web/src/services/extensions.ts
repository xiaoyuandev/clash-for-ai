import type { ExtensionPlugin } from "../types/extension";

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

export async function getExtensions(apiBase?: string): Promise<ExtensionPlugin[]> {
  return fetchJson<ExtensionPlugin[]>(
    `${getApiBase(apiBase)}/api/extensions`,
    {},
    "Extension request failed"
  );
}

export async function getExtension(id: string, apiBase?: string): Promise<ExtensionPlugin> {
  return fetchJson<ExtensionPlugin>(
    `${getApiBase(apiBase)}/api/extensions/${encodeURIComponent(id)}`,
    {},
    "Extension detail request failed"
  );
}

export async function rescanExtensions(apiBase?: string): Promise<ExtensionPlugin[]> {
  return fetchJson<ExtensionPlugin[]>(
    `${getApiBase(apiBase)}/api/extensions/rescan`,
    { method: "POST" },
    "Extension rescan failed"
  );
}

export async function enableExtension(id: string, apiBase?: string): Promise<ExtensionPlugin> {
  return fetchJson<ExtensionPlugin>(
    `${getApiBase(apiBase)}/api/extensions/${encodeURIComponent(id)}/enable`,
    { method: "POST" },
    "Enable extension failed"
  );
}

export async function disableExtension(id: string, apiBase?: string): Promise<ExtensionPlugin> {
  return fetchJson<ExtensionPlugin>(
    `${getApiBase(apiBase)}/api/extensions/${encodeURIComponent(id)}/disable`,
    { method: "POST" },
    "Disable extension failed"
  );
}
