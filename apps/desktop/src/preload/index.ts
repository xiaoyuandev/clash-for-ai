import { contextBridge, ipcRenderer } from "electron";
import { electronAPI } from "@electron-toolkit/preload";

const api = {
  ping: () => ipcRenderer.invoke("app:ping"),
  restartCore: () => ipcRenderer.invoke("app:restart-core"),
  updateCorePort: (port: number) => ipcRenderer.invoke("app:update-core-port", port),
  copyText: (text: string) => ipcRenderer.invoke("app:copy-text", text),
  listTools: () => ipcRenderer.invoke("tools:list"),
  configureTool: (toolId: string) => ipcRenderer.invoke("tools:configure", toolId),
  restoreTool: (toolId: string) => ipcRenderer.invoke("tools:restore", toolId),
  openCherryStudioImport: () => ipcRenderer.invoke("tools:open-cherry-studio-import"),
  checkUpdates: () => ipcRenderer.invoke("app:check-updates"),
  downloadUpdate: () => ipcRenderer.invoke("app:download-update"),
  quitAndInstallUpdate: () => ipcRenderer.invoke("app:quit-and-install-update"),
  openReleasePage: () => ipcRenderer.invoke("app:open-release-page")
};

if (process.contextIsolated) {
  try {
    contextBridge.exposeInMainWorld("electron", electronAPI);
    contextBridge.exposeInMainWorld("desktopBridge", api);
  } catch (error) {
    console.error(error);
  }
} else {
  // @ts-expect-error runtime fallback
  window.electron = electronAPI;
  // @ts-expect-error runtime fallback
  window.desktopBridge = api;
}
