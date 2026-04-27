import { app } from "electron";
import { spawnSync } from "node:child_process";
import { mkdir, readFile, readdir, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { homedir } from "node:os";
import { basename, dirname, join } from "node:path";

export type ToolIntegrationId =
  | "codex-cli"
  | "claude-code"
  | "cursor"
  | "cherry-studio"
  | "open-code"
  | "openai-sdk";

export interface ToolIntegrationState {
  id: ToolIntegrationId;
  detected: boolean;
  configured: boolean;
  supportsAdapter: boolean;
  configPath?: string;
  secondaryConfigPath?: string;
  executablePath?: string;
  backupPath?: string;
  message?: string;
}

const codexConfigPath = join(homedir(), ".codex", "config.toml");
const codexAuthPath = join(homedir(), ".codex", "auth.json");
const claudeConfigPath = join(homedir(), ".claude", "settings.json");

export async function listToolIntegrations(apiPort: number): Promise<ToolIntegrationState[]> {
  return Promise.all([
    inspectCodexIntegration(apiPort),
    inspectClaudeIntegration(apiPort),
    inspectStaticIntegration("cursor"),
    inspectStaticIntegration("cherry-studio"),
    inspectStaticIntegration("open-code"),
    inspectStaticIntegration("openai-sdk")
  ]);
}

export async function applyToolIntegration(
  toolId: ToolIntegrationId,
  apiPort: number
): Promise<ToolIntegrationState> {
  switch (toolId) {
    case "codex-cli":
      return applyCodexIntegration(apiPort);
    case "claude-code":
      return applyClaudeIntegration(apiPort);
    default:
      return inspectStaticIntegration(toolId);
  }
}

export async function restoreToolIntegration(
  toolId: ToolIntegrationId,
  apiPort: number
): Promise<ToolIntegrationState> {
  switch (toolId) {
    case "codex-cli":
      return restoreCodexIntegration(apiPort);
    case "claude-code":
      return restoreClaudeIntegration(apiPort);
    default:
      return inspectStaticIntegration(toolId);
  }
}

export function buildCherryStudioImportUrl(apiPort: number) {
  const payload = {
    id: "custom-provider",
    name: "Clash for AI",
    type: "openai",
    apiKey: "dummy",
    baseUrl: `http://127.0.0.1:${apiPort}/v1`
  };

  const encoded = Buffer.from(JSON.stringify(payload), "utf8").toString("base64");
  return `cherrystudio://providers/api-keys?v=1&data=${encodeURIComponent(encoded)}`;
}

async function inspectCodexIntegration(apiPort: number): Promise<ToolIntegrationState> {
  const executablePath = resolveExecutable(process.platform === "win32" ? "codex.exe" : "codex");
  const configured = await isCodexConfigured(apiPort);
  const backupPath = await getLatestBackupPath("codex-cli");

  return {
    id: "codex-cli",
    detected: Boolean(executablePath) || existsSync(codexConfigPath) || existsSync(codexAuthPath),
    configured,
    supportsAdapter: true,
    configPath: codexConfigPath,
    secondaryConfigPath: codexAuthPath,
    executablePath,
    backupPath,
    message: configured
      ? "Codex CLI is already pointed at the local Clash for AI gateway."
      : "Codex CLI can be configured by updating ~/.codex/config.toml and ~/.codex/auth.json."
  };
}

async function inspectClaudeIntegration(apiPort: number): Promise<ToolIntegrationState> {
  const executablePath = resolveExecutable(process.platform === "win32" ? "claude.exe" : "claude");
  const configured = await isClaudeConfigured(apiPort);
  const backupPath = await getLatestBackupPath("claude-code");

  return {
    id: "claude-code",
    detected: Boolean(executablePath) || existsSync(claudeConfigPath),
    configured,
    supportsAdapter: true,
    configPath: claudeConfigPath,
    executablePath,
    backupPath,
    message: configured
      ? "Claude Code is already configured with the local Clash for AI gateway variables."
      : "Claude Code can be configured by writing env overrides into ~/.claude/settings.json."
  };
}

async function inspectStaticIntegration(toolId: ToolIntegrationId): Promise<ToolIntegrationState> {
  return {
    id: toolId,
    detected: false,
    configured: false,
    supportsAdapter: false
  };
}

async function applyCodexIntegration(apiPort: number): Promise<ToolIntegrationState> {
  const nextContent = buildCodexConfig(await readOptionalText(codexConfigPath), apiPort);
  const backupPath = await backupCodexFiles();
  const nextAuth = buildCodexAuth(await readOptionalText(codexAuthPath));

  await mkdir(dirname(codexConfigPath), { recursive: true });
  await writeFile(codexConfigPath, nextContent, "utf8");
  await writeFile(codexAuthPath, `${JSON.stringify(nextAuth, null, 2)}\n`, "utf8");

  const state = await inspectCodexIntegration(apiPort);
  return {
    ...state,
    backupPath,
    message: "Configured Codex CLI to use the local Clash for AI gateway."
  };
}

async function applyClaudeIntegration(apiPort: number): Promise<ToolIntegrationState> {
  const backupPath = await backupIfExists("claude-code", claudeConfigPath);
  const currentRaw = await readOptionalText(claudeConfigPath);
  const parsed = currentRaw ? parseJsonObject(currentRaw) : {};
  const activeProviderModelMap = await fetchActiveClaudeCodeModelMap(apiPort);
  const nextEnv = {
    ...(isPlainObject(parsed.env) ? parsed.env : {}),
    ANTHROPIC_BASE_URL: `http://127.0.0.1:${apiPort}`,
    ANTHROPIC_AUTH_TOKEN: "dummy"
  };

  syncClaudeCodeModelEnv(nextEnv, activeProviderModelMap);
  const next = {
    ...parsed,
    env: nextEnv
  };

  await mkdir(dirname(claudeConfigPath), { recursive: true });
  await writeFile(claudeConfigPath, `${JSON.stringify(next, null, 2)}\n`, "utf8");

  const state = await inspectClaudeIntegration(apiPort);
  return {
    ...state,
    backupPath,
    message: "Configured Claude Code to use the local Clash for AI gateway and synced the active provider model slots."
  };
}

async function fetchActiveClaudeCodeModelMap(apiPort: number) {
  const response = await fetch(`http://127.0.0.1:${apiPort}/api/providers`);
  if (!response.ok) {
    throw new Error(`Failed to load providers from local core with ${response.status}.`);
  }

  const providers = (await response.json()) as Array<{
    status?: { is_active?: boolean };
    claude_code_model_map?: { opus?: string; sonnet?: string; haiku?: string };
  }>;
  const activeProvider = providers.find((item) => item.status?.is_active);
  const modelMap = activeProvider?.claude_code_model_map;

  return {
    opus: typeof modelMap?.opus === "string" ? modelMap.opus.trim() : "",
    sonnet: typeof modelMap?.sonnet === "string" ? modelMap.sonnet.trim() : "",
    haiku: typeof modelMap?.haiku === "string" ? modelMap.haiku.trim() : ""
  };
}

function syncClaudeCodeModelEnv(
  env: Record<string, any>,
  modelMap: { opus: string; sonnet: string; haiku: string }
) {
  assignOrDelete(env, "ANTHROPIC_MODEL", modelMap.sonnet);
  assignOrDelete(env, "ANTHROPIC_DEFAULT_OPUS_MODEL", modelMap.opus);
  assignOrDelete(env, "ANTHROPIC_DEFAULT_SONNET_MODEL", modelMap.sonnet);
  assignOrDelete(env, "ANTHROPIC_DEFAULT_HAIKU_MODEL", modelMap.haiku);
}

function assignOrDelete(env: Record<string, any>, key: string, value: string) {
  if (value) {
    env[key] = value;
    return;
  }

  delete env[key];
}

function buildCodexConfig(existingContent: string, apiPort: number) {
  const lines = existingContent.trim().length > 0 ? existingContent.replace(/\s+$/, "").split(/\r?\n/) : [];
  const nextModelProviderLine = 'model_provider = "OpenAI"';
  let replacedTopLevelProvider = false;
  let currentTable: string | null = null;
  let inOpenAISection = false;
  let replacedBaseUrl = false;
  const rewritten = lines.map((line) => {
    const trimmed = line.trim();
    if (trimmed.startsWith("[") && trimmed.endsWith("]")) {
      currentTable = trimmed.slice(1, -1).trim();
      inOpenAISection = currentTable === "model_providers.OpenAI";
      return line;
    }

    if (!trimmed || trimmed.startsWith("#")) {
      return line;
    }

    if (currentTable == null && /^model_provider\s*=/.test(trimmed)) {
      replacedTopLevelProvider = true;
      return nextModelProviderLine;
    }

    if (inOpenAISection && /^base_url\s*=/.test(trimmed)) {
      replacedBaseUrl = true;
      return `base_url = "http://127.0.0.1:${apiPort}/v1"`;
    }

    return line;
  });

  if (!replacedTopLevelProvider) {
    const firstTableIndex = rewritten.findIndex((line) => {
      const trimmed = line.trim();
      return trimmed.startsWith("[") && trimmed.endsWith("]");
    });

    if (firstTableIndex === -1) {
      rewritten.push(nextModelProviderLine);
    } else {
      rewritten.splice(firstTableIndex, 0, nextModelProviderLine, "");
    }
  }

  const hasOpenAISection = rewritten.some((line) => line.trim() === "[model_providers.OpenAI]");
  const nextLines = [...rewritten];

  if (!hasOpenAISection) {
    nextLines.push(
      "",
      "[model_providers.OpenAI]",
      'name = "OpenAI"',
      `base_url = "http://127.0.0.1:${apiPort}/v1"`,
      'wire_api = "responses"',
      "requires_openai_auth = true"
    );
  } else if (!replacedBaseUrl) {
    const sectionIndex = nextLines.findIndex((line) => line.trim() === "[model_providers.OpenAI]");
    if (sectionIndex >= 0) {
      let insertIndex = sectionIndex + 1;
      while (insertIndex < nextLines.length) {
        const trimmed = nextLines[insertIndex].trim();
        if (trimmed.startsWith("[") && trimmed.endsWith("]")) {
          break;
        }
        insertIndex += 1;
      }
      nextLines.splice(insertIndex, 0, `base_url = "http://127.0.0.1:${apiPort}/v1"`);
    }
  }

  return `${nextLines.join("\n").replace(/\s+$/, "")}\n`;
}

function buildCodexAuth(existingContent: string) {
  const parsed = existingContent ? parseJsonObject(existingContent) : {};
  return {
    ...parsed,
    OPENAI_API_KEY: "dummy"
  };
}

async function isCodexConfigured(apiPort: number) {
  const [content, authContent] = await Promise.all([
    readOptionalText(codexConfigPath),
    readOptionalText(codexAuthPath)
  ]);
  if (!content || !authContent) {
    return false;
  }

  const topLevelProvider = readTopLevelTomlValue(content, "model_provider");
  const auth = parseJsonObject(authContent);
  return (
    topLevelProvider === "OpenAI" &&
    content.includes("[model_providers.OpenAI]") &&
    content.includes(`base_url = "http://127.0.0.1:${apiPort}/v1"`) &&
    auth.OPENAI_API_KEY === "dummy"
  );
}

async function isClaudeConfigured(apiPort: number) {
  const content = await readOptionalText(claudeConfigPath);
  if (!content) {
    return false;
  }

  const parsed = parseJsonObject(content);
  if (!isPlainObject(parsed.env)) {
    return false;
  }

  return (
    parsed.env.ANTHROPIC_BASE_URL === `http://127.0.0.1:${apiPort}` &&
    parsed.env.ANTHROPIC_AUTH_TOKEN === "dummy"
  );
}

function isClaudeConfiguredFromRaw(raw: string, apiPort: number) {
  const parsed = parseJsonObject(raw);
  if (!isPlainObject(parsed.env)) {
    return false;
  }

  return (
    parsed.env.ANTHROPIC_BASE_URL === `http://127.0.0.1:${apiPort}` &&
    parsed.env.ANTHROPIC_AUTH_TOKEN === "dummy"
  );
}

async function backupIfExists(toolId: ToolIntegrationId, filePath: string) {
  if (!existsSync(filePath)) {
    return undefined;
  }

  const stamp = new Date().toISOString().replace(/[:.]/g, "-");
  const backupDir = join(app.getPath("userData"), "tool-backups", toolId);
  const backupPath = join(backupDir, `${stamp}-${basename(filePath) || "config.backup"}`);
  const content = await readFile(filePath);

  await mkdir(backupDir, { recursive: true });
  await writeFile(backupPath, content);
  return backupPath;
}

async function backupCodexFiles() {
  if (!existsSync(codexConfigPath) && !existsSync(codexAuthPath)) {
    return undefined;
  }

  const stamp = new Date().toISOString().replace(/[:.]/g, "-");
  const backupDir = join(app.getPath("userData"), "tool-backups", "codex-cli", stamp);
  await mkdir(backupDir, { recursive: true });

  if (existsSync(codexConfigPath)) {
    await writeFile(join(backupDir, "config.toml"), await readFile(codexConfigPath));
  }

  if (existsSync(codexAuthPath)) {
    await writeFile(join(backupDir, "auth.json"), await readFile(codexAuthPath));
  }

  return backupDir;
}

async function restoreCodexIntegration(apiPort: number) {
  const backupPath = await getPreferredCodexBackupPath(apiPort);
  if (!backupPath) {
    throw new Error("No backup file is available to restore.");
  }

  const backupName = basename(backupPath);
  if (backupName.endsWith(".toml") || backupName.endsWith(".json")) {
    const content = await readFile(backupPath);
    await mkdir(dirname(codexConfigPath), { recursive: true });
    await writeFile(codexConfigPath, content);
  } else {
    const configBackup = join(backupPath, "config.toml");
    const authBackup = join(backupPath, "auth.json");

    await mkdir(dirname(codexConfigPath), { recursive: true });

    if (existsSync(configBackup)) {
      await writeFile(codexConfigPath, await readFile(configBackup));
    }

    if (existsSync(authBackup)) {
      await writeFile(codexAuthPath, await readFile(authBackup));
    }
  }

  const state = await inspectCodexIntegration(apiPort);
  return {
    ...state,
    backupPath,
    message: "Restored the most recent backup for codex-cli."
  };
}

async function restoreClaudeIntegration(apiPort: number) {
  const backupPath = await getPreferredClaudeBackupPath(apiPort);
  if (!backupPath) {
    throw new Error("No backup file is available to restore.");
  }

  const content = await readFile(backupPath);
  await mkdir(dirname(claudeConfigPath), { recursive: true });
  await writeFile(claudeConfigPath, content);

  const state = await inspectClaudeIntegration(apiPort);
  return {
    ...state,
    backupPath,
    message: "Restored the most recent backup for claude-code."
  };
}

async function getLatestBackupPath(toolId: ToolIntegrationId) {
  const backupDir = join(app.getPath("userData"), "tool-backups", toolId);
  try {
    const entries = (await readdir(backupDir)).filter(Boolean).sort().reverse();
    const latest = entries[0];
    return latest ? join(backupDir, latest) : undefined;
  } catch {
    return undefined;
  }
}

async function getPreferredCodexBackupPath(apiPort: number) {
  const backupDir = join(app.getPath("userData"), "tool-backups", "codex-cli");
  try {
    const entries = (await readdir(backupDir)).filter(Boolean).sort().reverse();
    let fallbackPath: string | undefined;

    for (const entry of entries) {
      const fullPath = join(backupDir, entry);
      const configPath = join(fullPath, "config.toml");
      const authPath = join(fullPath, "auth.json");

      if (!fallbackPath) {
        fallbackPath = fullPath;
      }

      if (!existsSync(configPath) || !existsSync(authPath)) {
        continue;
      }

      const [configContent, authContent] = await Promise.all([
        readFile(configPath, "utf8"),
        readFile(authPath, "utf8")
      ]);
      const auth = parseJsonObject(authContent);
      const configured =
        readTopLevelTomlValue(configContent, "model_provider") === "OpenAI" &&
        configContent.includes(`base_url = "http://127.0.0.1:${apiPort}/v1"`) &&
        auth.OPENAI_API_KEY === "dummy";

      if (!configured) {
        return fullPath;
      }
    }

    return fallbackPath;
  } catch {
    return undefined;
  }
}

async function getPreferredClaudeBackupPath(apiPort: number) {
  const backupDir = join(app.getPath("userData"), "tool-backups", "claude-code");
  try {
    const entries = (await readdir(backupDir)).filter(Boolean).sort().reverse();
    let fallbackPath: string | undefined;

    for (const entry of entries) {
      const fullPath = join(backupDir, entry);
      if (!fallbackPath) {
        fallbackPath = fullPath;
      }

      const content = await readFile(fullPath, "utf8");
      if (!isClaudeConfiguredFromRaw(content, apiPort)) {
        return fullPath;
      }
    }

    return fallbackPath;
  } catch {
    return undefined;
  }
}

async function readOptionalText(filePath: string) {
  try {
    return await readFile(filePath, "utf8");
  } catch {
    return "";
  }
}

function parseJsonObject(raw: string): Record<string, any> {
  try {
    const parsed = JSON.parse(raw);
    return isPlainObject(parsed) ? parsed : {};
  } catch {
    return {};
  }
}

function isPlainObject(value: unknown): value is Record<string, any> {
  return typeof value === "object" && value != null && !Array.isArray(value);
}

function readTopLevelTomlValue(content: string, key: string) {
  let currentTable: string | null = null;
  for (const line of content.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (trimmed.startsWith("[") && trimmed.endsWith("]")) {
      currentTable = trimmed.slice(1, -1).trim();
      continue;
    }

    if (currentTable != null || !trimmed || trimmed.startsWith("#")) {
      continue;
    }

    const match = trimmed.match(new RegExp(`^${key}\\s*=\\s*"([^"]+)"\\s*$`));
    if (match) {
      return match[1];
    }
  }

  return undefined;
}

function resolveExecutable(command: string) {
  const lookupCommand = process.platform === "win32" ? "where" : "which";
  const result = spawnSync(lookupCommand, [command], { encoding: "utf8" });
  if (result.status !== 0) {
    return undefined;
  }

  const firstLine = result.stdout
    .split(/\r?\n/)
    .map((line) => line.trim())
    .find(Boolean);
  if (!firstLine) {
    return undefined;
  }

  try {
    const metadata = existsSync(firstLine) ? true : false;
    if (!metadata) {
      return undefined;
    }
  } catch {
    return undefined;
  }

  return firstLine;
}
