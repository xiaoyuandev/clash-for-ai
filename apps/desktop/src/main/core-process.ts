import { spawn, spawnSync, type ChildProcess } from "node:child_process";
import { existsSync, mkdirSync } from "node:fs";
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

  const workspaceRoot =
    process.env.ELECTRON_WORKSPACE_ROOT ?? resolveWorkspaceRoot(process.cwd());
  const coreDir = join(workspaceRoot, "core");
  const binaryName =
    process.platform === "win32" ? "clash-for-ai-core.exe" : "clash-for-ai-core";
  const binaryPath = join(coreDir, "bin", binaryName);
  const port = options.desiredPort;
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

  const explicitCoreExecutable = process.env.CORE_EXECUTABLE;
  if (explicitCoreExecutable) {
    return spawnCoreBinary(explicitCoreExecutable, coreDir, port, apiBase);
  }

  if (existsSync(binaryPath)) {
    return spawnCoreBinary(binaryPath, coreDir, port, apiBase);
  }

  const goBinary = resolveGoBinary();
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

  const builtBinary = buildCoreBinary(goBinary, coreDir, binaryPath);
  if (builtBinary) {
    return spawnCoreBinary(builtBinary, coreDir, port, apiBase);
  }

  return spawnGoCore(goBinary, coreDir, port, apiBase);
}

function spawnCoreBinary(
  executable: string,
  coreDir: string,
  port: number,
  apiBase: string
): CoreRuntimeHandle {
  console.info(`[core] starting binary ${executable} on port ${port}`);
  const child = spawn(executable, [], {
    cwd: coreDir,
    stdio: "inherit",
    env: {
      ...process.env,
      HTTP_PORT: String(port)
    }
  });

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

  void waitForHealth(`${apiBase}/health`, 20, 250).catch((error) => {
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
  apiBase: string
): Promise<CoreRuntimeHandle> {
  const workspaceRoot =
    process.env.ELECTRON_WORKSPACE_ROOT ?? resolveWorkspaceRoot(process.cwd());
  const cacheDir = join(workspaceRoot, ".gocache");
  const modCacheDir = join(workspaceRoot, ".gomodcache");
  mkdirSync(cacheDir, { recursive: true });
  mkdirSync(modCacheDir, { recursive: true });

  const command = `${goBinary} run cmd/clash-for-ai-core/main.go`;
  console.info(`[core] starting via go run on port ${port}`);
  const child = spawn(goBinary, ["run", "cmd/clash-for-ai-core/main.go"], {
    cwd: coreDir,
    stdio: "inherit",
    env: {
      ...process.env,
      HTTP_PORT: String(port),
      GOCACHE: cacheDir,
      GOMODCACHE: modCacheDir
    }
  });

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
    ["build", "-o", binaryPath, "./cmd/clash-for-ai-core"],
    {
      cwd: coreDir,
      stdio: "inherit",
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
  const candidates = [
    process.env.GO_BINARY,
    "/tmp/go-toolchain/go/bin/go",
    "go"
  ].filter(Boolean) as string[];

  for (const candidate of candidates) {
    const result = spawnSync(candidate, ["version"], { stdio: "ignore" });
    if (result.status === 0) {
      return candidate;
    }
  }

  return null;
}

function resolveWorkspaceRoot(startDir: string): string {
  let current = startDir;

  for (let depth = 0; depth < 6; depth += 1) {
    if (existsSync(join(current, "core", "cmd", "clash-for-ai-core", "main.go"))) {
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
