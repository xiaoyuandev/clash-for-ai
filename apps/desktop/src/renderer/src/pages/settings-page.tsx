import { useCallback, useEffect, useState, type ReactNode } from "react";
import { ToastRegion, type ToastItem } from "../components/toast-region";
import { useI18n } from "../i18n/i18n-provider";
import { getPortkeyTemplate, getRuntimeConfig, getRuntimeHealth, updateRuntimeConfig } from "../services/api";
import type { PortkeyTemplate } from "../types/portkey-template";
import type { RuntimeConfig, RuntimeHealth, RuntimeMode } from "../types/runtime";
import { getRuntimeLabel } from "../utils/runtime-label";
import {
  actionRowClass,
  buttonClass,
  eyebrowClass,
  fieldLabelClass,
  heroClass,
  heroCopyClass,
  heroTitleClass,
  iconBadgeClass,
  infoCardClass,
  inputClass,
  metaClass,
  metricValueClass,
  monoClass,
  pageShellClass,
  sectionCardClass,
  sectionHeadClass,
  sectionMetaClass,
  sectionTitleClass,
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

interface SettingsPageProps {
  desktopState: DesktopState;
  onCopyText: (text: string) => Promise<void>;
  onCheckUpdates: () => Promise<void>;
  onDownloadUpdate: () => Promise<void>;
  onQuitAndInstallUpdate: () => Promise<void>;
  onOpenReleasePage: () => Promise<void>;
  onCoreRestart: () => Promise<void>;
  onUpdateCorePort: (port: number) => Promise<void>;
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
  onCopyText,
  onCheckUpdates,
  onDownloadUpdate,
  onQuitAndInstallUpdate,
  onOpenReleasePage,
  onCoreRestart,
  onUpdateCorePort
}: SettingsPageProps) {
  const { t } = useI18n();
  const [busy, setBusy] = useState(false);
  const [updateBusy, setUpdateBusy] = useState(false);
  const [saveBusy, setSaveBusy] = useState(false);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [feedbackTone, setFeedbackTone] = useState<"success" | "error">("success");
  const [portInput, setPortInput] = useState(String(desktopState?.config.apiPort ?? 3456));
  const [runtimeMode, setRuntimeMode] = useState<RuntimeMode>("legacy");
  const [runtimeBaseURL, setRuntimeBaseURL] = useState("");
  const [runtimeHealth, setRuntimeHealth] = useState<RuntimeHealth | null>(null);
  const [portkeyTemplate, setPortkeyTemplate] = useState<PortkeyTemplate | null>(null);
  const [runtimeBusy, setRuntimeBusy] = useState(false);
  const [runtimeSaveBusy, setRuntimeSaveBusy] = useState(false);
  const [copyBusy, setCopyBusy] = useState(false);
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const runtimeLabel = getRuntimeLabel(desktopState?.runtime, {
    desktopApp: t("settings.value.desktopApp"),
    browser: t("settings.value.browser"),
    unknown: t("settings.value.unknown")
  });

  useEffect(() => {
    setPortInput(String(desktopState?.config.apiPort ?? 3456));
  }, [desktopState?.config.apiPort]);

  useEffect(() => {
    let cancelled = false;

    async function loadRuntimeState() {
      try {
        const [config, health] = await Promise.all([
          getRuntimeConfig(desktopState?.apiBase),
          getRuntimeHealth(desktopState?.apiBase)
        ]);
        if (cancelled) {
          return;
        }
        setRuntimeMode(config.mode);
        setRuntimeBaseURL(config.base_url);
        setRuntimeHealth(health);
        if (config.mode === "external-portkey") {
          const template = await getPortkeyTemplate(desktopState?.apiBase);
          if (cancelled) {
            return;
          }
          setPortkeyTemplate(template);
        } else {
          setPortkeyTemplate(null);
        }
      } catch (error) {
        if (cancelled) {
          return;
        }
        setFeedbackTone("error");
        setFeedback(error instanceof Error ? error.message : t("common.unknownError"));
      }
    }

    void loadRuntimeState();

    return () => {
      cancelled = true;
    };
  }, [desktopState?.apiBase, t]);

  const dismissToast = useCallback((id: string) => {
    setToasts((current) => current.filter((item) => item.id !== id));
  }, []);

  useEffect(() => {
    if (!feedback) {
      return;
    }
    setToasts((current) => [
      ...current,
      { id: `${Date.now()}-feedback`, message: feedback, tone: feedbackTone }
    ]);
    setFeedback(null);
  }, [feedback, feedbackTone]);

  async function handleRestart() {
    setBusy(true);
    setFeedback(null);

    try {
      await onCoreRestart();
      setFeedbackTone("success");
      setFeedback(t("settings.feedback.coreRestarted"));
    } catch (error) {
      setFeedbackTone("error");
      setFeedback(error instanceof Error ? error.message : t("settings.feedback.restartFailed"));
    } finally {
      setBusy(false);
    }
  }

  async function handleSavePort() {
    const nextPort = Number(portInput);
    if (!Number.isInteger(nextPort) || nextPort < 1 || nextPort > 65535) {
      setFeedbackTone("error");
      setFeedback(t("settings.feedback.invalidPort"));
      return;
    }

    setSaveBusy(true);
    setFeedback(null);

    try {
      await onUpdateCorePort(nextPort);
      setFeedbackTone("success");
      setFeedback(t("settings.feedback.portUpdated", { port: nextPort }));
    } catch (error) {
      setFeedbackTone("error");
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
      setFeedbackTone("error");
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
      setFeedbackTone("error");
      setFeedback(error instanceof Error ? error.message : t("settings.feedback.updateDownloadFailed"));
    } finally {
      setUpdateBusy(false);
    }
  }

  async function handleOpenReleasePage() {
    setUpdateBusy(true);
    setFeedback(null);

    try {
      await onOpenReleasePage();
    } catch (error) {
      setFeedbackTone("error");
      setFeedback(
        error instanceof Error ? error.message : t("settings.feedback.updateOpenReleaseFailed")
      );
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
      setFeedbackTone("error");
      setFeedback(
        error instanceof Error ? error.message : t("settings.feedback.updateInstallFailed")
      );
      setUpdateBusy(false);
    }
  }

  async function handleCheckRuntime() {
    setRuntimeBusy(true);
    setFeedback(null);

    try {
      const health = await getRuntimeHealth(desktopState?.apiBase);
      setRuntimeHealth(health);
      setFeedbackTone(health.status === "ok" ? "success" : "error");
      setFeedback(
        t("settings.runtime.feedback.checked", {
          status: health.status,
          message: health.message
        })
      );
    } catch (error) {
      setFeedbackTone("error");
      setFeedback(error instanceof Error ? error.message : t("common.unknownError"));
    } finally {
      setRuntimeBusy(false);
    }
  }

  async function handleSaveRuntime() {
    setRuntimeSaveBusy(true);
    setFeedback(null);

    const nextConfig: RuntimeConfig = {
      mode: runtimeMode,
      base_url: runtimeBaseURL.trim()
    };

    try {
      const saved = await updateRuntimeConfig(nextConfig, desktopState?.apiBase);
      setRuntimeMode(saved.mode);
      setRuntimeBaseURL(saved.base_url);
      const health = await getRuntimeHealth(desktopState?.apiBase);
      setRuntimeHealth(health);
      if (saved.mode === "external-portkey") {
        const template = await getPortkeyTemplate(desktopState?.apiBase);
        setPortkeyTemplate(template);
      } else {
        setPortkeyTemplate(null);
      }
      setFeedbackTone("success");
      setFeedback(t("settings.runtime.feedback.saved"));
    } catch (error) {
      setFeedbackTone("error");
      setFeedback(error instanceof Error ? error.message : t("common.unknownError"));
    } finally {
      setRuntimeSaveBusy(false);
    }
  }

  const portLocked = desktopState?.config.apiPortSource === "env";
  const isMacPlatform = desktopState?.platform === "darwin";
  const runtimeInitCommands =
    runtimeMode === "external-portkey"
      ? [
          "node -v",
          "npm -v",
          "npx @portkey-ai/gateway"
        ].join("\n")
      : "";

  async function handleCopyRuntimeCommands() {
    if (!runtimeInitCommands) {
      return;
    }

    setCopyBusy(true);
    setFeedback(null);

    try {
      await onCopyText(runtimeInitCommands);
      setFeedbackTone("success");
      setFeedback(t("settings.runtime.feedback.commandsCopied"));
    } catch (error) {
      setFeedbackTone("error");
      setFeedback(error instanceof Error ? error.message : t("settings.runtime.feedback.commandsCopyFailed"));
    } finally {
      setCopyBusy(false);
    }
  }

  return (
    <main className={pageShellClass}>
      <ToastRegion items={toasts} onDismiss={dismissToast} />
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
        </div>
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>{t("settings.section.connection")}</h2>
            <p className={sectionMetaClass}>{desktopState?.config.apiPortSource ?? t("settings.value.default")}</p>
          </div>
        </div>

        <div className="mt-4 grid gap-3 sm:grid-cols-2">
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

        <div className="mt-4 flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between">
          <p className={metaClass}>{t("settings.meta.configStored")}</p>
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
            <p className={sectionMetaClass}>{runtimeLabel}</p>
          </div>
        </div>

        <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
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

        <div className="mt-5 grid gap-3 xl:grid-cols-[1.1fr_1.4fr]">
          <div className={infoCardClass}>
            <p className={fieldLabelClass}>{t("settings.runtime.gatewayMode")}</p>
            <select
              className={`${inputClass} mt-3`}
              value={runtimeMode}
              onChange={(event) => setRuntimeMode(event.target.value as RuntimeMode)}
              disabled={runtimeSaveBusy}
            >
              <option value="legacy">{t("settings.runtime.mode.legacy")}</option>
              <option value="external-portkey">{t("settings.runtime.mode.externalPortkey")}</option>
            </select>

            <p className={`${fieldLabelClass} mt-4`}>{t("settings.runtime.gatewayBaseUrl")}</p>
            <input
              className={`${inputClass} mt-3`}
              value={runtimeBaseURL}
              onChange={(event) => setRuntimeBaseURL(event.target.value)}
              placeholder="http://127.0.0.1:8787"
              disabled={runtimeMode === "legacy" || runtimeSaveBusy}
            />

            <p className={`${metaClass} mt-2`}>{t("settings.runtime.meta.externalHint")}</p>

            <div className={`${actionRowClass} mt-4`}>
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={() => void handleCheckRuntime()}
                disabled={runtimeBusy}
              >
                {runtimeBusy
                  ? t("settings.runtime.button.checking")
                  : t("settings.runtime.button.check")}
              </button>
              <button
                type="button"
                className={buttonClass("primary")}
                onClick={() => void handleSaveRuntime()}
                disabled={runtimeSaveBusy}
              >
                {runtimeSaveBusy ? t("common.saving") : t("settings.runtime.button.save")}
              </button>
            </div>

            {runtimeMode === "external-portkey" ? (
              <div className="mt-5 rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-soft)] p-3.5">
                <div className="space-y-1">
                  <p className={fieldLabelClass}>{t("settings.runtime.init.title")}</p>
                  <p className={metaClass}>{t("settings.runtime.init.subtitle")}</p>
                </div>
                <pre className={`${monoClass} mt-3 overflow-x-auto whitespace-pre-wrap rounded-[12px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3`}>
                  {runtimeInitCommands}
                </pre>
                <div className={`${actionRowClass} mt-3`}>
                  <button
                    type="button"
                    className={buttonClass("secondary")}
                    onClick={() => void handleCopyRuntimeCommands()}
                    disabled={copyBusy}
                  >
                    {copyBusy
                      ? t("settings.runtime.button.copying")
                      : t("settings.runtime.button.copyCommands")}
                  </button>
                </div>
                {portkeyTemplate ? (
                  <>
                    <div className="mt-5 space-y-1">
                      <p className={fieldLabelClass}>{t("settings.runtime.template.title")}</p>
                      <p className={metaClass}>{t("settings.runtime.template.subtitle")}</p>
                    </div>
                    <div className="mt-3 grid gap-3 md:grid-cols-3">
                      <div className={infoCardClass}>
                        <p className={fieldLabelClass}>{t("settings.runtime.template.totalEntries")}</p>
                        <p className={metricValueClass}>{String(portkeyTemplate.total_entries)}</p>
                      </div>
                      <div className={infoCardClass}>
                        <p className={fieldLabelClass}>{t("settings.runtime.template.enabledEntries")}</p>
                        <p className={metricValueClass}>{String(portkeyTemplate.enabled_count)}</p>
                      </div>
                      <div className={infoCardClass}>
                        <p className={fieldLabelClass}>{t("settings.runtime.template.disabledEntries")}</p>
                        <p className={metricValueClass}>{String(portkeyTemplate.disabled_count)}</p>
                      </div>
                    </div>
                    <pre className={`${monoClass} mt-3 overflow-x-auto whitespace-pre-wrap rounded-[12px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3`}>
                      {JSON.stringify(portkeyTemplate, null, 2)}
                    </pre>
                  </>
                ) : null}
              </div>
            ) : null}
          </div>

          <div className={infoCardClass}>
            <p className={fieldLabelClass}>{t("settings.runtime.healthTitle")}</p>
            <p className={metricValueClass}>
              {runtimeHealth?.status ?? t("settings.runtime.healthUnknown")}
            </p>
            <p className={`${metaClass} mt-2`}>
              {runtimeHealth?.message ?? t("settings.runtime.healthUnknown")}
            </p>
            <div className="mt-4 space-y-2">
              <p className={metaClass}>
                {t("settings.runtime.healthMode", { mode: runtimeHealth?.mode ?? runtimeMode })}
              </p>
              <p className={metaClass}>
                {t("settings.runtime.healthBaseUrl", {
                  baseUrl: runtimeHealth?.base_url || runtimeBaseURL || "-"
                })}
              </p>
              <p className={metaClass}>
                {t("settings.runtime.healthCheckedAt", {
                  checkedAt: runtimeHealth?.checked_at ?? "-"
                })}
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>{t("settings.section.updates")}</h2>
            <p className={sectionMetaClass}>{desktopState?.updates.currentVersion ?? "-"}</p>
          </div>
        </div>

        <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
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

        <div className={`${actionRowClass} mt-4`}>
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
            onClick={() => void (isMacPlatform ? handleOpenReleasePage() : handleDownloadUpdate())}
            disabled={
              updateBusy ||
              (desktopState?.updates.status !== "available" &&
                desktopState?.updates.status !== "downloading" &&
                desktopState?.updates.status !== "downloaded")
            }
          >
            {isMacPlatform
              ? t("settings.button.openReleasePage")
              : desktopState?.updates.status === "downloading"
                ? t("settings.button.downloading", {
                    progress: Math.round(desktopState.updates.progressPercent ?? 0)
                  })
                : t("settings.button.downloadUpdate")}
          </button>
          <button
            type="button"
            className={buttonClass("primary")}
            onClick={() => void handleQuitAndInstallUpdate()}
            disabled={
              isMacPlatform || updateBusy || desktopState?.updates.status !== "downloaded"
            }
          >
            {isMacPlatform
              ? t("settings.button.manualInstallOnly")
              : t("settings.button.installUpdate")}
          </button>
        </div>
        {isMacPlatform ? <p className={`${metaClass} mt-4`}>{t("settings.meta.macManualUpdate")}</p> : null}
      </section>
    </main>
  );
}
