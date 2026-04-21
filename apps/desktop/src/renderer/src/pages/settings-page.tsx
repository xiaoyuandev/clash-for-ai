import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useI18n } from "../i18n/i18n-provider";
import {
  actionRowClass,
  buttonClass,
  emptyStateClass,
  eyebrowClass,
  fieldLabelClass,
  heroClass,
  heroCopyClass,
  heroTitleClass,
  iconBadgeClass,
  iconButtonClass,
  iconButtonSmallClass,
  infoCardClass,
  inputClass,
  metaClass,
  modalBackdropClass,
  modalPanelClass,
  metricValueClass,
  monoClass,
  pageShellClass,
  sectionCardClass,
  sectionHeadClass,
  sectionMetaClass,
  sectionTitleClass,
  successNoticeClass,
  statusPillClass
} from "../ui";

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

function StatCard({
  label,
  value,
  meta,
  icon
}: {
  label: string;
  value: string;
  meta?: string;
  icon?: ReactNode;
}) {
  return (
    <div className={infoCardClass}>
      <div className="flex items-center gap-3">
        {icon ? <span className={iconBadgeClass}>{icon}</span> : null}
        <p className={fieldLabelClass}>{label}</p>
      </div>
      <p className={metricValueClass}>{value}</p>
      {meta ? <p className={`${metaClass} mt-2`}>{meta}</p> : null}
    </div>
  );
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
          return pairs.map(([key, value]) => `$env:${key}="${value}"`).join("\n");
        case "unix":
        default:
          return pairs.map(([key, value]) => `export ${key}="${value}"`).join("\n");
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
    <main className={pageShellClass}>
      <section className={heroClass}>
        <div className="space-y-4">
          <div>
            <p className={eyebrowClass}>Clash for AI</p>
            <h1 className={heroTitleClass}>{t("settings.title")}</h1>
          </div>
          <p className={heroCopyClass}>{t("settings.subtitle")}</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <span className={statusPillClass(desktopState?.core.running ? "success" : "danger")}>
            {desktopState?.core.running ? t("app.coreRunning") : t("app.coreStopped")}
          </span>
          <span className={statusPillClass("default")}>
            {desktopState?.runtime ?? t("settings.value.unknown")}
          </span>
        </div>
      </section>

      {feedback ? (
        <p className={successNoticeClass}>{feedback}</p>
      ) : null}

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>{t("settings.section.connection")}</h2>
            <p className={sectionMetaClass}>
              {desktopState?.config.apiPortSource ?? t("settings.value.default")}
            </p>
          </div>
        </div>

        <div className="mt-6 grid gap-4 sm:grid-cols-2">
          <div className={infoCardClass}>
            <p className={fieldLabelClass}>{t("settings.fixedPort")}</p>
            <input
              className={`${inputClass} mt-3`}
              value={portInput}
              disabled={portLocked || saveBusy}
              onChange={(event) => setPortInput(event.target.value)}
              inputMode="numeric"
            />
            <p className={`${metaClass} mt-2`}>{t("settings.meta.portConflict")}</p>
          </div>
          <StatCard
            label={t("settings.connectedApiBase")}
            value={desktopState?.apiBase ?? "-"}
            meta={t("settings.meta.apiBase")}
            icon={
              <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                <path d="M12 4a8 8 0 1 0 8 8h-2a6 6 0 1 1-1.8-4.2L13 11h7V4l-2.4 2.4A7.9 7.9 0 0 0 12 4" />
              </svg>
            }
          />
        </div>

        <div className="mt-6 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <button
            type="button"
            className={buttonClass("primary")}
            onClick={() => setConnectOpen(true)}
          >
            {t("settings.button.connectTool")}
          </button>
          <div className={actionRowClass}>
            <button
              type="button"
              className={buttonClass("secondary")}
              onClick={() => void handleRestart()}
              disabled={busy}
            >
              {busy ? t("settings.button.restarting") : t("settings.button.restartCore")}
            </button>
            <button
              type="button"
              className={buttonClass("primary")}
              onClick={() => void handleSavePort()}
              disabled={portLocked || saveBusy}
            >
              {saveBusy ? t("common.saving") : t("settings.button.savePort")}
            </button>
          </div>
        </div>

        {desktopState?.core.lastError ? (
          <p className="mt-5 rounded-2xl border [border-color:var(--danger-border)] [background:var(--danger-soft)] px-4 py-3 text-sm text-[color:var(--danger-text)]">
            <span className={monoClass}>{desktopState.core.lastError}</span>
          </p>
        ) : null}
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>{t("settings.section.runtime")}</h2>
            <p className={sectionMetaClass}>{desktopState?.runtime ?? t("settings.value.unknown")}</p>
          </div>
        </div>

        <div className="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <StatCard
            label={t("settings.runtime.platform")}
            value={desktopState?.platform ?? "-"}
            icon={
              <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                <path d="M4 5h16v11H4zm2 2v7h12V7zm-2 11h16v2H4z" />
              </svg>
            }
          />
          <StatCard
            label={t("settings.runtime.corePort")}
            value={String(desktopState?.core.port ?? "-")}
            icon={
              <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                <path d="M7 7h10v10H7zm2 2v6h6V9zM4 11h2v2H4zm14 0h2v2h-2zM11 4h2v2h-2zm0 14h2v2h-2z" />
              </svg>
            }
          />
          <StatCard
            label={t("settings.runtime.corePid")}
            value={String(desktopState?.core.pid ?? "-")}
            meta={t("settings.meta.pid")}
          />
          <StatCard
            label={t("settings.runtime.coreManaged")}
            value={desktopState?.core.managed ? t("settings.runtime.yes") : t("settings.runtime.no")}
          />
          <StatCard
            label={t("settings.runtime.coreRunning")}
            value={desktopState?.core.running ? t("settings.runtime.yes") : t("settings.runtime.no")}
          />
          <StatCard
            label={t("settings.runtime.logRetention")}
            value={
              desktopState
                ? t("settings.runtime.daysRecords", {
                    days: desktopState.core.logRetentionDays,
                    records: desktopState.core.logMaxRecords.toLocaleString()
                  })
                : "-"
            }
            meta={t("settings.meta.logRetention")}
          />
        </div>
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>{t("settings.section.updates")}</h2>
            <p className={sectionMetaClass}>{desktopState?.updates.currentVersion ?? "-"}</p>
          </div>
        </div>

        <div className="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <StatCard
            label={t("settings.updates.currentVersion")}
            value={desktopState?.updates.currentVersion ?? "-"}
          />
          <StatCard
            label={t("settings.updates.status")}
            value={desktopState?.updates.status ?? "-"}
            meta={desktopState?.updates.message}
          />
          <StatCard
            label={t("settings.updates.availableVersion")}
            value={
              desktopState?.updates.availableVersion ??
              desktopState?.updates.downloadedVersion ??
              "-"
            }
          />
        </div>

        <div className={`${actionRowClass} mt-6`}>
          <button
            type="button"
            className={buttonClass("secondary")}
            onClick={() => void handleCheckUpdates()}
            disabled={updateBusy || desktopState?.updates.status === "unsupported"}
          >
            {updateBusy && desktopState?.updates.status === "checking"
              ? t("settings.button.checking")
              : t("settings.button.checkUpdates")}
          </button>
          <button
            type="button"
            className={buttonClass("secondary")}
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
            className={buttonClass("primary")}
            onClick={() => void handleQuitAndInstallUpdate()}
            disabled={updateBusy || desktopState?.updates.status !== "downloaded"}
          >
            {t("settings.button.installUpdate")}
          </button>
        </div>
      </section>

      {connectOpen ? (
        <div className={modalBackdropClass} role="presentation" onClick={() => setConnectOpen(false)}>
          <section
            className={modalPanelClass}
            role="dialog"
            aria-modal="true"
            aria-label={t("settings.modal.aria")}
            onClick={(event) => event.stopPropagation()}
          >
            <div className={sectionHeadClass}>
              <div className="space-y-1">
                <h2 className={sectionTitleClass}>{t("settings.modal.title")}</h2>
                <p className={sectionMetaClass}>{connectionGuide.summary}</p>
              </div>
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={() => setConnectOpen(false)}
              >
                {t("common.close")}
              </button>
            </div>

            <div className="mt-6 grid gap-4 sm:grid-cols-2">
              <label className="flex flex-col gap-2">
                <span className={fieldLabelClass}>{t("settings.modal.tool")}</span>
                <select
                  className={inputClass}
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
              <label className="flex flex-col gap-2">
                <span className={fieldLabelClass}>{t("settings.modal.platform")}</span>
                <select
                  className={inputClass}
                  value={platformPreset}
                  onChange={(event) => setPlatformPreset(event.target.value as PlatformPreset)}
                >
                  <option value="unix">{t("settings.platform.unix")}</option>
                  <option value="windows-cmd">{t("settings.platform.windowsCmd")}</option>
                  <option value="powershell">{t("settings.platform.powershell")}</option>
                </select>
              </label>
            </div>

            {connectionGuide.supportsManual ? (
              <div className="mt-6 grid grid-cols-2 rounded-[22px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-1">
                <button
                  type="button"
                  className={
                    connectMode === "command"
                      ? buttonClass("primary")
                      : buttonClass("ghost")
                  }
                  onClick={() => setConnectMode("command")}
                >
                  {t("settings.modal.mode.command")}
                </button>
                <button
                  type="button"
                  className={
                    connectMode === "manual"
                      ? buttonClass("primary")
                      : buttonClass("ghost")
                  }
                  onClick={() => setConnectMode("manual")}
                >
                  {t("settings.modal.mode.manual")}
                </button>
              </div>
            ) : null}

            {connectMode === "command" || !connectionGuide.supportsManual ? (
              <div className={`${infoCardClass} mt-6 p-5`}>
                <div className="mb-4 flex items-start justify-between gap-3">
                  <div>
                    <p className={fieldLabelClass}>{connectionGuide.title}</p>
                    {copyFeedback ? <p className={`${metaClass} mt-2`}>{copyFeedback}</p> : null}
                  </div>
                  <button
                    type="button"
                    className={
                      copyFeedback === t("settings.copy.copied")
                        ? `${iconButtonClass} [border-color:var(--success-border)] [background:var(--success-soft)] text-[color:var(--success-text)]`
                        : iconButtonClass
                    }
                    aria-label={t("settings.copy.command")}
                    title={t("settings.copy.command")}
                    onClick={() => {
                      void handleCopyCommand(connectionGuide.command, connectionGuide.title);
                    }}
                  >
                    {copyFeedback === t("settings.copy.copied") ? (
                      <svg className="h-5 w-5 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                        <path d="M9.2 16.6 4.9 12.3l1.4-1.4 2.9 2.9 8.5-8.5 1.4 1.4z" />
                      </svg>
                    ) : (
                      <svg className="h-5 w-5 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                        <path d="M9 9h10v10H9z" />
                        <path d="M5 5h10v2H7v8H5z" />
                      </svg>
                    )}
                  </button>
                </div>
                <pre className="rounded-3xl border [border-color:var(--border-soft)] [background:var(--panel-input)] p-4 text-sm leading-7 text-[color:var(--color-text)]">
                  <code>{connectionGuide.command}</code>
                </pre>
                <div className="mt-4 rounded-2xl border [border-color:var(--border-soft)] [background:var(--panel-input)] px-4 py-3">
                  <p className={metaClass}>{connectionGuide.note}</p>
                </div>
              </div>
            ) : (
              <div className={`${infoCardClass} mt-6 p-5`}>
                <div className="mb-4">
                  <p className={fieldLabelClass}>{connectionGuide.manualTitle}</p>
                </div>
                {connectionGuide.manualItems?.length ? (
                  <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                    {connectionGuide.manualItems.map((item) => (
                      <div key={item.label} className="rounded-3xl border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-4">
                        <div className="flex items-start justify-between gap-3">
                          <p className={fieldLabelClass}>{item.label}</p>
                          <button
                            type="button"
                            className={
                              manualCopyFeedback === item.label
                                ? `${iconButtonSmallClass} [border-color:var(--success-border)] [background:var(--success-soft)] text-[color:var(--success-text)]`
                                : iconButtonSmallClass
                            }
                            aria-label={t("settings.copy.value", { label: item.label })}
                            title={t("settings.copy.value", { label: item.label })}
                            onClick={() => {
                              void handleCopyValue(item.value, item.label);
                            }}
                          >
                            {manualCopyFeedback === item.label ? (
                              <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                                <path d="M9.2 16.6 4.9 12.3l1.4-1.4 2.9 2.9 8.5-8.5 1.4 1.4z" />
                              </svg>
                            ) : (
                              <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                                <path d="M9 9h10v10H9z" />
                                <path d="M5 5h10v2H7v8H5z" />
                              </svg>
                            )}
                          </button>
                        </div>
                        <p className={`${monoClass} mt-3`}>{item.value}</p>
                        {item.hint ? <p className={`${metaClass} mt-2`}>{item.hint}</p> : null}
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className={emptyStateClass}>No manual settings available.</div>
                )}
                <div className="mt-4 rounded-2xl border [border-color:var(--border-soft)] [background:var(--panel-input)] px-4 py-3">
                  <p className={metaClass}>{connectionGuide.note}</p>
                </div>
              </div>
            )}
          </section>
        </div>
      ) : null}
    </main>
  );
}
