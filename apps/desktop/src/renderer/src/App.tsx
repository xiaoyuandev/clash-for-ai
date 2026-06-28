import { useCallback, useEffect, useRef, useState } from "react";
import { ToastRegion, type ToastItem } from "./components/toast-region";
import { useI18n } from "./i18n/i18n-provider";
import {
  createLocalGatewaySource,
  createProvider,
  syncLocalGateway
} from "./services/api";
import { getModelPresets } from "./services/model-presets";
import { LogsPage } from "./pages/logs-page";
import { ModelsPage } from "./pages/models-page";
import { ProvidersPage } from "./pages/providers-page";
import { SettingsPage } from "./pages/settings-page";
import { ToolsPage } from "./pages/tools-page";
import { useTheme } from "./theme/theme-provider";
import type { AuthMode, Provider } from "./types/provider";
import type { CreateLocalGatewayModelSourceInput } from "./types/local-gateway";
import appIcon from "../../../build/icon.png";
import {
  appBackdropClass,
  appShellClass,
  buttonClass,
  eyebrowClass,
  fieldLabelClass,
  glassPanelClass,
  heroClass,
  heroCopyClass,
  heroTitleClass,
  modalBackdropClass,
  modalPanelClass,
  metaClass,
  monoClass,
  navButtonClass,
  pageShellClass,
  sectionMetaClass,
  sectionTitleClass,
  statusDotClass
} from "./ui";

interface DesktopState {
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
  updates: {
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
}

interface DeepLinkImportEvent {
  id: string;
  kind: "import";
  request: {
    resource: "provider" | "model";
    payload: Record<string, unknown>;
    originalURL: string;
  };
}

interface DeepLinkErrorEvent {
  id: string;
  kind: "error";
  message: string;
  originalURL?: string;
}

type DesktopDeepLinkEvent = DeepLinkImportEvent | DeepLinkErrorEvent;

type ImportRequest =
  | {
      id: string;
      resource: "provider";
      originalURL: string;
      data: {
        name: string;
        baseUrl: string;
        apiKey: string;
        authMode: AuthMode;
      };
    }
  | {
      id: string;
      resource: "model";
      originalURL: string;
      data: {
        name: string;
        baseUrl: string;
        apiKey: string;
        providerType: "openai-compatible" | "anthropic-compatible";
        modelIds: string[];
      };
    };

function readRequiredString(payload: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const value = payload[key];
    if (typeof value === "string" && value.trim()) {
      return value.trim();
    }
  }

  throw new Error(`Missing required field: ${keys[0]}.`);
}

function readOptionalString(payload: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const value = payload[key];
    if (typeof value === "string" && value.trim()) {
      return value.trim();
    }
  }

  return "";
}

function maskImportAPIKey(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }

  if (trimmed.length <= 4) {
    return "****";
  }

  if (trimmed.length <= 12) {
    return `${trimmed.slice(0, trimmed.length - 4)}****`;
  }

  return `${trimmed.slice(0, 8)}••••${trimmed.slice(-4)}`;
}

function normalizeImportRequest(event: DeepLinkImportEvent): ImportRequest {
  const { payload, resource, originalURL } = event.request;

  if (resource === "provider") {
    const authModeValue = readOptionalString(payload, ["authMode", "auth_mode"]).toLowerCase();
    const authMode: AuthMode =
      authModeValue === "x-api-key" || authModeValue === "both" ? authModeValue : "bearer";

    return {
      id: event.id,
      resource: "provider",
      originalURL,
      data: {
        name: readRequiredString(payload, ["name"]),
        baseUrl: readRequiredString(payload, ["baseUrl", "base_url", "endpoint"]),
        apiKey: readOptionalString(payload, ["apiKey", "api_key"]),
        authMode
      }
    };
  }

  const providerTypeValue = readOptionalString(payload, ["providerType", "provider_type"]).toLowerCase();
  const providerType =
    providerTypeValue === "anthropic-compatible" ? "anthropic-compatible" : "openai-compatible";
  const listCandidate =
    payload.modelIds ?? payload.model_ids ?? payload.models ?? payload.exposedModelIds ?? payload.exposed_model_ids;
  const modelIds = Array.isArray(listCandidate)
    ? listCandidate
        .filter((item): item is string => typeof item === "string")
        .map((item) => item.trim())
        .filter(Boolean)
    : [];
  const defaultModelId = readOptionalString(payload, ["defaultModelId", "default_model_id"]);
  const normalizedModelIds = Array.from(new Set([defaultModelId, ...modelIds].filter(Boolean)));

  if (normalizedModelIds.length === 0) {
    throw new Error("Missing required field: modelIds.");
  }

  return {
    id: event.id,
    resource: "model",
    originalURL,
    data: {
      name: readRequiredString(payload, ["name"]),
      baseUrl: readRequiredString(payload, ["baseUrl", "base_url", "endpoint"]),
      apiKey: readRequiredString(payload, ["apiKey", "api_key"]),
      providerType,
      modelIds: normalizedModelIds
    }
  };
}

export default function App() {
  const { locale, localeLabels, setLocale, t } = useI18n();
  const { resolvedTheme, toggleTheme } = useTheme();
  const [desktopState, setDesktopState] = useState<DesktopState | null>(null);
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [view, setView] = useState<"providers" | "tools" | "models" | "logs" | "settings">(
    "providers"
  );
  const [bootError, setBootError] = useState<string | null>(null);
  const [selectedProvider, setSelectedProvider] = useState<Provider | null>(null);
  const [providersRefreshToken, setProvidersRefreshToken] = useState(0);
  const [modelsRefreshToken, setModelsRefreshToken] = useState(0);
  const [pendingImportRequest, setPendingImportRequest] = useState<ImportRequest | null>(null);
  const [importBusy, setImportBusy] = useState(false);
  const [dismissedUpdateReminderKey] = useState<string | null>(null);
  const autoUpdateCheckStartedRef = useRef(false);
  const lastUpdateToastKeyRef = useRef<string | null>(null);
  const lastHandledDeepLinkEventIdRef = useRef<string | null>(null);
  const navItems = [
    {
      id: "providers",
      label: t("app.nav.providers"),
      icon: (
        <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
          <path d="M4 7.5A2.5 2.5 0 0 1 6.5 5h11A2.5 2.5 0 0 1 20 7.5v9A2.5 2.5 0 0 1 17.5 19h-11A2.5 2.5 0 0 1 4 16.5zM6.5 7a.5.5 0 0 0-.5.5V10h12V7.5a.5.5 0 0 0-.5-.5zM18 12H6v4.5a.5.5 0 0 0 .5.5h11a.5.5 0 0 0 .5-.5z" />
        </svg>
      )
    },
    {
      id: "tools",
      label: t("app.nav.tools"),
      icon: (
        <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
          <path d="M13.4 3.4a2 2 0 0 1 2.8 0l4.4 4.4a2 2 0 0 1 0 2.8l-2.1 2.1-7.2-7.2zM10.1 6.7 3 13.8V21h7.2l7.1-7.1zM6 18H5v-1l7.4-7.4 1 1z" />
        </svg>
      )
    },
    {
      id: "models",
      label: t("app.nav.models"),
      icon: (
        <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12 3 4 7v10l8 4 8-4V7zm0 2.2L17.8 8 12 10.8 6.2 8zM6 9.6l5 2.5v6.2l-5-2.5zm7 8.7v-6.2l5-2.5v6.2z" />
        </svg>
      )
    },
    {
      id: "logs",
      label: t("app.nav.logs"),
      icon: (
        <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
          <path d="M5 5h14v2H5zm0 6h14v2H5zm0 6h9v2H5z" />
        </svg>
      )
    },
    {
      id: "settings",
      label: t("app.nav.settings"),
      icon: (
        <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
          <path d="m19.4 13 .1-1-.1-1 2-1.6-2-3.4-2.4 1a7 7 0 0 0-1.7-1l-.4-2.5h-4l-.4 2.5a7 7 0 0 0-1.7 1l-2.4-1-2 3.4 2 1.6a8 8 0 0 0 0 2l-2 1.6 2 3.4 2.4-1a7 7 0 0 0 1.7 1l.4 2.5h4l.4-2.5a7 7 0 0 0 1.7-1l2.4 1 2-3.4zM12 15.5A3.5 3.5 0 1 1 12 8a3.5 3.5 0 0 1 0 7.5" />
        </svg>
      )
    }
  ] as const;

  const dismissToast = useCallback((id: string) => {
    setToasts((current) => current.filter((item) => item.id !== id));
  }, []);

  const pushToast = useCallback((message: string, tone: ToastItem["tone"]) => {
    setToasts((current) => [
      ...current,
      {
        id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        message,
        tone
      }
    ]);
  }, []);

  const updates = desktopState?.updates ?? null;
  const updateReminderKey =
    updates?.status === "available" && updates.availableVersion
      ? `available:${updates.availableVersion}`
      : updates?.status === "downloaded" && (updates.downloadedVersion ?? updates.availableVersion)
        ? `downloaded:${updates.downloadedVersion ?? updates.availableVersion}`
        : null;
  const showUpdateReminder =
    updateReminderKey !== null && updateReminderKey !== dismissedUpdateReminderKey;
  const coreLastError = desktopState?.core.lastError?.trim() ?? "";
  const corePort = desktopState?.core.port || desktopState?.config.apiPort || 3456;
  const corePortConflict =
    desktopState?.core.running === false &&
    /\bport\b.*\boccupied\b|\boccupied\b.*\bport\b/i.test(coreLastError);
  const coreStartError =
    desktopState?.core.running === false
      ? corePortConflict
        ? t("app.coreStartFailedPort", { port: corePort })
        : t("app.coreStartFailedGeneric", {
            message: coreLastError || t("common.unknownError")
          })
      : null;
  const canScheduleAutoUpdateCheck = Boolean(window.desktopBridge && desktopState);
  const updateStatus = desktopState?.updates.status;

  useEffect(() => {
    if (!import.meta.env.DEV) {
      return;
    }

    void getModelPresets();
  }, []);

  useEffect(() => {
    if (!window.desktopBridge) {
      return;
    }

    let cancelled = false;

    async function syncDesktopState() {
      try {
        const state = await window.desktopBridge.ping();
        if (cancelled) {
          return;
        }
        setDesktopState(state);
        setBootError(state.core.lastError ?? null);
      } catch (error) {
        if (cancelled) {
          return;
        }
        setBootError(error instanceof Error ? error.message : t("app.failedLoadState"));
      }
    }

    void syncDesktopState();
    const intervalId = window.setInterval(() => {
      void syncDesktopState();
    }, 2000);

    return () => {
      cancelled = true;
      window.clearInterval(intervalId);
    };
  }, []);

  useEffect(() => {
    if (!canScheduleAutoUpdateCheck || autoUpdateCheckStartedRef.current) {
      return;
    }

    if (updateStatus === "unsupported") {
      autoUpdateCheckStartedRef.current = true;
      return;
    }

    autoUpdateCheckStartedRef.current = true;
    const timeoutId = window.setTimeout(() => {
      void window.desktopBridge
        .checkUpdates()
        .then((nextUpdates) => {
          setDesktopState((current) =>
            current ? { ...current, updates: nextUpdates } : current
          );
        })
        .catch(() => undefined);
    }, 4000);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [canScheduleAutoUpdateCheck, updateStatus]);

  useEffect(() => {
    if (!window.desktopBridge) {
      return;
    }

    const handleDeepLinkEvent = (event: DesktopDeepLinkEvent | null) => {
      if (!event || event.id === lastHandledDeepLinkEventIdRef.current) {
        return;
      }

      lastHandledDeepLinkEventIdRef.current = event.id;

      if (event.kind === "error") {
        pushToast(
          t("importDeepLink.error.parse", { message: event.message }),
          "error"
        );
        return;
      }

      try {
        setPendingImportRequest(normalizeImportRequest(event));
      } catch (error) {
        pushToast(
          t("importDeepLink.error.invalidPayload", {
            message: error instanceof Error ? error.message : t("common.unknownError")
          }),
          "error"
        );
      }
    };

    const unsubscribe = window.desktopBridge.onDeepLinkEvent(handleDeepLinkEvent);
    void window.desktopBridge.consumeDeepLinkEvent().then(handleDeepLinkEvent);

    return () => {
      unsubscribe();
    };
  }, [pushToast, t]);

  useEffect(() => {
    if (!updates) {
      return;
    }

    const key =
      updates.status === "available" && updates.availableVersion
        ? `available:${updates.availableVersion}`
        : updates.status === "downloaded" && (updates.downloadedVersion ?? updates.availableVersion)
          ? `downloaded:${updates.downloadedVersion ?? updates.availableVersion}`
          : updates.status === "error" && updates.message
            ? `error:${updates.message}`
            : null;

    if (!key || key === lastUpdateToastKeyRef.current) {
      return;
    }

    lastUpdateToastKeyRef.current = key;
    setToasts((current) => [
      ...current,
      {
        id: `${Date.now()}-${key}`,
        tone: updates.status === "error" ? "error" : "default",
        message:
          updates.status === "available" && updates.availableVersion
            ? t("updates.toast.available", { version: updates.availableVersion })
            : updates.status === "downloaded"
              ? t("updates.toast.downloaded", {
                  version: updates.downloadedVersion ?? updates.availableVersion ?? ""
                })
              : t("updates.toast.error", {
                  message: updates.message ?? t("updates.status.error")
                })
      }
    ]);
  }, [t, updates]);

  async function handleCheckUpdates() {
    if (!window.desktopBridge) {
      return;
    }

    const nextUpdates = await window.desktopBridge.checkUpdates();
    setDesktopState((current) => (current ? { ...current, updates: nextUpdates } : current));
    setView("settings");
  }

  async function handleDownloadUpdate() {
    if (!window.desktopBridge) {
      return;
    }

    const nextUpdates = await window.desktopBridge.downloadUpdate();
    setDesktopState((current) => (current ? { ...current, updates: nextUpdates } : current));
  }

  async function handleQuitAndInstallUpdate() {
    if (!window.desktopBridge) {
      return;
    }

    const nextUpdates = await window.desktopBridge.quitAndInstallUpdate();
    setDesktopState((current) => (current ? { ...current, updates: nextUpdates } : current));
  }

  async function handleOpenReleasePage() {
    if (!window.desktopBridge) {
      return;
    }

    await window.desktopBridge.openReleasePage();
  }

  async function handleConfirmImport() {
    if (!desktopState || !pendingImportRequest) {
      return;
    }

    setImportBusy(true);
    try {
      if (pendingImportRequest.resource === "provider") {
        const created = await createProvider(
          {
            name: pendingImportRequest.data.name,
            base_url: pendingImportRequest.data.baseUrl,
            api_key: pendingImportRequest.data.apiKey,
            auth_mode: pendingImportRequest.data.authMode,
            extra_headers: {},
            claude_code_model_map: {
              opus: "",
              sonnet: "",
              haiku: ""
            }
          },
          desktopState.apiBase
        );
        setSelectedProvider(created);
        setProvidersRefreshToken((current) => current + 1);
        setView("providers");
        pushToast(t("importDeepLink.success.provider", { name: created.name }), "success");
        if (!pendingImportRequest.data.apiKey.trim()) {
          pushToast(t("importDeepLink.warning.emptyProviderApiKey"), "default");
        }
      } else {
        const payload: CreateLocalGatewayModelSourceInput = {
          name: pendingImportRequest.data.name,
          base_url: pendingImportRequest.data.baseUrl,
          api_key: pendingImportRequest.data.apiKey,
          provider_type: pendingImportRequest.data.providerType,
          default_model_id: pendingImportRequest.data.modelIds[0],
          exposed_model_ids: pendingImportRequest.data.modelIds.slice(1),
          enabled: true,
          position: 0
        };
        await createLocalGatewaySource(payload, desktopState.apiBase);
        await syncLocalGateway(desktopState.apiBase);
        setModelsRefreshToken((current) => current + 1);
        setView("models");
        pushToast(
          t("importDeepLink.success.model", { name: pendingImportRequest.data.name }),
          "success"
        );
      }

      setPendingImportRequest(null);
    } catch (error) {
      pushToast(
        error instanceof Error ? error.message : t("common.unknownError"),
        "error"
      );
    } finally {
      setImportBusy(false);
    }
  }

  if (!desktopState && window.desktopBridge) {
    return (
      <main className={pageShellClass}>
        <section className={heroClass}>
          <div>
            <p className={eyebrowClass}>Relay Switch</p>
            <h1 className={heroTitleClass}>{t("app.desktopBoot")}</h1>
            <p className={heroCopyClass}>{bootError ?? t("app.waitingRuntime")}</p>
          </div>
        </section>
      </main>
    );
  }

  return (
    <div className={appShellClass}>
      <ToastRegion items={toasts} onDismiss={dismissToast} />
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:left-3 focus:top-3 focus:z-[60] focus:rounded-lg focus:[background:var(--panel-glass)] focus:px-3 focus:py-2 focus:text-sm focus:text-[color:var(--color-text)] focus:shadow-[var(--shadow-panel)]"
      >
        Skip to main content
      </a>
      {pendingImportRequest ? (
        <div className={modalBackdropClass} role="presentation">
          <section
            className={`${modalPanelClass} max-w-2xl`}
            role="dialog"
            aria-modal="true"
            aria-label={t("importDeepLink.modal.title")}
          >
            <div className="space-y-1">
              <h2 className={sectionTitleClass}>{t("importDeepLink.modal.title")}</h2>
              <p className={sectionMetaClass}>
                {pendingImportRequest.resource === "provider"
                  ? t("importDeepLink.modal.providerSubtitle")
                  : t("importDeepLink.modal.modelSubtitle")}
              </p>
            </div>

            <div className="mt-4 grid gap-3 rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-4">
              <div>
                <p className={fieldLabelClass}>{t("importDeepLink.fields.resource")}</p>
                <p className="mt-1 text-sm text-[color:var(--color-text)]">
                  {pendingImportRequest.resource === "provider"
                    ? t("importDeepLink.resource.provider")
                    : t("importDeepLink.resource.model")}
                </p>
              </div>
              <div>
                <p className={fieldLabelClass}>{t("providers.form.name")}</p>
                <p className="mt-1 text-sm text-[color:var(--color-text)]">
                  {pendingImportRequest.data.name}
                </p>
              </div>
              <div>
                <p className={fieldLabelClass}>
                  {pendingImportRequest.resource === "provider"
                    ? t("providers.form.baseUrl")
                    : t("models.form.baseUrl")}
                </p>
                <p className="mt-1 break-all text-sm text-[color:var(--color-text)]">
                  {pendingImportRequest.data.baseUrl}
                </p>
              </div>
              {pendingImportRequest.resource === "model" ? (
                <div>
                  <p className={fieldLabelClass}>{t("models.form.models")}</p>
                  <p className="mt-1 break-all text-sm text-[color:var(--color-text)]">
                    {pendingImportRequest.data.modelIds.join(", ")}
                  </p>
                </div>
              ) : null}
              <div>
                <p className={fieldLabelClass}>{t("importDeepLink.fields.apiKeyMasked")}</p>
                <p className={`${monoClass} mt-1 break-all text-sm text-[color:var(--color-text)]`}>
                  {maskImportAPIKey(pendingImportRequest.data.apiKey) || t("importDeepLink.fields.apiKeyEmpty")}
                </p>
              </div>
            </div>

            <p className={`${metaClass} mt-4`}>{t("importDeepLink.modal.notice")}</p>
            {pendingImportRequest.resource === "provider" && !pendingImportRequest.data.apiKey.trim() ? (
              <p className="mt-3 rounded-[16px] border [border-color:var(--warning-border)] [background:var(--warning-soft)] px-4 py-3 text-sm leading-6 text-[color:var(--warning-text)]">
                {t("importDeepLink.warning.emptyProviderApiKey")}
              </p>
            ) : null}

            <div className="mt-4 flex flex-wrap items-center gap-2">
              <button
                type="button"
                className={buttonClass("primary")}
                onClick={() => void handleConfirmImport()}
                disabled={importBusy}
              >
                {importBusy ? t("importDeepLink.actions.importing") : t("importDeepLink.actions.import")}
              </button>
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={() => setPendingImportRequest(null)}
                disabled={importBusy}
              >
                {t("common.cancel")}
              </button>
            </div>
          </section>
        </div>
      ) : null}
      <div className={appBackdropClass} />
      <div className="relative mx-auto flex h-screen w-full max-w-[1600px] flex-row gap-3 overflow-hidden px-2.5 py-2.5 sm:px-3 sm:py-3 xl:px-4">
        <aside
          className={`${glassPanelClass} flex h-[calc(100vh-1.25rem)] w-[72px] min-w-[72px] flex-col items-center gap-3 overflow-visible p-3 sm:h-[calc(100vh-1.5rem)] xl:h-[calc(100vh-2rem)]`}
        >
          <div className="flex flex-col items-center">
            <div className="group relative flex items-center justify-center">
              <img
                src={appIcon}
                alt="Relay Switch"
                className="h-10 w-10 rounded-lg shadow-[0_8px_18px_rgba(15,23,42,0.14)]"
              />
              <div className="sr-only">
                <p className={`${eyebrowClass} mb-1`}>AI Gateway</p>
                <h2>Relay Switch</h2>
              </div>
              <span className="pointer-events-none absolute left-[calc(100%+0.5rem)] top-1/2 z-30 hidden -translate-y-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-popup)] px-2.5 py-1.5 text-xs font-medium text-[color:var(--color-text)] shadow-[var(--shadow-panel)] group-hover:block group-focus-within:block">
                Relay Switch
              </span>
            </div>
          </div>

          <nav className="grid w-full justify-items-center gap-1.5">
            {navItems.map(({ id, label, icon }) => (
              <div key={id} className="group relative flex justify-center">
                <button
                  type="button"
                  className={`${navButtonClass(view === id)} h-10 w-10 justify-center px-0`}
                  onClick={() => {
                    setView(id as typeof view);
                  }}
                  aria-label={label}
                  title={label}
                >
                  {icon}
                </button>
                <span className="pointer-events-none absolute left-[calc(100%+0.5rem)] top-1/2 z-30 hidden -translate-y-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-popup)] px-2.5 py-1.5 text-xs font-medium text-[color:var(--color-text)] shadow-[var(--shadow-panel)] group-hover:block group-focus-within:block">
                  {label}
                </span>
              </div>
            ))}
          </nav>

          <div className="grid justify-items-center gap-2">
            <label className="group relative block">
              <span className="sr-only">{t("app.language")}</span>
              <select
                className="h-10 w-10 cursor-pointer appearance-none rounded-lg border text-center text-xs font-semibold [border-color:var(--border-soft)] [background:var(--panel-solid)] text-[color:var(--color-text)] outline-none transition hover:[border-color:var(--border-strong)] focus:ring-2 focus:ring-[color:var(--accent-strong)]/20"
                value={locale}
                onChange={(event) => setLocale(event.target.value as typeof locale)}
                aria-label={t("app.language")}
                title={t("app.language")}
              >
                {Object.keys(localeLabels).map((key) => (
                  <option key={key} value={key}>
                    {key === "zh" ? "中" : "EN"}
                  </option>
                ))}
              </select>
              <span className="pointer-events-none absolute left-[calc(100%+0.5rem)] top-1/2 z-30 hidden -translate-y-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-popup)] px-2.5 py-1.5 text-xs font-medium text-[color:var(--color-text)] shadow-[var(--shadow-panel)] group-hover:block group-focus-within:block">
                {t("app.language")}
              </span>
            </label>

            <div className="group relative shrink-0">
              <span className="sr-only">
                {t("app.theme")}
              </span>
                <button
                  type="button"
                  className={`${buttonClass("secondary")} h-10 w-10 px-0`}
                  onClick={() => toggleTheme()}
                  aria-label={resolvedTheme === "dark" ? t("app.themeLight") : t("app.themeDark")}
                  title={resolvedTheme === "dark" ? t("app.themeLight") : t("app.themeDark")}
                >
                  {resolvedTheme === "dark" ? (
                    <svg className="h-5 w-5 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                      <path d="M6.8 5.4 5.4 4l-1.4 1.4 1.4 1.4zM12 2h-1v3h2V2zm6.6 3.4L20 4l-1.4-1.4-1.4 1.4zM19 11v2h3v-2zm-7 10h1v-3h-2v3zm6.6-2.4 1.4 1.4 1.4-1.4-1.4-1.4zM2 11v2h3v-2zm3.4 7.6L4 20l1.4 1.4 1.4-1.4zM12 7a5 5 0 1 0 0 10 5 5 0 0 0 0-10" />
                    </svg>
                  ) : (
                    <svg className="h-5 w-5 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                      <path d="M20 14.2A8 8 0 0 1 9.8 4 8 8 0 1 0 20 14.2" />
                    </svg>
                  )}
                </button>
              <span className="pointer-events-none absolute left-[calc(100%+0.5rem)] top-1/2 z-30 hidden -translate-y-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-popup)] px-2.5 py-1.5 text-xs font-medium text-[color:var(--color-text)] shadow-[var(--shadow-panel)] group-hover:block group-focus-within:block">
                {resolvedTheme === "dark" ? t("app.themeLight") : t("app.themeDark")}
              </span>
            </div>
          </div>

          <div className="mt-auto grid justify-items-center gap-2">
            {updates && showUpdateReminder ? (
              <div className="group relative">
                <button
                  type="button"
                  className={`${buttonClass(
                    desktopState?.platform === "darwin" || updates.status === "downloaded"
                      ? "primary"
                      : "secondary"
                  )} h-10 w-10 px-0`}
                  onClick={() =>
                    void (desktopState?.platform === "darwin"
                      ? handleOpenReleasePage()
                      : updates.status === "downloaded"
                        ? handleQuitAndInstallUpdate()
                        : handleDownloadUpdate())
                  }
                  aria-label={
                    updates.status === "downloaded"
                      ? t("updates.card.installReady")
                      : t("updates.card.availableCompact")
                  }
                  title={
                    updates.status === "downloaded"
                      ? t("updates.card.installReady")
                      : t("updates.card.availableCompact")
                  }
                >
                  <span className={statusDotClass("warning")} />
                </button>
                <span className="pointer-events-none absolute left-[calc(100%+0.5rem)] top-1/2 z-30 hidden -translate-y-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-popup)] px-2.5 py-1.5 text-xs font-medium text-[color:var(--color-text)] shadow-[var(--shadow-panel)] group-hover:block group-focus-within:block">
                  {updates.status === "downloaded"
                    ? t("updates.card.installReady")
                    : t("updates.card.availableCompact")}
                </span>
              </div>
            ) : null}
            <div className="group relative flex h-10 w-10 items-center justify-center rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-solid)]">
                <span
                  className={statusDotClass(
                    desktopState?.core.running ? "success" : "danger"
                  )}
                />
              <span className="pointer-events-none absolute left-[calc(100%+0.5rem)] top-1/2 z-30 hidden -translate-y-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-popup)] px-2.5 py-1.5 text-xs font-medium text-[color:var(--color-text)] shadow-[var(--shadow-panel)] group-hover:block group-focus-within:block">
                {t("app.runtimeChip", {
                  status: desktopState?.core.running ? t("app.coreRunning") : t("app.coreStopped"),
                  port: desktopState?.core.port ?? "-"
                })}
              </span>
            </div>
          </div>
        </aside>

        <section id="main-content" className="min-h-0 min-w-0 flex-1 overflow-y-auto">
          {coreStartError ? (
            <div className="mx-auto mb-3 flex w-full max-w-[1600px] flex-col gap-3 rounded-[18px] border [border-color:var(--danger-border)] [background:var(--danger-soft)] px-4 py-3 text-sm text-[color:var(--danger-text)] shadow-[var(--shadow-soft)] sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0">
                <p className="font-semibold">{t("app.coreStartFailedTitle")}</p>
                <p className="mt-1 leading-5">{coreStartError}</p>
                {coreLastError && coreStartError !== coreLastError ? (
                  <p className={`${monoClass} mt-2 text-[11px] text-[color:var(--danger-text)]`}>
                    {coreLastError}
                  </p>
                ) : null}
              </div>
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={() => setView("settings")}
              >
                {t("app.action.changePort")}
              </button>
            </div>
          ) : null}
          {view === "providers" ? (
            <ProvidersPage
              desktopState={desktopState}
              apiBase={desktopState?.apiBase}
              refreshToken={providersRefreshToken}
              selectedProviderId={selectedProvider?.id ?? null}
              onSelectedProviderChange={setSelectedProvider}
            />
          ) : view === "models" ? (
            <ModelsPage apiBase={desktopState?.apiBase} refreshToken={modelsRefreshToken} />
          ) : view === "tools" ? (
            <ToolsPage
              desktopState={desktopState}
              onCopyText={async (text) => {
                if (!window.desktopBridge) {
                  return;
                }

                await window.desktopBridge.copyText(text);
              }}
            />
          ) : view === "logs" ? (
            <LogsPage apiBase={desktopState?.apiBase} />
          ) : (
            <SettingsPage
              desktopState={desktopState}
              onUpdateCorePort={async (port) => {
                if (!window.desktopBridge) {
                  return;
                }

                const response = await window.desktopBridge.updateCorePort(port);
                setDesktopState((current) =>
                  current
                    ? {
                        ...current,
                        config: response.config,
                        updates: response.updates,
                        apiBase: response.core.apiBase,
                        core: response.core
                      }
                    : null
                );
              }}
              onUpdateLocalGatewayPort={async (port) => {
                if (!window.desktopBridge) {
                  return;
                }

                const response = await window.desktopBridge.updateLocalGatewayPort(port);
                setDesktopState((current) =>
                  current
                    ? {
                        ...current,
                        config: response.config,
                        updates: response.updates,
                        apiBase: response.core.apiBase,
                        core: response.core
                      }
                    : null
                );
              }}
              onUpdateLaunchSettings={async (settings) => {
                if (!window.desktopBridge) {
                  return;
                }

                const response = await window.desktopBridge.updateLaunchSettings(settings);
                setDesktopState((current) =>
                  current
                    ? {
                        ...current,
                        config: response.config,
                        updates: response.updates,
                        apiBase: response.core.apiBase,
                        core: response.core
                      }
                    : null
                );
              }}
              onCheckUpdates={async () => {
                await handleCheckUpdates();
              }}
              onDownloadUpdate={async () => {
                await handleDownloadUpdate();
              }}
              onQuitAndInstallUpdate={async () => {
                await handleQuitAndInstallUpdate();
              }}
              onOpenReleasePage={async () => {
                await handleOpenReleasePage();
              }}
              onOpenProjectPage={async () => {
                if (!window.desktopBridge) {
                  return;
                }

                await window.desktopBridge.openProjectPage();
              }}
              onCoreRestart={async () => {
                if (!window.desktopBridge) {
                  return;
                }

                const response = await window.desktopBridge.restartCore();
                setDesktopState((current) =>
                  current
                    ? {
                        ...current,
                        config: response.config,
                        updates: response.updates,
                        apiBase: response.core.apiBase,
                        core: response.core
                      }
                    : null
                );
              }}
            />
          )}
        </section>
      </div>
    </div>
  );
}
