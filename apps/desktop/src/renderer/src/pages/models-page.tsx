import { useCallback, useEffect, useMemo, useState } from "react";
import { ToastRegion, type ToastItem } from "../components/toast-region";
import { useI18n } from "../i18n/i18n-provider";
import {
  createGatewayModel,
  deleteGatewayModel,
  getGatewayModels,
  getLocalGatewayClaudeMap,
  getLocalGatewaySelectedModels,
  getRuntimeConfig,
  getRuntimeHealth,
  getProviderModels,
  getProviders,
  getSelectedProviderModels,
  updateGatewayModel,
  updateLocalGatewaySelectedModels,
  updateSelectedProviderModels
} from "../services/api";
import type { Provider } from "../types/provider";
import type { ProviderModel } from "../types/provider-model";
import type { RuntimeConfig, RuntimeHealth } from "../types/runtime";
import type { SelectedModel } from "../types/selected-model";
import type { GatewayModel, GatewayModelInput } from "../types/gateway-model";
import { buildLocalGatewayProvider, LOCAL_GATEWAY_PROVIDER_ID } from "../utils/local-gateway-provider";
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

interface ModelsPageProps {
  apiBase?: string;
  selectedProvider: Provider | null;
  onSelectedProviderChange: (provider: Provider | null) => void;
}

function decorateProviders(
  providers: Provider[],
  runtimeConfig: RuntimeConfig,
  runtimeHealth: RuntimeHealth | null,
  claudeCodeModelMap: Provider["claude_code_model_map"]
) {
  const localGatewayProvider = buildLocalGatewayProvider(runtimeConfig, runtimeHealth, claudeCodeModelMap);
  if (!localGatewayProvider) {
    return providers;
  }

  return [localGatewayProvider, ...providers.map((provider) => ({
    ...provider,
    status: {
      ...provider.status,
      is_active: false
    }
  }))];
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
  const [gatewayModels, setGatewayModels] = useState<GatewayModel[]>([]);
  const [runtimeConfig, setRuntimeConfig] = useState<RuntimeConfig>({ mode: "legacy", base_url: "" });
  const [runtimeHealth, setRuntimeHealth] = useState<RuntimeHealth | null>(null);
  const [gatewayForm, setGatewayForm] = useState<GatewayModelInput>({
    name: "",
    model_id: "",
    base_url: "",
    api_key: "",
    provider_type: "",
    protocol: "openai",
    enabled: true
  });
  const [editingGatewayModelId, setEditingGatewayModelId] = useState<string | null>(null);
  const [gatewayFormOpen, setGatewayFormOpen] = useState(false);
  const activeProvider = providers.find((provider) => provider.status.is_active) ?? null;
  const [leftPaneWidth, setLeftPaneWidth] = useState(48);
  const providerTypeOptions = ["anthropic", "openai", "google", "custom"] as const;
  const protocolOptions = ["anthropic", "openai", "gemini", "custom"] as const;

  useEffect(() => {
    let cancelled = false;

    async function loadProviders() {
      try {
        const [items, runtimeConfigData, runtimeHealthData, localGatewayClaudeMap] = await Promise.all([
          getProviders(apiBase),
          getRuntimeConfig(apiBase),
          getRuntimeHealth(apiBase),
          getLocalGatewayClaudeMap(apiBase)
        ]);
        if (cancelled) {
          return;
        }

        const decoratedProviders = decorateProviders(
          items,
          runtimeConfigData,
          runtimeHealthData,
          localGatewayClaudeMap
        );
        setRuntimeConfig(runtimeConfigData);
        setRuntimeHealth(runtimeHealthData);
        setProviders(decoratedProviders);
        onSelectedProviderChange(
          decoratedProviders.find((provider) => provider.status.is_active) ??
            decoratedProviders.find((provider) => provider.id === selectedProvider?.id) ??
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
        setGatewayModels([]);
        setLoading(false);
        return;
      }

      setLoading(true);
      try {
        if (activeProvider.id === LOCAL_GATEWAY_PROVIDER_ID) {
          const [entries, selected] = await Promise.all([
            getGatewayModels(apiBase),
            getLocalGatewaySelectedModels(apiBase)
          ]);
          if (cancelled) {
            return;
          }

          const converted = entries.map<ProviderModel>((item) => ({
            id: item.model_id,
            object: "gateway_model_entry",
            owned_by: item.provider_type || item.protocol || item.name
          }));

          setGatewayModels(entries);
          setAvailableModels(converted);
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

        setAvailableModels(available);
        setSelectedModels(selected);
        setGatewayModels([]);
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
      if (!isLocalGatewayProvider && selectedModelIds.has(model.id)) {
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
    if (isLocalGatewayProvider) {
      void persistLocalGatewaySelectedModels(
        [...selectedModels, { model_id: modelID, position: selectedModels.length }],
        t("models.feedback.orderUpdated")
      );
      return;
    }
    void persistSelectedModels(
      [...selectedModels, { model_id: modelID, position: selectedModels.length }],
      t("models.feedback.orderUpdated")
    );
  }

  function removeModel(modelID: string) {
    if (isLocalGatewayProvider) {
      void persistLocalGatewaySelectedModels(
        selectedModels
          .filter((item) => item.model_id !== modelID)
          .map((item, index) => ({ ...item, position: index })),
        t("models.feedback.removed")
      );
      return;
    }
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
    if (isLocalGatewayProvider) {
      void persistLocalGatewaySelectedModels(
        current.map((item, index) => ({ ...item, position: index })),
        t("models.feedback.orderUpdated")
      );
      return;
    }
    void persistSelectedModels(
      current.map((item, index) => ({ ...item, position: index })),
      t("models.feedback.orderUpdated")
    );
  }

  async function reloadGatewayModels() {
    const [entries, selected] = await Promise.all([
      getGatewayModels(apiBase),
      getLocalGatewaySelectedModels(apiBase)
    ]);
    const converted = entries.map<ProviderModel>((item) => ({
      id: item.model_id,
      object: "gateway_model_entry",
      owned_by: item.provider_type || item.protocol || item.name
    }));

    setGatewayModels(entries);
    setAvailableModels(converted);
    setSelectedModels(selected);
  }

  function resetGatewayForm() {
    setGatewayForm({
      name: "",
      model_id: "",
      base_url: "",
      api_key: "",
      provider_type: "",
      protocol: "openai",
      enabled: true
    });
    setEditingGatewayModelId(null);
    setGatewayFormOpen(false);
  }

  function validateGatewayForm() {
    if (!gatewayForm.name.trim() || !gatewayForm.model_id.trim() || !gatewayForm.base_url.trim()) {
      return t("models.gateway.validation.required");
    }

    try {
      new URL(gatewayForm.base_url.trim());
    } catch {
      return t("models.gateway.validation.baseUrl");
    }

    const duplicate = gatewayModels.find(
      (item) =>
        item.model_id.toLowerCase() === gatewayForm.model_id.trim().toLowerCase() &&
        item.id !== editingGatewayModelId
    );
    if (duplicate) {
      return t("models.gateway.validation.duplicateModelId");
    }

    if (!providerTypeOptions.includes(gatewayForm.provider_type as (typeof providerTypeOptions)[number])) {
      return t("models.gateway.validation.providerType");
    }

    if (!protocolOptions.includes(gatewayForm.protocol as (typeof protocolOptions)[number])) {
      return t("models.gateway.validation.protocol");
    }

    return null;
  }

  async function handleSaveGatewayModel() {
    const validationError = validateGatewayForm();
    if (validationError) {
      setError(validationError);
      return;
    }

    setSaving(true);
    setFeedback(null);
    setError(null);

    try {
      if (editingGatewayModelId) {
        await updateGatewayModel(editingGatewayModelId, gatewayForm, apiBase);
      } else {
        await createGatewayModel(gatewayForm, apiBase);
      }
      await reloadGatewayModels();
      resetGatewayForm();
      setFeedback(
        editingGatewayModelId
          ? t("models.gateway.feedback.updated")
          : t("models.gateway.feedback.created")
      );
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : t("common.unknownError"));
    } finally {
      setSaving(false);
    }
  }

  function startEditGatewayModel(item: GatewayModel) {
    setEditingGatewayModelId(item.id);
    setGatewayFormOpen(true);
      setGatewayForm({
        name: item.name,
        model_id: item.model_id,
        base_url: item.base_url,
        api_key: item.api_key,
      provider_type: item.provider_type,
      protocol: item.protocol,
      enabled: item.enabled
    });
  }

  async function handleDeleteGatewayModel(id: string) {
    setSaving(true);
    setFeedback(null);
    setError(null);

    try {
      await deleteGatewayModel(id, apiBase);
      await reloadGatewayModels();
      if (editingGatewayModelId === id) {
        resetGatewayForm();
      }
      setFeedback(t("models.gateway.feedback.deleted"));
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : t("common.unknownError"));
    } finally {
      setSaving(false);
    }
  }

  async function handleToggleGatewayModel(item: GatewayModel) {
    setSaving(true);
    setFeedback(null);
    setError(null);

    try {
      await updateGatewayModel(
        item.id,
        {
          name: item.name,
          model_id: item.model_id,
          base_url: item.base_url,
          api_key: item.api_key,
          provider_type: item.provider_type,
          protocol: item.protocol,
          enabled: !item.enabled
        },
        apiBase
      );
      await reloadGatewayModels();
      setFeedback(
        !item.enabled
          ? t("models.gateway.feedback.enabled")
          : t("models.gateway.feedback.disabled")
      );
    } catch (toggleError) {
      setError(toggleError instanceof Error ? toggleError.message : t("common.unknownError"));
    } finally {
      setSaving(false);
    }
  }

  async function persistLocalGatewaySelectedModels(
    nextItems: SelectedModel[],
    successMessage: string
  ) {
    setSaving(true);
    setFeedback(null);
    setError(null);

    try {
      const saved = await updateLocalGatewaySelectedModels(nextItems, apiBase);
      setSelectedModels(saved);
      setFeedback(successMessage);
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : t("common.unknownError"));
    } finally {
      setSaving(false);
    }
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
          {isLocalGatewayProvider ? (
            <button
              type="button"
              className={buttonClass("primary")}
              onClick={() => {
                setGatewayFormOpen(true);
                if (editingGatewayModelId) {
                  setEditingGatewayModelId(null);
                }
              }}
            >
              {t("models.gateway.hero.add")}
            </button>
          ) : null}
          <span className={statusPillClass(activeProvider ? "success" : "default")}>
            {activeProvider
              ? t("models.section.title", { name: activeProvider.name })
              : t("models.section.fallbackTitle")}
          </span>
          <span
            className={statusPillClass(
              runtimeHealth?.status === "ok" ? "success" : runtimeConfig.mode === "external-portkey" ? "warning" : "default"
            )}
          >
            {runtimeConfig.mode === "external-portkey"
              ? t("models.runtime.external", { status: runtimeHealth?.status ?? "pending" })
              : t("models.runtime.legacy")}
          </span>
          <span className={statusPillClass(saving ? "warning" : "default")}>
            {saving
              ? t("common.saving")
              : loading
                ? t("common.loading")
                : isLocalGatewayProvider
                  ? t("models.section.state.configured", { count: availableModels.length })
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
                ? isLocalGatewayProvider
                  ? t("models.localGateway.subtitle")
                  : t("models.subtitle")
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
                  {isLocalGatewayProvider
                    ? t("models.stats.configuredModels")
                    : t("models.stats.providerModels")}
                </p>
              </div>
              <div className="rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3">
                <p className={metaClass}>
                  {isLocalGatewayProvider
                    ? t("models.stats.localGatewayStatus")
                    : t("models.stats.availableToAdd")}
                </p>
                <p className={metricNumberClass}>{availableCount}</p>
                <p className="text-xs text-[color:var(--color-muted)]">
                  {t("models.stats.filteredBySearch")}
                </p>
              </div>
              <div className="rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3">
                <p className={metaClass}>
                  {isLocalGatewayProvider
                    ? t("models.stats.activeChain")
                    : t("models.stats.failoverSlots")}
                </p>
                <p className={metricNumberClass}>
                  {isLocalGatewayProvider ? selectedModels.length : selectedModels.length}
                </p>
                <p className="text-xs text-[color:var(--color-muted)]">
                  {isLocalGatewayProvider
                    ? t("models.localGateway.mappingHint")
                    : t("models.stats.activeChain")}
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
                      {isLocalGatewayProvider
                        ? t("models.available.addedTitle")
                        : t("models.available.title")}
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
                    {isLocalGatewayProvider
                      ? t("models.localGateway.searchHint")
                      : t("models.searchHint")}
                  </p>
                </div>

                <div className={`${scrollListClass} content-start auto-rows-max mt-3`}>
                  {filteredAvailableModels.length === 0 ? (
                    <div className={emptyStateClass}>
                      <p>{loading ? t("common.loading") : t("models.available.empty")}</p>
                    </div>
                  ) : (
                    filteredAvailableModels.map((model) => (
                      <article key={model.id} className={queueItemClass}>
                        {isLocalGatewayProvider ? (
                          <div className="absolute right-2.5 top-2.5 flex items-center gap-2">
                            <div className="relative">
                              <button
                                type="button"
                                className={`${iconButtonClass} peer min-h-8 min-w-8 rounded-lg`}
                                aria-label={t("models.gateway.action.toggle")}
                                title={t("models.gateway.action.toggle")}
                                onClick={() => {
                                  const target = gatewayModels.find((entry) => entry.model_id === model.id);
                                  if (target) {
                                    void handleToggleGatewayModel(target);
                                  }
                                }}
                              >
                                {gatewayModels.find((entry) => entry.model_id === model.id)?.enabled ? (
                                  <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                                    <path d="M3 12a9 9 0 1 0 18 0A9 9 0 0 0 3 12m9-7a7 7 0 1 1 0 14 7 7 0 0 1 0-14m-1 3h2v8h-2zm0 9h2v2h-2z" />
                                  </svg>
                                ) : (
                                  <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                                    <path d="M12 5a7 7 0 0 1 7 7h2a9 9 0 1 0-9 9v-2a7 7 0 0 1 0-14m4 6h-3V8h-2v5h5z" />
                                  </svg>
                                )}
                              </button>
                              <span className="pointer-events-none absolute left-1/2 top-full z-10 mt-1 hidden -translate-x-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-solid)] px-2 py-1 text-[11px] text-[color:var(--color-text)] shadow-[var(--shadow-soft)] peer-hover:block">
                                {t("models.gateway.action.toggle")}
                              </span>
                            </div>
                            <div className="relative">
                              <button
                                type="button"
                                className={`${iconButtonClass} peer min-h-8 min-w-8 rounded-lg`}
                                aria-label={t("common.edit")}
                                title={t("common.edit")}
                                onClick={() => {
                                  const target = gatewayModels.find((entry) => entry.model_id === model.id);
                                  if (target) {
                                    startEditGatewayModel(target);
                                  }
                                }}
                              >
                                <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                                  <path d="M13.4 3.4a2 2 0 0 1 2.8 0l4.4 4.4a2 2 0 0 1 0 2.8l-2.1 2.1-7.2-7.2zM10.1 6.7 3 13.8V21h7.2l7.1-7.1zM6 18H5v-1l7.4-7.4 1 1z" />
                                </svg>
                              </button>
                              <span className="pointer-events-none absolute left-1/2 top-full z-10 mt-1 hidden -translate-x-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-solid)] px-2 py-1 text-[11px] text-[color:var(--color-text)] shadow-[var(--shadow-soft)] peer-hover:block">
                                {t("common.edit")}
                              </span>
                            </div>
                            <div className="relative">
                              <button
                                type="button"
                                className={`${iconButtonClass} peer min-h-8 min-w-8 rounded-lg`}
                                aria-label={t("common.delete")}
                                title={t("common.delete")}
                                onClick={() => {
                                  const target = gatewayModels.find((entry) => entry.model_id === model.id);
                                  if (target) {
                                    void handleDeleteGatewayModel(target.id);
                                  }
                                }}
                              >
                                <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                                  <path d="M9 3h6l1 2h4v2H4V5h4zm1 6h2v8h-2zm4 0h2v8h-2zM7 9h2v8H7zm1 12a2 2 0 0 1-2-2V8h12v11a2 2 0 0 1-2 2z" />
                                </svg>
                              </button>
                              <span className="pointer-events-none absolute left-1/2 top-full z-10 mt-1 hidden -translate-x-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-solid)] px-2 py-1 text-[11px] text-[color:var(--color-text)] shadow-[var(--shadow-soft)] peer-hover:block">
                                {t("common.delete")}
                              </span>
                            </div>
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
                        <div className="flex items-start gap-2.5 pr-10">
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
                            <p className={`${metaClass} mt-1`}>
                              {gatewayModels.find((entry) => entry.model_id === model.id)?.enabled
                                ? t("models.gateway.status.enabled")
                                : t("models.gateway.status.disabled")}
                            </p>
                            {isLocalGatewayProvider ? (
                              <p className={`${metaClass} mt-1`}>
                                {
                                  gatewayModels.find((entry) => entry.model_id === model.id)?.base_url
                                }
                              </p>
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
                    <h3 className={sectionTitleClass}>
                      {isLocalGatewayProvider
                        ? t("models.fallback.localGatewayTitle")
                        : t("models.fallback.title")}
                    </h3>
                    <p className={sectionMetaClass}>{selectedModels.length}</p>
                  </div>
                </div>
                <p className={`${metaClass} mt-3`}>
                  {isLocalGatewayProvider
                    ? t("models.fallback.localGatewaySubtitle")
                    : t("models.fallback.subtitle")}
                </p>

                <div className={`${scrollListClass} content-start auto-rows-max mt-3`}>
                  {selectedModelDetails.length === 0 ? (
                    <div className={emptyStateClass}>
                      <p>{t("models.fallback.empty")}</p>
                    </div>
                  ) : isLocalGatewayProvider ? (
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
                              {item.details?.owned_by ?? t("models.available.ownerUnknown")}
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

      {isLocalGatewayProvider && gatewayFormOpen ? (
        <div className={modalBackdropClass} role="presentation" onClick={resetGatewayForm}>
          <section
            className={`${modalPanelClass} max-w-3xl`}
            role="dialog"
            aria-modal="true"
            aria-label={t("models.gateway.form.title")}
            onClick={(event) => event.stopPropagation()}
          >
            <div className={sectionHeadClass}>
              <div className="space-y-1">
                <h2 className={sectionTitleClass}>
                  {editingGatewayModelId
                    ? t("models.gateway.form.updateTitle")
                    : t("models.gateway.form.title")}
                </h2>
                <p className={sectionMetaClass}>{t("models.gateway.form.subtitle")}</p>
              </div>
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={resetGatewayForm}
              >
                {t("common.close")}
              </button>
            </div>

            <div className="mt-4 grid gap-3 md:grid-cols-2">
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("models.gateway.form.name")}</span>
                <input
                  className={inputClass}
                  value={gatewayForm.name}
                  onChange={(event) =>
                    setGatewayForm((current) => ({ ...current, name: event.target.value }))
                  }
                  placeholder="Claude Sonnet"
                />
              </label>
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("models.gateway.form.modelId")}</span>
                <input
                  className={inputClass}
                  value={gatewayForm.model_id}
                  onChange={(event) =>
                    setGatewayForm((current) => ({ ...current, model_id: event.target.value }))
                  }
                  placeholder="claude-sonnet-4-20250514"
                />
              </label>
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("models.gateway.form.baseUrl")}</span>
                <input
                  className={inputClass}
                  value={gatewayForm.base_url}
                  onChange={(event) =>
                    setGatewayForm((current) => ({ ...current, base_url: event.target.value }))
                  }
                  placeholder="https://api.example.com"
                />
              </label>
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("models.gateway.form.apiKey")}</span>
                <input
                  className={inputClass}
                  value={gatewayForm.api_key}
                  onChange={(event) =>
                    setGatewayForm((current) => ({ ...current, api_key: event.target.value }))
                  }
                  placeholder="sk-example"
                />
              </label>
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("models.gateway.form.providerType")}</span>
                <select
                  className={inputClass}
                  value={gatewayForm.provider_type}
                  onChange={(event) =>
                    setGatewayForm((current) => ({
                      ...current,
                      provider_type: event.target.value
                    }))
                  }
                >
                  {providerTypeOptions.map((option) => (
                    <option key={option} value={option}>
                      {t(`models.gateway.form.providerTypeOption.${option}` as never)}
                    </option>
                  ))}
                </select>
              </label>
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("models.gateway.form.protocol")}</span>
                <select
                  className={inputClass}
                  value={gatewayForm.protocol}
                  onChange={(event) =>
                    setGatewayForm((current) => ({ ...current, protocol: event.target.value }))
                  }
                >
                  {protocolOptions.map((option) => (
                    <option key={option} value={option}>
                      {t(`models.gateway.form.protocolOption.${option}` as never)}
                    </option>
                  ))}
                </select>
              </label>
            </div>
            <label className={`${labelClass} mt-4`}>
              <span className={fieldLabelClass}>{t("models.gateway.form.enabled")}</span>
              <button
                type="button"
                className={buttonClass(gatewayForm.enabled ? "primary" : "secondary")}
                onClick={() =>
                  setGatewayForm((current) => ({ ...current, enabled: !current.enabled }))
                }
              >
                {gatewayForm.enabled
                  ? t("models.gateway.form.enabledOn")
                  : t("models.gateway.form.enabledOff")}
              </button>
            </label>
            <p className={`${metaClass} mt-3`}>{t("models.gateway.form.hint")}</p>
            <div className={`${actionRowClass} mt-4`}>
              <button
                type="button"
                className={buttonClass("primary")}
                onClick={() => void handleSaveGatewayModel()}
                disabled={saving}
              >
                {saving
                  ? t("common.saving")
                  : editingGatewayModelId
                    ? t("models.gateway.form.update")
                    : t("models.gateway.form.create")}
              </button>
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={resetGatewayForm}
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
