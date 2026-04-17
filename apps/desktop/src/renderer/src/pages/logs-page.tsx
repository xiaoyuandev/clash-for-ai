import { useEffect, useState } from "react";
import { getLogs } from "../services/api";
import type { RequestLog } from "../types/request-log";

export function LogsPage() {
  const [logs, setLogs] = useState<RequestLog[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const items = await getLogs(50);
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
  }, []);

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
          <span>{loading ? "loading" : `${logs.length} rows`}</span>
        </div>

        {logs.length === 0 ? (
          <div className="empty-state">
            <p>No request logs yet.</p>
            <p>Send traffic through the local gateway to populate this view.</p>
          </div>
        ) : (
          <div className="log-list">
            {logs.map((log) => (
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
