import { contextBridge, ipcRenderer } from "electron";
import { electronAPI } from "@electron-toolkit/preload";

export interface DesktopDeepLinkImportEvent {
  id: string;
  kind: "import";
  request: {
    resource: "provider" | "model";
    payload: Record<string, unknown>;
    originalURL: string;
  };
}

export interface DesktopDeepLinkErrorEvent {
  id: string;
  kind: "error";
  message: string;
  originalURL?: string;
}

export type DesktopDeepLinkEvent = DesktopDeepLinkImportEvent | DesktopDeepLinkErrorEvent;

const api = {
  ping: () => ipcRenderer.invoke("app:ping"),
  restartCore: () => ipcRenderer.invoke("app:restart-core"),
  updateCorePort: (port: number) => ipcRenderer.invoke("app:update-core-port", port),
  updateLocalGatewayPort: (port: number) =>
    ipcRenderer.invoke("app:update-local-gateway-port", port),
  updateLaunchSettings: (settings: {
    launchAtLogin?: boolean;
    launchHidden?: boolean;
    closeToTray?: boolean;
  }) =>
    ipcRenderer.invoke("app:update-launch-settings", settings),
  copyText: (text: string) => ipcRenderer.invoke("app:copy-text", text),
  openCherryStudioImport: () => ipcRenderer.invoke("tools:open-cherry-studio-import"),
  checkUpdates: () => ipcRenderer.invoke("app:check-updates"),
  downloadUpdate: () => ipcRenderer.invoke("app:download-update"),
  quitAndInstallUpdate: () => ipcRenderer.invoke("app:quit-and-install-update"),
  openReleasePage: () => ipcRenderer.invoke("app:open-release-page"),
  openProjectPage: () => ipcRenderer.invoke("app:open-project-page"),
  consumeDeepLinkEvent: () => ipcRenderer.invoke("app:consume-deep-link-event"),
  onDeepLinkEvent: (listener: (event: DesktopDeepLinkEvent) => void) => {
    const wrapped = (_event: unknown, payload: DesktopDeepLinkEvent) => listener(payload);
    ipcRenderer.on("app:deep-link-event", wrapped);
    return () => {
      ipcRenderer.removeListener("app:deep-link-event", wrapped);
    };
  }
};

if (process.contextIsolated) {
  try {
    contextBridge.exposeInMainWorld("electron", electronAPI);
    contextBridge.exposeInMainWorld("desktopBridge", api);
  } catch (error) {
    console.error(error);
  }
} else {
  const nextWindow = window as typeof window & {
    electron: typeof electronAPI;
    desktopBridge: typeof api;
  };
  nextWindow.electron = electronAPI;
  nextWindow.desktopBridge = api;
}
