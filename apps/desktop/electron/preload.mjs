import { contextBridge, ipcRenderer } from "electron";

contextBridge.exposeInMainWorld("desktopBridge", {
  ping: () => ipcRenderer.invoke("app:ping")
});
