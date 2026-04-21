import { useCallback, useEffect, useState } from "react";
import { ToastRegion, type ToastItem } from "../components/toast-region";
import { useI18n } from "../i18n/i18n-provider";
import { getLogs } from "../services/api";
import type { RequestLog } from "../types/request-log";
import {
  actionRowClass,
  buttonClass,
  emptyStateClass,
  eyebrowClass,
  fieldLabelClass,
  heroClass,
  heroCopyClass,
  heroTitleClass,
  iconBadgeClass,
  inputClass,
  metaClass,
  monoClass,
  pageShellClass,
  sectionCardClass,
  sectionHeadClass,
  sectionMetaClass,
  sectionTitleClass,
  statusDotClass,
  surfaceCardClass,
  statusPillClass
} from "../ui";

interface LogsPageProps {
  apiBase?: string;
}

export function LogsPage({ apiBase }: LogsPageProps) {
  const { locale, t } = useI18n();
  const [logs, setLogs] = useState<RequestLog[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [providerFilter, setProviderFilter] = useState("all");
  const [errorFilter, setErrorFilter] = useState("all");
  const [search, setSearch] = useState("");
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const dismissToast = useCallback((id: string) => {
    setToasts((current) => current.filter((item) => item.id !== id));
  }, []);

  useEffect(() => {
    if (!error) {
      return;
    }
    setToasts((current) => [
      ...current,
      { id: `${Date.now()}-error`, message: error, tone: "error" }
    ]);
    setError(null);
  }, [error]);

  async function loadLogs() {
    setLoading(true);
    try {
      const items = await getLogs(50, apiBase);
      setLogs(items);
      setError(null);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : t("common.unknownError"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const items = await getLogs(50, apiBase);
        if (cancelled) {
          return;
        }
        setLogs(items);
        setError(null);
      } catch (loadError) {
        if (cancelled) {
          return;
        }
        setError(loadError instanceof Error ? loadError.message : t("common.unknownError"));
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [apiBase, t]);

  const providerOptions = Array.from(
    new Set(logs.map((log) => log.provider_name).filter(Boolean))
  );

  const filteredLogs = logs.filter((log) => {
    if (providerFilter !== "all" && log.provider_name !== providerFilter) {
      return false;
    }

    if (errorFilter === "errors" && !log.error_type) {
      return false;
    }

    if (errorFilter === "success" && log.error_type) {
      return false;
    }

    if (search.trim()) {
      const haystack =
        `${log.method} ${log.path} ${log.provider_name} ${log.model ?? ""} ${log.error_type ?? ""} ${log.error_message ?? ""}`.toLowerCase();
      if (!haystack.includes(search.trim().toLowerCase())) {
        return false;
      }
    }

    return true;
  });

  const successCount = filteredLogs.filter((log) => !log.error_type).length;
  const errorCount = filteredLogs.filter((log) => Boolean(log.error_type)).length;
  const averageLatency = filteredLogs.length
    ? Math.round(
        filteredLogs.reduce((total, log) => total + log.latency_ms, 0) / filteredLogs.length
      )
    : 0;

  return (
    <main className={pageShellClass}>
      <ToastRegion items={toasts} onDismiss={dismissToast} />
      <section className={heroClass}>
        <div className="space-y-4">
          <div>
            <p className={eyebrowClass}>Clash for AI</p>
            <h1 className={heroTitleClass}>{t("logs.title")}</h1>
          </div>
          <p className={heroCopyClass}>{t("logs.subtitle")}</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <span className={statusPillClass(loading ? "warning" : "default")}>
            {loading ? t("common.loading") : t("logs.section.rows", { count: filteredLogs.length })}
          </span>
        </div>
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>{t("logs.section.title")}</h2>
            <p className={sectionMetaClass}>
              {loading ? t("common.loading") : t("logs.section.rows", { count: filteredLogs.length })}
            </p>
          </div>
          <div className={actionRowClass}>
            <button
              type="button"
              className={buttonClass("secondary")}
              onClick={() => {
                void loadLogs();
              }}
            >
              {t("common.refresh")}
            </button>
          </div>
        </div>

        <div className="mt-6 grid gap-4 lg:grid-cols-[1fr_1fr_2fr]">
          <label className="flex flex-col gap-2">
            <span className={fieldLabelClass}>{t("logs.filter.provider")}</span>
            <select
              className={inputClass}
              value={providerFilter}
              onChange={(event) => {
                setProviderFilter(event.target.value);
              }}
            >
              <option value="all">{t("logs.filter.all")}</option>
              {providerOptions.map((providerName) => (
                <option key={providerName} value={providerName}>
                  {providerName}
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-col gap-2">
            <span className={fieldLabelClass}>{t("logs.filter.status")}</span>
            <select
              className={inputClass}
              value={errorFilter}
              onChange={(event) => {
                setErrorFilter(event.target.value);
              }}
            >
              <option value="all">{t("logs.filter.all")}</option>
              <option value="success">{t("logs.filter.success")}</option>
              <option value="errors">{t("logs.filter.errors")}</option>
            </select>
          </label>
          <label className="flex flex-col gap-2">
            <span className={fieldLabelClass}>{t("logs.filter.search")}</span>
            <input
              className={inputClass}
              value={search}
              onChange={(event) => {
                setSearch(event.target.value);
              }}
              placeholder={t("logs.filter.placeholder")}
            />
          </label>
        </div>

        <div className="mt-6 grid gap-3 sm:grid-cols-3">
          <div className={surfaceCardClass}>
            <p className={fieldLabelClass}>{t("logs.summary.success")}</p>
            <p className="mt-3 text-2xl font-semibold tracking-[-0.03em] text-[color:var(--color-heading)]">
              {successCount}
            </p>
          </div>
          <div className={surfaceCardClass}>
            <p className={fieldLabelClass}>{t("logs.summary.errors")}</p>
            <p className="mt-3 text-2xl font-semibold tracking-[-0.03em] text-[color:var(--color-heading)]">
              {errorCount}
            </p>
          </div>
          <div className={surfaceCardClass}>
            <p className={fieldLabelClass}>{t("logs.summary.avgLatency")}</p>
            <p className="mt-3 text-2xl font-semibold tracking-[-0.03em] text-[color:var(--color-heading)]">
              {averageLatency} ms
            </p>
          </div>
        </div>

        {filteredLogs.length === 0 ? (
          <div className="mt-6">
            <div className={emptyStateClass}>
              <p>{t("logs.empty.title")}</p>
              <p className="mt-2">{t("logs.empty.subtitle")}</p>
            </div>
          </div>
        ) : (
          <div className="mt-6 grid gap-4">
            {filteredLogs.map((log) => (
              <article
                key={log.id}
                className={surfaceCardClass}
              >
                <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                  <div className="flex items-start gap-3">
                    <span className={iconBadgeClass}>
                      <span
                        className={statusDotClass(log.error_type ? "danger" : "success")}
                      />
                    </span>
                    <div className="space-y-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className={statusPillClass("default")}>{log.method}</span>
                        <p className="text-lg font-semibold tracking-[-0.02em] text-[color:var(--color-heading)]">
                          {log.path}
                        </p>
                      </div>
                    <p className={metaClass}>
                      {t("logs.meta.provider")} <span className={monoClass}>{log.provider_name}</span>
                    </p>
                  </div>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <span className={statusPillClass("default")}>
                      {log.status_code ?? t("logs.status.na")}
                    </span>
                    {log.error_type ? (
                      <span className={statusPillClass("danger")}>{log.error_type}</span>
                    ) : (
                      <span className={statusPillClass("success")}>{t("logs.filter.success")}</span>
                    )}
                  </div>
                </div>

                <div className="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                  <p className={metaClass}>
                    {t("logs.meta.model")} <span className={monoClass}>{log.model ?? "-"}</span>
                  </p>
                  <p className={metaClass}>
                    {t("logs.meta.upstream")} <span className={monoClass}>{log.upstream_host}</span>
                  </p>
                  <p className={metaClass}>
                    {t("logs.meta.latency")} <span className={monoClass}>{log.latency_ms} ms</span>
                  </p>
                  <p className={metaClass}>
                    {t("logs.meta.firstByte")}{" "}
                    <span className={monoClass}>{log.first_byte_ms ?? "-"} ms</span>
                  </p>
                  <p className={metaClass}>
                    {t("logs.meta.at")} <span className={monoClass}>{log.timestamp}</span>
                  </p>
                  <p className={metaClass}>
                    {t("logs.meta.localTime")}{" "}
                    <span className={monoClass}>
                      {new Date(log.timestamp).toLocaleString(locale === "zh" ? "zh-CN" : "en-US")}
                    </span>
                  </p>
                </div>

                {log.error_message ? (
                  <p className="mt-5 rounded-2xl border [border-color:var(--danger-border)] [background:var(--danger-soft)] px-4 py-3 text-sm text-[color:var(--danger-text)]">
                    <span className={monoClass}>{log.error_message}</span>
                  </p>
                ) : null}
              </article>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}
