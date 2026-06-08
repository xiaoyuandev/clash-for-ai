import { resolve } from "node:path";
import { readModelPresetCatalog } from "@relay-switch/model-presets/node";

const sourcePath = resolve(process.cwd(), "config/model-presets.json");

try {
  const { catalog } = readModelPresetCatalog(sourcePath);
  console.log(`[model-presets] valid: ${catalog.presets.length} preset(s) in ${sourcePath}`);
} catch (error) {
  console.error(`[model-presets] invalid: ${error instanceof Error ? error.message : String(error)}`);
  process.exitCode = 1;
}
