import { useCallback, useEffect, useMemo, useState } from "react";
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
  inputClass,
  pageShellClass,
  sectionCardClass,
  sectionHeadClass,
  sectionMetaClass,
  sectionTitleClass,
  statusDotClass,
  statusPillClass
} from "../ui";

interface LogsPageProps {
  apiBase?: string;
}

const PAGE_SIZE = 50;

export function LogsPage({ apiBase }: LogsPageProps) {
  const { locale, t } = useI18n();
  const [logs, setLogs] = useState<RequestLog[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [limit, setLimit] = useState(PAGE_SIZE);
  const [refreshTick, setRefreshTick] = useState(0);
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

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (limit === PAGE_SIZE) {
        setLoading(true);
      } else {
        setLoadingMore(true);
      }

      try {
        const items = await getLogs(limit, apiBase);
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
          setLoadingMore(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [apiBase, limit, refreshTick, t]);

  const visibleLogs = useMemo(
    () => logs.filter((log) => log.path !== "/v1/models"),
    [logs]
  );

  const providerOptions = Array.from(
    new Set(visibleLogs.map((log) => log.provider_name).filter(Boolean))
  );

  const filteredLogs = visibleLogs.filter((log) => {
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

  const canLoadMore = logs.length >= limit;

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
                setLimit(PAGE_SIZE);
                setRefreshTick((current) => current + 1);
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

        {filteredLogs.length === 0 ? (
          <div className="mt-6">
            <div className={emptyStateClass}>
              <p>{t("logs.empty.title")}</p>
              <p className="mt-2">{t("logs.empty.subtitle")}</p>
            </div>
          </div>
        ) : (
          <div className="mt-6 overflow-hidden rounded-[28px] border [border-color:var(--border-soft)] [background:var(--panel-solid)]">
            <div className="grid grid-cols-[1.05fr_80px_1.25fr_1fr_1.2fr_100px_88px] gap-3 border-b [border-color:var(--border-soft)] px-4 py-3 text-[11px] font-semibold uppercase tracking-[0.22em] text-[color:var(--color-subtle)]">
              <span>{t("logs.columns.time")}</span>
              <span>{t("logs.columns.method")}</span>
              <span>{t("logs.columns.path")}</span>
              <span>{t("logs.columns.provider")}</span>
              <span>{t("logs.columns.model")}</span>
              <span>{t("logs.columns.status")}</span>
              <span>{t("logs.columns.latency")}</span>
            </div>

            <div className="divide-y [divide-color:var(--border-soft)]">
              {filteredLogs.map((log) => {
                const localTime = new Date(log.timestamp).toLocaleString(
                  locale === "zh" ? "zh-CN" : "en-US"
                );

                return (
                  <article
                    key={log.id}
                    className="grid grid-cols-[1.05fr_80px_1.25fr_1fr_1.2fr_100px_88px] items-center gap-3 px-4 py-3 text-sm"
                    title={
                      log.error_message
                        ? `${localTime}\n${log.method} ${log.path}\n${log.provider_name}\n${log.model ?? "-"}\n${log.error_message}`
                        : `${localTime}\n${log.method} ${log.path}\n${log.provider_name}\n${log.model ?? "-"}`
                    }
                  >
                    <span className="truncate text-[color:var(--color-muted)]">{localTime}</span>
                    <span className="truncate">
                      <span className={statusPillClass("default")}>{log.method}</span>
                    </span>
                    <span className="truncate font-mono text-[13px] text-[color:var(--color-text)]">
                      {log.path}
                    </span>
                    <span className="truncate text-[color:var(--color-text)]">{log.provider_name}</span>
                    <span className="truncate font-mono text-[13px] text-[color:var(--color-text)]">
                      {log.model ?? "-"}
                    </span>
                    <span className="inline-flex items-center gap-2 truncate">
                      <span className={statusDotClass(log.error_type ? "danger" : "success")} />
                      <span className="truncate text-[color:var(--color-muted)]">
                        {log.error_type ? t("logs.filter.errors") : t("logs.filter.success")}
                      </span>
                    </span>
                    <span className="truncate text-[color:var(--color-text)]">{log.latency_ms} ms</span>
                  </article>
                );
              })}
            </div>
          </div>
        )}

        {canLoadMore ? (
          <div className="mt-5 flex justify-center">
            <button
              type="button"
              className={buttonClass("secondary")}
              disabled={loadingMore}
              onClick={() => {
                setLimit((current) => current + PAGE_SIZE);
              }}
            >
              {loadingMore ? t("logs.button.loadingMore") : t("logs.button.loadMore")}
            </button>
          </div>
        ) : null}
      </section>
    </main>
  );
}
