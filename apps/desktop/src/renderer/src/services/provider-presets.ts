import { normalizeProviderPresetCatalog } from "@relay-switch/presets/provider";
import type { ProviderPresetCatalog } from "@relay-switch/presets/provider";

const DEFAULT_PROVIDER_PRESETS_URL =
  "https://www.relayswitch.dev/provider-presets.json";
const DEV_PROVIDER_PRESETS_URL = "/provider-presets.json";
const PROVIDER_PRESETS_CACHE_KEY = "relay-switch:provider-presets";

function providerPresetsURL() {
  return import.meta.env.VITE_PROVIDER_PRESETS_URL?.trim() || (import.meta.env.DEV ? DEV_PROVIDER_PRESETS_URL : DEFAULT_PROVIDER_PRESETS_URL);
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "Unknown error";
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function readCachedCatalog(sourceURL: string): ProviderPresetCatalog | null {
  try {
    const raw = window.localStorage.getItem(PROVIDER_PRESETS_CACHE_KEY);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as unknown;
    if (isObject(parsed) && typeof parsed.source_url === "string" && parsed.source_url !== sourceURL) {
      return null;
    }
    return normalizeProviderPresetCatalog(parsed, sourceURL);
  } catch {
    return null;
  }
}

function writeCachedCatalog(catalog: ProviderPresetCatalog) {
  try {
    window.localStorage.setItem(
      PROVIDER_PRESETS_CACHE_KEY,
      JSON.stringify({
        ...catalog,
        cached_at: new Date().toISOString()
      })
    );
  } catch {
    // Presets are an optional UI helper; storage failures should not block manual input.
  }
}

export async function getProviderPresets(): Promise<ProviderPresetCatalog> {
  const sourceURL = providerPresetsURL();
  try {
    const response = await fetch(sourceURL, {
      cache: "no-cache",
      headers: {
        Accept: "application/json"
      }
    });
    if (!response.ok) {
      throw new Error(`Provider presets request failed with ${response.status}`);
    }

    const catalog = normalizeProviderPresetCatalog(await response.json(), sourceURL);
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
