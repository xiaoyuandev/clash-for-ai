import { useEffect, useMemo, useState } from "react";
import { ToastRegion, type ToastItem } from "../components/toast-region";
import { useI18n } from "../i18n/i18n-provider";
import {
  getLocalGatewayRuntime,
  getLocalGatewaySelectedModels,
  getProviderModels,
  getProviders,
  replaceLocalGatewaySelectedModels
} from "../services/api";
import type { LocalGatewayRuntimeStatus } from "../types/local-gateway";
import type { Provider } from "../types/provider";
import type { ProviderModel } from "../types/provider-model";
import type { SelectedModel } from "../types/selected-model";
import {
  buttonClass,
  compactStatGridClass,
  emptyStateClass,
  eyebrowClass,
  heroClass,
  heroContentClass,
  heroCopyClass,
  heroLabelStackClass,
  heroTitleClass,
  iconBadgeClass,
  inputClass,
  metaClass,
  metricNumberClass,
  monoClass,
  pageShellClass,
  scrollListClass,
  sectionCardClass,
  sectionHeadClass,
  sectionMetaClass,
  sectionTitleClass,
  selectableItemClass,
  splitLayoutClass,
  statusPillClass
} from "../ui";

const LOCAL_GATEWAY_PROVIDER_ID = "system-local-gateway";

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
  const [models, setModels] = useState<ProviderModel[]>([]);
  const [selectedModels, setSelectedModels] = useState<SelectedModel[]>([]);
  const [runtimeStatus, setRuntimeStatus] = useState<LocalGatewayRuntimeStatus | null>(null);
  const [loadingProviders, setLoadingProviders] = useState(true);
  const [loadingModels, setLoadingModels] = useState(false);
  const [loadingSelectedModels, setLoadingSelectedModels] = useState(false);
  const [savingSelectedModels, setSavingSelectedModels] = useState(false);
  const [search, setSearch] = useState("");
  const [draggedModelId, setDraggedModelId] = useState<string | null>(null);
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const activeProvider = useMemo(
    () => providers.find((provider) => provider.status.is_active) ?? null,
    [providers]
  );

  const isLocalGateway = activeProvider?.id === LOCAL_GATEWAY_PROVIDER_ID;
  const selectedModelIDs = useMemo(() => new Set(selectedModels.map((item) => item.model_id)), [selectedModels]);
  const selectedModelInfos = useMemo(
    () =>
      selectedModels.map((item) => ({
        item,
        model: models.find((candidate) => candidate.id === item.model_id) ?? null
      })),
    [models, selectedModels]
  );

  const filteredModels = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    const availableModels = models.filter((model) => !selectedModelIDs.has(model.id));
    if (!keyword) {
      return availableModels;
    }

    return availableModels.filter((model) => {
      const owner = model.owned_by?.toLowerCase() ?? "";
      return model.id.toLowerCase().includes(keyword) || owner.includes(keyword);
    });
  }, [models, search, selectedModelIDs]);

  const localGatewayAdminWritable =
    runtimeStatus?.capabilities.supports_selected_model_admin ?? false;

  useEffect(() => {
    let cancelled = false;

    async function loadProviders() {
      setLoadingProviders(true);
      try {
        const items = await getProviders(apiBase);
        if (cancelled) {
          return;
        }

        setProviders(items);
        onSelectedProviderChange(
          items.find((provider) => provider.status.is_active) ??
            items.find((provider) => provider.id === selectedProvider?.id) ??
            items[0] ??
            null
        );
      } catch (error) {
        if (!cancelled) {
          setToasts((current) => [
            ...current,
            {
              id: `${Date.now()}-providers-error`,
              message: error instanceof Error ? error.message : t("common.unknownError"),
              tone: "error"
            }
          ]);
        }
      } finally {
        if (!cancelled) {
          setLoadingProviders(false);
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
        setModels([]);
        return;
      }

      setLoadingModels(true);
      try {
        const items = await getProviderModels(activeProvider.id, apiBase);
        if (cancelled) {
          return;
        }

        setModels(items);
      } catch (error) {
        if (!cancelled) {
          setModels([]);
          setToasts((current) => [
            ...current,
            {
              id: `${Date.now()}-models-error`,
              message: error instanceof Error ? error.message : t("common.unknownError"),
              tone: "error"
            }
          ]);
        }
      } finally {
        if (!cancelled) {
          setLoadingModels(false);
        }
      }
    }

    void loadModels();

    return () => {
      cancelled = true;
    };
  }, [activeProvider, apiBase, t]);

  useEffect(() => {
    let cancelled = false;

    async function loadLocalGatewayState() {
      if (!isLocalGateway) {
        setRuntimeStatus(null);
        setSelectedModels([]);
        return;
      }

      setLoadingSelectedModels(true);
      try {
        const [runtime, items] = await Promise.all([
          getLocalGatewayRuntime(apiBase),
          getLocalGatewaySelectedModels(apiBase)
        ]);
        if (cancelled) {
          return;
        }
        setRuntimeStatus(runtime);
        setSelectedModels(items);
      } catch (error) {
        if (!cancelled) {
          setRuntimeStatus(null);
          setSelectedModels([]);
          setToasts((current) => [
            ...current,
            {
              id: `${Date.now()}-local-gateway-error`,
              message: error instanceof Error ? error.message : t("common.unknownError"),
              tone: "error"
            }
          ]);
        }
      } finally {
        if (!cancelled) {
          setLoadingSelectedModels(false);
        }
      }
    }

    void loadLocalGatewayState();

    return () => {
      cancelled = true;
    };
  }, [apiBase, isLocalGateway, t]);

  async function persistSelectedModels(nextItems: SelectedModel[]) {
    if (!isLocalGateway || !localGatewayAdminWritable) {
      return;
    }

    try {
      setSavingSelectedModels(true);
      const normalized = nextItems.map((item, index) => ({
        model_id: item.model_id,
        position: index
      }));
      const updated = await replaceLocalGatewaySelectedModels(normalized, apiBase);
      setSelectedModels(updated);
      setToasts((current) => [
        ...current,
        {
          id: `${Date.now()}-selected-models-updated`,
          message: t("models.feedback.orderUpdated"),
          tone: "success"
        }
      ]);
    } catch (error) {
      setToasts((current) => [
        ...current,
        {
          id: `${Date.now()}-selected-models-error`,
          message: error instanceof Error ? error.message : t("common.unknownError"),
          tone: "error"
        }
      ]);
    } finally {
      setSavingSelectedModels(false);
    }
  }

  function addSelectedModel(modelID: string) {
    const nextItems = [...selectedModels, { model_id: modelID, position: selectedModels.length }];
    void persistSelectedModels(nextItems);
  }

  function removeSelectedModel(modelID: string) {
    const nextItems = selectedModels.filter((item) => item.model_id !== modelID);
    void persistSelectedModels(nextItems);
    setToasts((current) => [
      ...current,
      {
        id: `${Date.now()}-selected-model-removed`,
        message: t("models.feedback.removed"),
        tone: "success"
      }
    ]);
  }

  function moveSelectedModel(targetModelID: string) {
    if (!draggedModelId || draggedModelId === targetModelID) {
      return;
    }

    const reordered = [...selectedModels];
    const fromIndex = reordered.findIndex((item) => item.model_id === draggedModelId);
    const toIndex = reordered.findIndex((item) => item.model_id === targetModelID);
    if (fromIndex < 0 || toIndex < 0) {
      return;
    }

    const [moved] = reordered.splice(fromIndex, 1);
    reordered.splice(toIndex, 0, moved);
    void persistSelectedModels(reordered);
  }

  return (
    <main className={pageShellClass}>
      <ToastRegion
        items={toasts}
        onDismiss={(id) => setToasts((current) => current.filter((item) => item.id !== id))}
      />

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
          <span className={statusPillClass(loadingProviders || loadingModels ? "warning" : "default")}>
            {loadingProviders || loadingModels
              ? t("common.loading")
              : t("providers.detail.modelsCount", { count: models.length })}
          </span>
          {isLocalGateway && runtimeStatus ? (
            <span
              className={statusPillClass(
                localGatewayAdminWritable ? "success" : "warning"
              )}
            >
              {localGatewayAdminWritable
                ? t("providers.detail.localGatewayAdminEnabled")
                : t("providers.detail.localGatewayAdminReadOnly")}
            </span>
          ) : null}
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
              {activeProvider ? t("models.subtitle") : t("models.empty.noActiveProvider")}
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
                <p className={metricNumberClass}>{models.length}</p>
                <p className="text-xs text-[color:var(--color-muted)]">
                  {t("models.stats.providerModels")}
                </p>
              </div>
              <div className="rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3">
                <p className={metaClass}>{t("models.stats.availableToAdd")}</p>
                <p className={metricNumberClass}>{filteredModels.length}</p>
                <p className="text-xs text-[color:var(--color-muted)]">
                  {t("models.stats.filteredBySearch")}
                </p>
              </div>
              {isLocalGateway ? (
                <div className="rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3">
                  <p className={metaClass}>{t("models.stats.failoverSlots")}</p>
                  <p className={metricNumberClass}>{selectedModels.length}</p>
                  <p className="text-xs text-[color:var(--color-muted)]">
                    {savingSelectedModels
                      ? t("common.saving")
                      : t("models.stats.activeChain")}
                  </p>
                </div>
              ) : null}
            </div>

            {isLocalGateway && runtimeStatus && !localGatewayAdminWritable ? (
              <div className="mt-4 rounded-[16px] border [border-color:var(--success-border)] [background:color-mix(in_srgb,var(--success-soft)_70%,transparent)] px-3 py-2 text-sm text-[color:var(--accent)]">
                {t("providers.detail.localGatewayReadOnly")}
              </div>
            ) : null}

            <div className={`${isLocalGateway ? splitLayoutClass : ""} mt-4`}>
              <section className="min-h-0">
                <div className="mb-4">
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
                </div>

                <div className={`${scrollListClass} ${isLocalGateway ? "max-h-[540px]" : "mt-4"}`}>
                  {filteredModels.length === 0 ? (
                    <div className={emptyStateClass}>
                      <p>{loadingModels ? t("common.loading") : t("models.available.empty")}</p>
                    </div>
                  ) : (
                    filteredModels.map((model) => (
                      <article key={model.id} className={selectableItemClass(false)}>
                        <div className="flex items-start gap-2.5">
                          <span className={iconBadgeClass}>
                            <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                              <path d="M12 3 4 7v10l8 4 8-4V7zm0 2.2L17.8 8 12 10.8 6.2 8zM6 9.6l5 2.5v6.2l-5-2.5zm7 8.7v-6.2l5-2.5v6.2z" />
                            </svg>
                          </span>
                          <div className="min-w-0 flex-1">
                            <p className={monoClass}>{model.id}</p>
                            <p className={`${metaClass} mt-1.5`}>
                              {model.owned_by ?? t("models.available.ownerUnknown")}
                            </p>
                          </div>
                          {isLocalGateway ? (
                            <button
                              type="button"
                              className={buttonClass("secondary")}
                              disabled={!localGatewayAdminWritable}
                              onClick={() => addSelectedModel(model.id)}
                            >
                              {t("models.available.add", { id: model.id })}
                            </button>
                          ) : null}
                        </div>
                      </article>
                    ))
                  )}
                </div>
              </section>

              {isLocalGateway ? (
                <section className="min-h-0">
                  <div className={sectionHeadClass}>
                    <div className="space-y-1">
                      <h3 className={sectionTitleClass}>{t("models.available.addedTitle")}</h3>
                      <p className={sectionMetaClass}>{t("models.fallback.subtitle")}</p>
                    </div>
                  </div>

                  <div className={`${scrollListClass} mt-4 max-h-[540px]`}>
                    {loadingSelectedModels ? (
                      <div className={emptyStateClass}>
                        <p>{t("common.loading")}</p>
                      </div>
                    ) : selectedModelInfos.length === 0 ? (
                      <div className={emptyStateClass}>
                        <p>{t("models.fallback.empty")}</p>
                      </div>
                    ) : (
                      selectedModelInfos.map(({ item, model }, index) => (
                        <article
                          key={item.model_id}
                          className={`${selectableItemClass(false)} ${localGatewayAdminWritable ? "cursor-grab active:cursor-grabbing" : ""}`}
                          draggable={localGatewayAdminWritable}
                          onDragStart={() => setDraggedModelId(item.model_id)}
                          onDragEnd={() => setDraggedModelId(null)}
                          onDragOver={(event) => {
                            if (localGatewayAdminWritable) {
                              event.preventDefault();
                            }
                          }}
                          onDrop={() => {
                            if (localGatewayAdminWritable) {
                              moveSelectedModel(item.model_id);
                            }
                            setDraggedModelId(null);
                          }}
                        >
                          <div className="flex items-start gap-2.5">
                            <span className={iconBadgeClass}>
                              <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                                <path d="M12 3 4 7v10l8 4 8-4V7zm0 2.2L17.8 8 12 10.8 6.2 8zM6 9.6l5 2.5v6.2l-5-2.5zm7 8.7v-6.2l5-2.5v6.2z" />
                              </svg>
                            </span>
                            <div className="min-w-0 flex-1">
                              <p className={monoClass}>{item.model_id}</p>
                              <p className={`${metaClass} mt-1.5`}>
                                {index === 0
                                  ? t("models.fallback.primary")
                                  : t("models.fallback.secondary", { index: index + 1 })}
                              </p>
                              <p className={`${metaClass} mt-1.5`}>
                                {model?.owned_by ?? t("models.available.ownerUnknown")}
                              </p>
                            </div>
                            <button
                              type="button"
                              className={buttonClass("ghost")}
                              disabled={!localGatewayAdminWritable}
                              onClick={() => removeSelectedModel(item.model_id)}
                            >
                              {t("models.fallback.remove")}
                            </button>
                          </div>
                        </article>
                      ))
                    )}
                  </div>
                </section>
              ) : null}
            </div>
          </>
        )}
      </section>
    </main>
  );
}
