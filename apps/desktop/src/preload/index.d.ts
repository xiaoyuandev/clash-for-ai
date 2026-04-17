import { ElectronAPI } from "@electron-toolkit/preload";

declare global {
  interface Window {
    electron: ElectronAPI;
    desktopBridge: {
      ping: () => Promise<{
        ok: boolean;
        runtime: string;
        platform: string;
        apiBase: string;
        core: {
          managed: boolean;
          running: boolean;
          apiBase: string;
          port: number;
          lastError?: string;
          command?: string;
        };
      }>;
    };
  }
}
