import { useEffect, useState } from "react";
import {
  activateProvider,
  createProvider,
  deleteProvider,
  getHealth,
  getProviders,
  runProviderHealthcheck,
  updateProvider
} from "../services/api";
import type { Provider } from "../types/provider";

interface ProvidersPageProps {
  desktopState: {
    ok: boolean;
    runtime: string;
    platform: string;
  } | null;
}

export function ProvidersPage({ desktopState }: ProvidersPageProps) {
  const [health, setHealth] = useState("loading");
  const [providers, setProviders] = useState<Provider[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [authMode, setAuthMode] = useState<"bearer" | "x-api-key" | "both">(
    "bearer"
  );
  const [editingId, setEditingId] = useState<string | null>(null);
  const [healthFeedback, setHealthFeedback] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const [healthData, providersData] = await Promise.all([
          getHealth(),
          getProviders()
        ]);

        if (cancelled) {
          return;
        }

        setHealth(healthData.status);
        setProviders(providersData);
      } catch (loadError) {
        if (cancelled) {
          return;
        }

        setHealth("offline");
        setError(loadError instanceof Error ? loadError.message : "Unknown error");
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, []);

  async function refreshProviders() {
    const providersData = await getProviders();
    setProviders(providersData);
  }

  async function handleCreateProvider(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setHealthFeedback(null);

    try {
      const payload = {
        name,
        base_url: baseUrl,
        api_key: apiKey,
        auth_mode: authMode,
        extra_headers: {}
      };

      if (editingId) {
        await updateProvider(editingId, payload);
      } else {
        await createProvider(payload);
      }

      resetForm();
      setEditingId(null);
      await refreshProviders();
    } catch (createError) {
      setError(createError instanceof Error ? createError.message : "Unknown error");
    }
  }

  async function handleActivateProvider(id: string) {
    setError(null);
    setHealthFeedback(null);

    try {
      await activateProvider(id);
      await refreshProviders();
    } catch (activateError) {
      setError(
        activateError instanceof Error ? activateError.message : "Unknown error"
      );
    }
  }

  async function handleDeleteProvider(id: string) {
    setError(null);
    setHealthFeedback(null);

    try {
      await deleteProvider(id);
      if (editingId === id) {
        resetForm();
      }
      await refreshProviders();
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : "Unknown error");
    }
  }

  async function handleHealthcheck(id: string) {
    setError(null);

    try {
      const result = await runProviderHealthcheck(id);
      setHealthFeedback(
        `${result.status.toUpperCase()} ${result.status_code} in ${result.latency_ms}ms`
      );
      await refreshProviders();
    } catch (healthError) {
      setError(healthError instanceof Error ? healthError.message : "Unknown error");
    }
  }

  function startEditing(provider: Provider) {
    setEditingId(provider.id);
    setName(provider.name);
    setBaseUrl(provider.base_url);
    setApiKey("");
    setAuthMode(provider.auth_mode);
    setHealthFeedback(null);
    setError(null);
  }

  function resetForm() {
    setEditingId(null);
    setName("");
    setBaseUrl("");
    setApiKey("");
    setAuthMode("bearer");
  }

  return (
    <main className="page-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Clash for AI</p>
          <h1>Provider Control Plane</h1>
          <p className="subcopy">
            Electron-vite desktop shell aligned to the packaging baseline before
            further feature development.
          </p>
        </div>
        <div className="hero-pills">
          <div className={`health-pill health-${health}`}>core: {health}</div>
          <div
            className={`health-pill ${
              desktopState?.ok ? "health-ok" : "health-offline"
            }`}
          >
            desktop: {desktopState?.runtime ?? "browser"}
          </div>
        </div>
      </section>

      {error ? <p className="panel error-panel">{error}</p> : null}
      {healthFeedback ? <p className="panel info-panel">{healthFeedback}</p> : null}

      <section className="panel form-panel">
        <div className="section-head">
          <h2>{editingId ? "Edit Provider" : "Add Provider"}</h2>
          <span>{editingId ? "update existing" : "desktop baseline"}</span>
        </div>

        <form className="provider-form" onSubmit={handleCreateProvider}>
          <label>
            <span>Name</span>
            <input value={name} onChange={(event) => setName(event.target.value)} />
          </label>
          <label>
            <span>Base URL</span>
            <input
              value={baseUrl}
              onChange={(event) => setBaseUrl(event.target.value)}
              placeholder="https://api.example.com/v1"
            />
          </label>
          <label>
            <span>API Key</span>
            <input
              value={apiKey}
              onChange={(event) => setApiKey(event.target.value)}
              placeholder="sk-example"
            />
          </label>
          <label>
            <span>Auth Mode</span>
            <select
              value={authMode}
              onChange={(event) =>
                setAuthMode(event.target.value as "bearer" | "x-api-key" | "both")
              }
            >
              <option value="bearer">bearer</option>
              <option value="x-api-key">x-api-key</option>
              <option value="both">both</option>
            </select>
          </label>
          <button type="submit">
            {editingId ? "Save Provider" : "Create Provider"}
          </button>
          {editingId ? (
            <button
              type="button"
              className="secondary-button"
              onClick={resetForm}
            >
              Cancel
            </button>
          ) : null}
        </form>
      </section>

      <section className="panel">
        <div className="section-head">
          <h2>Providers</h2>
          <span>{providers.length} configured</span>
        </div>

        {providers.length === 0 ? (
          <div className="empty-state">
            <p>No providers configured yet.</p>
            <p>Next step is wiring the rest of the desktop management surface.</p>
          </div>
        ) : (
          <div className="provider-grid">
            {providers.map((provider) => (
              <article key={provider.id} className="provider-card">
                <div className="provider-card-head">
                  <h3>{provider.name}</h3>
                  {provider.status.is_active ? (
                    <span className="status-badge active">active</span>
                  ) : (
                    <span className="status-badge">standby</span>
                  )}
                </div>
                <p className="mono">{provider.base_url}</p>
                <p className="meta">
                  auth: <span className="mono">{provider.auth_mode}</span>
                </p>
                <p className="meta">
                  health:{" "}
                  <span className="mono">{provider.status.last_health_status}</span>
                </p>
                <div className="provider-actions">
                  <span className="meta mono">{provider.api_key_masked}</span>
                  <div className="action-row">
                    {!provider.status.is_active ? (
                      <button
                        type="button"
                        onClick={() => {
                          void handleActivateProvider(provider.id);
                        }}
                      >
                        Activate
                      </button>
                    ) : null}
                    <button
                      type="button"
                      className="secondary-button"
                      onClick={() => {
                        startEditing(provider);
                      }}
                    >
                      Edit
                    </button>
                    <button
                      type="button"
                      className="secondary-button"
                      onClick={() => {
                        void handleHealthcheck(provider.id);
                      }}
                    >
                      Check
                    </button>
                    <button
                      type="button"
                      className="danger-button"
                      onClick={() => {
                        void handleDeleteProvider(provider.id);
                      }}
                    >
                      Delete
                    </button>
                  </div>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}
