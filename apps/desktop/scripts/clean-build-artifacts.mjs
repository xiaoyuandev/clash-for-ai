import { existsSync, rmSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const desktopDir = join(__dirname, "..");
const workspaceRoot = join(desktopDir, "..", "..");

const targets = [
  join(desktopDir, "dist"),
  join(desktopDir, "out"),
  join(workspaceRoot, "core", "bin")
];

for (const target of targets) {
  if (!existsSync(target)) {
    continue;
  }

  rmSync(target, { recursive: true, force: true });
  console.log(`[clean] removed ${target}`);
}
