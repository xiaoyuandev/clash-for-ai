import { readFileSync } from "node:fs";
import { normalizeModelPresetCatalog, validateModelPresetCatalog } from "./core.mjs";

export { normalizeModelPresetCatalog, validateModelPresetCatalog };

export function readModelPresetCatalog(path) {
  const content = readFileSync(path, "utf8");
  let catalog;
  try {
    catalog = JSON.parse(content);
  } catch (error) {
    throw new Error(`parse ${path}: ${error instanceof Error ? error.message : "invalid JSON"}`);
  }
  return { catalog: normalizeModelPresetCatalog(catalog), content };
}
