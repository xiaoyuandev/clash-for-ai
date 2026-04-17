import { app, BrowserWindow, ipcMain, shell } from "electron";
import { join } from "path";
import { electronApp, is, optimizer } from "@electron-toolkit/utils";
import { startCoreProcess, type CoreRuntimeHandle } from "./core-process";

let coreRuntime: CoreRuntimeHandle = {
  state: {
    managed: false,
    running: false,
    apiBase: process.env.ELECTRON_API_BASE ?? "http://127.0.0.1:3456",
    port: Number(process.env.ELECTRON_API_PORT || 3456)
  },
  stop() {}
};
let isBootstrapped = false;

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

app.whenReady().then(() => {
  electronApp.setAppUserModelId("com.xiaoyuandev.clash-for-ai");

  app.on("browser-window-created", (_, window) => {
    optimizer.watchWindowShortcuts(window);
  });

  ipcMain.handle("app:ping", async () => ({
    ok: true,
    runtime: "electron",
    platform: process.platform,
    apiBase: coreRuntime.state.apiBase,
    core: coreRuntime.state
  }));

  void startCoreProcess()
    .then((runtime) => {
      coreRuntime = runtime;
    })
    .catch((error) => {
      coreRuntime = {
        state: {
          managed: false,
          running: false,
          apiBase: process.env.ELECTRON_API_BASE ?? "http://127.0.0.1:3456",
          port: 3456,
          lastError: error instanceof Error ? error.message : "failed to start core"
        },
        stop() {}
      };
    })
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
