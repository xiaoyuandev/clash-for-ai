import { useEffect, useMemo, useState } from "react";
import { useI18n } from "../i18n/i18n-provider";

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
  const { t } = useI18n();
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
      setFeedback(t("settings.feedback.coreRestarted"));
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : t("settings.feedback.restartFailed"));
    } finally {
      setBusy(false);
    }
  }

  async function handleSavePort() {
    const nextPort = Number(portInput);
    if (!Number.isInteger(nextPort) || nextPort < 1 || nextPort > 65535) {
      setFeedback(t("settings.feedback.invalidPort"));
      return;
    }

    setSaveBusy(true);
    setFeedback(null);

    try {
      await onUpdateCorePort(nextPort);
      setFeedback(t("settings.feedback.portUpdated", { port: nextPort }));
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : t("settings.feedback.portUpdateFailed"));
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
      setFeedback(error instanceof Error ? error.message : t("settings.feedback.updateCheckFailed"));
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
      setFeedback(error instanceof Error ? error.message : t("settings.feedback.updateDownloadFailed"));
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
        error instanceof Error ? error.message : t("settings.feedback.updateInstallFailed")
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
        ? t("settings.platform.unix")
        : platformPreset === "windows-cmd"
          ? t("settings.platform.windowsCmd")
          : t("settings.platform.powershell");

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
        title: t("settings.guide.codex.title", { platform: platformLabel }),
        summary: t("settings.guide.codex.summary"),
        command: openAICommand,
        note: t("settings.guide.codex.note"),
        supportsManual: false
      },
      "claude-code": {
        title: t("settings.guide.claude.title", { platform: platformLabel }),
        summary: t("settings.guide.claude.summary"),
        command: anthropicCommand,
        note: t("settings.guide.claude.note"),
        supportsManual: false
      },
      cursor: {
        title: t("settings.guide.cursor.title", { platform: platformLabel }),
        summary: t("settings.guide.cursor.summary"),
        command: openAICommand,
        note: t("settings.guide.cursor.note"),
        supportsManual: true,
        manualTitle: t("settings.guide.cursorManual"),
        manualItems: [
          { label: t("settings.guide.field.providerType"), value: t("settings.value.openaiCompatible") },
          { label: t("settings.guide.field.baseUrl"), value: openAIBase },
          { label: t("settings.guide.field.apiKey"), value: "dummy", hint: t("settings.guide.field.apiKeyHint") }
        ]
      },
      "cherry-studio": {
        title: t("settings.guide.cherry.title", { platform: platformLabel }),
        summary: t("settings.guide.cherry.summary"),
        command: openAICommand,
        note: t("settings.guide.cherry.note"),
        supportsManual: true,
        manualTitle: t("settings.guide.cherryManual"),
        manualItems: [
          { label: t("settings.guide.field.protocol"), value: t("settings.value.openaiCompatible") },
          { label: t("settings.guide.field.baseUrl"), value: openAIBase },
          { label: t("settings.guide.field.apiKey"), value: "dummy", hint: t("settings.guide.field.apiKeyHint") }
        ]
      },
      "open-code": {
        title: t("settings.guide.openCode.title", { platform: platformLabel }),
        summary: t("settings.guide.openCode.summary"),
        command: openAICommand,
        note: t("settings.guide.openCode.note"),
        supportsManual: false
      },
      "openai-sdk": {
        title: t("settings.guide.openaiSdk.title", { platform: platformLabel }),
        summary: t("settings.guide.openaiSdk.summary"),
        command: openAICommand,
        note: t("settings.guide.openaiSdk.note"),
        supportsManual: false
      }
    };

    return toolMetadata[toolPreset];
  }, [desktopState?.config.apiPort, desktopState?.core.port, platformPreset, t, toolPreset]);

  async function handleCopyCommand(text: string, title: string) {
    try {
      await onCopyText(text);
      setCopyFeedback(t("settings.copy.copied"));
      setFeedback(t("settings.feedback.commandCopied", { title }));
      window.setTimeout(() => {
        setCopyFeedback((current) => (current === t("settings.copy.copied") ? null : current));
      }, 1500);
    } catch (error) {
      setCopyFeedback(t("settings.copy.failed"));
      setFeedback(error instanceof Error ? error.message : t("settings.feedback.copyCommandFailed"));
      window.setTimeout(() => {
        setCopyFeedback((current) => (current === t("settings.copy.failed") ? null : current));
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
      setFeedback(t("settings.feedback.valueCopied", { label }));
      window.setTimeout(() => {
        setManualCopyFeedback((current) => (current === label ? null : current));
      }, 1500);
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : t("settings.feedback.copyValueFailed"));
    }
  }

  const portLocked = desktopState?.config.apiPortSource === "env";

  return (
    <main className="page-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Clash for AI</p>
          <h1>{t("settings.title")}</h1>
          <p className="subcopy">{t("settings.subtitle")}</p>
        </div>
      </section>

      {feedback ? <p className="panel info-panel">{feedback}</p> : null}

      <section className="panel">
        <div className="section-head">
          <h2>{t("settings.section.connection")}</h2>
          <span>{desktopState?.config.apiPortSource ?? t("settings.value.default")}</span>
        </div>

        <div className="settings-grid">
          <div className="settings-card">
            <p className="settings-label">{t("settings.fixedPort")}</p>
            <input
              className="settings-input"
              value={portInput}
              disabled={portLocked || saveBusy}
              onChange={(event) => setPortInput(event.target.value)}
              inputMode="numeric"
            />
            <p className="meta">{t("settings.meta.portConflict")}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">{t("settings.connectedApiBase")}</p>
            <p className="mono">{desktopState?.apiBase ?? "-"}</p>
            <p className="meta">{t("settings.meta.apiBase")}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">{t("settings.portSource")}</p>
            <p className="mono">{desktopState?.config.apiPortSource ?? "-"}</p>
            <p className="meta">
              {portLocked
                ? t("settings.meta.envLocked")
                : t("settings.meta.configStored")}
            </p>
          </div>
          <div className="settings-card">
            <p className="settings-label">{t("settings.launchFlow")}</p>
            <p className="mono">
              {desktopState?.core.managed ? t("settings.flow.managed") : t("settings.flow.reuse")}
            </p>
            <p className="meta">{t("settings.meta.managedLaunch")}</p>
          </div>
        </div>

        <div className="settings-actions settings-actions-split">
          <button
            type="button"
            className="secondary-button"
            onClick={() => setConnectOpen(true)}
          >
            {t("settings.button.connectTool")}
          </button>
          <div className="settings-action-row">
            <button
              type="button"
              className="secondary-button"
              onClick={() => void handleRestart()}
              disabled={busy}
            >
              {busy ? t("settings.button.restarting") : t("settings.button.restartCore")}
            </button>
            <button
              type="button"
              onClick={() => void handleSavePort()}
              disabled={portLocked || saveBusy}
            >
              {saveBusy ? t("common.saving") : t("settings.button.savePort")}
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
          <h2>{t("settings.section.runtime")}</h2>
          <span>{desktopState?.runtime ?? t("settings.value.unknown")}</span>
        </div>

        <div className="settings-grid">
          <div className="settings-card">
            <p className="settings-label">{t("settings.runtime.platform")}</p>
            <p className="mono">{desktopState?.platform ?? "-"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">{t("settings.runtime.corePort")}</p>
            <p className="mono">{desktopState?.core.port ?? "-"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">{t("settings.runtime.corePid")}</p>
            <p className="mono">{desktopState?.core.pid ?? "-"}</p>
            <p className="meta">{t("settings.meta.pid")}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">{t("settings.runtime.coreManaged")}</p>
            <p className="mono">{desktopState?.core.managed ? t("settings.runtime.yes") : t("settings.runtime.no")}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">{t("settings.runtime.coreRunning")}</p>
            <p className="mono">{desktopState?.core.running ? t("settings.runtime.yes") : t("settings.runtime.no")}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">{t("settings.runtime.launchCommand")}</p>
            <p className="mono">{desktopState?.core.command ?? "-"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">{t("settings.runtime.logRetention")}</p>
            <p className="mono">
              {desktopState
                ? t("settings.runtime.daysRecords", {
                    days: desktopState.core.logRetentionDays,
                    records: desktopState.core.logMaxRecords.toLocaleString()
                  })
                : "-"}
            </p>
            <p className="meta">{t("settings.meta.logRetention")}</p>
          </div>
        </div>
      </section>

      <section className="panel">
        <div className="section-head">
          <h2>{t("settings.section.updates")}</h2>
          <span>{desktopState?.updates.currentVersion ?? "-"}</span>
        </div>

        <div className="settings-grid">
          <div className="settings-card">
            <p className="settings-label">{t("settings.updates.currentVersion")}</p>
            <p className="mono">{desktopState?.updates.currentVersion ?? "-"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">{t("settings.updates.status")}</p>
            <p className="mono">{desktopState?.updates.status ?? "-"}</p>
            {desktopState?.updates.message ? (
              <p className="meta">{desktopState.updates.message}</p>
            ) : null}
          </div>
          <div className="settings-card">
            <p className="settings-label">{t("settings.updates.availableVersion")}</p>
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
              ? t("settings.button.checking")
              : t("settings.button.checkUpdates")}
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
              ? t("settings.button.downloading", {
                  progress: Math.round(desktopState.updates.progressPercent ?? 0)
                })
              : t("settings.button.downloadUpdate")}
          </button>
          <button
            type="button"
            onClick={() => void handleQuitAndInstallUpdate()}
            disabled={updateBusy || desktopState?.updates.status !== "downloaded"}
          >
            {t("settings.button.installUpdate")}
          </button>
        </div>
      </section>

      {connectOpen ? (
        <div className="modal-backdrop" role="presentation" onClick={() => setConnectOpen(false)}>
          <section
            className="modal-panel"
            role="dialog"
            aria-modal="true"
            aria-label={t("settings.modal.aria")}
            onClick={(event) => event.stopPropagation()}
          >
            <div className="section-head">
              <div>
                <h2>{t("settings.modal.title")}</h2>
                <p className="meta">{connectionGuide.summary}</p>
              </div>
              <button
                type="button"
                className="secondary-button"
                onClick={() => setConnectOpen(false)}
              >
                {t("common.close")}
              </button>
            </div>

            <div className="connect-grid">
              <label>
                <span>{t("settings.modal.tool")}</span>
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
                <span>{t("settings.modal.platform")}</span>
                <select
                  className="settings-input"
                  value={platformPreset}
                  onChange={(event) =>
                    setPlatformPreset(event.target.value as PlatformPreset)
                  }
                >
                  <option value="unix">{t("settings.platform.unix")}</option>
                  <option value="windows-cmd">{t("settings.platform.windowsCmd")}</option>
                  <option value="powershell">{t("settings.platform.powershell")}</option>
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
                  {t("settings.modal.mode.command")}
                </button>
                <button
                  type="button"
                  className={connectMode === "manual" ? "mode-tab active-mode-tab" : "mode-tab"}
                  onClick={() => setConnectMode("manual")}
                >
                  {t("settings.modal.mode.manual")}
                </button>
              </div>
            ) : null}

            {connectMode === "command" || !connectionGuide.supportsManual ? (
              <div className="settings-card connect-command-card">
                <div className="command-card-head">
                  <p className="settings-label">{connectionGuide.title}</p>
                  <button
                    type="button"
                    className={`icon-button ${copyFeedback === t("settings.copy.copied") ? "icon-button-success" : ""}`}
                    aria-label={t("settings.copy.command")}
                    title={t("settings.copy.command")}
                    onClick={() => {
                      void handleCopyCommand(connectionGuide.command, connectionGuide.title);
                    }}
                  >
                    {copyFeedback === t("settings.copy.copied") ? (
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
                          aria-label={t("settings.copy.value", { label: item.label })}
                          title={t("settings.copy.value", { label: item.label })}
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
