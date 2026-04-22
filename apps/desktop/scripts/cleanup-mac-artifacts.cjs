const { existsSync, rmSync } = require("node:fs");
const { join } = require("node:path");

module.exports = async function cleanupMacArtifacts(buildResult) {
  const outDir = buildResult?.outDir;
  const appId = buildResult?.configuration?.appId;

  if (!outDir || !appId) {
    return [];
  }

  const transientPkgPlist = join(outDir, `${appId}.plist`);
  if (existsSync(transientPkgPlist)) {
    rmSync(transientPkgPlist, { force: true });
    console.log(`[cleanup] removed ${transientPkgPlist}`);
  }

  return [];
};
