import { normalizeModelPresetCatalog } from "@relay-switch/presets/model";
import type { ModelPresetCatalog } from "@relay-switch/presets/model";

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
    return normalizeModelPresetCatalog(parsed, sourceURL);
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

    const catalog = normalizeModelPresetCatalog(await response.json(), sourceURL);
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
