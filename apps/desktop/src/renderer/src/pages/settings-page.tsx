import { useCallback, useEffect, useState, type ReactNode } from "react";
import { ToastRegion, type ToastItem } from "../components/toast-region";
import { useI18n } from "../i18n/i18n-provider";
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
    localGatewayPort: number;
    localGatewayPortSource: "default" | "config" | "env";
    launchAtLogin: boolean;
    launchHidden: boolean;
    closeToTray: boolean;
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
  onCheckUpdates: () => Promise<void>;
  onDownloadUpdate: () => Promise<void>;
  onQuitAndInstallUpdate: () => Promise<void>;
  onOpenReleasePage: () => Promise<void>;
  onOpenProjectPage: () => Promise<void>;
  onCoreRestart: () => Promise<void>;
  onUpdateCorePort: (port: number) => Promise<void>;
  onUpdateLocalGatewayPort: (port: number) => Promise<void>;
  onUpdateLaunchSettings: (settings: {
    launchAtLogin?: boolean;
    launchHidden?: boolean;
    closeToTray?: boolean;
  }) => Promise<void>;
}

function SettingsToggle({
  checked,
  label,
  meta,
  disabled,
  onChange
}: {
  checked: boolean;
  label: string;
  meta: string;
  disabled?: boolean;
  onChange: () => void;
}) {
  return (
    <div className={infoCardClass}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className={fieldLabelClass}>{label}</p>
          <p className={`${metaClass} mt-2`}>{meta}</p>
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={checked}
          aria-label={label}
          disabled={disabled}
          className={`inline-flex h-7 w-12 shrink-0 items-center rounded-full border px-1 transition ${
            checked
              ? "[border-color:var(--success-border)] [background:var(--success-soft)]"
              : "[border-color:var(--border-soft)] [background:var(--panel-soft)]"
          } disabled:cursor-not-allowed disabled:opacity-50`}
          onClick={onChange}
        >
          <span
            className={`h-5 w-5 rounded-full transition ${
              checked
                ? "translate-x-5 bg-[color:var(--accent-strong)]"
                : "translate-x-0 bg-[color:var(--color-subtle)]"
            }`}
          />
        </button>
      </div>
    </div>
  );
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
  onOpenReleasePage,
  onOpenProjectPage,
  onCoreRestart,
  onUpdateCorePort,
  onUpdateLocalGatewayPort,
  onUpdateLaunchSettings
}: SettingsPageProps) {
  const { t } = useI18n();
  const [busy, setBusy] = useState(false);
  const [updateBusy, setUpdateBusy] = useState(false);
  const [saveBusy, setSaveBusy] = useState(false);
  const [launchBusy, setLaunchBusy] = useState(false);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [feedbackTone, setFeedbackTone] = useState<"success" | "error">("success");
  const [portInput, setPortInput] = useState(String(desktopState?.config.apiPort ?? 3456));
  const [localGatewayPortInput, setLocalGatewayPortInput] = useState(
    String(desktopState?.config.localGatewayPort ?? 3457)
  );
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
    setLocalGatewayPortInput(String(desktopState?.config.localGatewayPort ?? 3457));
  }, [desktopState?.config.localGatewayPort]);

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

  async function handleSaveLocalGatewayPort() {
    const nextPort = Number(localGatewayPortInput);
    if (!Number.isInteger(nextPort) || nextPort < 1 || nextPort > 65535) {
      setFeedbackTone("error");
      setFeedback(t("settings.feedback.invalidPort"));
      return;
    }

    setSaveBusy(true);
    setFeedback(null);

    try {
      await onUpdateLocalGatewayPort(nextPort);
      setFeedbackTone("success");
      setFeedback(t("settings.feedback.localGatewayPortUpdated", { port: nextPort }));
    } catch (error) {
      setFeedbackTone("error");
      setFeedback(
        error instanceof Error ? error.message : t("settings.feedback.localGatewayPortUpdateFailed")
      );
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

  async function handleToggleLaunchAtLogin() {
    setLaunchBusy(true);
    setFeedback(null);

    try {
      const nextValue = !desktopState?.config.launchAtLogin;
      await onUpdateLaunchSettings({ launchAtLogin: nextValue });
      setFeedbackTone("success");
      setFeedback(
        nextValue
          ? t("settings.feedback.launchAtLoginEnabled")
          : t("settings.feedback.launchAtLoginDisabled")
      );
    } catch (error) {
      setFeedbackTone("error");
      setFeedback(error instanceof Error ? error.message : t("settings.feedback.launchSettingsFailed"));
    } finally {
      setLaunchBusy(false);
    }
  }

  async function handleToggleLaunchHidden() {
    setLaunchBusy(true);
    setFeedback(null);

    try {
      const nextValue = !desktopState?.config.launchHidden;
      await onUpdateLaunchSettings({ launchHidden: nextValue });
      setFeedbackTone("success");
      setFeedback(
        nextValue
          ? t("settings.feedback.launchHiddenEnabled")
          : t("settings.feedback.launchHiddenDisabled")
      );
    } catch (error) {
      setFeedbackTone("error");
      setFeedback(error instanceof Error ? error.message : t("settings.feedback.launchSettingsFailed"));
    } finally {
      setLaunchBusy(false);
    }
  }

  async function handleToggleCloseToTray() {
    setLaunchBusy(true);
    setFeedback(null);

    try {
      const nextValue = !desktopState?.config.closeToTray;
      await onUpdateLaunchSettings({ closeToTray: nextValue });
      setFeedbackTone("success");
      setFeedback(
        nextValue
          ? t("settings.feedback.closeToTrayEnabled")
          : t("settings.feedback.closeToTrayDisabled")
      );
    } catch (error) {
      setFeedbackTone("error");
      setFeedback(error instanceof Error ? error.message : t("settings.feedback.launchSettingsFailed"));
    } finally {
      setLaunchBusy(false);
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

  const portLocked = desktopState?.config.apiPortSource === "env";
  const localGatewayPortLocked = desktopState?.config.localGatewayPortSource === "env";
  const isMacOS = desktopState?.platform === "darwin";
  const hasAvailableMacUpdate =
    isMacOS &&
    (desktopState?.updates.status === "available" ||
      desktopState?.updates.status === "downloaded" ||
      desktopState?.updates.status === "downloading");
  const updateStatusLabel =
    desktopState?.updates.status === "checking"
      ? t("updates.status.checking")
      : desktopState?.updates.status === "available"
        ? t("updates.status.available")
        : desktopState?.updates.status === "downloading"
          ? t("updates.status.downloading")
          : desktopState?.updates.status === "downloaded"
            ? t("updates.status.downloaded")
            : desktopState?.updates.status === "error"
              ? t("updates.status.error")
              : desktopState?.updates.status === "unsupported"
                ? t("updates.status.unsupported")
                : t("updates.status.idle");
  const updateStatusMeta =
    desktopState?.updates.status === "available" && desktopState.updates.availableVersion
      ? t("updates.card.availableVersion", { version: desktopState.updates.availableVersion })
      : desktopState?.updates.status === "downloading"
        ? t("updates.card.downloading", {
            progress: Math.round(desktopState.updates.progressPercent ?? 0)
          })
        : desktopState?.updates.status === "downloaded"
          ? t("updates.card.downloadedVersion", {
              version:
                desktopState.updates.downloadedVersion ?? desktopState.updates.availableVersion ?? "-"
            })
          : desktopState?.updates.status === "error"
            ? desktopState.updates.message
            : desktopState?.updates.status === "not-available"
              ? t("updates.status.notAvailable")
              : desktopState?.updates.message;

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
          <div className={infoCardClass}>
            <p className={fieldLabelClass}>{t("settings.localGatewayFixedPort")}</p>
            <input
              className={`${inputClass} mt-3`}
              value={localGatewayPortInput}
              disabled={localGatewayPortLocked || saveBusy}
              onChange={(event) => setLocalGatewayPortInput(event.target.value)}
              inputMode="numeric"
            />
            <p className={`${metaClass} mt-2`}>{t("settings.meta.localGatewayPortConflict")}</p>
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
            <button
              type="button"
              className={buttonClass("secondary")}
              onClick={() => void handleSaveLocalGatewayPort()}
              disabled={localGatewayPortLocked || saveBusy}
            >
              {saveBusy ? t("common.saving") : t("settings.button.saveLocalGatewayPort")}
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
            <h2 className={sectionTitleClass}>{t("settings.section.launch")}</h2>
            <p className={sectionMetaClass}>{t("settings.meta.launchBehavior")}</p>
          </div>
        </div>

        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          <SettingsToggle
            checked={Boolean(desktopState?.config.launchAtLogin)}
            label={t("settings.launchAtLogin")}
            meta={t("settings.meta.launchAtLogin")}
            disabled={launchBusy}
            onChange={() => void handleToggleLaunchAtLogin()}
          />
          <SettingsToggle
            checked={Boolean(desktopState?.config.launchHidden)}
            label={t("settings.launchHidden")}
            meta={t("settings.meta.launchHidden")}
            disabled={launchBusy}
            onChange={() => void handleToggleLaunchHidden()}
          />
          <SettingsToggle
            checked={Boolean(desktopState?.config.closeToTray)}
            label={t("settings.closeToTray")}
            meta={t("settings.meta.closeToTray")}
            disabled={launchBusy}
            onChange={() => void handleToggleCloseToTray()}
          />
        </div>
        <p className={`${metaClass} mt-4`}>{t("settings.meta.trayHint")}</p>
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
            label={t("settings.runtime.localGatewayPort")}
            value={String(desktopState?.config.localGatewayPort ?? "-")}
          />
          <StatCard
            label={t("settings.runtime.corePid")}
            value={String(desktopState?.core.pid ?? "-")}
            meta={t("settings.meta.pid")}
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

        <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <StatCard
            label={t("settings.updates.currentVersion")}
            value={desktopState?.updates.currentVersion ?? "-"}
          />
          <StatCard
            label={t("settings.updates.status")}
            value={updateStatusLabel}
            meta={updateStatusMeta}
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

        {hasAvailableMacUpdate ? (
          <p className="mt-4 rounded-2xl border [border-color:var(--success-border)] [background:var(--success-soft)] px-4 py-3 text-sm leading-6 text-[color:var(--success-text)]">
            {t("settings.meta.macManualUpdate")}
          </p>
        ) : null}

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
          {isMacOS ? (
            <button
              type="button"
              className={buttonClass(hasAvailableMacUpdate ? "primary" : "secondary")}
              onClick={() => void handleOpenReleasePage()}
              disabled={updateBusy || !hasAvailableMacUpdate}
            >
              {hasAvailableMacUpdate
                ? t("settings.button.openReleasePage")
                : t("settings.button.manualInstallOnly")}
            </button>
          ) : (
            <>
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
            </>
          )}
        </div>
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>{t("settings.section.project")}</h2>
            <p className={sectionMetaClass}>{t("settings.meta.project")}</p>
          </div>
        </div>

        <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex min-w-0 items-center gap-3">
            <span className={iconBadgeClass}>
              <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                <path d="M12 .5A12 12 0 0 0 8.2 23.9c.6.1.8-.3.8-.6v-2c-3.3.7-4-1.4-4-1.4-.5-1.3-1.2-1.6-1.2-1.6-1-.7.1-.7.1-.7 1.1.1 1.7 1.2 1.7 1.2 1 1.7 2.6 1.2 3.3.9.1-.7.4-1.2.7-1.5-2.7-.3-5.5-1.3-5.5-5.9 0-1.3.5-2.4 1.2-3.2-.1-.3-.5-1.6.1-3.2 0 0 1-.3 3.3 1.2A11.4 11.4 0 0 1 12 6.6c1 0 2 .1 3 .4 2.3-1.5 3.3-1.2 3.3-1.2.6 1.6.2 2.9.1 3.2.8.9 1.2 1.9 1.2 3.2 0 4.6-2.8 5.6-5.5 5.9.4.4.8 1.1.8 2.2v3c0 .3.2.7.8.6A12 12 0 0 0 12 .5" />
              </svg>
            </span>
            <div className="min-w-0">
              <p className={fieldLabelClass}>{t("settings.project.github")}</p>
              <p className={`${monoClass} mt-2 truncate text-xs`}>
                https://github.com/xiaoyuandev/clash-for-ai
              </p>
            </div>
          </div>
          <button
            type="button"
            className={buttonClass("secondary")}
            onClick={() => void onOpenProjectPage()}
          >
            {t("settings.button.openProjectPage")}
          </button>
        </div>
      </section>
    </main>
  );
}
