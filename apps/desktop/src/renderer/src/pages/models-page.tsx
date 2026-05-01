import { useCallback, useEffect, useMemo, useState } from "react";
import { ToastRegion, type ToastItem } from "../components/toast-region";
import { useI18n } from "../i18n/i18n-provider";
import {
  createLocalGatewaySource,
  deleteLocalGatewaySource,
  getLocalGatewayCapabilities,
  getLocalGatewayRuntime,
  getLocalGatewaySelectedModels,
  getLocalGatewaySources,
  syncLocalGateway,
  updateLocalGatewaySelectedModels,
  updateLocalGatewaySource
} from "../services/api";
import type {
  CreateLocalGatewayModelSourceInput,
  LocalGatewayCapabilities,
  LocalGatewayModelSource,
  LocalGatewayRuntimeResponse
} from "../types/local-gateway";
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
  infoCardClass,
  inputClass,
  labelClass,
  metaClass,
  metricNumberClass,
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
}

interface AvailableModel {
  id: string;
  ownedBy: string;
  sourceName: string;
}

const emptyRuntime: LocalGatewayRuntimeResponse = {
  runtime: {
    runtime_kind: "ai-mini-gateway",
    state: "stopped",
    managed: true,
    running: false,
    healthy: false,
    api_base: "",
    host: "127.0.0.1",
    port: 3457
  },
  last_sync: {
    applied_sources: 0,
    applied_selected_models: 0,
    last_synced_at: ""
  },
  last_sync_error: ""
};

const emptyCapabilities: LocalGatewayCapabilities = {
  supports_openai_compatible: false,
  supports_anthropic_compatible: false,
  supports_models_api: false,
  supports_stream: false,
  supports_admin_api: false,
  supports_model_source_admin: false,
  supports_selected_model_admin: false,
  supports_source_capabilities: false,
  supports_atomic_source_sync: false,
  supports_runtime_version: false,
  supports_explicit_source_health: false
};

export function ModelsPage({ apiBase }: ModelsPageProps) {
  const { t } = useI18n();
  const [runtime, setRuntime] = useState<LocalGatewayRuntimeResponse>(emptyRuntime);
  const [capabilities, setCapabilities] = useState<LocalGatewayCapabilities>(emptyCapabilities);
  const [sources, setSources] = useState<LocalGatewayModelSource[]>([]);
  const [selectedModels, setSelectedModels] = useState<SelectedModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [search, setSearch] = useState("");
  const [draggedModelId, setDraggedModelId] = useState<string | null>(null);
  const [leftPaneWidth, setLeftPaneWidth] = useState(48);
  const [formOpen, setFormOpen] = useState(false);
  const [editingSourceId, setEditingSourceId] = useState<string | null>(null);
  const [sourceName, setSourceName] = useState("");
  const [sourceBaseURL, setSourceBaseURL] = useState("");
  const [sourceAPIKey, setSourceAPIKey] = useState("");
  const [sourceProviderType, setSourceProviderType] = useState<
    "openai-compatible" | "anthropic-compatible"
  >("openai-compatible");
  const [sourceDefaultModelID, setSourceDefaultModelID] = useState("");
  const [sourceExposedModelIDs, setSourceExposedModelIDs] = useState("");
  const [sourceEnabled, setSourceEnabled] = useState(true);

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

  const loadAll = useCallback(async () => {
    const [runtimeData, capabilityData, sourceData, selectedData] = await Promise.all([
      getLocalGatewayRuntime(apiBase),
      getLocalGatewayCapabilities(apiBase),
      getLocalGatewaySources(apiBase),
      getLocalGatewaySelectedModels(apiBase)
    ]);

    setRuntime(runtimeData);
    setCapabilities(capabilityData);
    setSources(sourceData);
    setSelectedModels(selectedData);
  }, [apiBase]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      try {
        const [runtimeData, capabilityData, sourceData, selectedData] = await Promise.all([
          getLocalGatewayRuntime(apiBase),
          getLocalGatewayCapabilities(apiBase),
          getLocalGatewaySources(apiBase),
          getLocalGatewaySelectedModels(apiBase)
        ]);

        if (cancelled) {
          return;
        }

        setRuntime(runtimeData);
        setCapabilities(capabilityData);
        setSources(sourceData);
        setSelectedModels(selectedData);
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

    void load();

    return () => {
      cancelled = true;
    };
  }, [apiBase, t]);

  const availableModels = useMemo<AvailableModel[]>(() => {
    const seen = new Set<string>();
    const items: AvailableModel[] = [];

    for (const source of sources) {
      if (!source.enabled) {
        continue;
      }

      const modelIDs = [source.default_model_id, ...source.exposed_model_ids];
      for (const modelID of modelIDs) {
        const trimmed = modelID.trim();
        if (!trimmed || seen.has(trimmed)) {
          continue;
        }
        seen.add(trimmed);
        items.push({
          id: trimmed,
          ownedBy: source.provider_type,
          sourceName: source.name
        });
      }
    }

    return items;
  }, [sources]);

  const selectedModelIDs = new Set(selectedModels.map((item) => item.model_id));

  const filteredAvailableModels = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    return availableModels.filter((model) => {
      if (selectedModelIDs.has(model.id)) {
        return false;
      }

      if (!keyword) {
        return true;
      }

      return (
        model.id.toLowerCase().includes(keyword) ||
        model.ownedBy.toLowerCase().includes(keyword) ||
        model.sourceName.toLowerCase().includes(keyword)
      );
    });
  }, [availableModels, search, selectedModelIDs]);

  const selectedModelDetails = selectedModels.map((item) => ({
    ...item,
    details: availableModels.find((model) => model.id === item.model_id) ?? null
  }));

  const runtimeStateTone =
    runtime.runtime.healthy && runtime.runtime.running
      ? "success"
      : runtime.runtime.last_error
        ? "danger"
        : "default";

  function resetForm() {
    setEditingSourceId(null);
    setSourceName("");
    setSourceBaseURL("");
    setSourceAPIKey("");
    setSourceProviderType("openai-compatible");
    setSourceDefaultModelID("");
    setSourceExposedModelIDs("");
    setSourceEnabled(true);
  }

  function startEditingSource(source: LocalGatewayModelSource) {
    setEditingSourceId(source.id);
    setSourceName(source.name);
    setSourceBaseURL(source.base_url);
    setSourceAPIKey("");
    setSourceProviderType(source.provider_type);
    setSourceDefaultModelID(source.default_model_id);
    setSourceExposedModelIDs(source.exposed_model_ids.join(", "));
    setSourceEnabled(source.enabled);
    setFormOpen(true);
  }

  function parseExposedModels(value: string) {
    return value
      .split(/[\n,]/)
      .map((item) => item.trim())
      .filter(Boolean);
  }

  async function handleSaveSource(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setFeedback(null);

    if (!sourceName.trim() || !sourceBaseURL.trim() || !sourceDefaultModelID.trim()) {
      setError(t("models.form.validation.required"));
      return;
    }

    if (!editingSourceId && !sourceAPIKey.trim()) {
      setError(t("models.form.validation.required"));
      return;
    }

    const payload: CreateLocalGatewayModelSourceInput = {
      name: sourceName.trim(),
      base_url: sourceBaseURL.trim(),
      api_key: sourceAPIKey.trim(),
      provider_type: sourceProviderType,
      default_model_id: sourceDefaultModelID.trim(),
      exposed_model_ids: parseExposedModels(sourceExposedModelIDs),
      enabled: sourceEnabled,
      position: editingSourceId
        ? sources.find((item) => item.id === editingSourceId)?.position ?? sources.length
        : sources.length
    };

    try {
      setSaving(true);
      if (editingSourceId) {
        await updateLocalGatewaySource(editingSourceId, payload, apiBase);
      } else {
        await createLocalGatewaySource(payload, apiBase);
      }

      await loadAll();
      resetForm();
      setFormOpen(false);
      setFeedback(
        editingSourceId ? t("models.feedback.sourceUpdated") : t("models.feedback.sourceCreated")
      );
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : t("common.unknownError"));
    } finally {
      setSaving(false);
    }
  }

  async function handleDeleteSource(sourceID: string) {
    setError(null);
    setFeedback(null);

    try {
      await deleteLocalGatewaySource(sourceID, apiBase);
      if (editingSourceId === sourceID) {
        resetForm();
        setFormOpen(false);
      }
      await loadAll();
      setFeedback(t("models.feedback.sourceDeleted"));
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : t("common.unknownError"));
    }
  }

  async function handleToggleSourceEnabled(source: LocalGatewayModelSource) {
    setError(null);
    setFeedback(null);

    try {
      await updateLocalGatewaySource(
        source.id,
        {
          name: source.name,
          base_url: source.base_url,
          api_key: "",
          provider_type: source.provider_type,
          default_model_id: source.default_model_id,
          exposed_model_ids: source.exposed_model_ids,
          enabled: !source.enabled,
          position: source.position
        },
        apiBase
      );
      await loadAll();
      setFeedback(t("models.feedback.sourceUpdated"));
    } catch (toggleError) {
      setError(toggleError instanceof Error ? toggleError.message : t("common.unknownError"));
    }
  }

  async function persistSelectedModels(nextItems: SelectedModel[], successMessage: string) {
    setSaving(true);
    setError(null);
    setFeedback(null);

    try {
      const saved = await updateLocalGatewaySelectedModels(nextItems, apiBase);
      setSelectedModels(saved);
      await loadAll();
      setFeedback(successMessage);
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : t("common.unknownError"));
    } finally {
      setSaving(false);
    }
  }

  async function handleSync() {
    setError(null);
    setFeedback(null);

    try {
      setSyncing(true);
      const result = await syncLocalGateway(apiBase);
      await loadAll();
      setFeedback(
        t("models.feedback.synced", {
          sources: result.applied_sources,
          models: result.applied_selected_models
        })
      );
    } catch (syncError) {
      setError(syncError instanceof Error ? syncError.message : t("common.unknownError"));
    } finally {
      setSyncing(false);
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
          <span className={statusPillClass(runtimeStateTone)}>
            {runtime.runtime.state.toUpperCase()}
          </span>
          <span className={statusPillClass(syncing ? "warning" : "default")}>
            {syncing ? t("models.runtime.syncing") : t("models.section.state.selected", { count: selectedModels.length })}
          </span>
        </div>
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>{t("models.runtime.title")}</h2>
            <p className={sectionMetaClass}>{t("models.runtime.subtitle")}</p>
          </div>
          <div className={actionRowClass}>
            <button
              type="button"
              className={buttonClass("secondary")}
              onClick={() => void loadAll()}
              disabled={loading || syncing}
            >
              {t("common.refresh")}
            </button>
            <button
              type="button"
              className={buttonClass("primary")}
              onClick={() => void handleSync()}
              disabled={syncing || loading}
            >
              {syncing ? t("models.runtime.syncing") : t("models.runtime.sync")}
            </button>
          </div>
        </div>

        <div className={`${compactStatGridClass} mt-4`}>
          <div className={infoCardClass}>
            <p className={metaClass}>{t("models.runtime.status")}</p>
            <p className={metricNumberClass}>{runtime.runtime.state}</p>
            <p className="text-xs text-[color:var(--color-muted)]">
              {runtime.runtime.last_error || (runtime.runtime.healthy ? "healthy" : "waiting")}
            </p>
          </div>
          <div className={infoCardClass}>
            <p className={metaClass}>{t("models.runtime.apiBase")}</p>
            <p className={monoClass}>{runtime.runtime.api_base || "-"}</p>
            <p className="text-xs text-[color:var(--color-muted)]">
              pid {runtime.runtime.pid ?? "-"}
            </p>
          </div>
          <div className={infoCardClass}>
            <p className={metaClass}>{t("models.runtime.lastSync")}</p>
            <p className={monoClass}>{runtime.last_sync.last_synced_at || "-"}</p>
            <p className="text-xs text-[color:var(--color-muted)]">
              {runtime.last_sync_error || `${runtime.last_sync.applied_sources} / ${runtime.last_sync.applied_selected_models}`}
            </p>
          </div>
        </div>

        <div className="mt-4 flex flex-wrap gap-2">
          {([
            ["OpenAI", capabilities.supports_openai_compatible],
            ["Anthropic", capabilities.supports_anthropic_compatible],
            ["Models API", capabilities.supports_models_api],
            ["Stream", capabilities.supports_stream],
            ["Admin API", capabilities.supports_admin_api]
          ] as const).map(([label, enabled]) => (
            <span
              key={label}
              className={statusPillClass(enabled ? "success" : "default")}
            >
              {label}
            </span>
          ))}
        </div>
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>{t("models.sources.title")}</h2>
            <p className={sectionMetaClass}>{t("models.sources.subtitle")}</p>
          </div>
          <div className={actionRowClass}>
            <button
              type="button"
              className={buttonClass(formOpen ? "secondary" : "primary")}
              onClick={() => {
                if (formOpen) {
                  resetForm();
                  setFormOpen(false);
                  return;
                }
                resetForm();
                setFormOpen(true);
              }}
            >
              {formOpen ? t("common.cancel") : t("models.sources.add")}
            </button>
          </div>
        </div>

        {formOpen ? (
          <form className="mt-4 grid gap-3 md:grid-cols-2" onSubmit={handleSaveSource}>
            <label className={labelClass}>
              <span className={fieldLabelClass}>{t("models.form.name")}</span>
              <input
                className={inputClass}
                value={sourceName}
                onChange={(event) => setSourceName(event.target.value)}
              />
            </label>
            <label className={labelClass}>
              <span className={fieldLabelClass}>{t("models.form.providerType")}</span>
              <select
                className={inputClass}
                value={sourceProviderType}
                onChange={(event) =>
                  setSourceProviderType(
                    event.target.value as "openai-compatible" | "anthropic-compatible"
                  )
                }
              >
                <option value="openai-compatible">{t("models.form.providerTypeOpenAI")}</option>
                <option value="anthropic-compatible">
                  {t("models.form.providerTypeAnthropic")}
                </option>
              </select>
            </label>
            <label className={labelClass}>
              <span className={fieldLabelClass}>{t("models.form.baseUrl")}</span>
              <input
                className={inputClass}
                value={sourceBaseURL}
                onChange={(event) => setSourceBaseURL(event.target.value)}
              />
            </label>
            <label className={labelClass}>
              <span className={fieldLabelClass}>{t("models.form.apiKey")}</span>
              <input
                className={inputClass}
                value={sourceAPIKey}
                onChange={(event) => setSourceAPIKey(event.target.value)}
                placeholder={editingSourceId ? t("models.form.apiKeyHintUpdate") : ""}
              />
            </label>
            <label className={labelClass}>
              <span className={fieldLabelClass}>{t("models.form.defaultModel")}</span>
              <input
                className={inputClass}
                value={sourceDefaultModelID}
                onChange={(event) => setSourceDefaultModelID(event.target.value)}
              />
            </label>
            <label className={labelClass}>
              <span className={fieldLabelClass}>{t("models.form.enabled")}</span>
              <select
                className={inputClass}
                value={sourceEnabled ? "enabled" : "disabled"}
                onChange={(event) => setSourceEnabled(event.target.value === "enabled")}
              >
                <option value="enabled">{t("models.sources.enabled")}</option>
                <option value="disabled">{t("models.sources.disabled")}</option>
              </select>
            </label>
            <label className={`${labelClass} md:col-span-2`}>
              <span className={fieldLabelClass}>{t("models.form.exposedModels")}</span>
              <textarea
                className={`${inputClass} min-h-24`}
                value={sourceExposedModelIDs}
                onChange={(event) => setSourceExposedModelIDs(event.target.value)}
                placeholder={t("models.form.exposedHint")}
              />
            </label>
            <div className={`${actionRowClass} md:col-span-2`}>
              <button type="submit" className={buttonClass("primary")} disabled={saving}>
                {saving
                  ? t("common.saving")
                  : editingSourceId
                    ? t("models.sources.update")
                    : t("models.sources.create")}
              </button>
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={() => {
                  resetForm();
                  setFormOpen(false);
                }}
              >
                {t("common.cancel")}
              </button>
            </div>
          </form>
        ) : null}

        <div className={`${scrollListClass} mt-4 max-h-[360px]`}>
          {sources.length === 0 ? (
            <div className={emptyStateClass}>
              <p>{loading ? t("common.loading") : t("models.sources.empty")}</p>
            </div>
          ) : (
            sources.map((source) => (
              <article key={source.id} className={queueItemClass}>
                <div className="min-w-0 flex-1 space-y-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <strong className="text-[15px] font-semibold text-[color:var(--color-heading)]">
                      {source.name}
                    </strong>
                    <span className={statusPillClass(source.enabled ? "success" : "default")}>
                      {source.enabled ? t("models.sources.enabled") : t("models.sources.disabled")}
                    </span>
                    <span
                      className={statusPillClass(
                        source.last_sync_status === "synced"
                          ? "success"
                          : source.last_sync_status === "error"
                            ? "danger"
                            : "warning"
                      )}
                    >
                      {source.last_sync_status}
                    </span>
                  </div>
                  <p className={monoClass}>{source.base_url}</p>
                  <p className={metaClass}>
                    {source.provider_type} · default {source.default_model_id}
                  </p>
                  <p className={metaClass}>
                    {source.exposed_model_ids.length > 0
                      ? source.exposed_model_ids.join(", ")
                      : t("models.available.ownerUnknown")}
                  </p>
                  {source.last_sync_error ? (
                    <p className="text-sm text-[color:var(--danger-text)]">
                      {source.last_sync_error}
                    </p>
                  ) : null}
                </div>
                <div className="flex flex-col items-end gap-2">
                  <button
                    type="button"
                    className={buttonClass("secondary")}
                    onClick={() => void handleToggleSourceEnabled(source)}
                  >
                    {source.enabled ? t("models.sources.disable") : t("models.sources.enable")}
                  </button>
                  <button
                    type="button"
                    className={buttonClass("secondary")}
                    onClick={() => startEditingSource(source)}
                  >
                    {t("common.edit")}
                  </button>
                  <button
                    type="button"
                    className={buttonClass("danger")}
                    onClick={() => void handleDeleteSource(source.id)}
                  >
                    {t("common.delete")}
                  </button>
                </div>
              </article>
            ))
          )}
        </div>
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>{t("models.fallback.title")}</h2>
            <p className={sectionMetaClass}>{t("models.fallback.subtitle")}</p>
          </div>
        </div>

        <div className={`${compactStatGridClass} mt-4`}>
          <div className={infoCardClass}>
            <p className={metaClass}>{t("models.stats.providerModels")}</p>
            <p className={metricNumberClass}>{availableModels.length}</p>
          </div>
          <div className={infoCardClass}>
            <p className={metaClass}>{t("models.stats.availableToAdd")}</p>
            <p className={metricNumberClass}>{filteredAvailableModels.length}</p>
          </div>
          <div className={infoCardClass}>
            <p className={metaClass}>{t("models.stats.failoverSlots")}</p>
            <p className={metricNumberClass}>{selectedModels.length}</p>
          </div>
        </div>

        <div
          className="mt-3 flex min-h-0 flex-col gap-3 xl:grid xl:h-[min(62vh,720px)] xl:items-stretch"
          style={{
            gridTemplateColumns: `minmax(0, ${leftPaneWidth}fr) 12px minmax(0, ${100 - leftPaneWidth}fr)`
          }}
        >
          <section className={columnCardClass}>
            <div className={sectionHeadClass}>
              <div className="space-y-1">
                <h3 className={sectionTitleClass}>{t("models.available.title")}</h3>
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
            </div>

            <div className={`${scrollListClass} mt-3`}>
              {filteredAvailableModels.length === 0 ? (
                <div className={emptyStateClass}>
                  <p>{loading ? t("common.loading") : t("models.available.empty")}</p>
                </div>
              ) : (
                filteredAvailableModels.map((model) => (
                  <article key={model.id} className={queueItemClass}>
                    <button
                      type="button"
                      className={`${buttonClass("secondary")} absolute right-3 top-3 min-h-8 px-2.5 py-1 text-xs`}
                      onClick={() => addModel(model.id)}
                    >
                      {t("models.available.add", { id: model.id })}
                    </button>
                    <div className="flex items-start gap-2.5 pr-28">
                      <span className={`${iconBadgeClass} mt-0.5`}>
                        <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                          <path d="M12 3 4 7v10l8 4 8-4V7zm0 2.2L17.8 8 12 10.8 6.2 8zM6 9.6l5 2.5v6.2l-5-2.5zm7 8.7v-6.2l5-2.5v6.2z" />
                        </svg>
                      </span>
                      <div>
                        <p className={monoClass}>{model.id}</p>
                        <p className={`${metaClass} mt-1.5`}>
                          {model.ownedBy} · {model.sourceName}
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

            <div className={`${scrollListClass} mt-3`}>
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
                          {item.details ? ` · ${item.details.ownedBy}` : ""}
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
      </section>
    </main>
  );
}
