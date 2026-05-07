import { NavLink, Route, Routes } from "react-router-dom";
import { useMemo } from "react";
import { useI18n } from "./i18n/i18n-provider";
import { useTheme } from "./theme/theme-provider";
import { ProvidersPage } from "./pages/providers-page";
import { ModelsPage } from "./pages/models-page";
import { LogsPage } from "./pages/logs-page";
import { SettingsPage } from "./pages/settings-page";
import { ToolsPage } from "./pages/tools-page";
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

  return (
    <div className={appShellClass}>
      <div className={appBackdropClass} />
      <div className={pageShellClass}>
        <div className="grid min-h-full gap-4 xl:grid-cols-[240px_minmax(0,1fr)]">
          <aside className={`${glassPanelClass} flex flex-col gap-4 p-4`}>
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

            <nav className="grid gap-2">
              {navItems.map((item) => (
                <NavLink key={item.id} to={item.path} className={({ isActive }) => navButtonClass(isActive)}>
                  {item.label}
                </NavLink>
              ))}
            </nav>

            <div className="mt-auto space-y-3">
              <label className="flex flex-col gap-2">
                <span className={fieldLabelClass}>{t("app.language")}</span>
                <select
                  className="min-h-9 rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-input)] px-3 py-2 text-sm"
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

              <button type="button" className={buttonClass("secondary")} onClick={toggleTheme}>
                {resolvedTheme === "dark" ? t("app.themeLight") : t("app.themeDark")}
              </button>
            </div>
          </aside>

          <main className="min-w-0">
            <Routes>
              <Route
                path="/"
                element={<ProvidersPage desktopState={null} selectedProviderId={null} onSelectedProviderChange={() => {}} />}
              />
              <Route
                path="/providers"
                element={<ProvidersPage desktopState={null} selectedProviderId={null} onSelectedProviderChange={() => {}} />}
              />
              <Route path="/models" element={<ModelsPage />} />
              <Route path="/logs" element={<LogsPage />} />
              <Route path="/settings" element={<SettingsPage />} />
              <Route
                path="/tools"
                element={<ToolsPage desktopState={null} onCopyText={(text) => navigator.clipboard.writeText(text)} />}
              />
            </Routes>
          </main>
        </div>
      </div>
    </div>
  );
}
