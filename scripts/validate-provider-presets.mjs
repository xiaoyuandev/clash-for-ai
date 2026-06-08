import { resolve } from "node:path";
import { readProviderPresetCatalog } from "@relay-switch/presets/provider/node";

const sourcePath = resolve(process.cwd(), "config/provider-presets.json");

try {
  const { catalog } = readProviderPresetCatalog(sourcePath);
  console.log(`[provider-presets] valid: ${catalog.presets.length} preset(s) in ${sourcePath}`);
} catch (error) {
  console.error(`[provider-presets] invalid: ${error instanceof Error ? error.message : String(error)}`);
  process.exitCode = 1;
}
