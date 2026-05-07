import { useI18n } from "../i18n/i18n-provider";
import {
  eyebrowClass,
  heroClass,
  heroCopyClass,
  heroTitleClass,
  infoCardClass,
  metricValueClass,
  pageShellClass,
  sectionCardClass,
  sectionHeadClass,
  sectionMetaClass,
  sectionTitleClass,
  statusPillClass
} from "../ui";

export function SettingsPage() {
  const { t } = useI18n();

  return (
    <main className={pageShellClass}>
      <section className={heroClass}>
        <div className="space-y-4">
          <div>
            <p className={eyebrowClass}>Clash for AI</p>
            <h1 className={heroTitleClass}>{t("settings.title")}</h1>
          </div>
          <p className={heroCopyClass}>
            This web view is a supplementary entry for WSL and Linux server environments.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <span className={statusPillClass("warning")}>WSL / Linux server</span>
        </div>
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>Scope</h2>
            <p className={sectionMetaClass}>
              Browser settings are intentionally limited here. Local runtime lifecycle and desktop integration remain
              in Electron.
            </p>
          </div>
        </div>

        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          <div className={infoCardClass}>
            <p className={sectionMetaClass}>Current view</p>
            <p className={metricValueClass}>{t("settings.value.browser")}</p>
          </div>
          <div className={infoCardClass}>
            <p className={sectionMetaClass}>Management mode</p>
            <p className={metricValueClass}>Supplementary web access</p>
          </div>
        </div>
      </section>
    </main>
  );
}
