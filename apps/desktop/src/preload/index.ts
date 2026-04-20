import { contextBridge, ipcRenderer } from "electron";
import { electronAPI } from "@electron-toolkit/preload";

const api = {
  ping: () => ipcRenderer.invoke("app:ping"),
  restartCore: () => ipcRenderer.invoke("app:restart-core"),
  updateCorePort: (port: number) => ipcRenderer.invoke("app:update-core-port", port),
  copyText: (text: string) => ipcRenderer.invoke("app:copy-text", text),
  checkUpdates: () => ipcRenderer.invoke("app:check-updates"),
  downloadUpdate: () => ipcRenderer.invoke("app:download-update"),
  quitAndInstallUpdate: () => ipcRenderer.invoke("app:quit-and-install-update")
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
