import { app } from "electron";
import { spawn, spawnSync, type ChildProcess } from "node:child_process";
import { existsSync, mkdirSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { createServer } from "node:net";
import {
  clearCoreProcessRecord,
  loadCoreProcessRecord,
  saveCoreProcessRecord
} from "./app-config";

export interface CoreRuntimeState {
  managed: boolean;
  running: boolean;
  apiBase: string;
  port: number;
  pid?: number;
  logRetentionDays: number;
  logMaxRecords: number;
  lastError?: string;
  command?: string;
}

export interface CoreRuntimeHandle {
  state: CoreRuntimeState;
  stop: () => void;
}

interface StartCoreProcessOptions {
  desiredPort: number;
  desiredLocalGatewayPort: number;
}

export async function startCoreProcess(
  options: StartCoreProcessOptions
): Promise<CoreRuntimeHandle> {
  const explicitApiBase = process.env.ELECTRON_API_BASE;
  if (explicitApiBase) {
    console.info(`[core] using external api base ${explicitApiBase}`);
    return {
      state: {
        managed: false,
        running: true,
        apiBase: explicitApiBase,
        port: parsePort(explicitApiBase),
        logRetentionDays: Number(process.env.LOG_RETENTION_DAYS || 30),
        logMaxRecords: Number(process.env.LOG_MAX_RECORDS || 10000)
      },
      stop() {}
    };
  }

  const runtimePaths = resolveCoreRuntimePaths();
  const port = options.desiredPort;
  const localGatewayPort = options.desiredLocalGatewayPort;
  const apiBase = `http://127.0.0.1:${port}`;
  console.info(`[core] using fixed port ${port}, api base ${apiBase}`);

  if (!(await isPortAvailable(port))) {
    const existingRecord = loadCoreProcessRecord();

    if (await isHealthyCore(apiBase)) {
      console.info(`[core] reusing existing core at ${apiBase}`);
      return {
        state: {
          managed: false,
          running: true,
          apiBase,
          port,
          pid: existingRecord?.port === port ? existingRecord.pid : undefined,
          logRetentionDays: Number(process.env.LOG_RETENTION_DAYS || 30),
          logMaxRecords: Number(process.env.LOG_MAX_RECORDS || 10000),
          command:
            existingRecord?.port === port
              ? `existing core instance (pid ${existingRecord.pid})`
              : "existing core instance"
        },
        stop() {
          if (existingRecord?.port === port) {
            terminatePid(existingRecord.pid);
          }
        }
      };
    }

    // Port occupied but core is not healthy — try to kill the stale process and retry
    if (existingRecord?.port === port) {
      console.info(`[core] port ${port} occupied by unhealthy process (pid ${existingRecord.pid}), terminating...`);
      terminatePid(existingRecord.pid);
      clearCoreProcessRecord();

      if (!(await waitForPortAvailable(port, 10000, 250))) {
        return {
          state: {
            managed: false,
            running: false,
            apiBase,
            port,
            logRetentionDays: Number(process.env.LOG_RETENTION_DAYS || 30),
            logMaxRecords: Number(process.env.LOG_MAX_RECORDS || 10000),
            lastError: `Port ${port} is still occupied after terminating stale process. Free the port or change the fixed port in Settings.`
          },
          stop() {}
        };
      }
    } else {
      return {
        state: {
          managed: false,
          running: false,
          apiBase,
          port,
          logRetentionDays: Number(process.env.LOG_RETENTION_DAYS || 30),
          logMaxRecords: Number(process.env.LOG_MAX_RECORDS || 10000),
          lastError: `Port ${port} is already occupied. Free the port or change the fixed port in Settings.`
        },
        stop() {}
      };
    }
  }

  const explicitCoreExecutable = process.env.CORE_EXECUTABLE;
  if (explicitCoreExecutable) {
    return spawnCoreBinary(
      explicitCoreExecutable,
      runtimePaths.coreDir,
      port,
      localGatewayPort,
      apiBase
    );
  }

  if (app.isPackaged && existsSync(runtimePaths.binaryPath)) {
    return spawnCoreBinary(
      runtimePaths.binaryPath,
      runtimePaths.coreDir,
      port,
      localGatewayPort,
      apiBase
    );
  }

  if (app.isPackaged) {
    return {
      state: {
        managed: false,
        running: false,
        apiBase,
        port,
        logRetentionDays: Number(process.env.LOG_RETENTION_DAYS || 30),
        logMaxRecords: Number(process.env.LOG_MAX_RECORDS || 10000),
        lastError: `Bundled core executable not found at ${runtimePaths.binaryPath}`
      },
      stop() {}
    };
  }

  const goBinary = resolveGoBinary();
  const hasFreshBinary =
    existsSync(runtimePaths.binaryPath) &&
    isDevelopmentBinaryFresh(runtimePaths.coreDir, runtimePaths.binaryPath);
  if (hasFreshBinary) {
    return spawnCoreBinary(
      runtimePaths.binaryPath,
      runtimePaths.coreDir,
      port,
      localGatewayPort,
      apiBase
    );
  }

  if (goBinary) {
    const builtBinary = buildCoreBinary(goBinary, runtimePaths.coreDir, runtimePaths.binaryPath);
    if (builtBinary) {
      return spawnCoreBinary(builtBinary, runtimePaths.coreDir, port, localGatewayPort, apiBase);
    }
  }

  if (!goBinary) {
    return {
      state: {
        managed: false,
        running: false,
        apiBase,
        port,
        logRetentionDays: Number(process.env.LOG_RETENTION_DAYS || 30),
        logMaxRecords: Number(process.env.LOG_MAX_RECORDS || 10000),
        lastError: "Go toolchain not found. Set CORE_EXECUTABLE or GO_BINARY."
      },
      stop() {}
    };
  }

  const builtBinary = buildCoreBinary(goBinary, runtimePaths.coreDir, runtimePaths.binaryPath);
  if (builtBinary) {
    return spawnCoreBinary(builtBinary, runtimePaths.coreDir, port, localGatewayPort, apiBase);
  }

  return spawnGoCore(goBinary, runtimePaths.coreDir, port, localGatewayPort, apiBase);
}

function resolveCoreRuntimePaths() {
  const binaryName =
    process.platform === "win32" ? "relay-switch-core.exe" : "relay-switch-core";

  if (app.isPackaged) {
    const coreDir = join(process.resourcesPath, "core");
    const dataDir = join(app.getPath("userData"), "core");
    return {
      coreDir,
      binaryPath: join(coreDir, "bin", binaryName),
      dataDir
    };
  }

  const workspaceRoot =
    process.env.ELECTRON_WORKSPACE_ROOT ?? resolveWorkspaceRoot(process.cwd());
  const coreDir = join(workspaceRoot, "core");
  return {
    coreDir,
    binaryPath: join(coreDir, "bin", binaryName),
    dataDir: join(coreDir, "data")
  };
}

function resolveLocalGatewayRuntimeExecutable(): string | null {
  if (process.env.LOCAL_GATEWAY_RUNTIME_EXECUTABLE) {
    return process.env.LOCAL_GATEWAY_RUNTIME_EXECUTABLE;
  }

  const binaryName =
    process.platform === "win32" ? "ai-mini-gateway.exe" : "ai-mini-gateway";

  if (app.isPackaged) {
    const bundledPath = join(process.resourcesPath, "ai-mini-gateway", "bin", binaryName);
    return existsSync(bundledPath) ? bundledPath : null;
  }

  const workspaceRoot =
    process.env.ELECTRON_WORKSPACE_ROOT ?? resolveWorkspaceRoot(process.cwd());
  const localPath = join(
    workspaceRoot,
    "apps",
    "desktop",
    "resources",
    "ai-mini-gateway",
    "bin",
    binaryName
  );
  return existsSync(localPath) ? localPath : null;
}

function localGatewayRuntimeEnv() {
  const executable = resolveLocalGatewayRuntimeExecutable();
  if (!executable) {
    return {};
  }

  return {
    LOCAL_GATEWAY_RUNTIME_EXECUTABLE: executable
  };
}

function spawnCoreBinary(
  executable: string,
  coreDir: string,
  port: number,
  localGatewayPort: number,
  apiBase: string
): CoreRuntimeHandle {
  console.info(`[core] starting binary ${executable} on port ${port}`);
  const dataDir = resolveCoreRuntimePaths().dataDir;
  mkdirSync(dataDir, { recursive: true });
  const isWindows = process.platform === "win32";
  const child = spawn(executable, [], {
    cwd: coreDir,
    stdio: isWindows ? "pipe" : "inherit",
    windowsHide: true,
    env: {
      ...process.env,
      ...localGatewayRuntimeEnv(),
      HTTP_PORT: String(port),
      LOCAL_GATEWAY_RUNTIME_PORT: String(localGatewayPort),
      CORE_DATA_DIR: dataDir
    }
  });

  if (isWindows && child.stdout && child.stderr) {
    child.stdout.on("data", (data: Buffer) => {
      process.stdout.write(data);
    });
    child.stderr.on("data", (data: Buffer) => {
      process.stderr.write(data);
    });
  }

  const state: CoreRuntimeState = {
    managed: true,
    running: true,
    apiBase,
    port,
    pid: child.pid ?? undefined,
    logRetentionDays: Number(process.env.LOG_RETENTION_DAYS || 30),
    logMaxRecords: Number(process.env.LOG_MAX_RECORDS || 10000),
    command: executable
  };

  if (child.pid) {
    saveCoreProcessRecord({
      pid: child.pid,
      port,
      apiBase,
      command: executable,
      managedByApp: true,
      recordedAt: new Date().toISOString()
    });
  }

  child.on("exit", (code, signal) => {
    state.running = false;
    state.lastError = `core exited (code=${code ?? "null"}, signal=${signal ?? "null"})`;
    console.error(`[core] process exited: ${state.lastError}`);
    if (child.pid) {
      const record = loadCoreProcessRecord();
      if (record?.pid === child.pid) {
        clearCoreProcessRecord();
      }
    }
  });

  const healthAttempts = process.platform === "win32" ? 40 : 20;
  void waitForHealth(`${apiBase}/health`, healthAttempts, 250).catch((error) => {
    state.lastError = error instanceof Error ? error.message : "core healthcheck failed";
    console.error(`[core] ${state.lastError}`);
  });

  return {
    state,
    stop() {
      terminateChild(child);
    }
  };
}

async function spawnGoCore(
  goBinary: string,
  coreDir: string,
  port: number,
  localGatewayPort: number,
  apiBase: string
): Promise<CoreRuntimeHandle> {
  const workspaceRoot =
    process.env.ELECTRON_WORKSPACE_ROOT ?? resolveWorkspaceRoot(process.cwd());
  const cacheDir = join(workspaceRoot, ".gocache");
  const modCacheDir = join(workspaceRoot, ".gomodcache");
  mkdirSync(cacheDir, { recursive: true });
  mkdirSync(modCacheDir, { recursive: true });
  const dataDir = resolveCoreRuntimePaths().dataDir;
  mkdirSync(dataDir, { recursive: true });

  const command = `${goBinary} run cmd/relay-switch-core/main.go`;
  console.info(`[core] starting via go run on port ${port}`);
  const isWindows = process.platform === "win32";
  const child = spawn(goBinary, ["run", "cmd/relay-switch-core/main.go"], {
    cwd: coreDir,
    stdio: isWindows ? "pipe" : "inherit",
    windowsHide: true,
    env: {
      ...process.env,
      ...localGatewayRuntimeEnv(),
      HTTP_PORT: String(port),
      LOCAL_GATEWAY_RUNTIME_PORT: String(localGatewayPort),
      CORE_DATA_DIR: dataDir,
      GOCACHE: cacheDir,
      GOMODCACHE: modCacheDir
    }
  });

  if (isWindows && child.stdout && child.stderr) {
    child.stdout.on("data", (data: Buffer) => {
      process.stdout.write(data);
    });
    child.stderr.on("data", (data: Buffer) => {
      process.stderr.write(data);
    });
  }

  const state: CoreRuntimeState = {
    managed: true,
    running: true,
    apiBase,
    port,
    pid: child.pid ?? undefined,
    logRetentionDays: Number(process.env.LOG_RETENTION_DAYS || 30),
    logMaxRecords: Number(process.env.LOG_MAX_RECORDS || 10000),
    command
  };

  if (child.pid) {
    saveCoreProcessRecord({
      pid: child.pid,
      port,
      apiBase,
      command,
      managedByApp: true,
      recordedAt: new Date().toISOString()
    });
  }

  child.on("exit", (code, signal) => {
    state.running = false;
    state.lastError = `core exited (code=${code ?? "null"}, signal=${signal ?? "null"})`;
    console.error(`[core] process exited: ${state.lastError}`);
    if (child.pid) {
      const record = loadCoreProcessRecord();
      if (record?.pid === child.pid) {
        clearCoreProcessRecord();
      }
    }
  });

  try {
    await waitForHealth(`${apiBase}/health`, 20, 250);
    console.info(`[core] healthcheck ready at ${apiBase}/health`);
  } catch (error) {
    state.lastError = error instanceof Error ? error.message : "core healthcheck failed";
    console.error(`[core] ${state.lastError}`);
  }

  return {
    state,
    stop() {
      terminateChild(child);
    }
  };
}

function buildCoreBinary(
  goBinary: string,
  coreDir: string,
  binaryPath: string
): string | null {
  const workspaceRoot =
    process.env.ELECTRON_WORKSPACE_ROOT ?? resolveWorkspaceRoot(process.cwd());
  const cacheDir = join(workspaceRoot, ".gocache");
  const modCacheDir = join(workspaceRoot, ".gomodcache");
  const binDir = join(coreDir, "bin");
  mkdirSync(cacheDir, { recursive: true });
  mkdirSync(modCacheDir, { recursive: true });
  mkdirSync(binDir, { recursive: true });

  const result = spawnSync(
    goBinary,
    ["build", "-o", binaryPath, "./cmd/relay-switch-core"],
    {
      cwd: coreDir,
      stdio: "inherit",
      windowsHide: true,
      env: {
        ...process.env,
        GOCACHE: cacheDir,
        GOMODCACHE: modCacheDir
      }
    }
  );

  if (result.status === 0 && existsSync(binaryPath)) {
    return binaryPath;
  }

  return null;
}

function resolveGoBinary(): string | null {
  const candidates = [process.env.GO_BINARY, "go"].filter(Boolean) as string[];

  for (const candidate of candidates) {
    if (isUsableGoBinary(candidate)) {
      return candidate;
    }
  }

  return null;
}

function isUsableGoBinary(candidate: string): boolean {
  const versionResult = spawnSync(candidate, ["version"], {
    stdio: "ignore",
    windowsHide: true
  });
  if (versionResult.status !== 0) {
    return false;
  }

  const gorootResult = spawnSync(candidate, ["env", "GOROOT"], {
    encoding: "utf8",
    windowsHide: true
  });
  if (gorootResult.status !== 0) {
    return false;
  }

  const goroot = gorootResult.stdout.trim();
  if (!goroot) {
    return false;
  }

  // Only accept Go binaries that can resolve a usable stdlib from GOROOT.
  const stdlibFiles = [
    join(goroot, "src", "context", "context.go"),
    join(goroot, "src", "fmt", "print.go"),
    join(goroot, "src", "log", "log.go")
  ];

  try {
    return stdlibFiles.every((file) => existsSync(file) && statSync(file).size > 0);
  } catch {
    return false;
  }
}

function isDevelopmentBinaryFresh(coreDir: string, binaryPath: string): boolean {
  try {
    const binaryMtime = statSync(binaryPath).mtimeMs;
    const latestSourceMtime = getLatestCoreSourceMtime(coreDir);
    return binaryMtime >= latestSourceMtime;
  } catch {
    return false;
  }
}

function getLatestCoreSourceMtime(coreDir: string): number {
  const pathsToCheck = [
    join(coreDir, "go.mod"),
    join(coreDir, "go.sum"),
    join(coreDir, "cmd"),
    join(coreDir, "internal")
  ];

  let latest = 0;
  for (const path of pathsToCheck) {
    latest = Math.max(latest, getPathLatestMtime(path));
  }

  return latest;
}

function getPathLatestMtime(targetPath: string): number {
  if (!existsSync(targetPath)) {
    return 0;
  }

  const stats = statSync(targetPath);
  if (!stats.isDirectory()) {
    return stats.mtimeMs;
  }

  let latest = stats.mtimeMs;
  for (const entry of readdirSync(targetPath, { withFileTypes: true })) {
    latest = Math.max(latest, getPathLatestMtime(join(targetPath, entry.name)));
  }

  return latest;
}

function resolveWorkspaceRoot(startDir: string): string {
  let current = startDir;

  for (let depth = 0; depth < 6; depth += 1) {
    if (existsSync(join(current, "core", "cmd", "relay-switch-core", "main.go"))) {
      return current;
    }

    const parent = join(current, "..");
    if (parent === current) {
      break;
    }
    current = parent;
  }

  return startDir;
}

function isPortAvailable(port: number): Promise<boolean> {
  return new Promise((resolve) => {
    const server = createServer();

    server.once("error", () => {
      resolve(false);
    });

    server.once("listening", () => {
      server.close(() => resolve(true));
    });

    server.listen(port, "127.0.0.1");
  });
}

async function waitForPortAvailable(port: number, timeoutMs: number, intervalMs: number) {
  const deadline = Date.now() + timeoutMs;

  while (Date.now() < deadline) {
    if (await isPortAvailable(port)) {
      return true;
    }
    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }

  return isPortAvailable(port);
}

async function waitForHealth(url: string, attempts: number, intervalMs: number) {
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        return;
      }
    } catch {
      // Ignore until attempts are exhausted.
    }

    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }

  throw new Error(`core healthcheck did not become ready at ${url}`);
}

async function isHealthyCore(apiBase: string) {
  try {
    await waitForHealth(`${apiBase}/health`, 2, 150);
    return true;
  } catch {
    return false;
  }
}

function terminateChild(child: ChildProcess) {
  if (child.killed) {
    return;
  }

  if (child.pid) {
    const record = loadCoreProcessRecord();
    if (record?.pid === child.pid) {
      clearCoreProcessRecord();
    }
  }

  child.kill("SIGTERM");
}

function terminatePid(pid: number) {
  try {
    process.kill(pid, "SIGTERM");
    const record = loadCoreProcessRecord();
    if (record?.pid === pid) {
      clearCoreProcessRecord();
    }
  } catch (error) {
    console.error("[core] failed to terminate existing pid:", error);
  }
}

function parsePort(apiBase: string): number {
  try {
    return Number(new URL(apiBase).port || 80);
  } catch {
    return 0;
  }
}
