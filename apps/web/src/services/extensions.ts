import type {
  ExtensionAuditLog,
  ExtensionCommand,
  ExtensionCommandResult,
  ExtensionDeclaredProcess,
  ExtensionPlugin,
  ExtensionSettings,
  ExtensionToolIntegration,
  ExtensionToolIntegrationAction,
  ExtensionToolIntegrationResult
} from "../types/extension";

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

export async function getExtensionCommands(apiBase?: string): Promise<ExtensionCommand[]> {
  return fetchJson<ExtensionCommand[]>(
    `${getApiBase(apiBase)}/api/extensions/commands`,
    {},
    "Extension commands request failed"
  );
}

export async function executeExtensionCommand(
  id: string,
  apiBase?: string
): Promise<ExtensionCommandResult> {
  return fetchJson<ExtensionCommandResult>(
    `${getApiBase(apiBase)}/api/extensions/commands/${encodeURIComponent(id)}/execute`,
    { method: "POST" },
    "Extension command execution failed"
  );
}

export async function getExtensionToolIntegrations(
  apiBase?: string
): Promise<ExtensionToolIntegration[]> {
  return fetchJson<ExtensionToolIntegration[]>(
    `${getApiBase(apiBase)}/api/extensions/tool-integrations`,
    {},
    "Extension tool integrations request failed"
  );
}

export async function executeExtensionToolIntegrationAction(
  id: string,
  action: ExtensionToolIntegrationAction,
  apiBase?: string
): Promise<ExtensionToolIntegrationResult> {
  return fetchJson<ExtensionToolIntegrationResult>(
    `${getApiBase(apiBase)}/api/extensions/tool-integrations/${encodeURIComponent(id)}/${action}`,
    { method: "POST" },
    "Extension tool integration action failed"
  );
}

export async function getExtensionDeclaredProcesses(
  apiBase?: string
): Promise<ExtensionDeclaredProcess[]> {
  return fetchJson<ExtensionDeclaredProcess[]>(
    `${getApiBase(apiBase)}/api/extensions/declared-processes`,
    {},
    "Extension declared processes request failed"
  );
}

export async function getExtensionSettings(id: string, apiBase?: string): Promise<ExtensionSettings> {
  return fetchJson<ExtensionSettings>(
    `${getApiBase(apiBase)}/api/extensions/${encodeURIComponent(id)}/settings`,
    {},
    "Extension settings request failed"
  );
}

export async function updateExtensionSettings(
  id: string,
  values: Record<string, unknown>,
  apiBase?: string
): Promise<ExtensionSettings> {
  return fetchJson<ExtensionSettings>(
    `${getApiBase(apiBase)}/api/extensions/${encodeURIComponent(id)}/settings`,
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({ values })
    },
    "Extension settings update failed"
  );
}

export async function getExtensionAuditLogs(
  id: string,
  limit = 20,
  apiBase?: string
): Promise<ExtensionAuditLog[]> {
  return fetchJson<ExtensionAuditLog[]>(
    `${getApiBase(apiBase)}/api/extensions/${encodeURIComponent(id)}/audit-logs?limit=${limit}`,
    {},
    "Extension audit logs request failed"
  );
}

export async function getAllExtensionAuditLogs(
  limit = 100,
  apiBase?: string
): Promise<ExtensionAuditLog[]> {
  return fetchJson<ExtensionAuditLog[]>(
    `${getApiBase(apiBase)}/api/extensions/audit-logs?limit=${limit}`,
    {},
    "Extension audit logs request failed"
  );
}
