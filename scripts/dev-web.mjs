import { spawn, spawnSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import process from "node:process";

const workspaceRoot = process.cwd();
const coreDir = join(workspaceRoot, "core");
const dataDir = join(coreDir, "data");
const cacheDir = join(workspaceRoot, ".gocache");
const modCacheDir = join(workspaceRoot, ".gomodcache");
const envLocalPath = join(workspaceRoot, ".env.local");
const localEnv = loadEnvFile(envLocalPath);
const devEnv = {
  ...localEnv,
  ...process.env
};
const localGatewayBinaryName = process.platform === "win32" ? "ai-mini-gateway.exe" : "ai-mini-gateway";
const localGatewayExecutable = join(
  workspaceRoot,
  "apps",
  "desktop",
  "resources",
  "ai-mini-gateway",
  "bin",
  localGatewayBinaryName
);

const configuredLocalGatewayExecutable =
  devEnv.LOCAL_GATEWAY_RUNTIME_EXECUTABLE ||
  (existsSync(localGatewayExecutable) ? localGatewayExecutable : "");

const httpPort = devEnv.HTTP_PORT || devEnv.CLASH_FOR_AI_HTTP_PORT || "3456";
const localGatewayPort =
  devEnv.LOCAL_GATEWAY_RUNTIME_PORT || devEnv.CLASH_FOR_AI_LOCAL_GATEWAY_PORT || "3457";

const children = new Set();
let shuttingDown = false;

mkdirSync(dataDir, { recursive: true });
mkdirSync(cacheDir, { recursive: true });
mkdirSync(modCacheDir, { recursive: true });

if (!existsSync(join(coreDir, "cmd", "clash-for-ai-core", "main.go"))) {
  console.error("[dev:web] core entrypoint not found. Run this command from the repository root.");
  process.exit(1);
}

if (existsSync(envLocalPath)) {
  console.log("[dev:web] loaded local environment from .env.local");
}

if (!configuredLocalGatewayExecutable) {
  console.warn(
    "[dev:web] LOCAL_GATEWAY_RUNTIME_EXECUTABLE is not configured. " +
      "Core will still start, but local gateway auto-launch is disabled."
  );
} else if (!existsSync(configuredLocalGatewayExecutable)) {
  console.warn(
    `[dev:web] LOCAL_GATEWAY_RUNTIME_EXECUTABLE points to a missing file: ${configuredLocalGatewayExecutable}. ` +
      "Core will still start, but local gateway auto-launch may fail."
  );
}

console.log(`[dev:web] starting core on http://127.0.0.1:${httpPort}`);
if (configuredLocalGatewayExecutable) {
  console.log(`[dev:web] using local gateway runtime ${configuredLocalGatewayExecutable}`);
}
console.log("[dev:web] starting web dev server");

await releasePort(Number(httpPort), "core");
await releasePort(Number(localGatewayPort), "local gateway");

if (!(await isPortAvailable(Number(httpPort)))) {
  console.error(`[dev:web] port ${httpPort} is still in use after cleanup.`);
  process.exit(1);
}

if (!(await isPortAvailable(Number(localGatewayPort)))) {
  console.error(`[dev:web] port ${localGatewayPort} is still in use after cleanup.`);
  process.exit(1);
}

const core = start("core", "go", ["run", "cmd/clash-for-ai-core/main.go"], {
  cwd: coreDir,
  env: {
    ...devEnv,
    HTTP_PORT: httpPort,
    LOCAL_GATEWAY_RUNTIME_PORT: localGatewayPort,
    LOCAL_GATEWAY_RUNTIME_EXECUTABLE: configuredLocalGatewayExecutable,
    CORE_DATA_DIR: dataDir,
    GOCACHE: cacheDir,
    GOMODCACHE: modCacheDir
  }
});

const web = start("web", "pnpm", ["--filter", "web", "dev"], {
  cwd: workspaceRoot,
  env: devEnv
});

for (const signal of ["SIGINT", "SIGTERM"]) {
  process.on(signal, () => shutdown(signal));
}

core.on("exit", (code, signal) => handleExit("core", core, code, signal));
web.on("exit", (code, signal) => handleExit("web", web, code, signal));

function start(name, command, args, options) {
  const child = spawn(command, args, {
    ...options,
    detached: process.platform !== "win32",
    stdio: "inherit",
    shell: process.platform === "win32"
  });

  children.add(child);

  child.on("error", (error) => {
    console.error(`[dev:web] failed to start ${name}: ${error.message}`);
    shutdown("START_ERROR", 1);
  });

  return child;
}

function loadEnvFile(filePath) {
  if (!existsSync(filePath)) {
    return {};
  }

  const env = {};
  const content = readFileSync(filePath, "utf8");

  for (const line of content.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }

    const match = /^(?:export\s+)?([\w.-]+)\s*=\s*(.*)$/.exec(trimmed);
    if (!match) {
      continue;
    }

    const [, key, rawValue] = match;
    env[key] = parseEnvValue(rawValue);
  }

  return env;
}

function parseEnvValue(rawValue) {
  const value = rawValue.trim();
  const quote = value[0];

  if ((quote === "\"" || quote === "'") && value.endsWith(quote)) {
    const unquoted = value.slice(1, -1);
    return quote === "\"" ? unescapeDoubleQuotedEnvValue(unquoted) : unquoted;
  }

  return value.replace(/\s+#.*$/, "");
}

function unescapeDoubleQuotedEnvValue(value) {
  return value.replace(/\\n/g, "\n").replace(/\\r/g, "\r").replace(/\\t/g, "\t").replace(/\\"/g, "\"");
}

async function isPortAvailable(port) {
  const { createServer } = await import("node:net");

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

async function releasePort(port, label) {
  if (!Number.isInteger(port) || port <= 0) {
    console.error(`[dev:web] invalid ${label} port: ${port}`);
    process.exit(1);
  }

  const pids = findListeningPids(port);
  if (pids.length === 0) {
    return;
  }

  console.log(`[dev:web] stopping existing ${label} listener on port ${port}: ${pids.join(", ")}`);
  for (const pid of pids) {
    try {
      process.kill(pid, "SIGTERM");
    } catch (error) {
      if (error?.code !== "ESRCH") {
        console.warn(`[dev:web] failed to stop process ${pid}: ${error.message}`);
      }
    }
  }

  if (await waitForPort(port, true, 2000)) {
    return;
  }

  for (const pid of pids) {
    try {
      process.kill(pid, "SIGKILL");
    } catch (error) {
      if (error?.code !== "ESRCH") {
        console.warn(`[dev:web] failed to force stop process ${pid}: ${error.message}`);
      }
    }
  }

  await waitForPort(port, true, 1000);
}

function findListeningPids(port) {
  if (process.platform === "win32") {
    return [];
  }

  const result = spawnSync("lsof", ["-nP", `-iTCP:${port}`, "-sTCP:LISTEN", "-t"], {
    encoding: "utf8"
  });

  if (result.status !== 0 || !result.stdout.trim()) {
    return [];
  }

  return [
    ...new Set(
      result.stdout
        .trim()
        .split(/\s+/)
        .map(Number)
        .filter((pid) => Number.isInteger(pid) && pid > 0)
    )
  ];
}

async function waitForPort(port, shouldBeAvailable, timeoutMs) {
  const startedAt = Date.now();

  while (Date.now() - startedAt < timeoutMs) {
    if ((await isPortAvailable(port)) === shouldBeAvailable) {
      return true;
    }
    await sleep(100);
  }

  return false;
}

function sleep(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

function handleExit(name, child, code, signal) {
  children.delete(child);

  if (shuttingDown) {
    return;
  }

  const reason = signal ? `${name} exited with ${signal}` : `${name} exited with code ${code ?? 0}`;
  console.log(`[dev:web] ${reason}; stopping remaining process.`);
  shutdown("CHILD_EXIT", code ?? 0);
}

function shutdown(reason, exitCode = 0) {
  if (shuttingDown) {
    return;
  }

  shuttingDown = true;

  for (const child of children) {
    terminateChild(child, reason === "SIGINT" ? "SIGINT" : "SIGTERM");
  }

  setTimeout(() => {
    for (const child of children) {
      terminateChild(child, "SIGKILL");
    }
    process.exit(exitCode);
  }, 1500).unref();
}

function terminateChild(child, signal) {
  if (child.killed || !child.pid) {
    return;
  }

  try {
    if (process.platform === "win32") {
      child.kill(signal);
    } else {
      process.kill(-child.pid, signal);
    }
  } catch (error) {
    if (error?.code !== "ESRCH") {
      console.error(`[dev:web] failed to stop child process ${child.pid}: ${error.message}`);
    }
  }
}
