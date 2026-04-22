import { mkdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const __dirname = dirname(fileURLToPath(import.meta.url));
const workspaceRoot = join(__dirname, "..", "..", "..");
const coreDir = join(workspaceRoot, "core");
const goBinary = process.env.GO_BINARY || "go";
const goos = process.env.CORE_GOOS || mapPlatform(process.platform);
const goarch = process.env.CORE_GOARCH || mapArch(process.arch);
const binaryName = goos === "windows" ? "clash-for-ai-core.exe" : "clash-for-ai-core";
const outputPath = join(coreDir, "bin", binaryName);
const cacheDir = join(workspaceRoot, ".gocache");
const modCacheDir = join(workspaceRoot, ".gomodcache");

if (!goos) {
  console.error(`[build-core] unsupported Node platform: ${process.platform}`);
  process.exit(1);
}

if (!goarch) {
  console.error(`[build-core] unsupported Node architecture: ${process.arch}`);
  process.exit(1);
}

mkdirSync(join(coreDir, "bin"), { recursive: true });
mkdirSync(cacheDir, { recursive: true });
mkdirSync(modCacheDir, { recursive: true });

console.log(`[build-core] building ${goos}/${goarch} -> ${outputPath}`);

const result = spawnSync(
  goBinary,
  ["build", "-o", outputPath, "./cmd/clash-for-ai-core"],
  {
    cwd: coreDir,
    stdio: "inherit",
    env: {
      ...process.env,
      GOOS: goos,
      GOARCH: goarch,
      CGO_ENABLED: process.env.CGO_ENABLED || "0",
      GOCACHE: cacheDir,
      GOMODCACHE: modCacheDir
    }
  }
);

if (result.status !== 0) {
  process.exit(result.status ?? 1);
}

function mapPlatform(platform) {
  switch (platform) {
    case "darwin":
      return "darwin";
    case "win32":
      return "windows";
    case "linux":
      return "linux";
    default:
      return null;
  }
}

function mapArch(arch) {
  switch (arch) {
    case "arm64":
      return "arm64";
    case "x64":
      return "amd64";
    default:
      return null;
  }
}
