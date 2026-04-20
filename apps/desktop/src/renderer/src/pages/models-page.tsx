import { useEffect, useMemo, useState } from "react";
import {
  getProviderModels,
  getProviders,
  getSelectedProviderModels,
  updateSelectedProviderModels
} from "../services/api";
import type { Provider } from "../types/provider";
import type { ProviderModel } from "../types/provider-model";
import type { SelectedModel } from "../types/selected-model";

interface ModelsPageProps {
  apiBase?: string;
  selectedProvider: Provider | null;
  onSelectedProviderChange: (provider: Provider | null) => void;
}

export function ModelsPage({
  apiBase,
  selectedProvider,
  onSelectedProviderChange
}: ModelsPageProps) {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [availableModels, setAvailableModels] = useState<ProviderModel[]>([]);
  const [selectedModels, setSelectedModels] = useState<SelectedModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [draggedModelId, setDraggedModelId] = useState<string | null>(null);
  const activeProvider = providers.find((provider) => provider.status.is_active) ?? null;
  const [leftPaneWidth, setLeftPaneWidth] = useState(48);

  useEffect(() => {
    let cancelled = false;

    async function loadProviders() {
      try {
        const items = await getProviders(apiBase);
        if (cancelled) {
          return;
        }

        setProviders(items);
        onSelectedProviderChange(
          items.find((provider) => provider.status.is_active) ??
            items.find((provider) => provider.id === selectedProvider?.id) ??
            null
        );
      } catch (loadError) {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : "Unknown error");
        }
      }
    }

    void loadProviders();

    return () => {
      cancelled = true;
    };
  }, [apiBase, onSelectedProviderChange, selectedProvider?.id]);

  useEffect(() => {
    let cancelled = false;

    async function loadModels() {
      if (!activeProvider) {
        setAvailableModels([]);
        setSelectedModels([]);
        setLoading(false);
        return;
      }

      setLoading(true);
      try {
        const [available, selected] = await Promise.all([
          getProviderModels(activeProvider.id, apiBase),
          getSelectedProviderModels(activeProvider.id, apiBase)
        ]);

        if (cancelled) {
          return;
        }

        setAvailableModels(available);
        setSelectedModels(selected);
        setError(null);
      } catch (loadError) {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : "Unknown error");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadModels();

    return () => {
      cancelled = true;
    };
  }, [activeProvider, apiBase]);

  const selectedModelIds = new Set(selectedModels.map((item) => item.model_id));

  const filteredAvailableModels = useMemo(() => {
    return availableModels.filter((model) => {
      if (selectedModelIds.has(model.id)) {
        return false;
      }

      if (!search.trim()) {
        return true;
      }

      return model.id.toLowerCase().includes(search.trim().toLowerCase());
    });
  }, [availableModels, search, selectedModelIds]);

  const selectedModelDetails = selectedModels.map((item) => ({
    ...item,
    details: availableModels.find((model) => model.id === item.model_id)
  }));

  async function persistSelectedModels(nextItems: SelectedModel[], successMessage: string) {
    if (!activeProvider) {
      return;
    }

    setSaving(true);
    setFeedback(null);
    setError(null);

    try {
      const saved = await updateSelectedProviderModels(activeProvider.id, nextItems, apiBase);
      setSelectedModels(saved);
      setFeedback(successMessage);
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : "Unknown error");
    } finally {
      setSaving(false);
    }
  }

  function addModel(modelID: string) {
    void persistSelectedModels(
      [...selectedModels, { model_id: modelID, position: selectedModels.length }],
      "Fallback order updated."
    );
  }

  function removeModel(modelID: string) {
    void persistSelectedModels(
      selectedModels
        .filter((item) => item.model_id !== modelID)
        .map((item, index) => ({ ...item, position: index })),
      "Model removed from fallback order."
    );
  }

  function moveModel(targetModelID: string) {
    if (!draggedModelId || draggedModelId === targetModelID) {
      return;
    }

    const current = [...selectedModels];
    const fromIndex = current.findIndex((item) => item.model_id === draggedModelId);
    const toIndex = current.findIndex((item) => item.model_id === targetModelID);
    if (fromIndex < 0 || toIndex < 0) {
      return;
    }

    const [moved] = current.splice(fromIndex, 1);
    current.splice(toIndex, 0, moved);
    void persistSelectedModels(
      current.map((item, index) => ({ ...item, position: index })),
      "Fallback order updated."
    );
  }

  function startResize(event: React.PointerEvent<HTMLDivElement>) {
    const container = event.currentTarget.parentElement;
    if (!container) {
      return;
    }

    const rect = container.getBoundingClientRect();
    const onMove = (moveEvent: PointerEvent) => {
      const nextWidth = ((moveEvent.clientX - rect.left) / rect.width) * 100;
      setLeftPaneWidth(Math.min(70, Math.max(30, nextWidth)));
    };

    const onUp = () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };

    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
  }

  return (
    <main className="page-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Clash for AI</p>
          <h1>Models</h1>
          <p className="subcopy">
            The left side shows the current active provider's supported models. Pick favorites into the ordered fallback list on the right.
          </p>
        </div>
      </section>

      {error ? <p className="panel error-panel">{error}</p> : null}
      {feedback ? <p className="panel info-panel">{feedback}</p> : null}

      <section className="panel">
        <div className="section-head">
          <h2>{activeProvider ? `Models / ${activeProvider.name}` : "Models"}</h2>
          <span>{saving ? "saving" : loading ? "loading" : `${selectedModels.length} selected`}</span>
        </div>

        {!activeProvider ? (
          <div className="empty-state">
            <p>Activate a provider first. This page only shows the current enabled provider.</p>
          </div>
        ) : (
          <div
            className="models-layout models-resizable-layout"
            style={{
              gridTemplateColumns: `${leftPaneWidth}fr 16px ${100 - leftPaneWidth}fr`
            }}
          >
            <section className="settings-card models-column">
              <div className="section-head">
                <h3>Supported Models</h3>
                <span>{filteredAvailableModels.length}</span>
              </div>
              <div className="sticky-search-shell">
                <label className="search-box compact-search-box">
                  <input
                    value={search}
                    onChange={(event) => setSearch(event.target.value)}
                    placeholder="model id..."
                  />
                  <span className="search-icon" aria-hidden="true">
                    <svg viewBox="0 0 24 24">
                      <path d="M10.5 4a6.5 6.5 0 1 1 0 13 6.5 6.5 0 0 1 0-13Zm0 2a4.5 4.5 0 1 0 0 9 4.5 4.5 0 0 0 0-9Z" />
                      <path d="m15.3 14 4.7 4.7-1.4 1.4-4.7-4.7z" />
                    </svg>
                  </span>
                </label>
              </div>

              <div className="models-list models-scroll-list">
                {filteredAvailableModels.length === 0 ? (
                  <div className="empty-state">
                    <p>{loading ? "Loading models..." : "No more models to add."}</p>
                  </div>
                ) : (
                  filteredAvailableModels.map((model) => (
                    <article key={model.id} className="queue-item supported-model-card">
                      <button
                        type="button"
                        className="icon-button icon-button-small supported-model-add"
                        aria-label={`Add ${model.id}`}
                        title={`Add ${model.id}`}
                        onClick={() => addModel(model.id)}
                      >
                        <svg viewBox="0 0 24 24" aria-hidden="true">
                          <path d="M11 5h2v14h-2z" />
                          <path d="M5 11h14v2H5z" />
                        </svg>
                      </button>
                      <div>
                        <p className="mono">{model.id}</p>
                        <p className="meta">{model.owned_by ?? "unknown owner"}</p>
                      </div>
                    </article>
                  ))
                )}
              </div>
            </section>

            <div
              className="pane-resizer"
              role="separator"
              aria-orientation="vertical"
              aria-label="Resize model columns"
              onPointerDown={startResize}
            >
              <span className="pane-resizer-line" />
            </div>

            <section className="settings-card models-column">
              <div className="section-head">
                <h3>Fallback Order</h3>
                <span>{selectedModels.length}</span>
              </div>
              <p className="meta">
                Drag to reorder. Requests use the first model by default, then fall back in this order.
              </p>

              <div className="models-list models-scroll-list">
                {selectedModelDetails.length === 0 ? (
                  <div className="empty-state">
                    <p>No models selected yet.</p>
                  </div>
                ) : (
                  selectedModelDetails.map((item, index) => (
                    <article
                      key={item.model_id}
                      className="queue-item draggable-item"
                      draggable
                      onDragStart={() => {
                        setDraggedModelId(item.model_id);
                      }}
                      onDragOver={(event) => {
                        event.preventDefault();
                      }}
                      onDrop={() => {
                        moveModel(item.model_id);
                      }}
                    >
                      <div className="queue-item-head">
                        <span className="drag-handle" aria-hidden="true">
                          :::
                        </span>
                        <div>
                          <p className="mono">
                            {index + 1}. {item.model_id}
                          </p>
                          <p className="meta">
                            {index === 0 ? "Primary model" : `Fallback ${index}`}
                            {item.details?.owned_by ? ` · ${item.details.owned_by}` : ""}
                          </p>
                        </div>
                      </div>
                      <button
                        type="button"
                        className="secondary-button"
                        onClick={() => removeModel(item.model_id)}
                      >
                        Remove
                      </button>
                    </article>
                  ))
                )}
              </div>
            </section>
          </div>
        )}
      </section>
    </main>
  );
}
