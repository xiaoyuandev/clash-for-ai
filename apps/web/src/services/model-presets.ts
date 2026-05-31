import type { ModelPreset, ModelPresetCatalog, ModelPresetProvider } from "../types/model-preset";

const DEFAULT_MODEL_PRESETS_URL =
  "https://www.relayswitch.dev/model-presets.json";
const DEV_MODEL_PRESETS_URL = "/model-presets.json";
const MODEL_PRESETS_CACHE_KEY = "relay-switch:model-presets";

function modelPresetsURL() {
  return import.meta.env.VITE_MODEL_PRESETS_URL?.trim() || (import.meta.env.DEV ? DEV_MODEL_PRESETS_URL : DEFAULT_MODEL_PRESETS_URL);
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "Unknown error";
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function optionalString(value: unknown, path: string) {
  if (value === undefined) {
    return undefined;
  }
  if (typeof value !== "string") {
    throw new Error(`${path} must be a string`);
  }
  return value.trim() ? value : undefined;
}

function optionalStringList(value: unknown, path: string) {
  if (value === undefined) {
    return undefined;
  }
  if (!Array.isArray(value)) {
    throw new Error(`${path} must be an array`);
  }
  return value.map((item, index) => {
    if (typeof item !== "string" || !item.trim()) {
      throw new Error(`${path}[${index}] must be a non-empty string`);
    }
    return item.trim();
  });
}

function optionalBoolean(value: unknown, path: string) {
  if (value === undefined) {
    return undefined;
  }
  if (typeof value !== "boolean") {
    throw new Error(`${path} must be a boolean`);
  }
  return value;
}

function normalizeProvider(value: unknown, path: string): ModelPresetProvider {
  if (!isObject(value)) {
    throw new Error(`${path} must be an object`);
  }

  const providerType = value.provider_type;
  const baseURL = value.base_url;
  const modelsAPI = value.models_api;
  if (providerType !== "openai-compatible" && providerType !== "anthropic-compatible") {
    throw new Error(`${path}.provider_type must be openai-compatible or anthropic-compatible`);
  }
  if (typeof baseURL !== "string" || !baseURL.trim()) {
    throw new Error(`${path}.base_url must be a non-empty string`);
  }
  if (
    modelsAPI !== undefined &&
    modelsAPI !== "auto" &&
    modelsAPI !== "supported" &&
    modelsAPI !== "unsupported"
  ) {
    throw new Error(`${path}.models_api must be auto, supported, or unsupported`);
  }

  const modelIDs = optionalStringList(value.model_ids, `${path}.model_ids`) ?? [];
  if (modelsAPI === "unsupported" && modelIDs.length === 0) {
    throw new Error(`${path}.model_ids must include at least one model when models_api is unsupported`);
  }

  return {
    id: optionalString(value.id, `${path}.id`),
    label: optionalString(value.label, `${path}.label`),
    provider_type: providerType,
    base_url: baseURL.trim(),
    models_api: modelsAPI,
    model_ids: modelIDs
  };
}

function normalizePreset(value: unknown, path: string): ModelPreset {
  if (!isObject(value)) {
    throw new Error(`${path} must be an object`);
  }
  if (typeof value.id !== "string" || !value.id.trim()) {
    throw new Error(`${path}.id must be a non-empty string`);
  }
  if (typeof value.label !== "string" || !value.label.trim()) {
    throw new Error(`${path}.label must be a non-empty string`);
  }
  if (!Array.isArray(value.providers)) {
    throw new Error(`${path}.providers must be an array`);
  }

  const providers = value.providers.map((provider, index) =>
    normalizeProvider(provider, `${path}.providers[${index}]`)
  );
  if (providers.length === 0) {
    throw new Error(`${path}.providers must include at least one provider`);
  }

  return {
    id: value.id.trim(),
    label: value.label.trim(),
    description: optionalString(value.description, `${path}.description`),
    aliases: optionalStringList(value.aliases, `${path}.aliases`),
    providers,
    tags: optionalStringList(value.tags, `${path}.tags`),
    deprecated: optionalBoolean(value.deprecated, `${path}.deprecated`),
    disabled: optionalBoolean(value.disabled, `${path}.disabled`)
  };
}

function normalizeCatalog(value: unknown, sourceURL: string): ModelPresetCatalog {
  if (!isObject(value)) {
    throw new Error("Model presets JSON must be an object");
  }
  if (value.schema_version !== 1) {
    throw new Error("Model presets schema_version must be 1");
  }
  if (!Array.isArray(value.presets)) {
    throw new Error("Model presets JSON must include presets");
  }

  const seenPresetIDs = new Set<string>();
  const presets: ModelPreset[] = [];
  for (const [index, item] of value.presets.entries()) {
    const preset = normalizePreset(item, `presets[${index}]`);
    if (seenPresetIDs.has(preset.id)) {
      throw new Error(`presets[${index}].id duplicates ${preset.id}`);
    }
    seenPresetIDs.add(preset.id);
    presets.push(preset);
  }

  return {
    schema_version: 1,
    updated_at: optionalString(value.updated_at, "updated_at") ?? "",
    presets,
    source_url: sourceURL,
    cached_at: optionalString(value.cached_at, "cached_at"),
    last_refresh_error: optionalString(value.last_refresh_error, "last_refresh_error")
  };
}

function readCachedCatalog(sourceURL: string): ModelPresetCatalog | null {
  try {
    const raw = window.localStorage.getItem(MODEL_PRESETS_CACHE_KEY);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as unknown;
    if (isObject(parsed) && typeof parsed.source_url === "string" && parsed.source_url !== sourceURL) {
      return null;
    }
    return normalizeCatalog(parsed, sourceURL);
  } catch {
    return null;
  }
}

function writeCachedCatalog(catalog: ModelPresetCatalog) {
  try {
    window.localStorage.setItem(
      MODEL_PRESETS_CACHE_KEY,
      JSON.stringify({
        ...catalog,
        cached_at: new Date().toISOString()
      })
    );
  } catch {
    // Presets are an optional UI helper; storage failures should not block manual input.
  }
}

export async function getModelPresets(): Promise<ModelPresetCatalog> {
  const sourceURL = modelPresetsURL();
  try {
    const response = await fetch(sourceURL, {
      cache: "no-cache",
      headers: {
        Accept: "application/json"
      }
    });
    if (!response.ok) {
      throw new Error(`Model presets request failed with ${response.status}`);
    }

    const catalog = normalizeCatalog(await response.json(), sourceURL);
    writeCachedCatalog(catalog);
    return catalog;
  } catch (error) {
    const cached = readCachedCatalog(sourceURL);
    if (cached) {
      return {
        ...cached,
        last_refresh_error: errorMessage(error)
      };
    }

    return {
      schema_version: 1,
      updated_at: "",
      presets: [],
      source_url: sourceURL,
      last_refresh_error: errorMessage(error)
    };
  }
}
