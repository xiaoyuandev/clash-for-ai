import { readFileSync } from "node:fs";
import { normalizeProviderPresetCatalog, validateProviderPresetCatalog } from "./core.mjs";

export { normalizeProviderPresetCatalog, validateProviderPresetCatalog };

export function readProviderPresetCatalog(path) {
  const content = readFileSync(path, "utf8");
  let catalog;
  try {
    catalog = JSON.parse(content);
  } catch (error) {
    throw new Error(`parse ${path}: ${error instanceof Error ? error.message : "invalid JSON"}`);
  }
  return { catalog: normalizeProviderPresetCatalog(catalog), content };
}
