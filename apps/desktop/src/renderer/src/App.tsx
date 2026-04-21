import { useEffect, useState } from "react";
import { useI18n } from "./i18n/i18n-provider";
import { LogsPage } from "./pages/logs-page";
import { ModelsPage } from "./pages/models-page";
import { ProvidersPage } from "./pages/providers-page";
import { SettingsPage } from "./pages/settings-page";
import type { Provider } from "./types/provider";

interface DesktopState {
  ok: boolean;
  runtime: string;
  platform: string;
  apiBase: string;
  config: {
    apiPort: number;
    apiPortSource: "default" | "config" | "env";
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

export default function App() {
  const { locale, localeLabels, setLocale, t } = useI18n();
  const [desktopState, setDesktopState] = useState<DesktopState | null>(null);
  const [view, setView] = useState<"providers" | "models" | "logs" | "settings">("providers");
  const [bootError, setBootError] = useState<string | null>(null);
  const [selectedProvider, setSelectedProvider] = useState<Provider | null>(null);

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

  if (!desktopState && window.desktopBridge) {
    return (
      <main className="page-shell">
        <section className="hero">
          <div>
            <p className="eyebrow">Clash for AI</p>
            <h1>{t("app.desktopBoot")}</h1>
            <p className="subcopy">{bootError ?? t("app.waitingRuntime")}</p>
          </div>
        </section>
      </main>
    );
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div>
          <p className="eyebrow">Clash for AI</p>
          <h2 className="sidebar-title">{t("app.desktopGateway")}</h2>
          <p className="meta">
            {selectedProvider
              ? t("app.currentProvider", { name: selectedProvider.name })
              : t("app.selectProviderHint")}
          </p>
        </div>

        <nav className="sidebar-nav">
          {[
            ["providers", t("app.nav.providers")],
            ["models", t("app.nav.models")],
            ["logs", t("app.nav.logs")],
            ["settings", t("app.nav.settings")]
          ].map(([id, label]) => (
            <button
              key={id}
              type="button"
              className={view === id ? "nav-button active-nav" : "nav-button"}
              onClick={() => {
                setView(id as typeof view);
              }}
            >
              {label}
            </button>
          ))}
        </nav>

        <div className="sidebar-runtime">
          <label className="sidebar-language">
            <span className="settings-label">{t("app.language")}</span>
            <select
              className="settings-input sidebar-language-select"
              value={locale}
              onChange={(event) => setLocale(event.target.value as typeof locale)}
            >
              {Object.entries(localeLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </label>
        </div>

        <div className="sidebar-runtime">
          <span className="runtime-chip">
            {t("app.runtimeChip", {
              status: desktopState?.core.running ? t("app.coreRunning") : t("app.coreStopped"),
              port: desktopState?.core.port ?? "-"
            })}
          </span>
        </div>
      </aside>

      <section className="content-shell">
        {view === "providers" ? (
          <ProvidersPage
            desktopState={desktopState}
            apiBase={desktopState?.apiBase}
            selectedProviderId={selectedProvider?.id ?? null}
            onSelectedProviderChange={setSelectedProvider}
          />
        ) : view === "models" ? (
          <ModelsPage
            apiBase={desktopState?.apiBase}
            selectedProvider={selectedProvider}
            onSelectedProviderChange={setSelectedProvider}
          />
        ) : view === "logs" ? (
          <LogsPage apiBase={desktopState?.apiBase} />
        ) : (
          <SettingsPage
            desktopState={desktopState}
            onCopyText={async (text) => {
              if (!window.desktopBridge) {
                return;
              }

              await window.desktopBridge.copyText(text);
            }}
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
            onCheckUpdates={async () => {
              if (!window.desktopBridge) {
                return;
              }

              const updates = await window.desktopBridge.checkUpdates();
              setDesktopState((current) => (current ? { ...current, updates } : current));
            }}
            onDownloadUpdate={async () => {
              if (!window.desktopBridge) {
                return;
              }

              const updates = await window.desktopBridge.downloadUpdate();
              setDesktopState((current) => (current ? { ...current, updates } : current));
            }}
            onQuitAndInstallUpdate={async () => {
              if (!window.desktopBridge) {
                return;
              }

              const updates = await window.desktopBridge.quitAndInstallUpdate();
              setDesktopState((current) => (current ? { ...current, updates } : current));
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
  );
}
