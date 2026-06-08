export {
  type ModelPreset,
  type ModelPresetCatalog,
  type ModelPresetProvider,
  normalizeModelPresetCatalog,
  validateModelPresetCatalog
} from "./index";

export function readModelPresetCatalog(path: string): {
  catalog: import("./index").ModelPresetCatalog;
  content: string;
};
