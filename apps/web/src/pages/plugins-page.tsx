import { useCallback, useEffect, useMemo, useState } from "react";
import { useI18n } from "../i18n/i18n-provider";
import {
  disableExtension,
  enableExtension,
  getExtensions,
  rescanExtensions
} from "../services/extensions";
import type { ExtensionPlugin, PluginStatus } from "../types/extension";
import {
  actionRowClass,
  buttonClass,
  dangerNoticeClass,
  emptyStateClass,
  eyebrowClass,
  fieldLabelClass,
  heroClass,
  heroCopyClass,
  heroTitleClass,
  hintClass,
  metaClass,
  monoClass,
  pageShellClass,
  sectionCardClass,
  sectionHeadClass,
  sectionMetaClass,
  sectionTitleClass,
  selectableItemClass,
  statusDotClass,
  statusPillClass,
  successNoticeClass
} from "../ui";

interface PluginsPageProps {
  apiBase?: string;
}

function statusTone(status: PluginStatus, enabled: boolean) {
  if (status === "invalid" || status === "incompatible") {
    return "danger" as const;
  }
  if (enabled || status === "enabled") {
    return "success" as const;
  }
  if (status === "disabled") {
    return "warning" as const;
  }
  return "default" as const;
}

function formatContributionSummary(item: ExtensionPlugin) {
  return Object.entries(item.contributes)
    .filter(([, count]) => count > 0)
    .sort(([left], [right]) => left.localeCompare(right));
}

export function PluginsPage({ apiBase }: PluginsPageProps) {
  const { t } = useI18n();
  const [items, setItems] = useState<ExtensionPlugin[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [busyAction, setBusyAction] = useState<"rescan" | "enable" | "disable" | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const selected = useMemo(
    () => items.find((item) => item.id === selectedId) ?? items[0] ?? null,
    [items, selectedId]
  );

  const stats = useMemo(
    () => ({
      total: items.length,
      enabled: items.filter((item) => item.enabled).length,
      invalid: items.filter((item) => item.status === "invalid").length
    }),
    [items]
  );

  const syncItems = useCallback(
    async (mode: "load" | "rescan" = "load") => {
      setError(null);
      if (mode === "load") {
        setLoading(true);
      } else {
        setBusyAction("rescan");
      }

      try {
        const nextItems = mode === "rescan" ? await rescanExtensions(apiBase) : await getExtensions(apiBase);
        setItems(nextItems);
        setSelectedId((current) =>
          current && nextItems.some((item) => item.id === current)
            ? current
            : nextItems[0]?.id ?? null
        );
        if (mode === "rescan") {
          setFeedback(t("plugins.feedback.rescanned"));
        }
      } catch (nextError) {
        setError(nextError instanceof Error ? nextError.message : t("common.unknownError"));
      } finally {
        setLoading(false);
        setBusyAction(null);
      }
    },
    [apiBase, t]
  );

  useEffect(() => {
    void syncItems();
  }, [syncItems]);

  async function handleToggle(item: ExtensionPlugin) {
    setBusyAction(item.enabled ? "disable" : "enable");
    setError(null);
    setFeedback(null);

    try {
      const updated = item.enabled
        ? await disableExtension(item.id, apiBase)
        : await enableExtension(item.id, apiBase);
      setItems((current) => current.map((entry) => (entry.id === updated.id ? updated : entry)));
      setFeedback(item.enabled ? t("plugins.feedback.disabled") : t("plugins.feedback.enabled"));
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : t("common.unknownError"));
    } finally {
      setBusyAction(null);
    }
  }

  const selectedContributions = selected ? formatContributionSummary(selected) : [];

  return (
    <main className={pageShellClass}>
      <section className={heroClass}>
        <div className="space-y-4">
          <div>
            <p className={eyebrowClass}>Relay Switch</p>
            <h1 className={heroTitleClass}>{t("plugins.title")}</h1>
          </div>
          <p className={heroCopyClass}>{t("plugins.subtitle")}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className={statusPillClass()}>{t("plugins.stats.total", { count: stats.total })}</span>
          <span className={statusPillClass("success")}>
            {t("plugins.stats.enabled", { count: stats.enabled })}
          </span>
          {stats.invalid > 0 ? (
            <span className={statusPillClass("danger")}>
              {t("plugins.stats.invalid", { count: stats.invalid })}
            </span>
          ) : null}
        </div>
      </section>

      {feedback ? <div className={successNoticeClass}>{feedback}</div> : null}
      {error ? <div className={dangerNoticeClass}>{error}</div> : null}

      <section className="grid min-h-0 gap-4 xl:grid-cols-[360px_minmax(0,1fr)]">
        <div className={sectionCardClass}>
          <div className={sectionHeadClass}>
            <div className="space-y-1">
              <h2 className={sectionTitleClass}>{t("plugins.list.title")}</h2>
              <p className={sectionMetaClass}>{t("plugins.list.subtitle")}</p>
            </div>
            <button
              type="button"
              className={buttonClass("secondary")}
              disabled={busyAction === "rescan"}
              onClick={() => void syncItems("rescan")}
            >
              {busyAction === "rescan" ? t("plugins.button.rescanning") : t("plugins.button.rescan")}
            </button>
          </div>

          <div className="mt-4 grid gap-3">
            {loading ? (
              <div className={emptyStateClass}>{t("common.loading")}</div>
            ) : items.length === 0 ? (
              <div className={emptyStateClass}>{t("plugins.empty")}</div>
            ) : (
              items.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  className={selectableItemClass(selected?.id === item.id)}
                  onClick={() => setSelectedId(item.id)}
                >
                  <span className="flex items-start justify-between gap-3">
                    <span className="min-w-0">
                      <span className="block truncate text-sm font-semibold text-[color:var(--color-heading)]">
                        {item.name}
                      </span>
                      <span className={`${monoClass} mt-1 block text-[11px]`}>{item.id}</span>
                    </span>
                    <span className={statusPillClass(statusTone(item.status, item.enabled))}>
                      {item.status}
                    </span>
                  </span>
                </button>
              ))
            )}
          </div>
        </div>

        <div className={sectionCardClass}>
          {selected ? (
            <div className="space-y-5">
              <div className={sectionHeadClass}>
                <div className="min-w-0 space-y-1">
                  <h2 className={sectionTitleClass}>{selected.name}</h2>
                  <p className={sectionMetaClass}>{selected.description || t("plugins.detail.noDescription")}</p>
                </div>
                <div className={actionRowClass}>
                  <span className={statusPillClass(statusTone(selected.status, selected.enabled))}>
                    {selected.status}
                  </span>
                  <button
                    type="button"
                    className={buttonClass(selected.enabled ? "danger" : "primary")}
                    disabled={
                      selected.status === "invalid" ||
                      selected.status === "incompatible" ||
                      busyAction === "enable" ||
                      busyAction === "disable"
                    }
                    onClick={() => void handleToggle(selected)}
                  >
                    {busyAction === "enable" || busyAction === "disable"
                      ? t("common.saving")
                      : selected.enabled
                        ? t("plugins.button.disable")
                        : t("plugins.button.enable")}
                  </button>
                </div>
              </div>

              <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                <div>
                  <p className={fieldLabelClass}>{t("plugins.detail.version")}</p>
                  <p className="mt-1 text-sm font-medium text-[color:var(--color-heading)]">{selected.version}</p>
                </div>
                <div>
                  <p className={fieldLabelClass}>{t("plugins.detail.scope")}</p>
                  <p className="mt-1 text-sm font-medium text-[color:var(--color-heading)]">{selected.scope}</p>
                </div>
                <div>
                  <p className={fieldLabelClass}>{t("plugins.detail.publisher")}</p>
                  <p className="mt-1 text-sm font-medium text-[color:var(--color-heading)]">
                    {selected.publisher || t("settings.value.unknown")}
                  </p>
                </div>
                <div>
                  <p className={fieldLabelClass}>{t("plugins.detail.entry")}</p>
                  <p className="mt-1 text-sm font-medium text-[color:var(--color-heading)]">
                    {selected.manifest.entry.type}
                  </p>
                </div>
              </div>

              <div className="space-y-2">
                <p className={fieldLabelClass}>{t("plugins.detail.manifestPath")}</p>
                <code className={`${monoClass} block rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-input)] p-3`}>
                  {selected.manifest_path}
                </code>
              </div>

              {selected.last_error ? (
                <div className={dangerNoticeClass}>{selected.last_error}</div>
              ) : null}

              {selected.warnings.length > 0 ? (
                <div className="rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3 text-sm text-[color:var(--color-muted)]">
                  <p className="mb-2 font-medium text-[color:var(--color-heading)]">
                    {t("plugins.detail.warnings")}
                  </p>
                  <ul className="grid gap-1.5">
                    {selected.warnings.map((warning) => (
                      <li key={warning} className="flex items-start gap-2">
                        <span className={`${statusDotClass("warning")} mt-1.5`} />
                        <span>{warning}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              ) : null}

              <div className="grid gap-4 xl:grid-cols-2">
                <div className="space-y-2">
                  <p className={fieldLabelClass}>{t("plugins.detail.permissions")}</p>
                  {selected.permissions.length > 0 ? (
                    <div className="flex flex-wrap gap-2">
                      {selected.permissions.map((permission) => (
                        <span key={permission} className={statusPillClass("warning")}>
                          {permission}
                        </span>
                      ))}
                    </div>
                  ) : (
                    <p className={metaClass}>{t("plugins.detail.noPermissions")}</p>
                  )}
                </div>

                <div className="space-y-2">
                  <p className={fieldLabelClass}>{t("plugins.detail.contributions")}</p>
                  {selectedContributions.length > 0 ? (
                    <div className="flex flex-wrap gap-2">
                      {selectedContributions.map(([name, count]) => (
                        <span key={name} className={statusPillClass()}>
                          {name}: {count}
                        </span>
                      ))}
                    </div>
                  ) : (
                    <p className={metaClass}>{t("plugins.detail.noContributions")}</p>
                  )}
                </div>
              </div>

              <p className={hintClass}>{t("plugins.detail.runtimeHint")}</p>
            </div>
          ) : (
            <div className={emptyStateClass}>{t("plugins.empty")}</div>
          )}
        </div>
      </section>
    </main>
  );
}
