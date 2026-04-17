import { useEffect, useState } from "react";
import { getLogs } from "../services/api";
import type { RequestLog } from "../types/request-log";

interface LogsPageProps {
  apiBase?: string;
}

export function LogsPage({ apiBase }: LogsPageProps) {
  const [logs, setLogs] = useState<RequestLog[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [providerFilter, setProviderFilter] = useState("all");
  const [errorFilter, setErrorFilter] = useState("all");
  const [search, setSearch] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const items = await getLogs(50, apiBase);
        if (cancelled) {
          return;
        }
        setLogs(items);
      } catch (loadError) {
        if (cancelled) {
          return;
        }
        setError(loadError instanceof Error ? loadError.message : "Unknown error");
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
          <h1>Request Log</h1>
          <p className="subcopy">
            Latest gateway activity with status, latency, upstream target and
            error classification.
          </p>
        </div>
      </section>

      {error ? <p className="panel error-panel">{error}</p> : null}

      <section className="panel">
        <div className="section-head">
          <h2>Recent Requests</h2>
          <span>{loading ? "loading" : `${filteredLogs.length} rows`}</span>
        </div>

        <div className="log-filters">
          <label>
            <span>Provider</span>
            <select
              value={providerFilter}
              onChange={(event) => {
                setProviderFilter(event.target.value);
              }}
            >
              <option value="all">all</option>
              {providerOptions.map((providerName) => (
                <option key={providerName} value={providerName}>
                  {providerName}
                </option>
              ))}
            </select>
          </label>
          <label>
            <span>Status</span>
            <select
              value={errorFilter}
              onChange={(event) => {
                setErrorFilter(event.target.value);
              }}
            >
              <option value="all">all</option>
              <option value="success">success</option>
              <option value="errors">errors</option>
            </select>
          </label>
          <label className="search-filter">
            <span>Search</span>
            <input
              value={search}
              onChange={(event) => {
                setSearch(event.target.value);
              }}
              placeholder="method, path, model, error..."
            />
          </label>
        </div>

        {filteredLogs.length === 0 ? (
          <div className="empty-state">
            <p>No request logs yet.</p>
            <p>Send traffic through the local gateway to populate this view.</p>
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
                      provider: <span className="mono">{log.provider_name}</span>
                    </p>
                  </div>
                  <div className="log-chip-row">
                    <span className="status-badge">
                      {log.status_code ?? "n/a"}
                    </span>
                    {log.error_type ? (
                      <span className="status-badge warning-chip">
                        {log.error_type}
                      </span>
                    ) : (
                      <span className="status-badge active">success</span>
                    )}
                  </div>
                </div>

                <div className="log-metadata">
                  <p className="meta">
                    model: <span className="mono">{log.model ?? "-"}</span>
                  </p>
                  <p className="meta">
                    upstream: <span className="mono">{log.upstream_host}</span>
                  </p>
                  <p className="meta">
                    latency: <span className="mono">{log.latency_ms} ms</span>
                  </p>
                  <p className="meta">
                    first byte:{" "}
                    <span className="mono">{log.first_byte_ms ?? "-"} ms</span>
                  </p>
                  <p className="meta">
                    at: <span className="mono">{log.timestamp}</span>
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
