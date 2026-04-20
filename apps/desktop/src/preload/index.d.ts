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
        updates: {
          currentVersion: string;
          status:
            | "idle"
            | "checking"
            | "available"
            | "not-available"
            | "downloading"
            | "downloaded"
            | "error"
            | "unsupported";
          availableVersion?: string;
          downloadedVersion?: string;
          progressPercent?: number;
          message?: string;
        };
        core: {
          managed: boolean;
          running: boolean;
          apiBase: string;
          port: number;
          logRetentionDays: number;
          logMaxRecords: number;
          lastError?: string;
          command?: string;
        };
      }>;
      restartCore: () => Promise<{
        ok: boolean;
        updates: {
          currentVersion: string;
          status:
            | "idle"
            | "checking"
            | "available"
            | "not-available"
            | "downloading"
            | "downloaded"
            | "error"
            | "unsupported";
          availableVersion?: string;
          downloadedVersion?: string;
          progressPercent?: number;
          message?: string;
        };
        core: {
          managed: boolean;
          running: boolean;
          apiBase: string;
          port: number;
          logRetentionDays: number;
          logMaxRecords: number;
          lastError?: string;
          command?: string;
        };
      }>;
      checkUpdates: () => Promise<{
        currentVersion: string;
        status:
          | "idle"
          | "checking"
          | "available"
          | "not-available"
          | "downloading"
          | "downloaded"
          | "error"
          | "unsupported";
        availableVersion?: string;
        downloadedVersion?: string;
        progressPercent?: number;
        message?: string;
      }>;
      downloadUpdate: () => Promise<{
        currentVersion: string;
        status:
          | "idle"
          | "checking"
          | "available"
          | "not-available"
          | "downloading"
          | "downloaded"
          | "error"
          | "unsupported";
        availableVersion?: string;
        downloadedVersion?: string;
        progressPercent?: number;
        message?: string;
      }>;
      quitAndInstallUpdate: () => Promise<{
        currentVersion: string;
        status:
          | "idle"
          | "checking"
          | "available"
          | "not-available"
          | "downloading"
          | "downloaded"
          | "error"
          | "unsupported";
        availableVersion?: string;
        downloadedVersion?: string;
        progressPercent?: number;
        message?: string;
      }>;
    };
  }
}
