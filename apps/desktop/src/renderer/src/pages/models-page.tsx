import { useCallback, useEffect, useMemo, useState } from "react";
import { ToastRegion, type ToastItem } from "../components/toast-region";
import { useI18n } from "../i18n/i18n-provider";
import {
  getProviderModels,
  getProviders,
  getSelectedProviderModels,
  updateSelectedProviderModels
} from "../services/api";
import type { Provider } from "../types/provider";
import type { ProviderModel } from "../types/provider-model";
import type { SelectedModel } from "../types/selected-model";
import {
  buttonClass,
  columnCardClass,
  compactStatGridClass,
  emptyStateClass,
  eyebrowClass,
  heroClass,
  heroContentClass,
  heroCopyClass,
  heroLabelStackClass,
  heroTitleClass,
  iconBadgeClass,
  iconButtonClass,
  inputClass,
  metricNumberClass,
  metaClass,
  monoClass,
  pageShellClass,
  queueItemClass,
  scrollListClass,
  sectionCardClass,
  sectionHeadClass,
  sectionMetaClass,
  sectionTitleClass,
  statusPillClass,
  stickySearchClass
} from "../ui";

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
  const { t } = useI18n();
  const [providers, setProviders] = useState<Provider[]>([]);
  const [availableModels, setAvailableModels] = useState<ProviderModel[]>([]);
  const [selectedModels, setSelectedModels] = useState<SelectedModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [toasts, setToasts] = useState<ToastItem[]>([]);
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
          setError(loadError instanceof Error ? loadError.message : t("common.unknownError"));
        }
      }
    }

    void loadProviders();

    return () => {
      cancelled = true;
    };
  }, [apiBase, onSelectedProviderChange, selectedProvider?.id, t]);

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
          setError(loadError instanceof Error ? loadError.message : t("common.unknownError"));
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
  }, [activeProvider, apiBase, t]);

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
  const availableCount = filteredAvailableModels.length;
  const providerModelCount = availableModels.length;

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
    if (!feedback) {
      return;
    }
    setToasts((current) => [
      ...current,
      { id: `${Date.now()}-success`, message: feedback, tone: "success" }
    ]);
    setFeedback(null);
  }, [feedback]);

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
      setError(saveError instanceof Error ? saveError.message : t("common.unknownError"));
    } finally {
      setSaving(false);
    }
  }

  function addModel(modelID: string) {
    void persistSelectedModels(
      [...selectedModels, { model_id: modelID, position: selectedModels.length }],
      t("models.feedback.orderUpdated")
    );
  }

  function removeModel(modelID: string) {
    void persistSelectedModels(
      selectedModels
        .filter((item) => item.model_id !== modelID)
        .map((item, index) => ({ ...item, position: index })),
      t("models.feedback.removed")
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
      t("models.feedback.orderUpdated")
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
    <main className={pageShellClass}>
      <ToastRegion items={toasts} onDismiss={dismissToast} />
      <section className={heroClass}>
        <div className={heroContentClass}>
          <div className={heroLabelStackClass}>
            <p className={eyebrowClass}>Clash for AI</p>
            <h1 className={heroTitleClass}>{t("models.title")}</h1>
          </div>
          <p className={heroCopyClass}>{t("models.subtitle")}</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <span className={statusPillClass(activeProvider ? "success" : "default")}>
            {activeProvider
              ? t("models.section.title", { name: activeProvider.name })
              : t("models.section.fallbackTitle")}
          </span>
          <span className={statusPillClass(saving ? "warning" : "default")}>
            {saving
              ? t("common.saving")
              : loading
                ? t("common.loading")
                : t("models.section.state.selected", { count: selectedModels.length })}
          </span>
        </div>
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>
              {activeProvider
                ? t("models.section.title", { name: activeProvider.name })
                : t("models.section.fallbackTitle")}
            </h2>
            <p className={sectionMetaClass}>
              {activeProvider
                ? t("models.subtitle")
                : t("models.empty.noActiveProvider")}
            </p>
          </div>
        </div>

        {!activeProvider ? (
          <div className="mt-6">
            <div className={emptyStateClass}>
              <p>{t("models.empty.noActiveProvider")}</p>
            </div>
          </div>
        ) : (
          <>
            <div className={`${compactStatGridClass} mt-6`}>
              <div className="rounded-3xl border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-4">
                <p className={metaClass}>{activeProvider.name}</p>
                <p className={metricNumberClass}>{providerModelCount}</p>
                <p className="text-xs text-[color:var(--color-muted)]">
                  {t("models.stats.providerModels")}
                </p>
              </div>
              <div className="rounded-3xl border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-4">
                <p className={metaClass}>{t("models.stats.availableToAdd")}</p>
                <p className={metricNumberClass}>{availableCount}</p>
                <p className="text-xs text-[color:var(--color-muted)]">
                  {t("models.stats.filteredBySearch")}
                </p>
              </div>
              <div className="rounded-3xl border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-4">
                <p className={metaClass}>{t("models.stats.failoverSlots")}</p>
                <p className={metricNumberClass}>{selectedModels.length}</p>
                <p className="text-xs text-[color:var(--color-muted)]">
                  {t("models.stats.activeChain")}
                </p>
              </div>
            </div>

            <div
              className="mt-4 flex min-h-0 flex-col gap-4 xl:grid xl:h-[min(62vh,720px)] xl:items-stretch"
              style={{
                gridTemplateColumns: `minmax(0, ${leftPaneWidth}fr) 16px minmax(0, ${
                  100 - leftPaneWidth
                }fr)`
              }}
            >
              <section className={columnCardClass}>
                <div className={sectionHeadClass}>
                  <div className="space-y-1">
                    <h3 className={sectionTitleClass}>{t("models.available.title")}</h3>
                    <p className={sectionMetaClass}>{filteredAvailableModels.length}</p>
                  </div>
                </div>

                <div className={`${stickySearchClass} mt-4`}>
                  <label className="relative block">
                    <input
                      className={`${inputClass} pr-11`}
                      value={search}
                      onChange={(event) => setSearch(event.target.value)}
                      placeholder={t("models.available.searchPlaceholder")}
                    />
                    <span
                      className="pointer-events-none absolute inset-y-0 right-4 inline-flex items-center text-[color:var(--color-subtle)]"
                      aria-hidden="true"
                    >
                      <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24">
                        <path d="M10.5 4a6.5 6.5 0 1 1 0 13 6.5 6.5 0 0 1 0-13Zm0 2a4.5 4.5 0 1 0 0 9 4.5 4.5 0 0 0 0-9Z" />
                        <path d="m15.3 14 4.7 4.7-1.4 1.4-4.7-4.7z" />
                      </svg>
                    </span>
                  </label>
                  <p className="px-1 pt-3 text-xs text-[color:var(--color-muted)]">
                    Search by exact model id or provider-assigned alias.
                  </p>
                </div>

                <div className={`${scrollListClass} mt-4`}>
                  {filteredAvailableModels.length === 0 ? (
                    <div className={emptyStateClass}>
                      <p>{loading ? t("common.loading") : t("models.available.empty")}</p>
                    </div>
                  ) : (
                    filteredAvailableModels.map((model) => (
                      <article key={model.id} className={queueItemClass}>
                        <button
                          type="button"
                          className={`${iconButtonClass} absolute right-3 top-3 min-h-9 min-w-9 rounded-xl`}
                          aria-label={t("models.available.add", { id: model.id })}
                          title={t("models.available.add", { id: model.id })}
                          onClick={() => addModel(model.id)}
                        >
                          <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                            <path d="M11 5h2v14h-2z" />
                            <path d="M5 11h14v2H5z" />
                          </svg>
                        </button>
                        <div className="flex items-start gap-3 pr-12">
                          <span className={`${iconBadgeClass} mt-0.5`}>
                            <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                              <path d="M12 3 4 7v10l8 4 8-4V7zm0 2.2L17.8 8 12 10.8 6.2 8zM6 9.6l5 2.5v6.2l-5-2.5zm7 8.7v-6.2l5-2.5v6.2z" />
                            </svg>
                          </span>
                          <div>
                            <p className={monoClass}>{model.id}</p>
                            <p className={`${metaClass} mt-2`}>
                              {model.owned_by ?? t("models.available.ownerUnknown")}
                            </p>
                          </div>
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
                aria-label={t("models.resizeColumns")}
                onPointerDown={startResize}
              >
                <span className="pane-resizer-line" />
              </div>

              <section className={columnCardClass}>
                <div className={sectionHeadClass}>
                  <div className="space-y-1">
                    <h3 className={sectionTitleClass}>{t("models.fallback.title")}</h3>
                    <p className={sectionMetaClass}>{selectedModels.length}</p>
                  </div>
                </div>
                <p className={`${metaClass} mt-4`}>{t("models.fallback.subtitle")}</p>

                <div className={`${scrollListClass} mt-4`}>
                  {selectedModelDetails.length === 0 ? (
                    <div className={emptyStateClass}>
                      <p>{t("models.fallback.empty")}</p>
                    </div>
                  ) : (
                    selectedModelDetails.map((item, index) => (
                      <article
                        key={item.model_id}
                        className={`${queueItemClass} cursor-grab`}
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
                        <div className="flex items-start gap-4">
                          <div className="grid gap-2">
                            <span
                              className="pt-1 text-[11px] font-bold uppercase tracking-[0.3em] text-[color:var(--accent)]/75"
                              aria-hidden="true"
                            >
                              :::
                            </span>
                            <span className="text-xs font-semibold text-[color:var(--color-subtle)]">
                              #{index + 1}
                            </span>
                          </div>
                          <div>
                            <p className={monoClass}>{item.model_id}</p>
                            <p className={`${metaClass} mt-2`}>
                              {index === 0
                                ? t("models.fallback.primary")
                                : t("models.fallback.secondary", { index })}
                              {item.details?.owned_by ? ` · ${item.details.owned_by}` : ""}
                            </p>
                          </div>
                        </div>
                        <button
                          type="button"
                          className={buttonClass("secondary")}
                          onClick={() => removeModel(item.model_id)}
                        >
                          {t("models.fallback.remove")}
                        </button>
                      </article>
                    ))
                  )}
                </div>
              </section>
            </div>
          </>
        )}
      </section>
    </main>
  );
}
