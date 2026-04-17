import { spawn, spawnSync, type ChildProcess } from "node:child_process";
import { existsSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import { createServer } from "node:net";

export interface CoreRuntimeState {
  managed: boolean;
  running: boolean;
  apiBase: string;
  port: number;
  lastError?: string;
  command?: string;
}

export interface CoreRuntimeHandle {
  state: CoreRuntimeState;
  stop: () => void;
}

export async function startCoreProcess(): Promise<CoreRuntimeHandle> {
  const explicitApiBase = process.env.ELECTRON_API_BASE;
  if (explicitApiBase) {
    return {
      state: {
        managed: false,
        running: true,
        apiBase: explicitApiBase,
        port: parsePort(explicitApiBase)
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
  const desiredPort = Number(process.env.ELECTRON_API_PORT || 3456);
  const port = await pickPort([desiredPort, 3457, 3458, 3459, 3460]);
  const apiBase = `http://127.0.0.1:${port}`;

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
    command: executable
  };

  child.on("exit", (code, signal) => {
    state.running = false;
    state.lastError = `core exited (code=${code ?? "null"}, signal=${signal ?? "null"})`;
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
    command
  };

  child.on("exit", (code, signal) => {
    state.running = false;
    state.lastError = `core exited (code=${code ?? "null"}, signal=${signal ?? "null"})`;
  });

  try {
    await waitForHealth(`${apiBase}/health`, 20, 250);
  } catch (error) {
    state.lastError = error instanceof Error ? error.message : "core healthcheck failed";
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

async function pickPort(candidates: number[]): Promise<number> {
  for (const candidate of candidates) {
    if (await isPortAvailable(candidate)) {
      return candidate;
    }
  }

  return createEphemeralPort();
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

function createEphemeralPort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = createServer();

    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      if (typeof address === "object" && address?.port) {
        const port = address.port;
        server.close(() => resolve(port));
        return;
      }
      server.close(() => reject(new Error("failed to allocate ephemeral port")));
    });
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

function terminateChild(child: ChildProcess) {
  if (child.killed) {
    return;
  }

  child.kill("SIGTERM");
}

function parsePort(apiBase: string): number {
  try {
    return Number(new URL(apiBase).port || 80);
  } catch {
    return 0;
  }
}
