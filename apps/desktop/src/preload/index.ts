import { contextBridge, ipcRenderer } from "electron";
import { electronAPI } from "@electron-toolkit/preload";

const api = {
  ping: () => ipcRenderer.invoke("app:ping")
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
