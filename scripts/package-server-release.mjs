import { cpSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const __dirname = dirname(fileURLToPath(import.meta.url));
const workspaceRoot = join(__dirname, "..");
const coreDir = join(workspaceRoot, "core");
const webDistDir = join(workspaceRoot, "apps", "web", "dist");
const runtimeResourcesDir = join(workspaceRoot, "apps", "desktop", "resources", "ai-mini-gateway");
const runtimeManifestPath = join(runtimeResourcesDir, "manifest.json");

const releaseVersion =
  process.env.RELEASE_VERSION ||
  process.env.GITHUB_REF_NAME ||
  `v${JSON.parse(readFileSync(join(workspaceRoot, "package.json"), "utf8")).version}`;
const goos = process.env.RELEASE_GOOS || "linux";
const goarch = process.env.RELEASE_GOARCH || "amd64";
const coreBinaryName = goos === "windows" ? "relay-switch-core.exe" : "relay-switch-core";
const runtimeBinaryName = goos === "windows" ? "ai-mini-gateway.exe" : "ai-mini-gateway";
const coreBinaryPath = join(coreDir, "bin", coreBinaryName);
const runtimeBinaryPath = join(runtimeResourcesDir, "bin", runtimeBinaryName);
const outputDir = join(workspaceRoot, "dist", "server");
const packageRootName = `relay-switch-server_${releaseVersion}_${goos}_${goarch}`;
const stagingDir = join(outputDir, packageRootName);
const archivePath = join(outputDir, `${packageRootName}.tar.gz`);

if (!existsSync(coreBinaryPath)) {
  console.error(`[package-server-release] missing core binary: ${coreBinaryPath}`);
  process.exit(1);
}

if (!existsSync(runtimeBinaryPath)) {
  console.error(`[package-server-release] missing runtime binary: ${runtimeBinaryPath}`);
  process.exit(1);
}

if (!existsSync(webDistDir)) {
  console.error(`[package-server-release] missing web dist: ${webDistDir}`);
  process.exit(1);
}

rmSync(stagingDir, { recursive: true, force: true });
mkdirSync(join(stagingDir, "bin"), { recursive: true });
mkdirSync(join(stagingDir, "web"), { recursive: true });

cpSync(coreBinaryPath, join(stagingDir, "bin", coreBinaryName));
cpSync(runtimeBinaryPath, join(stagingDir, "bin", runtimeBinaryName));
cpSync(webDistDir, join(stagingDir, "web"), { recursive: true });
cpSync(runtimeManifestPath, join(stagingDir, "ai-mini-gateway-manifest.json"));

const runtimeManifest = JSON.parse(readFileSync(runtimeManifestPath, "utf8"));
writeFileSync(
  join(stagingDir, "release.json"),
  `${JSON.stringify(
    {
      release_version: releaseVersion,
      platform: goos,
      arch: goarch,
      core_binary: coreBinaryName,
      runtime_binary: runtimeBinaryName,
      runtime_kind: runtimeManifest.runtime_kind,
      runtime_version: runtimeManifest.version,
      runtime_commit: runtimeManifest.commit,
      packaged_at: new Date().toISOString()
    },
    null,
    2
  )}\n`
);

mkdirSync(outputDir, { recursive: true });
rmSync(archivePath, { force: true });

const tar = spawnSync("tar", ["-czf", archivePath, "-C", outputDir, packageRootName], {
  stdio: "inherit"
});

if (tar.status !== 0) {
  process.exit(tar.status ?? 1);
}

console.log(`[package-server-release] created ${archivePath}`);
