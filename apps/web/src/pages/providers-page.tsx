import { useCallback, useEffect, useMemo, useState } from "react";
import { ToastRegion, type ToastItem } from "../components/toast-region";
import { useI18n } from "../i18n/i18n-provider";
import {
  activateProvider,
  createProvider,
  deleteProvider,
  getHealth,
  getLocalGatewayRuntime,
  getProviderModels,
  getProviders,
  runProviderHealthcheck,
  updateProvider
} from "../services/api";
import type { LocalGatewayRuntimeStatus } from "../types/local-gateway";
import type { ClaudeCodeModelMap, Provider } from "../types/provider";
import type { ProviderModel } from "../types/provider-model";
import {
  buttonClass,
  columnCardClass,
  emptyStateClass,
  fieldLabelClass,
  iconBadgeClass,
  iconButtonSmallClass,
  inputClass,
  labelClass,
  listClass,
  metaClass,
  modalBackdropClass,
  modalPanelClass,
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

interface ProvidersPageProps {
  apiBase?: string;
  refreshToken?: number;
  selectedProviderId: string | null;
  onSelectedProviderChange: (provider: Provider | null) => void;
}

export function ProvidersPage({
  apiBase,
  refreshToken = 0,
  selectedProviderId,
  onSelectedProviderChange
}: ProvidersPageProps) {
  const { t } = useI18n();
  const [health, setHealth] = useState("loading");
  const [providers, setProviders] = useState<Provider[]>([]);
  const [providerModels, setProviderModels] = useState<ProviderModel[]>([]);
  const [loadingModels, setLoadingModels] = useState(false);
  const [modelSearch, setModelSearch] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [name, setName] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [showApiKey, setShowApiKey] = useState(false);
  const [showSelectedProviderApiKey, setShowSelectedProviderApiKey] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [detailProviderID, setDetailProviderID] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [claudeCodeModelMap, setClaudeCodeModelMap] = useState<ClaudeCodeModelMap>({
    opus: "",
    sonnet: "",
    haiku: ""
  });
  const [localGatewayRuntime, setLocalGatewayRuntime] = useState<LocalGatewayRuntimeStatus | null>(null);
  const [savingClaudeMap, setSavingClaudeMap] = useState(false);
  const [draggedProviderModelId, setDraggedProviderModelId] = useState<string | null>(null);
  const [draggedClaudeSlot, setDraggedClaudeSlot] = useState<keyof ClaudeCodeModelMap | null>(null);
  const [dragOverClaudeSlot, setDragOverClaudeSlot] = useState<keyof ClaudeCodeModelMap | null>(null);

  const selectedProvider =
    providers.find((provider) => provider.id === selectedProviderId) ??
    providers.find((provider) => provider.status.is_active) ??
    providers[0] ??
    null;
  const detailProvider =
    providers.find((provider) => provider.id === detailProviderID) ?? null;

  const filteredModels = useMemo(() => {
    const keyword = modelSearch.trim().toLowerCase();
    if (!keyword) {
      return providerModels;
    }

    return providerModels.filter((model) => {
      const owner = model.owned_by?.toLowerCase() ?? "";
      return model.id.toLowerCase().includes(keyword) || owner.includes(keyword);
    });
  }, [modelSearch, providerModels]);

  const localGatewayReady =
    localGatewayRuntime?.running === true && localGatewayRuntime.healthy === true;

  const maskApiKey = useCallback((value: string) => {
    const trimmed = value.trim();
    if (trimmed.length <= 4) {
      return "****";
    }

    if (trimmed.length <= 12) {
      return `${trimmed.slice(0, trimmed.length - 4)}****`;
    }

    return `${trimmed.slice(0, 8)}••••${trimmed.slice(-4)}`;
  }, []);

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

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const [healthData, providersData] = await Promise.all([
          getHealth(apiBase),
          getProviders(apiBase)
        ]);
        const localGatewayRuntimeData = await getLocalGatewayRuntime(apiBase).catch(() => null);

        if (cancelled) {
          return;
        }

        setHealth(healthData.status);
        setProviders(providersData);
        setLocalGatewayRuntime(localGatewayRuntimeData?.runtime ?? null);
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
        setError(loadError instanceof Error ? loadError.message : t("common.unknownError"));
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [apiBase, onSelectedProviderChange, refreshToken, selectedProviderId, t]);

  useEffect(() => {
    let cancelled = false;

    async function loadProviderModels() {
      if (!selectedProvider) {
        setProviderModels([]);
        return;
      }

      setLoadingModels(true);
      try {
        const items = await getProviderModels(selectedProvider.id, apiBase);
        if (cancelled) {
          return;
        }
        setProviderModels(items);
      } catch (loadError) {
        if (!cancelled) {
          setProviderModels([]);
          setError(loadError instanceof Error ? loadError.message : t("common.unknownError"));
        }
      } finally {
        if (!cancelled) {
          setLoadingModels(false);
        }
      }
    }

    void loadProviderModels();

    return () => {
      cancelled = true;
    };
  }, [apiBase, selectedProvider, t]);

  useEffect(() => {
    setShowSelectedProviderApiKey(false);
  }, [selectedProvider?.id]);

  useEffect(() => {
    setClaudeCodeModelMap(
      selectedProvider?.claude_code_model_map ?? {
        opus: "",
        sonnet: "",
        haiku: ""
      }
    );
  }, [selectedProvider?.claude_code_model_map, selectedProvider?.id]);

  async function refreshProviders(preferredProviderId?: string) {
    const [providersData, localGatewayRuntimeData] = await Promise.all([
      getProviders(apiBase),
      getLocalGatewayRuntime(apiBase).catch(() => null)
    ]);
    setProviders(providersData);
    setLocalGatewayRuntime(localGatewayRuntimeData?.runtime ?? null);
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
      setError(t("providers.form.validation.required"));
      return;
    }

    const existingProvider = editingId
      ? providers.find((provider) => provider.id === editingId) ?? null
      : null;

    try {
      setSubmitting(true);
      const payload = {
        name: name.trim(),
        base_url: baseUrl.trim(),
        api_key: apiKey.trim(),
        auth_mode: existingProvider?.auth_mode,
        extra_headers: {},
        claude_code_model_map:
          existingProvider?.claude_code_model_map ?? {
            opus: "",
            sonnet: "",
            haiku: ""
          }
      };

      const provider = editingId
        ? await updateProvider(editingId, payload, apiBase)
        : await createProvider(payload, apiBase);

      resetForm();
      setFormOpen(false);
      setFeedback(editingId ? t("providers.feedback.updated") : t("providers.feedback.created"));
      await refreshProviders(provider.id);
    } catch (createError) {
      setError(createError instanceof Error ? createError.message : t("common.unknownError"));
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
      await syncClaudeCodeIntegrationIfConfigured();
      setFeedback(t("providers.feedback.activated", { name: provider.name }));
    } catch (activateError) {
      setError(activateError instanceof Error ? activateError.message : t("common.unknownError"));
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
      setError(deleteError instanceof Error ? deleteError.message : t("common.unknownError"));
    }
  }

  async function handleHealthcheck(id: string) {
    setError(null);
    setFeedback(null);

    try {
      const result = await runProviderHealthcheck(id, apiBase);
      setFeedback(
        t("providers.feedback.healthcheck", {
          status: result.status.toUpperCase(),
          code: result.status_code,
          latency: result.latency_ms
        })
      );
      await refreshProviders(id);
    } catch (healthError) {
      setError(healthError instanceof Error ? healthError.message : t("common.unknownError"));
    }
  }

  async function persistClaudeCodeModelMap(nextMap: ClaudeCodeModelMap) {
    if (!selectedProvider) {
      return;
    }

    setClaudeCodeModelMap(nextMap);
    setSavingClaudeMap(true);
    setError(null);

    try {
      await updateProvider(
        selectedProvider.id,
        {
          name: selectedProvider.name,
          base_url: selectedProvider.base_url,
          api_key: selectedProvider.api_key,
          auth_mode: selectedProvider.auth_mode,
          extra_headers: selectedProvider.extra_headers ?? {},
          claude_code_model_map: {
            opus: nextMap.opus.trim(),
            sonnet: nextMap.sonnet.trim(),
            haiku: nextMap.haiku.trim()
          }
        },
        apiBase
      );
      await refreshProviders(selectedProvider.id);
      if (selectedProvider.status.is_active) {
        await syncClaudeCodeIntegrationIfConfigured();
      }
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : t("common.unknownError"));
    } finally {
      setSavingClaudeMap(false);
    }
  }

  function assignClaudeSlot(slot: keyof ClaudeCodeModelMap, modelID: string) {
    const nextMap = {
      ...claudeCodeModelMap,
      [slot]: modelID
    };
    void persistClaudeCodeModelMap(nextMap);
  }

  function clearClaudeSlot(slot: keyof ClaudeCodeModelMap) {
    const nextMap = {
      ...claudeCodeModelMap,
      [slot]: ""
    };
    void persistClaudeCodeModelMap(nextMap);
  }

  function resetClaudeDragState() {
    setDraggedProviderModelId(null);
    setDraggedClaudeSlot(null);
    setDragOverClaudeSlot(null);
  }

  function startEditing(provider: Provider) {
    setEditingId(provider.id);
    setName(provider.name);
    setBaseUrl(provider.base_url);
    setApiKey(provider.api_key ?? "");
    setFeedback(null);
    setError(null);
    setShowApiKey(false);
    setFormOpen(true);
  }

  function resetForm() {
    setEditingId(null);
    setName("");
    setBaseUrl("");
    setApiKey("");
    setShowApiKey(false);
  }

  function startCreating() {
    resetForm();
    setFeedback(null);
    setError(null);
    setFormOpen(true);
  }

  async function syncClaudeCodeIntegrationIfConfigured() {
    return;
  }

  function openProviderDetail(provider: Provider) {
    setDetailProviderID(provider.id);
  }

  function isManagedLocalGatewayProvider(provider: Provider) {
    return provider.is_system_managed && provider.runtime_kind === "local-gateway";
  }

  function localGatewayActivationHint() {
    if (localGatewayReady) {
      return t("providers.localGateway.runtimeReady");
    }
    if (localGatewayRuntime?.last_error) {
      return t("providers.localGateway.runtimeUnavailableWithReason", {
        reason: localGatewayRuntime.last_error
      });
    }
    return t("providers.localGateway.runtimeUnavailable");
  }

  return (
    <main className={`${pageShellClass} h-full overflow-hidden`}>
      <ToastRegion items={toasts} onDismiss={dismissToast} />
      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div>
            <h1 className={sectionTitleClass}>{t("providers.title")}</h1>
            <p className={`${sectionMetaClass} mt-1`}>{t("providers.subtitle")}</p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              className={buttonClass("primary")}
              onClick={startCreating}
            >
              {t("providers.form.addTitle")}
            </button>
            <span className={statusPillClass(
              health === "ok" ? "success" : health === "offline" ? "danger" : "default"
            )}>
              {t("providers.coreHealth", { status: health })}
            </span>
            <span className={statusPillClass("default")}>
              <span className={monoClass}>
                {apiBase ?? "http://127.0.0.1:3456"}
              </span>
            </span>
            <span className={statusPillClass(localGatewayReady ? "success" : "default")}>
              {t("providers.localGateway.runtimeSummary", {
                status: localGatewayReady ? t("providers.localGateway.runtimeReady") : localGatewayActivationHint()
              })}
            </span>
          </div>
        </div>
      </section>

      <section className={`${splitLayoutClass} min-h-0 flex-1 overflow-hidden`}>
        <section className={`${columnCardClass} min-h-0 h-full overflow-hidden`}>
          <div className={sectionHeadClass}>
            <div className="space-y-1">
              <h2 className={sectionTitleClass}>{t("providers.list.title")}</h2>
              <p className={sectionMetaClass}>
                {t("providers.list.configured", { count: providers.length })}
              </p>
            </div>
          </div>

          {providers.length === 0 ? (
            <div className="mt-4 min-h-0 flex-1">
              <div className={emptyStateClass}>
                <p>{t("providers.list.empty")}</p>
              </div>
            </div>
          ) : (
            <div className="mt-4 flex min-h-0 flex-1 flex-col gap-2.5 overflow-y-auto pr-1">
              {providers.map((provider) => (
                <article
                  key={provider.id}
                  className={selectableItemClass(selectedProvider?.id === provider.id)}
                >
                  <div className="flex flex-col gap-2.5">
                    <button
                      type="button"
                      className="min-w-0 text-left"
                      onClick={() => {
                        onSelectedProviderChange(provider);
                        setFeedback(null);
                        setError(null);
                      }}
                    >
                      <div className="flex items-center gap-2.5">
                        <span className={iconBadgeClass}>
                          <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                            <path d="M4 7.5A2.5 2.5 0 0 1 6.5 5h11A2.5 2.5 0 0 1 20 7.5v9A2.5 2.5 0 0 1 17.5 19h-11A2.5 2.5 0 0 1 4 16.5zM6.5 7a.5.5 0 0 0-.5.5V10h12V7.5a.5.5 0 0 0-.5-.5zM18 12H6v4.5a.5.5 0 0 0 .5.5h11a.5.5 0 0 0 .5-.5z" />
                          </svg>
                        </span>
                        <div className="min-w-0 flex-1">
                          <strong className="block truncate text-[15px] font-semibold text-[color:var(--color-heading)]">
                            {provider.name}
                          </strong>
                          <p className={`${metaClass} mt-1 truncate`}>
                            {provider.base_url}
                          </p>
                          <p className={`${metaClass} mt-1 truncate`}>
                            {maskApiKey(provider.api_key ?? "")}
                          </p>
                        </div>
                      </div>
                    </button>

                    <div className="flex flex-wrap items-center gap-1.5">
                      {provider.status.is_active ? (
                        <span className={statusPillClass("success")}>
                          {t("providers.status.active")}
                        </span>
                      ) : (
                        <button
                          type="button"
                          className={buttonClass("primary")}
                          onClick={() => void handleActivateProvider(provider)}
                          disabled={isManagedLocalGatewayProvider(provider) && !localGatewayReady}
                          title={
                            isManagedLocalGatewayProvider(provider) && !localGatewayReady
                              ? localGatewayActivationHint()
                              : undefined
                          }
                        >
                          {t("providers.action.activate")}
                        </button>
                      )}
                      {isManagedLocalGatewayProvider(provider) ? (
                        <span className={statusPillClass(localGatewayReady ? "success" : "default")}>
                          {localGatewayReady
                            ? t("providers.localGateway.runtimeReady")
                            : t("providers.localGateway.runtimeUnavailable")}
                        </span>
                      ) : null}
                      <div className="relative">
                        <button
                          type="button"
                          className={`${iconButtonSmallClass} peer`}
                          aria-label={t("providers.action.view")}
                          onClick={() => {
                            openProviderDetail(provider);
                          }}
                        >
                          <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                            <path d="M12 5c5.5 0 9.5 4.6 10.7 6.2.4.5.4 1.1 0 1.6C21.5 14.4 17.5 19 12 19S2.5 14.4 1.3 12.8a1.3 1.3 0 0 1 0-1.6C2.5 9.6 6.5 5 12 5m0 2C8.2 7 5 10 3.4 12 5 14 8.2 17 12 17s7-3 8.6-5C19 10 15.8 7 12 7m0 2.5a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5" />
                          </svg>
                        </button>
                        <span className="pointer-events-none absolute left-1/2 top-full z-10 mt-1 hidden -translate-x-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-solid)] px-2 py-1 text-[11px] text-[color:var(--color-text)] shadow-[var(--shadow-soft)] peer-hover:block">
                          {t("providers.action.view")}
                        </span>
                      </div>
                      {provider.is_editable ? (
                        <div className="relative">
                          <button
                            type="button"
                            className={`${iconButtonSmallClass} peer`}
                            aria-label={t("common.edit")}
                            onClick={() => {
                              onSelectedProviderChange(provider);
                              startEditing(provider);
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
                      ) : null}
                      <div className="relative">
                        <button
                          type="button"
                          className={`${iconButtonSmallClass} peer`}
                          aria-label={t("providers.action.test")}
                          onClick={() => {
                            void handleHealthcheck(provider.id);
                          }}
                        >
                          <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                            <path d="M4 13h3l2-6 3 10 2-6h6v2h-4.6l-3 9-3-10-1.8 5H4z" />
                          </svg>
                        </button>
                        <span className="pointer-events-none absolute left-1/2 top-full z-10 mt-1 hidden -translate-x-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-solid)] px-2 py-1 text-[11px] text-[color:var(--color-text)] shadow-[var(--shadow-soft)] peer-hover:block">
                          {t("providers.action.test")}
                        </span>
                      </div>
                      {provider.is_deletable ? (
                        <div className="relative">
                          <button
                            type="button"
                            className={`${iconButtonSmallClass} peer`}
                            aria-label={t("common.delete")}
                            onClick={() => {
                              void handleDeleteProvider(provider.id);
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
                      ) : null}
                    </div>
                  </div>
                </article>
              ))}
            </div>
          )}
        </section>

        <section className={`${columnCardClass} min-h-0 h-full overflow-hidden`}>
          <div className={sectionHeadClass}>
            <div className="space-y-1">
              <h2 className={sectionTitleClass}>
                {selectedProvider
                  ? t("providers.detail.title", { name: selectedProvider.name })
                  : t("providers.detail.fallbackTitle")}
              </h2>
              <p className={sectionMetaClass}>
                {selectedProvider
                  ? t("providers.detail.inspectHint")
                  : t("providers.list.empty")}
              </p>
            </div>
          </div>

          {!selectedProvider ? (
            <div className="mt-4 min-h-0 flex-1">
              <div className={emptyStateClass}>
                <p>{t("providers.list.empty")}</p>
              </div>
            </div>
          ) : (
            <div className="mt-4 flex min-h-0 flex-1 flex-col overflow-hidden pr-1">
              <div className="grid min-h-0 flex-1 gap-4 overflow-y-auto pr-1 xl:grid-cols-[minmax(0,1.15fr)_minmax(280px,0.85fr)]">
                  <section className="flex min-h-0 flex-col overflow-hidden">
                    <div className={sectionHeadClass}>
                      <div className="space-y-1">
                        <h3 className={sectionTitleClass}>{t("models.available.title")}</h3>
                        <p className={sectionMetaClass}>
                          {loadingModels
                            ? t("common.loading")
                            : t("providers.detail.modelsCount", { count: filteredModels.length })}
                        </p>
                      </div>
                    </div>

                    <div className="mt-3">
                      <label className={labelClass}>
                        <span className={fieldLabelClass}>{t("logs.filter.search")}</span>
                        <input
                          className={inputClass}
                          value={modelSearch}
                          onChange={(event) => setModelSearch(event.target.value)}
                          placeholder={t("models.available.searchPlaceholder")}
                        />
                      </label>
                    </div>

                    {filteredModels.length === 0 ? (
                      <div className="mt-4 min-h-0 flex-1">
                        <div className={emptyStateClass}>
                          <p>{loadingModels ? t("common.loading") : t("providers.detail.modelsEmpty")}</p>
                        </div>
                      </div>
                    ) : (
                      <div className={`${scrollListClass} mt-4`}>
                        {filteredModels.map((model) => (
                          <article
                            key={model.id}
                            className={`${selectableItemClass(false)} cursor-grab active:cursor-grabbing`}
                            draggable
                            onDragStart={() => {
                              setDraggedProviderModelId(model.id);
                              setDraggedClaudeSlot(null);
                            }}
                            onDragEnd={() => {
                              resetClaudeDragState();
                            }}
                          >
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
                              <span className="pt-1 text-[11px] font-bold uppercase tracking-[0.3em] text-[color:var(--accent)]/75">
                                :::
                              </span>
                            </div>
                          </article>
                        ))}
                      </div>
                    )}
                  </section>

                  <section className="flex min-h-0 flex-col overflow-hidden">
                    <div className={sectionHeadClass}>
                      <div className="space-y-1">
                        <h3 className={sectionTitleClass}>{t("providers.detail.claudeSlotsTitle")}</h3>
                        <p className={sectionMetaClass}>{t("providers.detail.claudeSlotsMeta")}</p>
                      </div>
                    </div>
                    <p className={`${metaClass} mt-3`}>
                      {savingClaudeMap
                        ? t("providers.detail.claudeSlotsSaving")
                        : t("providers.detail.claudeSlotsAuto")}
                    </p>

                    <div className="mt-3 min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
                      {(
                        [
                          ["opus", t("providers.detail.claudeSlot.opus")],
                          ["sonnet", t("providers.detail.claudeSlot.sonnet")],
                          ["haiku", t("providers.detail.claudeSlot.haiku")]
                        ] as const
                      ).map(([slot, label]) => {
                        const assignedModelID = claudeCodeModelMap[slot];
                        const assignedModel = providerModels.find((model) => model.id === assignedModelID) ?? null;
                        const isDragOver = dragOverClaudeSlot === slot;

                        return (
                          <article
                            key={slot}
                            className={`rounded-[20px] border p-4 transition-[background,border-color,box-shadow,transform] duration-200 ${
                              isDragOver
                                ? "[border-color:var(--accent)] [background:color-mix(in_srgb,var(--panel-soft)_84%,var(--accent)_16%)] shadow-[0_18px_32px_rgba(15,23,42,0.12)]"
                                : "[border-color:var(--border-soft)] [background:var(--panel-soft)]"
                            }`}
                            onDragOver={(event) => {
                              event.preventDefault();
                              setDragOverClaudeSlot(slot);
                            }}
                            onDragLeave={() => {
                              setDragOverClaudeSlot((current) => (current === slot ? null : current));
                            }}
                            onDrop={() => {
                              if (draggedProviderModelId) {
                                assignClaudeSlot(slot, draggedProviderModelId);
                              } else if (draggedClaudeSlot) {
                                const nextModelID = claudeCodeModelMap[draggedClaudeSlot];
                                if (nextModelID) {
                                  const nextMap = {
                                    ...claudeCodeModelMap,
                                    [slot]: nextModelID
                                  };
                                  if (draggedClaudeSlot !== slot) {
                                    nextMap[draggedClaudeSlot] = "";
                                  }
                                  void persistClaudeCodeModelMap(nextMap);
                                }
                              }
                              resetClaudeDragState();
                            }}
                          >
                            <div className="flex items-start justify-between gap-3">
                              <div>
                                <p className={fieldLabelClass}>{label}</p>
                                <p className={`${metaClass} mt-1.5`}>
                                  {assignedModelID
                                    ? t("providers.detail.claudeSlot.ready")
                                    : t("providers.detail.claudeSlot.dropHint")}
                                </p>
                              </div>
                              {assignedModelID ? (
                                <button
                                  type="button"
                                  className={buttonClass("ghost")}
                                  onClick={() => clearClaudeSlot(slot)}
                                >
                                  {t("providers.detail.claudeSlot.clear")}
                                </button>
                              ) : null}
                            </div>

                            <div className="mt-4 rounded-[18px] border border-dashed p-4">
                              {assignedModelID ? (
                                <div
                                  className="flex cursor-grab items-start gap-3 active:cursor-grabbing"
                                  draggable
                                  onDragStart={() => {
                                    setDraggedProviderModelId(null);
                                    setDraggedClaudeSlot(slot);
                                  }}
                                  onDragEnd={() => {
                                    resetClaudeDragState();
                                  }}
                                >
                                  <span className={iconBadgeClass}>
                                    <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                                      <path d="M12 3 4 7v10l8 4 8-4V7zm0 2.2L17.8 8 12 10.8 6.2 8zM6 9.6l5 2.5v6.2l-5-2.5zm7 8.7v-6.2l5-2.5v6.2z" />
                                    </svg>
                                  </span>
                                  <div className="min-w-0">
                                    <p className={monoClass}>{assignedModelID}</p>
                                    <p className={`${metaClass} mt-1.5`}>
                                      {assignedModel?.owned_by ?? t("models.available.ownerUnknown")}
                                    </p>
                                  </div>
                                  <span className="pt-1 text-[11px] font-bold uppercase tracking-[0.3em] text-[color:var(--accent)]/75">
                                    :::
                                  </span>
                                </div>
                              ) : (
                                <div className="py-3 text-center">
                                  <p className={metaClass}>{t("providers.detail.claudeSlot.unset")}</p>
                                </div>
                              )}
                            </div>
                          </article>
                        );
                      })}
                    </div>
                  </section>
              </div>
            </div>
          )}
        </section>
      </section>

      {formOpen ? (
        <div className={modalBackdropClass} role="presentation">
          <section
            className={`${modalPanelClass} max-w-3xl`}
            role="dialog"
            aria-modal="true"
            aria-label={editingId ? t("providers.form.editTitle") : t("providers.form.addTitle")}
            onClick={(event) => event.stopPropagation()}
          >
            <div className={sectionHeadClass}>
              <div className="space-y-1">
                <h2 className={sectionTitleClass}>
                  {editingId ? t("providers.form.editTitle") : t("providers.form.addTitle")}
                </h2>
                <p className={sectionMetaClass}>
                  {editingId ? t("providers.form.editMeta") : t("providers.form.addMeta")}
                </p>
              </div>
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={() => setFormOpen(false)}
              >
                {t("common.close")}
              </button>
            </div>

            <form className="mt-4 grid gap-3" onSubmit={handleCreateProvider}>
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("providers.form.name")}</span>
                <input
                  required
                  className={inputClass}
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                />
              </label>
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("providers.form.baseUrl")}</span>
                <input
                  required
                  className={inputClass}
                  value={baseUrl}
                  onChange={(event) => setBaseUrl(event.target.value)}
                  placeholder="https://api.example.com/v1"
                />
              </label>
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("providers.form.apiKey")}</span>
                <div className="relative">
                  <input
                    required
                    className={`${inputClass} pr-11`}
                    value={apiKey}
                    onChange={(event) => setApiKey(event.target.value)}
                    placeholder="sk-example"
                    type={showApiKey ? "text" : "password"}
                  />
                  <button
                    type="button"
                    className="absolute inset-y-0 right-2 inline-flex items-center justify-center px-2 text-[color:var(--color-subtle)] transition hover:text-[color:var(--color-text)]"
                    onClick={() => setShowApiKey((current) => !current)}
                    aria-label={showApiKey ? t("providers.form.hideApiKey") : t("providers.form.showApiKey")}
                    title={showApiKey ? t("providers.form.hideApiKey") : t("providers.form.showApiKey")}
                  >
                    {showApiKey ? (
                      <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                        <path d="M2.7 1.3 1.3 2.7l3 3C2.9 6.9 1.9 8.2 1.3 9.2a1.3 1.3 0 0 0 0 1.6C2.5 12.4 6.5 17 12 17c2 0 3.8-.6 5.3-1.5l4 4 1.4-1.4zM9.9 11.3l2.8 2.8a2.5 2.5 0 0 1-2.8-2.8m4.1 1.3-3.6-3.6A2.5 2.5 0 0 1 14 12.6M12 7c3.8 0 7 3 8.6 5-.5.6-1.1 1.3-2 2l1.4 1.4c1.1-.8 2-1.8 2.7-2.8.4-.5.4-1.1 0-1.6C21.5 9.4 17.5 5 12 5c-1.4 0-2.7.3-3.9.8l1.7 1.7A9 9 0 0 1 12 7" />
                      </svg>
                    ) : (
                      <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                        <path d="M12 5c5.5 0 9.5 4.6 10.7 6.2.4.5.4 1.1 0 1.6C21.5 14.4 17.5 19 12 19S2.5 14.4 1.3 12.8a1.3 1.3 0 0 1 0-1.6C2.5 9.6 6.5 5 12 5m0 2C8.2 7 5 10 3.4 12 5 14 8.2 17 12 17s7-3 8.6-5C19 10 15.8 7 12 7m0 2.5a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5" />
                      </svg>
                    )}
                  </button>
                </div>
              </label>
              <div className="flex flex-wrap items-center gap-2">
                <button type="submit" className={buttonClass("primary")} disabled={submitting}>
                  {submitting
                    ? t("common.saving")
                    : editingId
                      ? t("providers.form.save")
                      : t("providers.form.create")}
                </button>
                <button
                  type="button"
                  className={buttonClass("secondary")}
                  onClick={() => {
                    setFormOpen(false);
                    resetForm();
                  }}
                  disabled={submitting}
                >
                  {t("common.cancel")}
                </button>
              </div>
            </form>
          </section>
        </div>
      ) : null}

      {detailProvider ? (
        <div className={modalBackdropClass} role="presentation" onClick={() => setDetailProviderID(null)}>
          <section
            className={`${modalPanelClass} max-w-3xl`}
            role="dialog"
            aria-modal="true"
            aria-label={t("providers.detail.title", { name: detailProvider.name })}
            onClick={(event) => event.stopPropagation()}
          >
            <div className={sectionHeadClass}>
              <div className="space-y-1">
                <h2 className={sectionTitleClass}>
                  {t("providers.detail.title", { name: detailProvider.name })}
                </h2>
                <p className={sectionMetaClass}>{t("providers.detail.inspectHint")}</p>
              </div>
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={() => setDetailProviderID(null)}
              >
                {t("common.close")}
              </button>
            </div>

            <div className="mt-4">
              <div className={listClass}>
                <div className={selectableItemClass(true)}>
                  <div className="flex flex-col gap-2.5">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className={statusPillClass(detailProvider.status.is_active ? "success" : "default")}>
                        {detailProvider.status.is_active
                          ? t("providers.status.active")
                          : t("providers.status.standby")}
                      </span>
                      <span className={statusPillClass("default")}>
                        {detailProvider.status.last_health_status}
                      </span>
                    </div>
                    <p className={metaClass}>
                      {t("providers.detail.baseUrl")}{" "}
                      <span className={monoClass}>{detailProvider.base_url}</span>
                    </p>
                    <p className={metaClass}>
                      {t("providers.detail.apiKey")}{" "}
                      <span className={monoClass}>
                        {showSelectedProviderApiKey
                          ? detailProvider.api_key
                          : maskApiKey(detailProvider.api_key ?? "")}
                      </span>
                      <button
                        type="button"
                        className={`${iconButtonSmallClass} ml-2 align-middle`}
                        aria-label={
                          showSelectedProviderApiKey
                            ? t("providers.form.hideApiKey")
                            : t("providers.form.showApiKey")
                        }
                        title={
                          showSelectedProviderApiKey
                            ? t("providers.form.hideApiKey")
                            : t("providers.form.showApiKey")
                        }
                        onClick={() => setShowSelectedProviderApiKey((current) => !current)}
                      >
                        {showSelectedProviderApiKey ? (
                          <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                            <path d="M2.7 1.3 1.3 2.7l3 3C2.9 6.9 1.9 8.2 1.3 9.2a1.3 1.3 0 0 0 0 1.6C2.5 12.4 6.5 17 12 17c2 0 3.8-.6 5.3-1.5l4 4 1.4-1.4zM9.9 11.3l2.8 2.8a2.5 2.5 0 0 1-2.8-2.8m4.1 1.3-3.6-3.6A2.5 2.5 0 0 1 14 12.6M12 7c3.8 0 7 3 8.6 5-.5.6-1.1 1.3-2 2l1.4 1.4c1.1-.8 2-1.8 2.7-2.8.4-.5.4-1.1 0-1.6C21.5 9.4 17.5 5 12 5c-1.4 0-2.7.3-3.9.8l1.7 1.7A9 9 0 0 1 12 7" />
                          </svg>
                        ) : (
                          <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                            <path d="M12 5c5.5 0 9.5 4.6 10.7 6.2.4.5.4 1.1 0 1.6C21.5 14.4 17.5 19 12 19S2.5 14.4 1.3 12.8a1.3 1.3 0 0 1 0-1.6C2.5 9.6 6.5 5 12 5m0 2C8.2 7 5 10 3.4 12 5 14 8.2 17 12 17s7-3 8.6-5C19 10 15.8 7 12 7m0 2.5a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5" />
                          </svg>
                        )}
                      </button>
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </div>
      ) : null}
    </main>
  );
}
