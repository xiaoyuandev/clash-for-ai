function isObject(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function optionalString(value, path) {
  if (value === undefined) {
    return undefined;
  }
  if (typeof value !== "string") {
    throw new Error(`${path} must be a string`);
  }
  return value.trim() ? value.trim() : undefined;
}

function optionalBoolean(value, path) {
  if (value === undefined) {
    return undefined;
  }
  if (typeof value !== "boolean") {
    throw new Error(`${path} must be a boolean`);
  }
  return value;
}

function optionalStringList(value, path) {
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

function normalizeProvider(value, path) {
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

function normalizePreset(value, path) {
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

export function normalizeModelPresetCatalog(value, sourceURL) {
  if (!isObject(value)) {
    throw new Error("Model presets JSON must be an object");
  }
  if (value.schema_version !== 1) {
    throw new Error("Model presets schema_version must be 1");
  }
  if (!Array.isArray(value.presets)) {
    throw new Error("Model presets JSON must include presets");
  }

  const seenPresetIDs = new Set();
  const presets = [];
  for (const [index, item] of value.presets.entries()) {
    const preset = normalizePreset(item, `presets[${index}]`);
    if (seenPresetIDs.has(preset.id)) {
      throw new Error(`presets[${index}].id duplicates ${preset.id}`);
    }
    seenPresetIDs.add(preset.id);
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

export function validateModelPresetCatalog(value) {
  normalizeModelPresetCatalog(value);
}
