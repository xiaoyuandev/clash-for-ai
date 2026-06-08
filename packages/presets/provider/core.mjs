import {
  isObject,
  optionalString,
  requiredString,
  validateAbsoluteURL
} from "../shared.mjs";

function assertKnownKeys(value, allowedKeys, path) {
  for (const key of Object.keys(value)) {
    if (!allowedKeys.has(key)) {
      throw new Error(`${path}.${key} is not supported`);
    }
  }
}

const catalogKeys = new Set([
  "schema_version",
  "updated_at",
  "presets",
  "source_url",
  "cached_at",
  "last_refresh_error"
]);
const presetKeys = new Set(["name", "base_url"]);

function normalizePreset(value, path) {
  if (!isObject(value)) {
    throw new Error(`${path} must be an object`);
  }
  assertKnownKeys(value, presetKeys, path);

  const name = requiredString(value.name, `${path}.name`);
  const baseURL = requiredString(value.base_url, `${path}.base_url`);
  validateAbsoluteURL(baseURL, `${path}.base_url`);

  return {
    name,
    base_url: baseURL
  };
}

export function normalizeProviderPresetCatalog(value, sourceURL) {
  if (!isObject(value)) {
    throw new Error("Provider presets JSON must be an object");
  }
  assertKnownKeys(value, catalogKeys, "Provider presets JSON");
  if (value.schema_version !== 1) {
    throw new Error("Provider presets schema_version must be 1");
  }
  if (!Array.isArray(value.presets)) {
    throw new Error("Provider presets JSON must include presets");
  }

  const seenNames = new Set();
  const presets = [];
  for (const [index, item] of value.presets.entries()) {
    const preset = normalizePreset(item, `presets[${index}]`);
    const key = preset.name.toLowerCase();
    if (seenNames.has(key)) {
      throw new Error(`presets[${index}].name duplicates ${preset.name}`);
    }
    seenNames.add(key);
    presets.push(preset);
  }

  const catalog = {
    schema_version: 1,
    updated_at: optionalString(value.updated_at, "updated_at") ?? "",
    presets,
    cached_at: optionalString(value.cached_at, "cached_at"),
    last_refresh_error: optionalString(value.last_refresh_error, "last_refresh_error")
  };

  if (sourceURL !== undefined) {
    catalog.source_url = sourceURL;
  }

  return catalog;
}

export function validateProviderPresetCatalog(value) {
  normalizeProviderPresetCatalog(value);
}
