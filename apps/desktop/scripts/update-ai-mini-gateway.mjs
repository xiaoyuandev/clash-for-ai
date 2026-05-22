import { readFileSync, writeFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const workspaceRoot = join(__dirname, "..", "..", "..");
const desktopDir = join(workspaceRoot, "apps", "desktop");
const resourcesDir = join(desktopDir, "resources", "ai-mini-gateway");
const manifestPath = join(resourcesDir, "manifest.json");
const prepareScriptPath = join(desktopDir, "scripts", "prepare-ai-mini-gateway.mjs");

const args = process.argv.slice(2);
const shouldPrepare = args.includes("--prepare");
const versionArg = args.find((arg) => !arg.startsWith("--")) || "latest";

const manifest = JSON.parse(readFileSync(manifestPath, "utf8"));
const repo = manifest.release_repo;
const tag = normalizeTag(versionArg);

try {
  const release = tag === "latest" ? await fetchLatestRelease(repo) : await fetchReleaseByTag(repo, tag);
  const releaseTag = readReleaseTag(release);
  const commitSha = await resolveTagCommit(repo, releaseTag);
  const commitDate = await fetchCommitDate(repo, commitSha);

  const nextManifest = {
    ...manifest,
    version: releaseTag,
    commit: commitSha.slice(0, 7),
    source_commit_time: toShanghaiOffset(commitDate)
  };

  writeFileSync(manifestPath, `${JSON.stringify(nextManifest, null, 2)}\n`);

  console.log(
    `[update-ai-mini-gateway] updated manifest to ${nextManifest.version} (${nextManifest.commit})`
  );

  if (shouldPrepare) {
    console.log("[update-ai-mini-gateway] refreshing bundled runtime via prepare script");
    runNodeScript(prepareScriptPath);
  }
} catch (error) {
  console.error(
    `[update-ai-mini-gateway] ${error instanceof Error ? error.message : String(error)}`
  );
  process.exit(1);
}

function normalizeTag(value) {
  if (value === "latest") {
    return value;
  }

  return value.startsWith("v") ? value : `v${value}`;
}

function readReleaseTag(release) {
  const tagName = release?.tagName || release?.tag_name;
  if (typeof tagName !== "string" || !tagName) {
    throw new Error("unable to resolve release tag name from GitHub response");
  }
  return tagName;
}

async function fetchLatestRelease(repo) {
  return fetchGitHubJSON(`https://api.github.com/repos/${repo}/releases/latest`);
}

async function fetchReleaseByTag(repo, tagName) {
  return fetchGitHubJSON(`https://api.github.com/repos/${repo}/releases/tags/${tagName}`);
}

async function resolveTagCommit(repo, tagName) {
  const ref = await fetchGitHubJSON(`https://api.github.com/repos/${repo}/git/ref/tags/${tagName}`);

  if (ref.object?.type === "commit") {
    return ref.object.sha;
  }

  if (ref.object?.type === "tag") {
    const annotatedTag = await fetchGitHubJSON(ref.object.url);
    if (annotatedTag.object?.type === "commit") {
      return annotatedTag.object.sha;
    }
  }

  throw new Error(`unable to resolve commit sha for tag ${tagName}`);
}

async function fetchCommitDate(repo, sha) {
  const commit = await fetchGitHubJSON(`https://api.github.com/repos/${repo}/commits/${sha}`);
  const authoredAt = commit?.commit?.author?.date;
  if (typeof authoredAt !== "string" || !authoredAt) {
    throw new Error(`unable to resolve commit date for ${sha}`);
  }
  return authoredAt;
}

async function fetchGitHubJSON(url) {
  const token = process.env.GITHUB_TOKEN || process.env.GH_TOKEN || "";
  const headers = {
    Accept: "application/vnd.github+json",
    "User-Agent": "relay-switch-update-ai-mini-gateway-script"
  };

  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const response = await fetch(url, {
    headers
  });

  if (!response.ok) {
    let details = `${response.status} ${response.statusText}`;
    try {
      const body = await response.json();
      if (typeof body?.message === "string" && body.message) {
        details = `${details}: ${body.message}`;
      }
    } catch {
      // Ignore JSON parse failures for non-JSON error bodies.
    }

    throw new Error(`GitHub API request failed for ${url}: ${details}`);
  }

  return response.json();
}

function toShanghaiOffset(isoString) {
  const date = new Date(isoString);
  if (Number.isNaN(date.getTime())) {
    throw new Error(`invalid commit date: ${isoString}`);
  }

  const shanghaiMs = date.getTime() + 8 * 60 * 60 * 1000;
  const shanghaiDate = new Date(shanghaiMs);
  const yyyy = shanghaiDate.getUTCFullYear();
  const mm = String(shanghaiDate.getUTCMonth() + 1).padStart(2, "0");
  const dd = String(shanghaiDate.getUTCDate()).padStart(2, "0");
  const hh = String(shanghaiDate.getUTCHours()).padStart(2, "0");
  const min = String(shanghaiDate.getUTCMinutes()).padStart(2, "0");
  const ss = String(shanghaiDate.getUTCSeconds()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd}T${hh}:${min}:${ss}+08:00`;
}

function runNodeScript(scriptPath) {
  const result = spawnSync(process.execPath, [scriptPath], {
    cwd: workspaceRoot,
    stdio: "inherit",
    env: process.env
  });

  if (result.status !== 0) {
    throw new Error(`prepare script failed with exit code ${result.status ?? 1}`);
  }
}
