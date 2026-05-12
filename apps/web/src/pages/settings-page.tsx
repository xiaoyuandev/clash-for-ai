import { useEffect, useState } from "react";
import { useI18n } from "../i18n/i18n-provider";
import { getReleaseMetadata, type GitHubRelease, type ReleaseMetadata } from "../services/api";
import { copyText } from "../bridge/platform-bridge";
import { compareVersions } from "../utils/version";
import {
  actionRowClass,
  buttonClass,
  eyebrowClass,
  heroClass,
  heroCopyClass,
  heroTitleClass,
  hintClass,
  infoCardClass,
  metricValueClass,
  monoClass,
  pageShellClass,
  sectionCardClass,
  sectionHeadClass,
  sectionMetaClass,
  sectionTitleClass,
  statusPillClass
} from "../ui";

const installCommand = "curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/clash-for-ai/main/scripts/install.sh | bash";
const latestReleaseUrl = "https://github.com/xiaoyuandev/clash-for-ai/releases/latest";

interface SettingsPageProps {
  releaseMetadata?: ReleaseMetadata | null;
  latestRelease?: GitHubRelease | null;
  latestReleaseError?: string | null;
}

export function SettingsPage({
  releaseMetadata: initialReleaseMetadata = null,
  latestRelease,
  latestReleaseError
}: SettingsPageProps) {
  const { t } = useI18n();
  const [releaseMetadata, setReleaseMetadata] = useState<ReleaseMetadata | null>(initialReleaseMetadata);
  const [copyFeedback, setCopyFeedback] = useState<string | null>(null);

  useEffect(() => {
    setReleaseMetadata(initialReleaseMetadata);
  }, [initialReleaseMetadata]);

  useEffect(() => {
    let cancelled = false;

    void getReleaseMetadata()
      .then((payload) => {
        if (!cancelled) {
          setReleaseMetadata(payload);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setReleaseMetadata({ available: false });
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  const currentVersion = releaseMetadata?.available ? releaseMetadata.release?.release_version : undefined;
  const latestVersion = latestRelease?.tag_name;
  const updateComparison = compareVersions(currentVersion, latestVersion);
  const updateAvailable = updateComparison > 0;
  const updateStatus = latestReleaseError
    ? t("settings.updates.latestUnavailable")
    : latestRelease
      ? updateAvailable
        ? t("settings.updates.updateAvailable")
        : t("settings.updates.upToDate")
      : t("settings.updates.checking");

  async function handleCopyInstallCommand() {
    try {
      await copyText(installCommand);
      setCopyFeedback(t("settings.updates.installCommandCopied"));
    } catch {
      setCopyFeedback(t("settings.updates.installCommandCopyFailed"));
    }
  }

  return (
    <main className={pageShellClass}>
      <section className={heroClass}>
        <div className="space-y-4">
          <div>
            <p className={eyebrowClass}>Clash for AI</p>
            <h1 className={heroTitleClass}>{t("settings.title")}</h1>
          </div>
          <p className={heroCopyClass}>
            {t("settings.web.subtitle")}
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
              {t("settings.web.scope")}
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
            <p className={metricValueClass}>{t("settings.web.managementMode")}</p>
          </div>
          <div className={infoCardClass}>
            <p className={sectionMetaClass}>Release version</p>
            <p className={metricValueClass}>
              {releaseMetadata?.available ? releaseMetadata.release?.release_version : t("settings.value.unknown")}
            </p>
          </div>
          <div className={infoCardClass}>
            <p className={sectionMetaClass}>Runtime version</p>
            <p className={metricValueClass}>
              {releaseMetadata?.available
                ? `${releaseMetadata.release?.runtime_version} (${releaseMetadata.release?.runtime_commit})`
                : t("settings.value.unknown")}
            </p>
          </div>
        </div>
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>{t("settings.webUpdate.title")}</h2>
            <p className={sectionMetaClass}>{t("settings.webUpdate.meta")}</p>
          </div>
          <span className={statusPillClass(updateAvailable ? "success" : latestReleaseError ? "danger" : "default")}>
            {updateStatus}
          </span>
        </div>

        <div className="mt-4 grid gap-3 sm:grid-cols-3">
          <div className={infoCardClass}>
            <p className={sectionMetaClass}>{t("settings.updates.currentVersion")}</p>
            <p className={metricValueClass}>{currentVersion ?? t("settings.value.unknown")}</p>
          </div>
          <div className={infoCardClass}>
            <p className={sectionMetaClass}>{t("settings.updates.latestVersion")}</p>
            <p className={metricValueClass}>{latestVersion ?? t("settings.value.unknown")}</p>
          </div>
          <div className={infoCardClass}>
            <p className={sectionMetaClass}>{t("settings.updates.status")}</p>
            <p className={metricValueClass}>{updateStatus}</p>
          </div>
        </div>

        <div className="mt-4 rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3.5">
          <p className={hintClass}>{t("settings.webUpdate.commandHint")}</p>
          <code className={`${monoClass} mt-3 block rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-input)] p-3`}>
            {installCommand}
          </code>
          {copyFeedback ? <p className={`${hintClass} mt-2`}>{copyFeedback}</p> : null}
        </div>

        {latestReleaseError ? <p className={`${hintClass} mt-3`}>{latestReleaseError}</p> : null}

        <div className={`${actionRowClass} mt-4`}>
          <a className={buttonClass(updateAvailable ? "primary" : "secondary")} href={latestRelease?.html_url ?? latestReleaseUrl} target="_blank" rel="noreferrer">
            {t("settings.button.openReleasePage")}
          </a>
          <button type="button" className={buttonClass("secondary")} onClick={() => void handleCopyInstallCommand()}>
            {t("settings.webUpdate.copyInstallCommand")}
          </button>
        </div>
      </section>
    </main>
  );
}
