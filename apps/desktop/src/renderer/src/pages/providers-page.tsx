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
    apiBase: string;
  } | null;
  apiBase?: string;
  selectedProviderId: string | null;
  onSelectedProviderChange: (provider: Provider | null) => void;
}

export function ProvidersPage({
  desktopState,
  apiBase,
  selectedProviderId,
  onSelectedProviderChange
}: ProvidersPageProps) {
  const [health, setHealth] = useState("loading");
  const [providers, setProviders] = useState<Provider[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const selectedProvider =
    providers.find((provider) => provider.id === selectedProviderId) ??
    providers.find((provider) => provider.status.is_active) ??
    providers[0] ??
    null;

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const [healthData, providersData] = await Promise.all([
          getHealth(apiBase),
          getProviders(apiBase)
        ]);

        if (cancelled) {
          return;
        }

        setHealth(healthData.status);
        setProviders(providersData);
        const nextSelected =
          providersData.find((provider) => provider.id === selectedProviderId) ??
          providersData.find((provider) => provider.status.is_active) ??
          providersData[0] ??
          null;
        onSelectedProviderChange(nextSelected);
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
  }, [apiBase, onSelectedProviderChange, selectedProviderId]);

  async function refreshProviders(preferredProviderId?: string) {
    const providersData = await getProviders(apiBase);
    setProviders(providersData);
    const nextSelected =
      providersData.find((provider) => provider.id === preferredProviderId) ??
      providersData.find((provider) => provider.id === selectedProviderId) ??
      providersData.find((provider) => provider.status.is_active) ??
      providersData[0] ??
      null;
    onSelectedProviderChange(nextSelected);
  }

  async function handleCreateProvider(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setFeedback(null);

    if (!name.trim() || !baseUrl.trim() || !apiKey.trim()) {
      setError("Name, Base URL, and API Key are required.");
      return;
    }

    try {
      setSubmitting(true);
      const payload = {
        name: name.trim(),
        base_url: baseUrl.trim(),
        api_key: apiKey.trim(),
        extra_headers: {}
      };

      const provider = editingId
        ? await updateProvider(editingId, payload, apiBase)
        : await createProvider(payload, apiBase);

      resetForm();
      setFeedback(editingId ? "Provider updated." : "Provider created.");
      await refreshProviders(provider.id);
    } catch (createError) {
      setError(createError instanceof Error ? createError.message : "Unknown error");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleActivateProvider(provider: Provider) {
    setError(null);
    setFeedback(null);

    try {
      await activateProvider(provider.id, apiBase);
      await refreshProviders(provider.id);
      setFeedback(`${provider.name} activated.`);
    } catch (activateError) {
      setError(activateError instanceof Error ? activateError.message : "Unknown error");
    }
  }

  async function handleDeleteProvider(id: string) {
    setError(null);
    setFeedback(null);

    try {
      await deleteProvider(id, apiBase);
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
    setFeedback(null);

    try {
      const result = await runProviderHealthcheck(id, apiBase);
      setFeedback(
        `${result.status.toUpperCase()} ${result.status_code} in ${result.latency_ms}ms`
      );
      await refreshProviders(id);
    } catch (healthError) {
      setError(healthError instanceof Error ? healthError.message : "Unknown error");
    }
  }

  function startEditing(provider: Provider) {
    setEditingId(provider.id);
    setName(provider.name);
    setBaseUrl(provider.base_url);
    setApiKey("");
    setFeedback(null);
    setError(null);
  }

  function resetForm() {
    setEditingId(null);
    setName("");
    setBaseUrl("");
    setApiKey("");
  }

  return (
    <main className="page-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Clash for AI</p>
          <h1>Providers</h1>
          <p className="subcopy">
            Manage provider connections here. Supported models are now managed from the Models menu for the active provider.
          </p>
          <p className="meta">
            connected api base:{" "}
            <span className="mono">{desktopState?.apiBase ?? apiBase ?? "http://127.0.0.1:3456"}</span>
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
      {feedback ? <p className="panel info-panel">{feedback}</p> : null}

      <section className="panel form-panel">
        <div className="section-head">
          <h2>{editingId ? "Edit Provider" : "Add Provider"}</h2>
          <span>{editingId ? "update existing" : "add new"}</span>
        </div>

        <form className="provider-form" onSubmit={handleCreateProvider}>
          <label>
            <span>Name</span>
            <input required value={name} onChange={(event) => setName(event.target.value)} />
          </label>
          <label>
            <span>Base URL</span>
            <input
              required
              value={baseUrl}
              onChange={(event) => setBaseUrl(event.target.value)}
              placeholder="https://api.example.com/v1"
            />
          </label>
          <label>
            <span>API Key</span>
            <input
              required
              value={apiKey}
              onChange={(event) => setApiKey(event.target.value)}
              placeholder="sk-example"
              type="password"
            />
          </label>
          <button type="submit" disabled={submitting}>
            {submitting ? "Saving..." : editingId ? "Save Provider" : "Create Provider"}
          </button>
          {editingId ? (
            <button
              type="button"
              className="secondary-button"
              onClick={resetForm}
              disabled={submitting}
            >
              Cancel
            </button>
          ) : null}
        </form>
      </section>

      <section className="providers-layout">
        <aside className="panel providers-sidebar">
          <div className="section-head">
            <h2>Provider List</h2>
            <span>{providers.length} configured</span>
          </div>

          {providers.length === 0 ? (
            <div className="empty-state">
              <p>No providers configured yet.</p>
            </div>
          ) : (
            <div className="provider-list">
              {providers.map((provider) => (
                <button
                  key={provider.id}
                  type="button"
                  className={
                    selectedProvider?.id === provider.id
                      ? "provider-list-item active-provider-list-item"
                      : "provider-list-item"
                  }
                  onClick={() => {
                    onSelectedProviderChange(provider);
                  }}
                >
                  <div className="provider-list-head">
                    <strong>{provider.name}</strong>
                    <span className={provider.status.is_active ? "status-badge active" : "status-badge"}>
                      {provider.status.is_active ? "active" : "standby"}
                    </span>
                  </div>
                  <p className="meta mono">{provider.base_url}</p>
                </button>
              ))}
            </div>
          )}
        </aside>

        <section className="providers-main">
          <section className="panel detail-panel">
            <div className="section-head">
              <h2>{selectedProvider ? `Providers / ${selectedProvider.name}` : "Providers"}</h2>
              <span>{selectedProvider?.status.is_active ? "active" : "standby"}</span>
            </div>

            {!selectedProvider ? (
              <div className="empty-state">
                <p>Select a provider to inspect its models and actions.</p>
              </div>
            ) : (
              <>
                <div className="settings-grid">
                  <div className="settings-card">
                    <p className="settings-label">Base URL</p>
                    <p className="mono">{selectedProvider.base_url}</p>
                  </div>
                  <div className="settings-card">
                    <p className="settings-label">Health</p>
                    <p className="mono">{selectedProvider.status.last_health_status}</p>
                  </div>
                  <div className="settings-card">
                    <p className="settings-label">API Key</p>
                    <p className="mono">{selectedProvider.api_key_masked}</p>
                  </div>
                  <div className="settings-card">
                    <p className="settings-label">Capabilities</p>
                    <p className="mono">
                      {selectedProvider.capabilities.supports_models_api ? "models " : ""}
                      {selectedProvider.capabilities.supports_balance_api ? "balance " : ""}
                      {selectedProvider.capabilities.supports_stream ? "stream" : ""}
                    </p>
                  </div>
                </div>

                <div className="settings-actions settings-actions-split">
                  <div className="settings-action-row">
                    {!selectedProvider.status.is_active ? (
                      <button type="button" onClick={() => void handleActivateProvider(selectedProvider)}>
                        Activate
                      </button>
                    ) : null}
                    <button
                      type="button"
                      className="secondary-button"
                      onClick={() => {
                        startEditing(selectedProvider);
                      }}
                    >
                      Edit
                    </button>
                    <button
                      type="button"
                      className="secondary-button"
                      onClick={() => {
                        void handleHealthcheck(selectedProvider.id);
                      }}
                    >
                      Check
                    </button>
                    <button
                      type="button"
                      className="danger-button"
                      onClick={() => {
                        void handleDeleteProvider(selectedProvider.id);
                      }}
                    >
                      Delete
                    </button>
                  </div>
                </div>
              </>
            )}
          </section>
        </section>
      </section>
    </main>
  );
}
