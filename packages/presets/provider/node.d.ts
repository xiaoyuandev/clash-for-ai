export {
  type ProviderPreset,
  type ProviderPresetCatalog,
  normalizeProviderPresetCatalog,
  validateProviderPresetCatalog
} from "./index";

export function readProviderPresetCatalog(path: string): {
  catalog: import("./index").ProviderPresetCatalog;
  content: string;
};
