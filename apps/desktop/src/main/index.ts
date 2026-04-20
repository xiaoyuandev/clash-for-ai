import { app, BrowserWindow, ipcMain, shell } from "electron";
import { join } from "path";
import { electronApp, is, optimizer } from "@electron-toolkit/utils";
import { autoUpdater } from "electron-updater";
import { startCoreProcess, type CoreRuntimeHandle } from "./core-process";

interface UpdateState {
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
}

let coreRuntime: CoreRuntimeHandle = {
  state: {
    managed: false,
    running: false,
    apiBase: process.env.ELECTRON_API_BASE ?? "http://127.0.0.1:3456",
    port: Number(process.env.ELECTRON_API_PORT || 3456),
    logRetentionDays: Number(process.env.LOG_RETENTION_DAYS || 30),
    logMaxRecords: Number(process.env.LOG_MAX_RECORDS || 10000)
  },
  stop() {}
};
let isBootstrapped = false;
let updateState: UpdateState = {
  currentVersion: app.getVersion(),
  status: app.isPackaged ? "idle" : "unsupported",
  message: app.isPackaged
    ? undefined
    : "Update checks are only available in packaged builds."
};

async function bootstrapCoreRuntime() {
  try {
    coreRuntime = await startCoreProcess();
  } catch (error) {
    coreRuntime = {
      state: {
        managed: false,
        running: false,
        apiBase: process.env.ELECTRON_API_BASE ?? "http://127.0.0.1:3456",
        port: Number(process.env.ELECTRON_API_PORT || 3456),
        logRetentionDays: Number(process.env.LOG_RETENTION_DAYS || 30),
        logMaxRecords: Number(process.env.LOG_MAX_RECORDS || 10000),
        lastError: error instanceof Error ? error.message : "failed to start core"
      },
      stop() {}
    };
  }
}

function createWindow(): void {
  const mainWindow = new BrowserWindow({
    width: 1280,
    height: 860,
    minWidth: 960,
    minHeight: 680,
    show: false,
    autoHideMenuBar: true,
    backgroundColor: "#171310",
    webPreferences: {
      preload: join(__dirname, "../preload/index.js"),
      sandbox: false
    }
  });

  mainWindow.on("ready-to-show", () => {
    mainWindow.show();
  });

  if (is.dev) {
    mainWindow.webContents.once("did-frame-finish-load", () => {
      mainWindow.webContents.openDevTools({ mode: "detach", activate: true });
    });
  }

  mainWindow.webContents.setWindowOpenHandler((details) => {
    void shell.openExternal(details.url);
    return { action: "deny" };
  });

  if (is.dev && process.env.ELECTRON_RENDERER_URL) {
    void mainWindow.loadURL(process.env.ELECTRON_RENDERER_URL);
  } else {
    void mainWindow.loadFile(join(__dirname, "../renderer/index.html"));
  }
}

function configureAutoUpdater() {
  if (!app.isPackaged) {
    return;
  }

  autoUpdater.autoDownload = false;
  autoUpdater.autoInstallOnAppQuit = true;

  autoUpdater.on("checking-for-update", () => {
    updateState = {
      currentVersion: app.getVersion(),
      status: "checking"
    };
  });

  autoUpdater.on("update-available", (info) => {
    updateState = {
      currentVersion: app.getVersion(),
      status: "available",
      availableVersion: info.version,
      message: info.releaseName ?? "Update available"
    };
  });

  autoUpdater.on("update-not-available", () => {
    updateState = {
      currentVersion: app.getVersion(),
      status: "not-available",
      message: "You are on the latest version."
    };
  });

  autoUpdater.on("download-progress", (progress) => {
    updateState = {
      currentVersion: app.getVersion(),
      status: "downloading",
      availableVersion: updateState.availableVersion,
      progressPercent: progress.percent,
      message: `Downloading update: ${Math.round(progress.percent)}%`
    };
  });

  autoUpdater.on("update-downloaded", (info) => {
    updateState = {
      currentVersion: app.getVersion(),
      status: "downloaded",
      downloadedVersion: info.version,
      message: "Update downloaded. Restart to install."
    };
  });

  autoUpdater.on("error", (error) => {
    updateState = {
      currentVersion: app.getVersion(),
      status: "error",
      message: error == null ? "Unknown update error" : error.message
    };
  });
}

app.whenReady().then(() => {
  electronApp.setAppUserModelId("com.xiaoyuandev.clash-for-ai");
  configureAutoUpdater();

  app.on("browser-window-created", (_, window) => {
    optimizer.watchWindowShortcuts(window);
  });

  ipcMain.handle("app:ping", async () => ({
    ok: true,
    runtime: "electron",
    platform: process.platform,
    apiBase: coreRuntime.state.apiBase,
    updates: updateState,
    core: coreRuntime.state
  }));

  ipcMain.handle("app:restart-core", async () => {
    coreRuntime.stop();
    await bootstrapCoreRuntime();
    return {
      ok: true,
      updates: updateState,
      core: coreRuntime.state
    };
  });

  ipcMain.handle("app:check-updates", async () => {
    if (!app.isPackaged) {
      return updateState;
    }

    await autoUpdater.checkForUpdates();
    return updateState;
  });

  ipcMain.handle("app:download-update", async () => {
    if (!app.isPackaged) {
      return updateState;
    }

    await autoUpdater.downloadUpdate();
    return updateState;
  });

  ipcMain.handle("app:quit-and-install-update", async () => {
    if (!app.isPackaged) {
      return updateState;
    }

    autoUpdater.quitAndInstall();
    return updateState;
  });

  void bootstrapCoreRuntime()
    .finally(() => {
      isBootstrapped = true;
      createWindow();
    });

  app.on("activate", function () {
    if (isBootstrapped && BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") {
    app.quit();
  }
});

app.on("before-quit", () => {
  coreRuntime?.stop();
});
