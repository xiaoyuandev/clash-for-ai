import { useEffect, useMemo, useState } from "react";

type UpdateState = {
  currentVersion: string;
  status:
    | "idle"
    | "checking"
    | "available"
    | "not-available"
    | "downloading"
    | "downloaded"
    | "error"
    | "unsupported";
  availableVersion?: string;
  downloadedVersion?: string;
  progressPercent?: number;
  message?: string;
};

type DesktopState = {
  ok: boolean;
  runtime: string;
  platform: string;
  apiBase: string;
  config: {
    apiPort: number;
    apiPortSource: "default" | "config" | "env";
  };
  updates: UpdateState;
  core: {
    managed: boolean;
    running: boolean;
    apiBase: string;
    port: number;
    pid?: number;
    logRetentionDays: number;
    logMaxRecords: number;
    lastError?: string;
    command?: string;
  };
} | null;

type ToolPreset =
  | "codex-cli"
  | "claude-code"
  | "cursor"
  | "cherry-studio"
  | "open-code"
  | "openai-sdk";
type PlatformPreset = "unix" | "windows-cmd" | "powershell";
type ConnectMode = "command" | "manual";

interface SettingsPageProps {
  desktopState: DesktopState;
  onCheckUpdates: () => Promise<void>;
  onDownloadUpdate: () => Promise<void>;
  onQuitAndInstallUpdate: () => Promise<void>;
  onCoreRestart: () => Promise<void>;
  onUpdateCorePort: (port: number) => Promise<void>;
  onCopyText: (text: string) => Promise<void>;
}

export function SettingsPage({
  desktopState,
  onCheckUpdates,
  onDownloadUpdate,
  onQuitAndInstallUpdate,
  onCoreRestart,
  onUpdateCorePort,
  onCopyText
}: SettingsPageProps) {
  const [busy, setBusy] = useState(false);
  const [updateBusy, setUpdateBusy] = useState(false);
  const [saveBusy, setSaveBusy] = useState(false);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [portInput, setPortInput] = useState(String(desktopState?.config.apiPort ?? 3456));
  const [connectOpen, setConnectOpen] = useState(false);
  const [toolPreset, setToolPreset] = useState<ToolPreset>("codex-cli");
  const [platformPreset, setPlatformPreset] = useState<PlatformPreset>("unix");
  const [copyFeedback, setCopyFeedback] = useState<string | null>(null);
  const [manualCopyFeedback, setManualCopyFeedback] = useState<string | null>(null);
  const [connectMode, setConnectMode] = useState<ConnectMode>("command");

  useEffect(() => {
    setPortInput(String(desktopState?.config.apiPort ?? 3456));
  }, [desktopState?.config.apiPort]);

  async function handleRestart() {
    setBusy(true);
    setFeedback(null);

    try {
      await onCoreRestart();
      setFeedback("Core restarted on the configured fixed port.");
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : "Failed to restart core");
    } finally {
      setBusy(false);
    }
  }

  async function handleSavePort() {
    const nextPort = Number(portInput);
    if (!Number.isInteger(nextPort) || nextPort < 1 || nextPort > 65535) {
      setFeedback("Port must be an integer between 1 and 65535.");
      return;
    }

    setSaveBusy(true);
    setFeedback(null);

    try {
      await onUpdateCorePort(nextPort);
      setFeedback(`Fixed port updated to ${nextPort}. Core restarted.`);
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : "Failed to update core port");
    } finally {
      setSaveBusy(false);
    }
  }

  async function handleCheckUpdates() {
    setUpdateBusy(true);
    setFeedback(null);

    try {
      await onCheckUpdates();
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : "Failed to check updates");
    } finally {
      setUpdateBusy(false);
    }
  }

  async function handleDownloadUpdate() {
    setUpdateBusy(true);
    setFeedback(null);

    try {
      await onDownloadUpdate();
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : "Failed to download update");
    } finally {
      setUpdateBusy(false);
    }
  }

  async function handleQuitAndInstallUpdate() {
    setUpdateBusy(true);
    setFeedback(null);

    try {
      await onQuitAndInstallUpdate();
    } catch (error) {
      setFeedback(
        error instanceof Error ? error.message : "Failed to install downloaded update"
      );
      setUpdateBusy(false);
    }
  }

  const connectionGuide = useMemo(() => {
    const port = desktopState?.core.port ?? desktopState?.config.apiPort ?? 3456;
    const openAIBase = `http://127.0.0.1:${port}/v1`;
    const anthropicBase = `http://127.0.0.1:${port}`;

    const platformLabel =
      platformPreset === "unix"
        ? "macOS / Linux / WSL"
        : platformPreset === "windows-cmd"
          ? "Windows CMD"
          : "PowerShell";

    const formatEnvCommands = (pairs: Array<[string, string]>) => {
      switch (platformPreset) {
        case "windows-cmd":
          return pairs.map(([key, value]) => `set ${key}=${value}`).join("\n");
        case "powershell":
          return pairs
            .map(([key, value]) => `$env:${key}="${value}"`)
            .join("\n");
        case "unix":
        default:
          return pairs
            .map(([key, value]) => `export ${key}="${value}"`)
            .join("\n");
      }
    };

    const openAICommand = formatEnvCommands([
      ["OPENAI_BASE_URL", openAIBase],
      ["OPENAI_API_KEY", "dummy"]
    ]);
    const anthropicCommand = formatEnvCommands([
      ["ANTHROPIC_BASE_URL", anthropicBase],
      ["ANTHROPIC_AUTH_TOKEN", "dummy"]
    ]);

    const toolMetadata: Record<
      ToolPreset,
      {
        title: string;
        summary: string;
        command: string;
        note: string;
        supportsManual: boolean;
        manualTitle?: string;
        manualItems?: Array<{ label: string; value: string; hint?: string }>;
      }
    > = {
      "codex-cli": {
        title: `Codex CLI + ${platformLabel}`,
        summary: "Set OpenAI-compatible environment variables in the current shell before launching Codex CLI.",
        command: openAICommand,
        note: "These commands only affect the current terminal session. Codex CLI can stay on one stable local /v1 endpoint while Clash for AI rotates the upstream provider.",
        supportsManual: false
      },
      "claude-code": {
        title: `Claude Code + ${platformLabel}`,
        summary: "Set Anthropic-style gateway variables in the current shell before launching Claude Code.",
        command: anthropicCommand,
        note: "These commands only affect the current terminal session. Use the local root URL without /v1 here. Clash for AI will forward the Anthropic-style requests upstream.",
        supportsManual: false
      },
      cursor: {
        title: `Cursor + ${platformLabel}`,
        summary: "Prepare the OpenAI-compatible endpoint values you can paste into Cursor's custom provider fields.",
        command: openAICommand,
        note: "Cursor primarily uses in-app API key and Base URL settings. These session-only commands are still useful if you launch Cursor from a shell or want the values ready to paste.",
        supportsManual: true,
        manualTitle: "Cursor In-App Fields",
        manualItems: [
          { label: "Provider Type", value: "OpenAI Compatible" },
          { label: "Base URL", value: openAIBase },
          { label: "API Key", value: "dummy", hint: "Any non-empty string is fine." }
        ]
      },
      "cherry-studio": {
        title: `Cherry Studio + ${platformLabel}`,
        summary: "Prepare the OpenAI-compatible endpoint values for Cherry Studio provider setup.",
        command: openAICommand,
        note: "Cherry Studio still expects you to enter Base URL, API Key, and model names in its provider UI. The copied values match that UI, and the shell commands are session-only.",
        supportsManual: true,
        manualTitle: "Cherry Studio In-App Fields",
        manualItems: [
          { label: "Provider Protocol", value: "OpenAI Compatible" },
          { label: "Base URL", value: openAIBase },
          { label: "API Key", value: "dummy", hint: "Any non-empty string is fine." }
        ]
      },
      "open-code": {
        title: `Open Code + ${platformLabel}`,
        summary: "Set standard OpenAI environment variables in the current shell for Open Code style CLIs.",
        command: openAICommand,
        note: "These commands only affect the current terminal session. This preset assumes the tool reads standard OPENAI-compatible variables from the current shell.",
        supportsManual: false
      },
      "openai-sdk": {
        title: `OpenAI SDK + ${platformLabel}`,
        summary: "Set process-level defaults in the current shell so OpenAI-compatible SDK clients can reuse the local gateway.",
        command: openAICommand,
        note: "These commands only affect the current terminal session. Useful when your app or scripts rely on OPENAI_BASE_URL and OPENAI_API_KEY instead of hardcoding values in source.",
        supportsManual: false
      }
    };

    return toolMetadata[toolPreset];
  }, [desktopState?.config.apiPort, desktopState?.core.port, platformPreset, toolPreset]);

  async function handleCopyCommand(text: string, title: string) {
    try {
      await onCopyText(text);
      setCopyFeedback("Copied");
      setFeedback(`${title} command copied.`);
      window.setTimeout(() => {
        setCopyFeedback((current) => (current === "Copied" ? null : current));
      }, 1500);
    } catch (error) {
      setCopyFeedback("Failed");
      setFeedback(error instanceof Error ? error.message : "Failed to copy command");
      window.setTimeout(() => {
        setCopyFeedback((current) => (current === "Failed" ? null : current));
      }, 2000);
    }
  }

  useEffect(() => {
    setConnectMode("command");
    setCopyFeedback(null);
    setManualCopyFeedback(null);
  }, [toolPreset, platformPreset]);

  async function handleCopyValue(text: string, label: string) {
    try {
      await onCopyText(text);
      setManualCopyFeedback(label);
      setFeedback(`${label} copied.`);
      window.setTimeout(() => {
        setManualCopyFeedback((current) => (current === label ? null : current));
      }, 1500);
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : "Failed to copy value");
    }
  }

  const portLocked = desktopState?.config.apiPortSource === "env";

  return (
    <main className="page-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Clash for AI</p>
          <h1>Settings</h1>
          <p className="subcopy">
            Fix the local port once, then generate copy-ready connection commands for each coding tool.
          </p>
        </div>
      </section>

      {feedback ? <p className="panel info-panel">{feedback}</p> : null}

      <section className="panel">
        <div className="section-head">
          <h2>Connection</h2>
          <span>{desktopState?.config.apiPortSource ?? "default"}</span>
        </div>

        <div className="settings-grid">
          <div className="settings-card">
            <p className="settings-label">Fixed Port</p>
            <input
              className="settings-input"
              value={portInput}
              disabled={portLocked || saveBusy}
              onChange={(event) => setPortInput(event.target.value)}
              inputMode="numeric"
            />
            <p className="meta">
              No fallback ports. If this port is occupied by another process, startup now fails with a clear error.
            </p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Connected API Base</p>
            <p className="mono">{desktopState?.apiBase ?? "-"}</p>
            <p className="meta">Use this value when a tool expects an OpenAI-compatible base URL.</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Port Source</p>
            <p className="mono">{desktopState?.config.apiPortSource ?? "-"}</p>
            <p className="meta">
              {portLocked
                ? "ELECTRON_API_PORT is set, so the desktop UI keeps the value read-only."
                : "Stored in the desktop app config and reused on every launch."}
            </p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Launch Flow</p>
            <p className="mono">
              {desktopState?.core.managed ? "managed core" : "reuse existing instance"}
            </p>
            <p className="meta">
              A second app launch focuses the existing window instead of starting a duplicate instance.
            </p>
          </div>
        </div>

        <div className="settings-actions settings-actions-split">
          <button
            type="button"
            className="secondary-button"
            onClick={() => setConnectOpen(true)}
          >
            Connect a Tool
          </button>
          <div className="settings-action-row">
            <button
              type="button"
              className="secondary-button"
              onClick={() => void handleRestart()}
              disabled={busy}
            >
              {busy ? "Restarting..." : "Restart Core"}
            </button>
            <button
              type="button"
              onClick={() => void handleSavePort()}
              disabled={portLocked || saveBusy}
            >
              {saveBusy ? "Saving..." : "Save Port and Restart"}
            </button>
          </div>
        </div>

        {desktopState?.core.lastError ? (
          <p className="log-error">
            <span className="mono">{desktopState.core.lastError}</span>
          </p>
        ) : null}
      </section>

      <section className="panel">
        <div className="section-head">
          <h2>Runtime</h2>
          <span>{desktopState?.runtime ?? "unknown"}</span>
        </div>

        <div className="settings-grid">
          <div className="settings-card">
            <p className="settings-label">Platform</p>
            <p className="mono">{desktopState?.platform ?? "-"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Core Port</p>
            <p className="mono">{desktopState?.core.port ?? "-"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Core PID</p>
            <p className="mono">{desktopState?.core.pid ?? "-"}</p>
            <p className="meta">
              Restart now uses the recorded PID when the desktop app reattaches to an existing managed core.
            </p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Core Managed</p>
            <p className="mono">{desktopState?.core.managed ? "yes" : "no"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Core Running</p>
            <p className="mono">{desktopState?.core.running ? "yes" : "no"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Launch Command</p>
            <p className="mono">{desktopState?.core.command ?? "-"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Log Retention</p>
            <p className="mono">
              {desktopState
                ? `${desktopState.core.logRetentionDays} days / ${desktopState.core.logMaxRecords.toLocaleString()} records`
                : "-"}
            </p>
            <p className="meta">
              Request logs are pruned automatically to keep SQLite size under control.
            </p>
          </div>
        </div>
      </section>

      <section className="panel">
        <div className="section-head">
          <h2>Updates</h2>
          <span>{desktopState?.updates.currentVersion ?? "-"}</span>
        </div>

        <div className="settings-grid">
          <div className="settings-card">
            <p className="settings-label">Current Version</p>
            <p className="mono">{desktopState?.updates.currentVersion ?? "-"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Update Status</p>
            <p className="mono">{desktopState?.updates.status ?? "-"}</p>
            {desktopState?.updates.message ? (
              <p className="meta">{desktopState.updates.message}</p>
            ) : null}
          </div>
          <div className="settings-card">
            <p className="settings-label">Available Version</p>
            <p className="mono">
              {desktopState?.updates.availableVersion ??
                desktopState?.updates.downloadedVersion ??
                "-"}
            </p>
          </div>
        </div>

        <div className="settings-actions">
          <button
            type="button"
            className="secondary-button"
            onClick={() => void handleCheckUpdates()}
            disabled={updateBusy || desktopState?.updates.status === "unsupported"}
          >
            {updateBusy && desktopState?.updates.status === "checking"
              ? "Checking..."
              : "Check for Updates"}
          </button>
          <button
            type="button"
            className="secondary-button"
            onClick={() => void handleDownloadUpdate()}
            disabled={
              updateBusy ||
              (desktopState?.updates.status !== "available" &&
                desktopState?.updates.status !== "downloading")
            }
          >
            {desktopState?.updates.status === "downloading"
              ? `Downloading ${Math.round(desktopState.updates.progressPercent ?? 0)}%`
              : "Download Update"}
          </button>
          <button
            type="button"
            onClick={() => void handleQuitAndInstallUpdate()}
            disabled={updateBusy || desktopState?.updates.status !== "downloaded"}
          >
            Restart to Install
          </button>
        </div>
      </section>

      {connectOpen ? (
        <div className="modal-backdrop" role="presentation" onClick={() => setConnectOpen(false)}>
          <section
            className="modal-panel"
            role="dialog"
            aria-modal="true"
            aria-label="Connect a coding tool"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="section-head">
              <div>
                <h2>Connect a Tool</h2>
                <p className="meta">{connectionGuide.summary}</p>
              </div>
              <button
                type="button"
                className="secondary-button"
                onClick={() => setConnectOpen(false)}
              >
                Close
              </button>
            </div>

            <div className="connect-grid">
              <label>
                <span>Tool</span>
                <select
                  className="settings-input"
                  value={toolPreset}
                  onChange={(event) => setToolPreset(event.target.value as ToolPreset)}
                >
                  <option value="codex-cli">Codex CLI</option>
                  <option value="claude-code">Claude Code</option>
                  <option value="cursor">Cursor</option>
                  <option value="cherry-studio">Cherry Studio</option>
                  <option value="open-code">Open Code</option>
                  <option value="openai-sdk">OpenAI SDK</option>
                </select>
              </label>
              <label>
                <span>Shell / Platform</span>
                <select
                  className="settings-input"
                  value={platformPreset}
                  onChange={(event) =>
                    setPlatformPreset(event.target.value as PlatformPreset)
                  }
                >
                  <option value="unix">macOS / Linux / WSL</option>
                  <option value="windows-cmd">Windows CMD</option>
                  <option value="powershell">PowerShell</option>
                </select>
              </label>
            </div>

            {connectionGuide.supportsManual ? (
              <div className="mode-switch">
                <button
                  type="button"
                  className={connectMode === "command" ? "mode-tab active-mode-tab" : "mode-tab"}
                  onClick={() => setConnectMode("command")}
                >
                  Terminal Command
                </button>
                <button
                  type="button"
                  className={connectMode === "manual" ? "mode-tab active-mode-tab" : "mode-tab"}
                  onClick={() => setConnectMode("manual")}
                >
                  In-App Fields
                </button>
              </div>
            ) : null}

            {connectMode === "command" || !connectionGuide.supportsManual ? (
              <div className="settings-card connect-command-card">
                <div className="command-card-head">
                  <p className="settings-label">{connectionGuide.title}</p>
                  <button
                    type="button"
                    className={`icon-button ${copyFeedback === "Copied" ? "icon-button-success" : ""}`}
                    aria-label="Copy command"
                    title="Copy command"
                    onClick={() => {
                      void handleCopyCommand(connectionGuide.command, connectionGuide.title);
                    }}
                  >
                    {copyFeedback === "Copied" ? (
                      <svg viewBox="0 0 24 24" aria-hidden="true">
                        <path d="M9.2 16.6 4.9 12.3l1.4-1.4 2.9 2.9 8.5-8.5 1.4 1.4z" />
                      </svg>
                    ) : (
                      <svg viewBox="0 0 24 24" aria-hidden="true">
                        <path d="M9 9h10v10H9z" />
                        <path d="M5 5h10v2H7v8H5z" />
                      </svg>
                    )}
                  </button>
                </div>
                <pre className="command-preview">
                  <code>{connectionGuide.command}</code>
                </pre>
                {copyFeedback ? <p className="copy-feedback">{copyFeedback}</p> : null}
                <p className="meta">{connectionGuide.note}</p>
              </div>
            ) : (
              <div className="settings-card connect-command-card">
                <div className="command-card-head">
                  <p className="settings-label">{connectionGuide.manualTitle}</p>
                </div>
                <div className="manual-grid">
                  {connectionGuide.manualItems?.map((item) => (
                    <div key={item.label} className="manual-item">
                      <div className="manual-item-head">
                        <p className="settings-label">{item.label}</p>
                        <button
                          type="button"
                          className={`icon-button icon-button-small ${manualCopyFeedback === item.label ? "icon-button-success" : ""}`}
                          aria-label={`Copy ${item.label}`}
                          title={`Copy ${item.label}`}
                          onClick={() => {
                            void handleCopyValue(item.value, item.label);
                          }}
                        >
                          {manualCopyFeedback === item.label ? (
                            <svg viewBox="0 0 24 24" aria-hidden="true">
                              <path d="M9.2 16.6 4.9 12.3l1.4-1.4 2.9 2.9 8.5-8.5 1.4 1.4z" />
                            </svg>
                          ) : (
                            <svg viewBox="0 0 24 24" aria-hidden="true">
                              <path d="M9 9h10v10H9z" />
                              <path d="M5 5h10v2H7v8H5z" />
                            </svg>
                          )}
                        </button>
                      </div>
                      <p className="mono">{item.value}</p>
                      {item.hint ? <p className="meta">{item.hint}</p> : null}
                    </div>
                  ))}
                </div>
                <p className="meta">{connectionGuide.note}</p>
              </div>
            )}
          </section>
        </div>
      ) : null}
    </main>
  );
}
