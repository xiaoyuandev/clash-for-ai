import { ElectronAPI } from "@electron-toolkit/preload";

declare global {
  interface Window {
    electron: ElectronAPI;
    desktopBridge: {
      ping: () => Promise<{
        ok: boolean;
        runtime: string;
        platform: string;
      }>;
    };
  }
}
