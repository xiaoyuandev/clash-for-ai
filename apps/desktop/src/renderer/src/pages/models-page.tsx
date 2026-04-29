import { useCallback, useEffect, useMemo, useState } from "react";
import { ToastRegion, type ToastItem } from "../components/toast-region";
import { useI18n } from "../i18n/i18n-provider";
import {
  createModelSource,
  deleteModelSource,
  getModelSources,
  getLocalGatewaySelectedModels,
  getProviderModels,
  getProviders,
  getSelectedProviderModels,
  updateLocalGatewaySelectedModels,
  updateModelSource,
  updateModelSourceOrder,
  updateSelectedProviderModels
} from "../services/api";
import type { ModelSource } from "../types/model-source";
import type { Provider } from "../types/provider";
import type { ProviderModel } from "../types/provider-model";
import type { SelectedModel } from "../types/selected-model";
import {
  actionRowClass,
  buttonClass,
  columnCardClass,
  compactStatGridClass,
  emptyStateClass,
  eyebrowClass,
  fieldLabelClass,
  heroClass,
  heroContentClass,
  heroCopyClass,
  heroLabelStackClass,
  heroTitleClass,
  iconBadgeClass,
  iconButtonClass,
  inputClass,
  labelClass,
  metricNumberClass,
  metaClass,
  modalBackdropClass,
  modalPanelClass,
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
import {
  decorateProvidersWithLocalGateway,
  LOCAL_GATEWAY_PROVIDER_ID
} from "../utils/local-gateway-provider";

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
  const [modelSources, setModelSources] = useState<ModelSource[]>([]);
  const [availableModels, setAvailableModels] = useState<ProviderModel[]>([]);
  const [selectedModels, setSelectedModels] = useState<SelectedModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [search, setSearch] = useState("");
  const [draggedModelId, setDraggedModelId] = useState<string | null>(null);
  const [modelSourceFormOpen, setModelSourceFormOpen] = useState(false);
  const [editingModelSourceId, setEditingModelSourceId] = useState<string | null>(null);
  const [modelSourceForm, setModelSourceForm] = useState({
    name: "",
    base_url: "",
    provider_type: "openai-compatible",
    default_model_id: "",
    enabled: true,
    api_key: ""
  });
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

        const decoratedProviders = decorateProvidersWithLocalGateway(items, apiBase);
        setProviders(decoratedProviders);
        onSelectedProviderChange(
          decoratedProviders.find((provider) => provider.status.is_active) ??
            decoratedProviders.find((provider) => provider.id === selectedProvider?.id) ??
            decoratedProviders[0] ??
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
        setModelSources([]);
        setAvailableModels([]);
        setSelectedModels([]);
        setLoading(false);
        return;
      }

      setLoading(true);
      try {
        if (activeProvider.id === LOCAL_GATEWAY_PROVIDER_ID) {
          const [sources, selected] = await Promise.all([
            getModelSources(apiBase),
            getLocalGatewaySelectedModels(apiBase)
          ]);

          if (cancelled) {
            return;
          }

          setModelSources(sources);
          setAvailableModels(
            sources
              .filter((source) => source.enabled)
              .map((source) => ({
                id: source.default_model_id,
                object: "model_source",
                owned_by: source.provider_type || source.name
              }))
          );
          setSelectedModels(selected);
          setError(null);
          setLoading(false);
          return;
        }

        const [available, selected] = await Promise.all([
          getProviderModels(activeProvider.id, apiBase),
          getSelectedProviderModels(activeProvider.id, apiBase)
        ]);

        if (cancelled) {
          return;
        }

        setModelSources([]);
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
  const isLocalGatewayProvider = activeProvider?.id === LOCAL_GATEWAY_PROVIDER_ID;

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
      const saved = isLocalGatewayProvider
        ? await updateLocalGatewaySelectedModels(nextItems, apiBase)
        : await updateSelectedProviderModels(activeProvider.id, nextItems, apiBase);
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

  async function reloadModelSources() {
    const [sources, selected] = await Promise.all([
      getModelSources(apiBase),
      getLocalGatewaySelectedModels(apiBase)
    ]);
    setModelSources(sources);
    setAvailableModels(
      sources
        .filter((source) => source.enabled)
        .map((source) => ({
          id: source.default_model_id,
          object: "model_source",
          owned_by: source.provider_type || source.name
        }))
    );
    setSelectedModels(selected);
  }

  function resetModelSourceForm() {
    setModelSourceForm({
      name: "",
      base_url: "",
      provider_type: "openai-compatible",
      default_model_id: "",
      enabled: true,
      api_key: ""
    });
    setEditingModelSourceId(null);
    setModelSourceFormOpen(false);
  }

  async function handleSaveModelSource() {
    setSaving(true);
    setFeedback(null);
    setError(null);

    try {
      if (editingModelSourceId) {
        await updateModelSource(editingModelSourceId, modelSourceForm, apiBase);
      } else {
        await createModelSource(modelSourceForm, apiBase);
      }
      await reloadModelSources();
      resetModelSourceForm();
      setFeedback(editingModelSourceId ? t("models.feedback.orderUpdated") : t("models.feedback.orderUpdated"));
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : t("common.unknownError"));
    } finally {
      setSaving(false);
    }
  }

  async function handleDeleteModelSource(id: string) {
    setSaving(true);
    setFeedback(null);
    setError(null);

    try {
      const target = modelSources.find((source) => source.id === id);
      await deleteModelSource(id, apiBase);
      if (target) {
        await updateLocalGatewaySelectedModels(
          selectedModels
            .filter((item) => item.model_id !== target.default_model_id)
            .map((item, index) => ({ ...item, position: index })),
          apiBase
        );
      }
      await reloadModelSources();
      setFeedback(t("models.feedback.removed"));
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : t("common.unknownError"));
    } finally {
      setSaving(false);
    }
  }

  function startEditModelSource(source: ModelSource) {
    setEditingModelSourceId(source.id);
    setModelSourceForm({
      name: source.name,
      base_url: source.base_url,
      provider_type: source.provider_type,
      default_model_id: source.default_model_id,
      enabled: source.enabled,
      api_key: source.api_key
    });
    setModelSourceFormOpen(true);
  }

  async function moveModelSource(targetModelID: string) {
    const current = [...modelSources].filter((source) => source.enabled);
    const dragged = current.find((source) => source.default_model_id === draggedModelId);
    const toIndex = current.findIndex((source) => source.default_model_id === targetModelID);
    const fromIndex = current.findIndex((source) => source.default_model_id === draggedModelId);
    if (!dragged || fromIndex < 0 || toIndex < 0) {
      return;
    }
    current.splice(fromIndex, 1);
    current.splice(toIndex, 0, dragged);

    setSaving(true);
    setFeedback(null);
    setError(null);

    try {
      const reordered = await updateModelSourceOrder(
        current.map((source, index) => ({ ...source, position: index })),
        apiBase
      );
      const currentByID = new Map(reordered.map((source) => [source.id, source]));
      setModelSources((previous) =>
        previous.map((source) => currentByID.get(source.id) ?? source)
      );
      setFeedback(t("models.feedback.orderUpdated"));
    } catch (moveError) {
      setError(moveError instanceof Error ? moveError.message : t("common.unknownError"));
    } finally {
      setSaving(false);
    }
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
          {isLocalGatewayProvider ? (
            <button
              type="button"
              className={buttonClass("primary")}
              onClick={() => setModelSourceFormOpen(true)}
            >
              {t("providers.form.addTitle")}
            </button>
          ) : null}
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
            <div className={`${compactStatGridClass} mt-4`}>
              <div className="rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3">
                <p className={metaClass}>{activeProvider.name}</p>
                <p className={metricNumberClass}>{providerModelCount}</p>
                <p className="text-xs text-[color:var(--color-muted)]">
                  {t("models.stats.providerModels")}
                </p>
              </div>
              <div className="rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3">
                <p className={metaClass}>{t("models.stats.availableToAdd")}</p>
                <p className={metricNumberClass}>{availableCount}</p>
                <p className="text-xs text-[color:var(--color-muted)]">
                  {t("models.stats.filteredBySearch")}
                </p>
              </div>
              <div className="rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3">
                <p className={metaClass}>{t("models.stats.failoverSlots")}</p>
                <p className={metricNumberClass}>{selectedModels.length}</p>
                <p className="text-xs text-[color:var(--color-muted)]">
                  {t("models.stats.activeChain")}
                </p>
              </div>
            </div>

            <div
              className="mt-3 flex min-h-0 flex-col gap-3 xl:grid xl:h-[min(62vh,720px)] xl:items-stretch"
              style={{
                gridTemplateColumns: `minmax(0, ${leftPaneWidth}fr) 12px minmax(0, ${
                  100 - leftPaneWidth
                }fr)`
              }}
            >
              <section className={columnCardClass}>
                <div className={sectionHeadClass}>
                  <div className="space-y-1">
                    <h3 className={sectionTitleClass}>
                      {isLocalGatewayProvider ? t("models.available.addedTitle") : t("models.available.title")}
                    </h3>
                    <p className={sectionMetaClass}>{filteredAvailableModels.length}</p>
                  </div>
                </div>

                <div className={`${stickySearchClass} mt-3`}>
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
                  <p className="px-1 pt-2 text-xs text-[color:var(--color-muted)]">
                    Search by exact model id or provider-assigned alias.
                  </p>
                </div>

                <div className={`${scrollListClass} mt-3`}>
                  {filteredAvailableModels.length === 0 ? (
                    <div className={emptyStateClass}>
                      <p>{loading ? t("common.loading") : t("models.available.empty")}</p>
                    </div>
                  ) : (
                    filteredAvailableModels.map((model) => (
                      <article key={model.id} className={queueItemClass}>
                        {isLocalGatewayProvider ? (
                          <div className="absolute right-2.5 top-2.5 flex items-center gap-2">
                            <button
                              type="button"
                              className={`${iconButtonClass} min-h-8 min-w-8 rounded-lg`}
                              aria-label={t("models.available.add", { id: model.id })}
                              title={t("models.available.add", { id: model.id })}
                              onClick={() => addModel(model.id)}
                            >
                              <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                                <path d="M11 5h2v14h-2z" />
                                <path d="M5 11h14v2H5z" />
                              </svg>
                            </button>
                            <button
                              type="button"
                              className={`${iconButtonClass} min-h-8 min-w-8 rounded-lg`}
                              aria-label={t("common.edit")}
                              title={t("common.edit")}
                              onClick={() => {
                                const target = modelSources.find((source) => source.default_model_id === model.id)
                                if (target) {
                                  startEditModelSource(target)
                                }
                              }}
                            >
                              <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                                <path d="M13.4 3.4a2 2 0 0 1 2.8 0l4.4 4.4a2 2 0 0 1 0 2.8l-2.1 2.1-7.2-7.2zM10.1 6.7 3 13.8V21h7.2l7.1-7.1zM6 18H5v-1l7.4-7.4 1 1z" />
                              </svg>
                            </button>
                            <button
                              type="button"
                              className={`${iconButtonClass} min-h-8 min-w-8 rounded-lg`}
                              aria-label={t("common.delete")}
                              title={t("common.delete")}
                              onClick={() => {
                                const target = modelSources.find((source) => source.default_model_id === model.id)
                                if (target) {
                                  void handleDeleteModelSource(target.id)
                                }
                              }}
                            >
                              <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                                <path d="M9 3h6l1 2h4v2H4V5h4zm1 6h2v8h-2zm4 0h2v8h-2zM7 9h2v8H7zm1 12a2 2 0 0 1-2-2V8h12v11a2 2 0 0 1-2 2z" />
                              </svg>
                            </button>
                          </div>
                        ) : (
                          <button
                            type="button"
                            className={`${iconButtonClass} absolute right-2.5 top-2.5 min-h-8 min-w-8 rounded-lg`}
                            aria-label={t("models.available.add", { id: model.id })}
                            title={t("models.available.add", { id: model.id })}
                            onClick={() => addModel(model.id)}
                          >
                            <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                              <path d="M11 5h2v14h-2z" />
                              <path d="M5 11h14v2H5z" />
                            </svg>
                          </button>
                        )}
                        <div
                          className={`flex items-start gap-2.5 ${
                            isLocalGatewayProvider ? "pr-32" : "pr-10"
                          }`}
                        >
                          <span className={`${iconBadgeClass} mt-0.5`}>
                            <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                              <path d="M12 3 4 7v10l8 4 8-4V7zm0 2.2L17.8 8 12 10.8 6.2 8zM6 9.6l5 2.5v6.2l-5-2.5zm7 8.7v-6.2l5-2.5v6.2z" />
                            </svg>
                          </span>
                          <div>
                            <p className={monoClass}>{model.id}</p>
                            <p className={`${metaClass} mt-1.5`}>
                              {model.owned_by ?? t("models.available.ownerUnknown")}
                            </p>
                            {isLocalGatewayProvider ? (
                              <>
                                <p className={`${metaClass} mt-1`}>
                                  {
                                    modelSources.find((source) => source.default_model_id === model.id)?.name ??
                                    "-"
                                  }
                                </p>
                                <p className={`${metaClass} mt-1`}>
                                  {
                                    modelSources.find((source) => source.default_model_id === model.id)?.base_url ??
                                    "-"
                                  }
                                </p>
                              </>
                            ) : null}
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
                <p className={`${metaClass} mt-3`}>{t("models.fallback.subtitle")}</p>

                <div className={`${scrollListClass} content-start auto-rows-max mt-3`}>
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
                          if (isLocalGatewayProvider) {
                            void moveModelSource(item.model_id);
                          } else {
                            moveModel(item.model_id);
                          }
                        }}
                      >
                        <div className="flex items-start gap-3">
                          <div className="grid gap-1.5">
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
                            <p className={`${metaClass} mt-1.5`}>
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

      {isLocalGatewayProvider && modelSourceFormOpen ? (
        <div className={modalBackdropClass} role="presentation" onClick={resetModelSourceForm}>
          <section
            className={`${modalPanelClass} max-w-3xl`}
            role="dialog"
            aria-modal="true"
            aria-label={t("providers.form.addTitle")}
            onClick={(event) => event.stopPropagation()}
          >
            <div className={sectionHeadClass}>
              <div className="space-y-1">
                <h2 className={sectionTitleClass}>
                  {editingModelSourceId ? t("providers.form.editTitle") : t("providers.form.addTitle")}
                </h2>
                <p className={sectionMetaClass}>{t("models.subtitle")}</p>
              </div>
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={resetModelSourceForm}
              >
                {t("common.close")}
              </button>
            </div>

            <div className="mt-4 grid gap-3 md:grid-cols-2">
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("providers.form.name")}</span>
                <input
                  className={inputClass}
                  value={modelSourceForm.name}
                  onChange={(event) =>
                    setModelSourceForm((current) => ({ ...current, name: event.target.value }))
                  }
                />
              </label>
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("providers.form.baseUrl")}</span>
                <input
                  className={inputClass}
                  value={modelSourceForm.base_url}
                  onChange={(event) =>
                    setModelSourceForm((current) => ({ ...current, base_url: event.target.value }))
                  }
                />
              </label>
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("models.available.title")}</span>
                <input
                  className={inputClass}
                  value={modelSourceForm.default_model_id}
                  onChange={(event) =>
                    setModelSourceForm((current) => ({
                      ...current,
                      default_model_id: event.target.value
                    }))
                  }
                />
              </label>
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("providers.form.apiKey")}</span>
                <input
                  className={inputClass}
                  value={modelSourceForm.api_key}
                  onChange={(event) =>
                    setModelSourceForm((current) => ({ ...current, api_key: event.target.value }))
                  }
                />
              </label>
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("settings.guide.field.providerType")}</span>
                <select
                  className={inputClass}
                  value={modelSourceForm.provider_type}
                  onChange={(event) =>
                    setModelSourceForm((current) => ({
                      ...current,
                      provider_type: event.target.value
                    }))
                  }
                >
                  <option value="openai-compatible">OpenAI Compatible</option>
                  <option value="anthropic-compatible">Anthropic Compatible</option>
                  <option value="gemini-compatible">Gemini Compatible</option>
                </select>
              </label>
            </div>
            <div className={`${actionRowClass} mt-4`}>
              <button
                type="button"
                className={buttonClass("primary")}
                onClick={() => void handleSaveModelSource()}
                disabled={saving}
              >
                {saving ? t("common.saving") : editingModelSourceId ? t("providers.form.save") : t("providers.form.create")}
              </button>
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={resetModelSourceForm}
              >
                {t("common.cancel")}
              </button>
            </div>
          </section>
        </div>
      ) : null}
    </main>
  );
}
