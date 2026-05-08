import { Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { useEffect, useMemo, useState } from "react";
import { useI18n } from "./i18n/i18n-provider";
import { useTheme } from "./theme/theme-provider";
import { ProvidersPage } from "./pages/providers-page";
import { ModelsPage } from "./pages/models-page";
import { LogsPage } from "./pages/logs-page";
import { SettingsPage } from "./pages/settings-page";
import { ToolsPage } from "./pages/tools-page";
import {
  getHealth,
  getLocalGatewayRuntime,
  getRuntime,
  type WebRuntimeOverview
} from "./services/api";
import {
  appBackdropClass,
  appShellClass,
  buttonClass,
  fieldLabelClass,
  glassPanelClass,
  metaClass,
  navButtonClass,
  pageShellClass,
  statusPillClass
} from "./ui";

const navIDs = ["providers", "models", "logs", "settings", "tools"] as const;

export default function App() {
  const { locale, localeLabels, setLocale, t } = useI18n();
  const { resolvedTheme, toggleTheme } = useTheme();
  const navigate = useNavigate();
  const location = useLocation();
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
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

  const navItems = useMemo(
    () => [
      { id: "providers", path: "/providers", label: t("app.nav.providers") },
      { id: "models", path: "/models", label: t("app.nav.models") },
      { id: "logs", path: "/logs", label: t("app.nav.logs") },
      { id: "settings", path: "/settings", label: t("app.nav.settings") },
      { id: "tools", path: "/tools", label: t("app.nav.tools") }
    ],
    [t]
  );

  useEffect(() => {
    setMobileNavOpen(false);
  }, [location.pathname]);

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
      <div className={appBackdropClass} />
      <div className="relative mx-auto h-full min-h-0 w-full max-w-[1600px] px-3 py-3 sm:px-4 sm:py-4 xl:px-6">
        <div className={`${glassPanelClass} mb-4 flex items-center justify-between gap-3 px-4 py-3 xl:hidden`}>
          <div className="min-w-0">
            <p className={fieldLabelClass}>Clash for AI</p>
            <p className="text-base font-semibold text-[color:var(--color-heading)]">Clash for AI Web</p>
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
        <div className="grid h-full min-h-0 gap-4 xl:grid-cols-[240px_minmax(0,1fr)]">
          <aside
            className={`${glassPanelClass} ${
              mobileNavOpen ? "flex" : "hidden"
            } min-h-0 flex-col gap-4 overflow-y-auto p-4 xl:flex`}
          >
            <div className="space-y-2">
              <p className={fieldLabelClass}>Clash for AI</p>
              <h1 className="text-2xl font-semibold tracking-[-0.04em] text-[color:var(--color-heading)]">
                Clash for AI Web
              </h1>
              <p className={metaClass}>{t("tools.overview.subtitle")}</p>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <span className={statusPillClass("warning")}>WSL / Linux server</span>
              <span className={statusPillClass()}>{t("settings.value.browser")}</span>
            </div>

            <div className="grid gap-2 rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3">
              <div>
                <p className={fieldLabelClass}>Runtime</p>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <span className={statusPillClass(runtimeOverview.core.available ? "success" : "danger")}>
                  {runtimeOverview.core.available ? "Core ready" : "Core unavailable"}
                </span>
                <span className={statusPillClass(localGatewayTone)}>
                  {runtimeOverview.localGateway.running && runtimeOverview.localGateway.healthy
                    ? "Gateway ready"
                    : runtimeOverview.localGateway.last_error
                      ? "Gateway error"
                      : runtimeOverview.localGateway.configured
                        ? "Gateway starting"
                        : "Gateway not configured"}
                </span>
              </div>
              <div className="space-y-1">
                <p className={metaClass}>
                  {runtimeOverview.environment
                    ? `${runtimeOverview.environment.os} / ${runtimeOverview.environment.arch}${
                        runtimeOverview.environment.is_wsl ? " / WSL" : ""
                      }`
                    : "Runtime information unavailable"}
                </p>
                {runtimeOverview.localGateway.api_base ? (
                  <p className={metaClass}>Gateway: {runtimeOverview.localGateway.api_base}</p>
                ) : null}
                {runtimeOverview.localGateway.last_error ? (
                  <p className="text-sm text-[color:var(--danger-text)]">
                    {runtimeOverview.localGateway.last_error}
                  </p>
                ) : null}
              </div>
            </div>

            <nav className="grid gap-2">
              {navItems.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  className={navButtonClass(location.pathname === item.path || (item.path === "/providers" && location.pathname === "/"))}
                  onClick={() => navigate(item.path)}
                >
                  {item.label}
                </button>
              ))}
            </nav>

            <div className="mt-auto space-y-3">
              <div className="grid gap-2 rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-2.5">
                <div className="flex items-center justify-between gap-3">
                  <label className="min-w-0 flex-1">
                    <span className="mb-2 block text-[11px] font-semibold uppercase tracking-[0.22em] text-[color:var(--color-subtle)]">
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

                  <div className="shrink-0">
                    <span className="mb-2 block text-[11px] font-semibold uppercase tracking-[0.22em] text-[color:var(--color-subtle)]">
                      {t("app.theme")}
                    </span>
                    <button
                      type="button"
                      className={buttonClass("secondary")}
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
                  </div>
                </div>
              </div>
            </div>
          </aside>

          <main className="min-h-0 min-w-0 overflow-y-auto">
            <Routes>
              <Route
                path="/"
                element={<ProvidersPage selectedProviderId={null} onSelectedProviderChange={() => {}} />}
              />
              <Route
                path="/providers"
                element={<ProvidersPage selectedProviderId={null} onSelectedProviderChange={() => {}} />}
              />
              <Route path="/models" element={<ModelsPage />} />
              <Route path="/logs" element={<LogsPage />} />
              <Route path="/settings" element={<SettingsPage />} />
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
