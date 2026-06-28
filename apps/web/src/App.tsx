import { Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useI18n } from "./i18n/i18n-provider";
import { useTheme } from "./theme/theme-provider";
import { ProvidersPage } from "./pages/providers-page";
import { ModelsPage } from "./pages/models-page";
import { LogsPage } from "./pages/logs-page";
import { SettingsPage } from "./pages/settings-page";
import { ToolsPage } from "./pages/tools-page";
import { ToastRegion, type ToastItem } from "./components/toast-region";
import {
  getHealth,
  getLatestGitHubRelease,
  getLocalGatewayRuntime,
  getReleaseMetadata,
  getRuntime,
  type GitHubRelease,
  type ReleaseMetadata,
  type WebRuntimeOverview
} from "./services/api";
import { getModelPresets } from "./services/model-presets";
import { compareVersions } from "./utils/version";
import {
  appBackdropClass,
  appShellClass,
  buttonClass,
  fieldLabelClass,
  glassPanelClass,
  metaClass,
  navButtonClass,
  pageShellClass,
  statusDotClass,
  statusPillClass
} from "./ui";

function WebNavIcon({ id }: { id: string }) {
  const className = "h-4 w-4 fill-current";

  if (id === "providers") {
    return (
      <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
        <path d="M4 7.5A2.5 2.5 0 0 1 6.5 5h11A2.5 2.5 0 0 1 20 7.5v9A2.5 2.5 0 0 1 17.5 19h-11A2.5 2.5 0 0 1 4 16.5zM6.5 7a.5.5 0 0 0-.5.5V10h12V7.5a.5.5 0 0 0-.5-.5zM18 12H6v4.5a.5.5 0 0 0 .5.5h11a.5.5 0 0 0 .5-.5z" />
      </svg>
    );
  }

  if (id === "models") {
    return (
      <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
        <path d="M12 3 4 7v10l8 4 8-4V7zm0 2.2L17.8 8 12 10.8 6.2 8zM6 9.6l5 2.5v6.2l-5-2.5zm7 8.7v-6.2l5-2.5v6.2z" />
      </svg>
    );
  }

  if (id === "tools") {
    return (
      <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
        <path d="M13.4 3.4a2 2 0 0 1 2.8 0l4.4 4.4a2 2 0 0 1 0 2.8l-2.1 2.1-7.2-7.2zM10.1 6.7 3 13.8V21h7.2l7.1-7.1zM6 18H5v-1l7.4-7.4 1 1z" />
      </svg>
    );
  }

  if (id === "logs") {
    return (
      <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
        <path d="M5 5h14v2H5zm0 6h14v2H5zm0 6h9v2H5z" />
      </svg>
    );
  }

  return (
    <svg className={className} viewBox="0 0 24 24" aria-hidden="true">
      <path d="m19.4 13 .1-1-.1-1 2-1.6-2-3.4-2.4 1a7 7 0 0 0-1.7-1l-.4-2.5h-4l-.4 2.5a7 7 0 0 0-1.7 1l-2.4-1-2 3.4 2 1.6a8 8 0 0 0 0 2l-2 1.6 2 3.4 2.4-1a7 7 0 0 0 1.7 1l.4 2.5h4l.4-2.5a7 7 0 0 0 1.7-1l2.4 1 2-3.4zM12 15.5A3.5 3.5 0 1 1 12 8a3.5 3.5 0 0 1 0 7.5" />
    </svg>
  );
}

export default function App() {
  const { locale, localeLabels, setLocale, t } = useI18n();
  const { resolvedTheme, toggleTheme } = useTheme();
  const navigate = useNavigate();
  const location = useLocation();
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [releaseMetadata, setReleaseMetadata] = useState<ReleaseMetadata | null>(null);
  const [latestRelease, setLatestRelease] = useState<GitHubRelease | null>(null);
  const [latestReleaseError, setLatestReleaseError] = useState<string | null>(null);
  const [runtimeOverview, setRuntimeOverview] = useState<WebRuntimeOverview>({
    core: {
      available: false
    },
    environment: null,
    localGateway: {
      configured: false,
      running: false,
      healthy: false
    }
  });
  const ignoreSelectedProviderChange = useCallback(() => {}, []);

  const dismissToast = useCallback((id: string) => {
    setToasts((current) => current.filter((item) => item.id !== id));
  }, []);

  const pushToast = useCallback((message: string, tone: ToastItem["tone"]) => {
    setToasts((current) => [
      ...current,
      {
        id: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
        message,
        tone
      }
    ]);
  }, []);

  const navItems = useMemo(
    () => [
      { id: "providers", path: "/providers", label: t("app.nav.providers") },
      { id: "models", path: "/models", label: t("app.nav.models") },
      { id: "tools", path: "/tools", label: t("app.nav.tools") },
      { id: "logs", path: "/logs", label: t("app.nav.logs") },
      { id: "settings", path: "/settings", label: t("app.nav.settings") },
    ],
    [t]
  );

  useEffect(() => {
    setMobileNavOpen(false);
  }, [location.pathname]);

  useEffect(() => {
    if (!import.meta.env.DEV) {
      return;
    }

    void getModelPresets();
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function checkLatestRelease() {
      const [localRelease, remoteRelease] = await Promise.all([
        getReleaseMetadata().catch(() => null),
        getLatestGitHubRelease()
      ]);

      if (cancelled) {
        return;
      }

      setReleaseMetadata(localRelease);
      setLatestRelease(remoteRelease);
      setLatestReleaseError(null);

      const currentVersion = localRelease?.available ? localRelease.release?.release_version : undefined;
      if (compareVersions(currentVersion, remoteRelease.tag_name) > 0) {
        pushToast(t("updates.toast.available", { version: remoteRelease.tag_name }), "success");
      }
    }

    void checkLatestRelease().catch((error) => {
      if (cancelled) {
        return;
      }

      setLatestRelease(null);
      setLatestReleaseError(error instanceof Error ? error.message : t("settings.updates.latestError"));
    });

    return () => {
      cancelled = true;
    };
  }, [pushToast, t]);

  useEffect(() => {
    let cancelled = false;

    async function syncRuntimeOverview() {
      try {
        const [health, environment, localGateway] = await Promise.all([
          getHealth().catch(() => null),
          getRuntime().catch(() => null),
          getLocalGatewayRuntime().catch(() => null)
        ]);

        if (cancelled) {
          return;
        }

        setRuntimeOverview({
          core: {
            available: health?.status === "ok",
            version: health?.version
          },
          environment,
          localGateway: {
            configured: Boolean(localGateway?.runtime.runtime_kind),
            running: localGateway?.runtime.running ?? false,
            healthy: localGateway?.runtime.healthy ?? false,
            state: localGateway?.runtime.state,
            api_base: localGateway?.runtime.api_base,
            last_error: localGateway?.runtime.last_error
          }
        });
      } catch {
        if (cancelled) {
          return;
        }
      }
    }

    void syncRuntimeOverview();
    const intervalId = window.setInterval(() => {
      void syncRuntimeOverview();
    }, 4000);

    return () => {
      cancelled = true;
      window.clearInterval(intervalId);
    };
  }, []);

  const localGatewayTone =
    runtimeOverview.localGateway.running && runtimeOverview.localGateway.healthy
      ? "success"
      : runtimeOverview.localGateway.last_error
        ? "danger"
        : "warning";

  return (
    <div className={appShellClass}>
      <ToastRegion items={toasts} onDismiss={dismissToast} />
      <div className={appBackdropClass} />
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:left-3 focus:top-3 focus:z-[60] focus:rounded-lg focus:[background:var(--panel-glass)] focus:px-3 focus:py-2 focus:text-sm focus:text-[color:var(--color-text)] focus:shadow-[var(--shadow-panel)]"
      >
        Skip to main content
      </a>
      <div className="relative mx-auto h-full min-h-0 w-full max-w-[1600px] px-2.5 py-2.5 sm:px-3 sm:py-3 xl:px-4">
        <div className={`${glassPanelClass} mb-3 flex items-center justify-between gap-3 px-3 py-2 xl:hidden`}>
          <div className="min-w-0">
            <p className={fieldLabelClass}>Relay Switch</p>
            <p className="truncate text-sm font-semibold text-[color:var(--color-heading)]">Relay Switch Web</p>
          </div>
          <button
            type="button"
            className={buttonClass("secondary")}
            onClick={() => setMobileNavOpen((current) => !current)}
            aria-label={mobileNavOpen ? "Close navigation" : "Open navigation"}
            title={mobileNavOpen ? "Close navigation" : "Open navigation"}
          >
            {mobileNavOpen ? "Close" : "Menu"}
          </button>
        </div>
        <div className="grid h-full min-h-0 gap-3 xl:grid-cols-[72px_minmax(0,1fr)]">
          <aside
            className={`${glassPanelClass} ${
              mobileNavOpen ? "flex" : "hidden"
            } min-h-0 flex-col gap-3 overflow-visible p-3 xl:flex xl:items-center`}
          >
            <div className="space-y-1.5 xl:flex xl:flex-col xl:items-center xl:space-y-0">
              <p className={`${fieldLabelClass} xl:hidden`}>Relay Switch</p>
              <div className="hidden h-10 w-10 items-center justify-center rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-soft)] text-sm font-semibold text-[color:var(--accent)] xl:flex">
                RS
              </div>
              <h1 className="text-lg font-semibold text-[color:var(--color-heading)] xl:sr-only">
                Relay Switch Web
              </h1>
              <p className={`${metaClass} xl:hidden`}>{t("tools.overview.subtitle")}</p>
            </div>

            <div className="flex flex-wrap items-center gap-2 xl:flex-col">
              <span className={`${statusPillClass("warning")} xl:hidden`}>WSL / Linux server</span>
              <span className={`${statusPillClass()} xl:hidden`}>{t("settings.value.browser")}</span>
              <span className="group relative hidden h-8 w-8 items-center justify-center rounded-lg border [border-color:var(--warning-border)] [background:var(--warning-soft)] xl:inline-flex">
                <span className={statusDotClass("warning")} />
                <span className="pointer-events-none absolute left-[calc(100%+0.5rem)] top-1/2 z-30 hidden -translate-y-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-popup)] px-2.5 py-1.5 text-xs font-medium text-[color:var(--color-text)] shadow-[var(--shadow-panel)] group-hover:block">
                  WSL / Linux server
                </span>
              </span>
              <span className="group relative hidden h-8 w-8 items-center justify-center rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-solid)] xl:inline-flex">
                <span className={statusDotClass()} />
                <span className="pointer-events-none absolute left-[calc(100%+0.5rem)] top-1/2 z-30 hidden -translate-y-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-popup)] px-2.5 py-1.5 text-xs font-medium text-[color:var(--color-text)] shadow-[var(--shadow-panel)] group-hover:block">
                  {t("settings.value.browser")}
                </span>
              </span>
            </div>

            <nav className="grid gap-1.5 xl:w-full xl:justify-items-center">
              {navItems.map((item) => (
                <div key={item.id} className="group relative w-full xl:flex xl:justify-center">
                  <button
                    type="button"
                    className={`${navButtonClass(location.pathname === item.path || (item.path === "/providers" && location.pathname === "/"))} xl:h-10 xl:w-10 xl:justify-center xl:px-0`}
                    onClick={() => navigate(item.path)}
                    aria-label={item.label}
                    title={item.label}
                  >
                    <span className="hidden xl:block">
                      <WebNavIcon id={item.id} />
                    </span>
                    <span className="xl:hidden">{item.label}</span>
                  </button>
                  <span className="pointer-events-none absolute left-[calc(100%+0.5rem)] top-1/2 z-30 hidden -translate-y-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-popup)] px-2.5 py-1.5 text-xs font-medium text-[color:var(--color-text)] shadow-[var(--shadow-panel)] group-hover:block group-focus-within:block">
                    {item.label}
                  </span>
                </div>
              ))}
            </nav>

            <div className="mt-auto space-y-2.5 xl:w-full">
              <div className="grid gap-2 rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-2.5 xl:border-0 xl:bg-transparent xl:p-0">
                <div className="flex items-center justify-between gap-3 xl:flex-col xl:justify-center xl:gap-2">
                  <label className="min-w-0 flex-1 xl:hidden">
                    <span className="mb-1.5 block text-[11px] font-semibold uppercase tracking-[0.14em] text-[color:var(--color-subtle)]">
                      {t("app.language")}
                    </span>
                    <select
                      className="min-h-9 w-full rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-input)] px-3 py-2 text-sm"
                      value={locale}
                      onChange={(event) => setLocale(event.target.value as typeof locale)}
                    >
                      {Object.entries(localeLabels).map(([key, label]) => (
                        <option key={key} value={key}>
                          {label}
                        </option>
                      ))}
                    </select>
                  </label>

                  <label className="group relative hidden xl:block">
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
                    <span className="mb-1.5 block text-[11px] font-semibold uppercase tracking-[0.14em] text-[color:var(--color-subtle)] xl:sr-only xl:mb-0">
                      {t("app.theme")}
                    </span>
                    <button
                      type="button"
                      className={`${buttonClass("secondary")} xl:h-10 xl:w-10 xl:px-0`}
                      onClick={toggleTheme}
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
              </div>

              <div className="grid gap-2 rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3 xl:place-items-center xl:border-0 xl:bg-transparent xl:p-0">
                <div>
                  <p className={`${fieldLabelClass} xl:sr-only`}>Runtime</p>
                </div>
                <div className="flex flex-wrap items-center gap-3 xl:flex-col xl:gap-2">
                  <span className="group relative inline-flex items-center gap-2 text-sm text-[color:var(--color-text)]">
                    <span className={statusDotClass(runtimeOverview.core.available ? "success" : "danger")} />
                    <span className="xl:sr-only">Core</span>
                    <span className="pointer-events-none absolute left-[calc(100%+0.75rem)] top-1/2 z-30 hidden -translate-y-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-popup)] px-2.5 py-1.5 text-xs font-medium text-[color:var(--color-text)] shadow-[var(--shadow-panel)] group-hover:block">
                      Core
                    </span>
                  </span>
                  <span className="group relative inline-flex items-center gap-2 text-sm text-[color:var(--color-text)]">
                    <span className={statusDotClass(localGatewayTone === "success" ? "success" : "danger")} />
                    <span className="xl:sr-only">Gateway</span>
                    <span className="pointer-events-none absolute left-[calc(100%+0.75rem)] top-1/2 z-30 hidden -translate-y-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-popup)] px-2.5 py-1.5 text-xs font-medium text-[color:var(--color-text)] shadow-[var(--shadow-panel)] group-hover:block">
                      Gateway
                    </span>
                  </span>
                </div>
                <div className="space-y-1 xl:hidden">
                  <p className={metaClass}>
                    {runtimeOverview.environment
                      ? `${runtimeOverview.environment.os} / ${runtimeOverview.environment.arch}${
                        runtimeOverview.environment.is_wsl ? " / WSL" : ""
                      }`
                      : "Runtime information unavailable"}
                  </p>
                  {runtimeOverview.localGateway.last_error ? (
                    <p className="text-sm text-[color:var(--danger-text)]">
                      {runtimeOverview.localGateway.last_error}
                    </p>
                  ) : null}
                </div>
              </div>
            </div>
          </aside>

          <main id="main-content" className="min-h-0 min-w-0 overflow-y-auto">
            <Routes>
              <Route
                path="/"
                element={
                  <ProvidersPage
                    selectedProviderId={null}
                    onSelectedProviderChange={ignoreSelectedProviderChange}
                  />
                }
              />
              <Route
                path="/providers"
                element={
                  <ProvidersPage
                    selectedProviderId={null}
                    onSelectedProviderChange={ignoreSelectedProviderChange}
                  />
                }
              />
              <Route path="/models" element={<ModelsPage />} />
              <Route path="/logs" element={<LogsPage />} />
              <Route
                path="/settings"
                element={
                  <SettingsPage
                    releaseMetadata={releaseMetadata}
                    latestRelease={latestRelease}
                    latestReleaseError={latestReleaseError}
                  />
                }
              />
              <Route
                path="/tools"
                element={<ToolsPage onCopyText={(text) => navigator.clipboard.writeText(text)} />}
              />
            </Routes>
          </main>
        </div>
      </div>
    </div>
  );
}
