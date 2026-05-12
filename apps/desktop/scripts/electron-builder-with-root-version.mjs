import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const desktopDir = join(__dirname, "..");
const workspaceRoot = join(desktopDir, "..", "..");
const rootPackagePath = join(workspaceRoot, "package.json");
const rootPackage = JSON.parse(readFileSync(rootPackagePath, "utf8"));
const rootVersion = rootPackage.version;

if (!rootVersion) {
  console.error("[electron-builder] root package.json is missing version");
  process.exit(1);
}

const expectedTag = `v${rootVersion}`;
const releaseRef = process.env.GITHUB_REF_NAME || process.env.RELEASE_VERSION || "";

if (releaseRef && releaseRef.startsWith("v") && releaseRef !== expectedTag) {
  console.error(
    `[electron-builder] release tag ${releaseRef} does not match root package version ${expectedTag}`
  );
  process.exit(1);
}

const electronBuilderBin = process.platform === "win32" ? "electron-builder.cmd" : "electron-builder";
const args = [`--config.extraMetadata.version=${rootVersion}`, ...process.argv.slice(2)];

console.log(`[electron-builder] using root package version ${rootVersion}`);

const result = spawnSync(electronBuilderBin, args, {
  cwd: desktopDir,
  stdio: "inherit",
  shell: process.platform === "win32"
});

process.exit(result.status ?? 1);
