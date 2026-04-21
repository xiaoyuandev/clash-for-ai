import { useEffect, useState } from "react";
import { useI18n } from "../i18n/i18n-provider";
import { getLogs } from "../services/api";
import type { RequestLog } from "../types/request-log";

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
  }, [apiBase]);

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
      const haystack = `${log.method} ${log.path} ${log.provider_name} ${log.model ?? ""} ${log.error_type ?? ""} ${log.error_message ?? ""}`.toLowerCase();
      if (!haystack.includes(search.trim().toLowerCase())) {
        return false;
      }
    }

    return true;
  });

  return (
    <main className="page-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Clash for AI</p>
          <h1>{t("logs.title")}</h1>
          <p className="subcopy">{t("logs.subtitle")}</p>
        </div>
      </section>

      {error ? <p className="panel error-panel">{error}</p> : null}

      <section className="panel">
        <div className="section-head">
          <h2>{t("logs.section.title")}</h2>
          <div className="section-actions">
            <span>
              {loading ? t("common.loading") : t("logs.section.rows", { count: filteredLogs.length })}
            </span>
            <button
              type="button"
              className="secondary-button"
              onClick={() => {
                void loadLogs();
              }}
            >
              {t("common.refresh")}
            </button>
          </div>
        </div>

        <div className="log-filters">
          <label>
            <span>{t("logs.filter.provider")}</span>
            <select
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
          <label>
            <span>{t("logs.filter.status")}</span>
            <select
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
          <label className="search-filter">
            <span>{t("logs.filter.search")}</span>
            <input
              value={search}
              onChange={(event) => {
                setSearch(event.target.value);
              }}
              placeholder={t("logs.filter.placeholder")}
            />
          </label>
        </div>

        {filteredLogs.length === 0 ? (
          <div className="empty-state">
            <p>{t("logs.empty.title")}</p>
            <p>{t("logs.empty.subtitle")}</p>
          </div>
        ) : (
          <div className="log-list">
            {filteredLogs.map((log) => (
              <article key={log.id} className="log-card">
                <div className="log-card-head">
                  <div>
                    <p className="log-title">
                      {log.method} {log.path}
                    </p>
                    <p className="meta">
                      {t("logs.meta.provider")} <span className="mono">{log.provider_name}</span>
                    </p>
                  </div>
                  <div className="log-chip-row">
                    <span className="status-badge">
                      {log.status_code ?? t("logs.status.na")}
                    </span>
                    {log.error_type ? (
                      <span className="status-badge warning-chip">
                        {log.error_type}
                      </span>
                    ) : (
                      <span className="status-badge active">{t("logs.filter.success")}</span>
                    )}
                  </div>
                </div>

                <div className="log-metadata">
                  <p className="meta">
                    {t("logs.meta.model")} <span className="mono">{log.model ?? "-"}</span>
                  </p>
                  <p className="meta">
                    {t("logs.meta.upstream")} <span className="mono">{log.upstream_host}</span>
                  </p>
                  <p className="meta">
                    {t("logs.meta.latency")} <span className="mono">{log.latency_ms} ms</span>
                  </p>
                  <p className="meta">
                    {t("logs.meta.firstByte")}{" "}
                    <span className="mono">{log.first_byte_ms ?? "-"} ms</span>
                  </p>
                  <p className="meta">
                    {t("logs.meta.at")} <span className="mono">{log.timestamp}</span>
                  </p>
                  <p className="meta">
                    {t("logs.meta.localTime")}{" "}
                    <span className="mono">
                      {new Date(log.timestamp).toLocaleString(locale === "zh" ? "zh-CN" : "en-US")}
                    </span>
                  </p>
                </div>

                {log.error_message ? (
                  <p className="log-error">
                    <span className="mono">{log.error_message}</span>
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
