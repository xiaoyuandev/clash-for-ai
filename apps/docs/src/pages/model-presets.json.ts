import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import type { APIRoute } from "astro";

export const prerender = true;

const sourcePath = resolve(dirname(fileURLToPath(import.meta.url)), "../../../../config/model-presets.json");

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function validateOptionalString(value: unknown, path: string) {
  if (value === undefined) {
    return;
  }
  if (typeof value !== "string") {
    throw new Error(`${path} must be a string`);
  }
}

function validateOptionalBoolean(value: unknown, path: string) {
  if (value === undefined) {
    return;
  }
  if (typeof value !== "boolean") {
    throw new Error(`${path} must be a boolean`);
  }
}

function validateOptionalStringArray(value: unknown, path: string) {
  if (value === undefined) {
    return;
  }
  if (!Array.isArray(value)) {
    throw new Error(`${path} must be an array`);
  }
  value.forEach((item, index) => {
    if (typeof item !== "string" || !item.trim()) {
      throw new Error(`${path}[${index}] must be a non-empty string`);
    }
  });
}

function validateProvider(value: unknown, path: string) {
  if (!isObject(value)) {
    throw new Error(`${path} must be an object`);
  }

  if (value.provider_type !== "openai-compatible" && value.provider_type !== "anthropic-compatible") {
    throw new Error(`${path}.provider_type must be openai-compatible or anthropic-compatible`);
  }
  if (typeof value.base_url !== "string" || !value.base_url.trim()) {
    throw new Error(`${path}.base_url must be a non-empty string`);
  }
  if (
    value.models_api !== undefined &&
    value.models_api !== "auto" &&
    value.models_api !== "supported" &&
    value.models_api !== "unsupported"
  ) {
    throw new Error(`${path}.models_api must be auto, supported, or unsupported`);
  }

  validateOptionalString(value.id, `${path}.id`);
  validateOptionalString(value.label, `${path}.label`);
  validateOptionalStringArray(value.model_ids, `${path}.model_ids`);

  const modelIDs = Array.isArray(value.model_ids) ? value.model_ids : [];
  if (value.models_api === "unsupported" && modelIDs.length === 0) {
    throw new Error(`${path}.model_ids must include at least one model when models_api is unsupported`);
  }
}

function validatePreset(value: unknown, path: string) {
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
  if (value.providers.length === 0) {
    throw new Error(`${path}.providers must include at least one provider`);
  }

  validateOptionalString(value.description, `${path}.description`);
  validateOptionalStringArray(value.aliases, `${path}.aliases`);
  validateOptionalStringArray(value.tags, `${path}.tags`);
  validateOptionalBoolean(value.deprecated, `${path}.deprecated`);
  validateOptionalBoolean(value.disabled, `${path}.disabled`);

  value.providers.forEach((provider, index) => validateProvider(provider, `${path}.providers[${index}]`));
}

function validateCatalog(value: unknown) {
  if (!isObject(value)) {
    throw new Error("Model presets JSON must be an object");
  }
  if (value.schema_version !== 1) {
    throw new Error("Model presets schema_version must be 1");
  }
  validateOptionalString(value.updated_at, "updated_at");
  if (!Array.isArray(value.presets)) {
    throw new Error("Model presets JSON must include presets");
  }

  const ids = new Set<string>();
  value.presets.forEach((preset, index) => {
    validatePreset(preset, `presets[${index}]`);
    const id = (preset as { id: string }).id.trim();
    if (ids.has(id)) {
      throw new Error(`presets[${index}].id duplicates ${id}`);
    }
    ids.add(id);
  });
}

export const GET: APIRoute = () => {
  const content = readFileSync(sourcePath, "utf8");
  const catalog = JSON.parse(content) as unknown;
  validateCatalog(catalog);

  return new Response(`${JSON.stringify(catalog, null, 2)}\n`, {
    headers: {
      "Cache-Control": "public, max-age=300",
      "Content-Type": "application/json; charset=utf-8"
    }
  });
};
