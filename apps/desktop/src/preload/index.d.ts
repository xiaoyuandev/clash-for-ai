import { ElectronAPI } from "@electron-toolkit/preload";
import type { DesktopDeepLinkEvent } from "./index";

declare global {
  interface Window {
    electron: ElectronAPI;
    desktopBridge: {
      ping: () => Promise<{
        ok: boolean;
        runtime: string;
        platform: string;
        apiBase: string;
        config: {
          apiPort: number;
          apiPortSource: "default" | "config" | "env";
          localGatewayPort: number;
          localGatewayPortSource: "default" | "config" | "env";
          launchAtLogin: boolean;
          launchHidden: boolean;
          closeToTray: boolean;
        };
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
          pid?: number;
          logRetentionDays: number;
          logMaxRecords: number;
          lastError?: string;
          command?: string;
        };
      }>;
      restartCore: () => Promise<{
        ok: boolean;
        config: {
          apiPort: number;
          apiPortSource: "default" | "config" | "env";
          localGatewayPort: number;
          localGatewayPortSource: "default" | "config" | "env";
          launchAtLogin: boolean;
          launchHidden: boolean;
          closeToTray: boolean;
        };
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
          pid?: number;
          logRetentionDays: number;
          logMaxRecords: number;
          lastError?: string;
          command?: string;
        };
      }>;
      updateCorePort: (port: number) => Promise<{
        ok: boolean;
        config: {
          apiPort: number;
          apiPortSource: "default" | "config" | "env";
          localGatewayPort: number;
          localGatewayPortSource: "default" | "config" | "env";
          launchAtLogin: boolean;
          launchHidden: boolean;
          closeToTray: boolean;
        };
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
          pid?: number;
          logRetentionDays: number;
          logMaxRecords: number;
          lastError?: string;
          command?: string;
        };
      }>;
      updateLocalGatewayPort: (port: number) => Promise<{
        ok: boolean;
        config: {
          apiPort: number;
          apiPortSource: "default" | "config" | "env";
          localGatewayPort: number;
          localGatewayPortSource: "default" | "config" | "env";
          launchAtLogin: boolean;
          launchHidden: boolean;
          closeToTray: boolean;
        };
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
          pid?: number;
          logRetentionDays: number;
          logMaxRecords: number;
          lastError?: string;
          command?: string;
        };
      }>;
      copyText: (text: string) => Promise<{ ok: boolean }>;
      openCherryStudioImport: () => Promise<{ ok: boolean; url: string }>;
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
      updateLaunchSettings: (settings: {
        launchAtLogin?: boolean;
        launchHidden?: boolean;
        closeToTray?: boolean;
      }) => Promise<{
        ok: boolean;
        config: {
          apiPort: number;
          apiPortSource: "default" | "config" | "env";
          localGatewayPort: number;
          localGatewayPortSource: "default" | "config" | "env";
          launchAtLogin: boolean;
          launchHidden: boolean;
          closeToTray: boolean;
        };
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
          pid?: number;
          logRetentionDays: number;
          logMaxRecords: number;
          lastError?: string;
          command?: string;
        };
      }>;
      openReleasePage: () => Promise<{ ok: boolean; url: string }>;
      consumeDeepLinkEvent: () => Promise<DesktopDeepLinkEvent | null>;
      onDeepLinkEvent: (listener: (event: DesktopDeepLinkEvent) => void) => () => void;
    };
  }
}
