import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ToastRegion, type ToastItem } from "../components/toast-region";
import { useI18n } from "../i18n/i18n-provider";
import {
  checkLocalGatewaySourceHealth,
  createLocalGatewaySource,
  deleteLocalGatewaySource,
  getLocalGatewayCapabilities,
  getLocalGatewayRuntime,
  getLocalGatewaySourceCapabilities,
  getLocalGatewaySources,
  previewLocalGatewaySourceModels,
  syncLocalGateway,
  updateLocalGatewaySource
} from "../services/api";
import type {
  CreateLocalGatewayModelSourceInput,
  LocalGatewayCapabilities,
  LocalGatewayModelSource,
  LocalGatewaySourceCapability,
  LocalGatewaySourceHealthcheck,
  LocalGatewayRuntimeResponse
} from "../types/local-gateway";
import {
  actionRowClass,
  buttonClass,
  compactStatGridClass,
  emptyStateClass,
  eyebrowClass,
  fieldLabelClass,
  heroClass,
  heroContentClass,
  heroCopyClass,
  heroLabelStackClass,
  heroTitleClass,
  iconButtonSmallClass,
  infoCardClass,
  inputClass,
  labelClass,
  metaClass,
  metricNumberClass,
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
  statusPillClass
} from "../ui";

interface ModelsPageProps {
  apiBase?: string;
  refreshToken?: number;
}

function normalizeModelSource(source: LocalGatewayModelSource): LocalGatewayModelSource {
  return {
    ...source,
    exposed_model_ids: Array.isArray(source.exposed_model_ids) ? source.exposed_model_ids : []
  };
}

function collectModelIDs(source: LocalGatewayModelSource): string[] {
  return Array.from(
    new Set([source.default_model_id, ...source.exposed_model_ids].map((item) => item.trim()).filter(Boolean))
  );
}

const emptyRuntime: LocalGatewayRuntimeResponse = {
  runtime: {
    runtime_kind: "",
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

export function ModelsPage({ apiBase, refreshToken = 0 }: ModelsPageProps) {
  const { t } = useI18n();
  const [runtime, setRuntime] = useState<LocalGatewayRuntimeResponse>(emptyRuntime);
  const [capabilities, setCapabilities] = useState<LocalGatewayCapabilities>(emptyCapabilities);
  const [sources, setSources] = useState<LocalGatewayModelSource[]>([]);
  const [sourceCapabilities, setSourceCapabilities] = useState<LocalGatewaySourceCapability[]>([]);
  const [sourceHealthchecks, setSourceHealthchecks] = useState<Record<string, LocalGatewaySourceHealthcheck>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [runtimePanelOpen, setRuntimePanelOpen] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [editingSourceId, setEditingSourceId] = useState<string | null>(null);
  const [sourceName, setSourceName] = useState("");
  const [sourceBaseURL, setSourceBaseURL] = useState("");
  const [sourceAPIKey, setSourceAPIKey] = useState("");
  const [showSourceAPIKey, setShowSourceAPIKey] = useState(false);
  const [sourceProviderType, setSourceProviderType] = useState<
    "openai-compatible" | "anthropic-compatible"
  >("openai-compatible");
  const [sourceModelIDsInput, setSourceModelIDsInput] = useState("");
  const [sourceModelOptions, setSourceModelOptions] = useState<string[]>([]);
  const [sourceModelsMessage, setSourceModelsMessage] = useState<string | null>(null);
  const [sourceModelsMessageTone, setSourceModelsMessageTone] = useState<"default" | "success" | "error">("default");
  const [checkingSourceHealthID, setCheckingSourceHealthID] = useState<string | null>(null);
  const autoDetectRequestRef = useRef(0);

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
    const [runtimeData, capabilityData, sourceData] = await Promise.all([
      getLocalGatewayRuntime(apiBase),
      getLocalGatewayCapabilities(apiBase),
      getLocalGatewaySources(apiBase)
    ]);
    const sourceCapabilityData = capabilityData.supports_source_capabilities
      ? await getLocalGatewaySourceCapabilities(apiBase).catch(() => [])
      : [];

    setRuntime(runtimeData);
    setCapabilities(capabilityData);
    setSources(sourceData.map(normalizeModelSource));
    setSourceCapabilities(sourceCapabilityData);
  }, [apiBase]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      try {
        const [runtimeData, capabilityData, sourceData] = await Promise.all([
          getLocalGatewayRuntime(apiBase),
          getLocalGatewayCapabilities(apiBase),
          getLocalGatewaySources(apiBase)
        ]);
        const sourceCapabilityData = capabilityData.supports_source_capabilities
          ? await getLocalGatewaySourceCapabilities(apiBase).catch(() => [])
          : [];

        if (cancelled) {
          return;
        }

        setRuntime(runtimeData);
        setCapabilities(capabilityData);
        setSources(sourceData.map(normalizeModelSource));
        setSourceCapabilities(sourceCapabilityData);
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
  }, [apiBase, refreshToken, t]);

  const runtimeStateTone =
    runtime.runtime.healthy && runtime.runtime.running
      ? "success"
      : runtime.runtime.last_error
        ? "danger"
        : "default";

  const sourceCapabilityByID = useMemo(() => {
    return new Map(sourceCapabilities.map((item) => [item.source_id, item]));
  }, [sourceCapabilities]);

  function closeForm() {
    resetForm();
    setFormOpen(false);
  }

  function resetForm() {
    setEditingSourceId(null);
    setSourceName("");
    setSourceBaseURL("");
    setSourceAPIKey("");
    setShowSourceAPIKey(false);
    setSourceProviderType("openai-compatible");
    setSourceModelIDsInput("");
    setSourceModelOptions([]);
    setSourceModelsMessage(null);
    setSourceModelsMessageTone("default");
  }

  function startEditingSource(source: LocalGatewayModelSource) {
    const modelIDs = collectModelIDs(source);
    setEditingSourceId(source.id);
    setSourceName(source.name);
    setSourceBaseURL(source.base_url);
    setSourceAPIKey(source.api_key || "");
    setShowSourceAPIKey(false);
    setSourceProviderType(source.provider_type);
    setSourceModelIDsInput(modelIDs.join(", "));
    setSourceModelOptions(modelIDs);
    setSourceModelsMessage(null);
    setSourceModelsMessageTone("default");
    setFormOpen(true);
  }

  function startCreatingSource() {
    resetForm();
    setFormOpen(true);
  }

  function clearFetchedSourceModels() {
    setSourceModelOptions([]);
    setSourceModelsMessage(null);
    setSourceModelsMessageTone("default");
  }

  function handleSourceBaseURLChange(value: string) {
    setSourceBaseURL(value);
    clearFetchedSourceModels();
  }

  function handleSourceAPIKeyChange(value: string) {
    setSourceAPIKey(value);
  }

  function handleSourceProviderTypeChange(value: "openai-compatible" | "anthropic-compatible") {
    setSourceProviderType(value);
    clearFetchedSourceModels();
  }

  function applyFetchedSourceModels(modelIDs: string[]) {
    setSourceModelOptions(modelIDs);
    setSourceModelIDsInput(modelIDs.join(", "));
  }

  function parseModelIDs(value: string) {
    return value
      .split(/[\n,]/)
      .map((item) => item.trim())
      .filter(Boolean);
  }

  useEffect(() => {
    if (!formOpen) {
      return;
    }

    if (!sourceBaseURL.trim() || !sourceAPIKey.trim()) {
      setSourceModelsMessage(null);
      setSourceModelsMessageTone("default");
      return;
    }

    const timeoutID = window.setTimeout(() => {
      const requestID = autoDetectRequestRef.current + 1;
      autoDetectRequestRef.current = requestID;

      setSourceModelsMessage(t("models.form.autoDetecting"));
      setSourceModelsMessageTone("default");

      void previewLocalGatewaySourceModels(
        {
          base_url: sourceBaseURL.trim(),
          api_key: sourceAPIKey.trim(),
          provider_type: sourceProviderType
        },
        apiBase
      )
        .then((items) => {
          if (autoDetectRequestRef.current != requestID) {
            return;
          }

          const modelIDs = Array.from(new Set(items.map((item) => item.id.trim()).filter(Boolean)));
          if (modelIDs.length === 0) {
            setSourceModelOptions([]);
            setSourceModelsMessage(t("models.form.fetchEmpty"));
            setSourceModelsMessageTone("error");
            return;
          }

          applyFetchedSourceModels(modelIDs);
          setSourceModelsMessage(t("models.form.fetchSuccess", { count: modelIDs.length }));
          setSourceModelsMessageTone("success");
        })
        .catch((fetchError) => {
          if (autoDetectRequestRef.current != requestID) {
            return;
          }

          setSourceModelOptions([]);
          setSourceModelsMessage(
            t("models.form.autoDetectFailed", {
              message: fetchError instanceof Error ? fetchError.message : t("common.unknownError")
            })
          );
          setSourceModelsMessageTone("error");
        })
        .finally(() => undefined);
    }, 700);

    return () => {
      window.clearTimeout(timeoutID);
    };
  }, [apiBase, formOpen, sourceAPIKey, sourceBaseURL, sourceProviderType, t]);

  async function handleSaveSource(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setFeedback(null);

    const modelIDs = parseModelIDs(sourceModelIDsInput);

    if (!sourceName.trim() || !sourceBaseURL.trim() || modelIDs.length === 0) {
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
      default_model_id: modelIDs[0],
      exposed_model_ids: modelIDs.slice(1),
      enabled: editingSourceId
        ? sources.find((item) => item.id === editingSourceId)?.enabled ?? true
        : true,
      position: editingSourceId
        ? sources.find((item) => item.id === editingSourceId)?.position ?? sources.length
        : sources.length
    };

    let sourceSaved = false;

    try {
      setSaving(true);
      setSyncing(true);
      if (editingSourceId) {
        await updateLocalGatewaySource(editingSourceId, payload, apiBase);
      } else {
        await createLocalGatewaySource(payload, apiBase);
      }
      sourceSaved = true;

      await syncLocalGateway(apiBase);
      await loadAll();
      setSourceHealthchecks({});
      resetForm();
      setFormOpen(false);
      setFeedback(
        editingSourceId ? t("models.feedback.sourceUpdated") : t("models.feedback.sourceCreated")
      );
    } catch (saveError) {
      if (sourceSaved) {
        await loadAll().catch(() => undefined);
        setSourceHealthchecks({});
        closeForm();
      }
      setError(saveError instanceof Error ? saveError.message : t("common.unknownError"));
    } finally {
      setSaving(false);
      setSyncing(false);
    }
  }

  async function handleDeleteSource(sourceID: string) {
    setError(null);
    setFeedback(null);

    let sourceDeleted = false;

    try {
      setSyncing(true);
      await deleteLocalGatewaySource(sourceID, apiBase);
      sourceDeleted = true;
      if (editingSourceId === sourceID) {
        closeForm();
      }
      await syncLocalGateway(apiBase);
      await loadAll();
      setSourceHealthchecks({});
      setFeedback(t("models.feedback.sourceDeleted"));
    } catch (deleteError) {
      if (sourceDeleted) {
        await loadAll().catch(() => undefined);
        setSourceHealthchecks({});
      }
      setError(deleteError instanceof Error ? deleteError.message : t("common.unknownError"));
    } finally {
      setSyncing(false);
    }
  }

  async function handleToggleSourceEnabled(source: LocalGatewayModelSource) {
    setError(null);
    setFeedback(null);

    let sourceUpdated = false;

    try {
      setSyncing(true);
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
      sourceUpdated = true;
      await syncLocalGateway(apiBase);
      await loadAll();
      setSourceHealthchecks({});
      setFeedback(t("models.feedback.sourceUpdated"));
    } catch (toggleError) {
      if (sourceUpdated) {
        await loadAll().catch(() => undefined);
        setSourceHealthchecks({});
      }
      setError(toggleError instanceof Error ? toggleError.message : t("common.unknownError"));
    } finally {
      setSyncing(false);
    }
  }

  async function handleCheckSourceHealth(sourceID: string) {
    setError(null);
    setFeedback(null);

    const source = sources.find((item) => item.id === sourceID);
    const requiresSync = source?.last_sync_status !== "synced";

    try {
      setCheckingSourceHealthID(sourceID);
      if (requiresSync) {
        setSyncing(true);
        await syncLocalGateway(apiBase);
        await loadAll();
      }
      const result = await checkLocalGatewaySourceHealth(sourceID, apiBase);
      setSourceHealthchecks((current) => ({
        ...current,
        [sourceID]: result
      }));
      const message = t("models.feedback.healthcheck", {
        source: source?.name ?? sourceID,
        status: result.status.toUpperCase(),
        code: result.status_code,
        latency: result.latency_ms
      });
      if (result.status === "ok") {
        setFeedback(message);
      } else {
        setError(message);
      }
    } catch (healthError) {
      setError(healthError instanceof Error ? healthError.message : t("common.unknownError"));
    } finally {
      if (requiresSync) {
        setSyncing(false);
      }
      setCheckingSourceHealthID(null);
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
          <span className={statusPillClass(runtimeStateTone)}>
            {runtime.runtime.state.toUpperCase()}
          </span>
          <span className={statusPillClass(syncing ? "warning" : "default")}>
            {syncing ? t("models.runtime.syncing") : t("models.section.state.sources", { count: sources.length })}
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
              className={buttonClass("ghost")}
              onClick={() => setRuntimePanelOpen((current) => !current)}
            >
              {runtimePanelOpen ? t("models.runtime.collapse") : t("models.runtime.expand")}
            </button>
          </div>
        </div>

        <div className="mt-4 flex flex-wrap items-center gap-3 rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] px-4 py-3">
          <span className={statusPillClass(runtimeStateTone)}>
            {runtime.runtime.state.toUpperCase()}
          </span>
          <span className={monoClass}>{runtime.runtime.api_base || "-"}</span>
          <span className={metaClass}>
            pid {runtime.runtime.pid ?? "-"} · {runtime.runtime.runtime_kind || "-"}
          </span>
        </div>

        {runtimePanelOpen ? (
          <>
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
                  pid {runtime.runtime.pid ?? "-"} · {runtime.runtime.runtime_kind || "-"}
                </p>
              </div>
              <div className={infoCardClass}>
                <p className={metaClass}>{t("models.runtime.version")}</p>
                <p className={monoClass}>{runtime.runtime.version || "-"}</p>
                <p className="text-xs text-[color:var(--color-muted)]">
                  commit {runtime.runtime.commit || "-"}
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
          </>
        ) : null}
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
              className={buttonClass("primary")}
              onClick={startCreatingSource}
            >
              {t("models.sources.add")}
            </button>
          </div>
        </div>

        <div className={`${scrollListClass} mt-4 max-h-[360px]`}>
          {sources.length === 0 ? (
            <div className={emptyStateClass}>
              <p>{loading ? t("common.loading") : t("models.sources.empty")}</p>
            </div>
          ) : (
            sources.map((source) => {
              const capability = sourceCapabilityByID.get(source.id);
              const healthcheck = sourceHealthchecks[source.id];
              const modelIDs = collectModelIDs(source);

              return (
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
                    <p className={metaClass}>{source.provider_type}</p>
                    <p className={metaClass}>
                      API Key <span className={monoClass}>{source.api_key_masked || source.api_key}</span>
                    </p>
                    <div className="space-y-1">
                      <p className={metaClass}>{t("models.sources.modelIDs")}</p>
                      <div className="flex flex-wrap gap-2">
                        {modelIDs.map((modelID) => (
                          <span
                            key={modelID}
                            className="inline-flex rounded-full border [border-color:var(--border-soft)] [background:var(--panel-soft)] px-2.5 py-1 font-mono text-[11px] text-[color:var(--color-text)]"
                          >
                            {modelID}
                          </span>
                        ))}
                      </div>
                    </div>
                    {capability ? (
                      <div className="flex flex-wrap gap-2">
                        <span className={statusPillClass(
                          capability.models_api_status === "supported"
                            ? "success"
                            : capability.models_api_status === "unsupported"
                              ? "default"
                              : "warning"
                        )}>
                          models {capability.models_api_status}
                        </span>
                        <span className={statusPillClass(
                          capability.openai_chat_completions_status === "supported"
                            ? "success"
                            : capability.openai_chat_completions_status === "unsupported"
                              ? "default"
                              : "warning"
                        )}>
                          chat {capability.openai_chat_completions_status}
                        </span>
                        <span className={statusPillClass(
                          capability.stream_status === "supported"
                            ? "success"
                            : capability.stream_status === "unsupported"
                              ? "default"
                              : "warning"
                        )}>
                          stream {capability.stream_status}
                        </span>
                      </div>
                    ) : null}
                    {healthcheck ? (
                      <p className={metaClass}>
                        health {healthcheck.status} · {healthcheck.status_code} · {healthcheck.latency_ms}ms
                        {healthcheck.summary ? ` · ${healthcheck.summary}` : ""}
                      </p>
                    ) : null}
                    {source.last_sync_error ? (
                      <p className="text-sm text-[color:var(--danger-text)]">
                        {source.last_sync_error}
                      </p>
                    ) : null}
                  </div>
                  <div className="flex flex-col items-end gap-2">
                    <div className="relative flex items-center">
                      <button
                        type="button"
                        role="switch"
                        aria-checked={source.enabled}
                        aria-label={source.enabled ? t("models.sources.disable") : t("models.sources.enable")}
                        title={source.enabled ? t("models.sources.disable") : t("models.sources.enable")}
                        className={`peer inline-flex h-7 w-12 items-center rounded-full border px-1 transition ${
                          source.enabled
                            ? "[border-color:var(--success-border)] [background:var(--success-soft)]"
                            : "[border-color:var(--border-soft)] [background:var(--panel-soft)]"
                        }`}
                        onClick={() => void handleToggleSourceEnabled(source)}
                      >
                        <span
                          className={`h-5 w-5 rounded-full transition ${
                            source.enabled
                              ? "translate-x-5 bg-[color:var(--accent-strong)]"
                              : "translate-x-0 bg-[color:var(--color-subtle)]"
                          }`}
                        />
                      </button>
                      <span className="pointer-events-none absolute left-1/2 top-full z-10 mt-1 hidden -translate-x-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-solid)] px-2 py-1 text-[11px] text-[color:var(--color-text)] shadow-[var(--shadow-soft)] peer-hover:block">
                        {source.enabled ? t("models.sources.disable") : t("models.sources.enable")}
                      </span>
                    </div>

                    <div className="flex flex-wrap justify-end gap-2">
                      {capabilities.supports_explicit_source_health ? (
                        <div className="relative">
                          <button
                            type="button"
                            className={`${iconButtonSmallClass} peer`}
                            aria-label={t("common.check")}
                            onClick={() => void handleCheckSourceHealth(source.id)}
                            disabled={checkingSourceHealthID === source.id}
                            title={t("common.check")}
                          >
                            <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                              <path d="M4 13h3l2-6 3 10 2-6h6v2h-4.6l-3 9-3-10-1.8 5H4z" />
                            </svg>
                          </button>
                          <span className="pointer-events-none absolute left-1/2 top-full z-10 mt-1 hidden -translate-x-1/2 whitespace-nowrap rounded-md border [border-color:var(--border-soft)] [background:var(--panel-solid)] px-2 py-1 text-[11px] text-[color:var(--color-text)] shadow-[var(--shadow-soft)] peer-hover:block">
                            {t("common.check")}
                          </span>
                        </div>
                      ) : null}

                      <div className="relative">
                        <button
                          type="button"
                          className={`${iconButtonSmallClass} peer`}
                          aria-label={t("common.edit")}
                          onClick={() => startEditingSource(source)}
                          title={t("common.edit")}
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
                          className={`${iconButtonSmallClass} peer`}
                          aria-label={t("common.delete")}
                          onClick={() => void handleDeleteSource(source.id)}
                          title={t("common.delete")}
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
                  </div>
                </article>
              );
            })
          )}
        </div>
      </section>

      {formOpen ? (
        <div className={modalBackdropClass} role="presentation">
          <div
            className={`${modalPanelClass} max-w-3xl`}
            role="dialog"
            aria-modal="true"
            aria-labelledby="models-source-form-title"
            onClick={(event) => event.stopPropagation()}
          >
            <div className={sectionHeadClass}>
              <div className="space-y-1">
                <h2 id="models-source-form-title" className={sectionTitleClass}>
                  {editingSourceId ? t("models.form.editTitle") : t("models.form.addTitle")}
                </h2>
                <p className={sectionMetaClass}>{t("models.form.subtitle")}</p>
              </div>
              <button type="button" className={buttonClass("ghost")} onClick={closeForm}>
                {t("common.cancel")}
              </button>
            </div>

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
                    handleSourceProviderTypeChange(
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
                  onChange={(event) => handleSourceBaseURLChange(event.target.value)}
                />
              </label>
              <label className={labelClass}>
                <span className={fieldLabelClass}>{t("models.form.apiKey")}</span>
                <div className="relative">
                  <input
                    className={`${inputClass} pr-11`}
                    value={sourceAPIKey}
                    onChange={(event) => handleSourceAPIKeyChange(event.target.value)}
                    placeholder={editingSourceId ? t("models.form.apiKeyHintUpdate") : ""}
                    type={showSourceAPIKey ? "text" : "password"}
                  />
                  <button
                    type="button"
                    className="absolute inset-y-0 right-2 inline-flex items-center justify-center px-2 text-[color:var(--color-subtle)] transition hover:text-[color:var(--color-text)]"
                    onClick={() => setShowSourceAPIKey((current) => !current)}
                    aria-label={
                      showSourceAPIKey ? t("providers.form.hideApiKey") : t("providers.form.showApiKey")
                    }
                    title={
                      showSourceAPIKey ? t("providers.form.hideApiKey") : t("providers.form.showApiKey")
                    }
                  >
                    {showSourceAPIKey ? (
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
              <label className={`${labelClass} md:col-span-2`}>
                <span className={fieldLabelClass}>{t("models.form.models")}</span>
                <div className="grid gap-3 rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-soft)] p-3">
                  <div className="space-y-2">
                    <span
                      className={
                        sourceModelsMessageTone === "error"
                          ? "text-sm text-[color:var(--danger-text)]"
                          : sourceModelsMessageTone === "success"
                            ? "text-sm text-[color:var(--success-text)]"
                            : metaClass
                      }
                    >
                      {sourceModelsMessage
                        ? sourceModelsMessage
                        : t("models.form.fetchHint")}
                    </span>
                    {sourceModelsMessageTone === "error" ? (
                      <span className="text-sm text-[color:var(--danger-text)]">
                        {t("models.form.manualFallback")}
                      </span>
                    ) : null}
                  </div>

                  {sourceModelOptions.length > 0 ? (
                    <div className="space-y-2">
                      <span className={fieldLabelClass}>{t("models.form.detectedModels")}</span>
                      <div className="flex flex-wrap gap-2">
                        {sourceModelOptions.map((modelID) => (
                          <span
                            key={modelID}
                            className="inline-flex rounded-full border [border-color:var(--border-soft)] [background:var(--panel-solid)] px-2.5 py-1 font-mono text-[11px] text-[color:var(--color-text)]"
                          >
                            {modelID}
                          </span>
                        ))}
                      </div>
                    </div>
                  ) : null}

                  <div className="grid gap-3">
                    <label className={labelClass}>
                      <span className={fieldLabelClass}>{t("models.form.models")}</span>
                      <textarea
                        className={`${inputClass} min-h-24`}
                        value={sourceModelIDsInput}
                        onChange={(event) => setSourceModelIDsInput(event.target.value)}
                        placeholder={t("models.form.modelsPlaceholder")}
                      />
                    </label>
                  </div>
                </div>
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
                  onClick={closeForm}
                >
                  {t("common.cancel")}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </main>
  );
}
