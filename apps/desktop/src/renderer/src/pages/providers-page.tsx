import { useCallback, useEffect, useState } from "react";
import { ToastRegion, type ToastItem } from "../components/toast-region";
import { useI18n } from "../i18n/i18n-provider";
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
import {
  buttonClass,
  emptyStateClass,
  eyebrowClass,
  fieldLabelClass,
  heroClass,
  heroCopyClass,
  heroPillsClass,
  heroTitleClass,
  iconBadgeClass,
  inputClass,
  labelClass,
  listClass,
  metaClass,
  monoClass,
  pageShellClass,
  sectionCardClass,
  sectionHeadClass,
  sectionMetaClass,
  sectionTitleClass,
  selectableItemClass,
  statusPillClass
} from "../ui";

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
  const { t } = useI18n();
  const [health, setHealth] = useState("loading");
  const [providers, setProviders] = useState<Provider[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [name, setName] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [expandedProviderId, setExpandedProviderId] = useState<string | null>(null);

  const selectedProvider =
    providers.find((provider) => provider.id === selectedProviderId) ??
    providers.find((provider) => provider.status.is_active) ??
    providers[0] ??
    null;

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
        setExpandedProviderId(nextSelected?.id ?? null);
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
  }, [apiBase, onSelectedProviderChange, selectedProviderId, t]);

  async function refreshProviders(preferredProviderId?: string) {
    const providersData = await getProviders(apiBase);
    setProviders(providersData);
    const nextSelected =
      providersData.find((provider) => provider.id === preferredProviderId) ??
      providersData.find((provider) => provider.id === selectedProviderId) ??
      providersData.find((provider) => provider.status.is_active) ??
      providersData[0] ??
      null;
    setExpandedProviderId(nextSelected?.id ?? null);
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
      setFeedback(
        editingId ? t("providers.feedback.updated") : t("providers.feedback.created")
      );
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
      setFeedback(t("providers.feedback.healthcheck", {
        status: result.status.toUpperCase(),
        code: result.status_code,
        latency: result.latency_ms
      }));
      await refreshProviders(id);
    } catch (healthError) {
      setError(healthError instanceof Error ? healthError.message : t("common.unknownError"));
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
    <main className={pageShellClass}>
      <ToastRegion items={toasts} onDismiss={dismissToast} />
      <section className={heroClass}>
        <div className="space-y-4">
          <div>
            <p className={eyebrowClass}>Clash for AI</p>
            <h1 className={heroTitleClass}>{t("providers.title")}</h1>
          </div>
          <p className={heroCopyClass}>{t("providers.subtitle")}</p>
          <p className={metaClass}>
            {t("providers.connectedApiBase")}{" "}
            <span className={monoClass}>
              {desktopState?.apiBase ?? apiBase ?? "http://127.0.0.1:3456"}
            </span>
          </p>
        </div>
        <div className={heroPillsClass}>
          <div
            className={statusPillClass(
              health === "ok" ? "success" : health === "offline" ? "danger" : "default"
            )}
          >
            {t("providers.coreHealth", { status: health })}
          </div>
        </div>
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>
              {editingId ? t("providers.form.editTitle") : t("providers.form.addTitle")}
            </h2>
            <p className={sectionMetaClass}>
              {editingId ? t("providers.form.editMeta") : t("providers.form.addMeta")}
            </p>
          </div>
        </div>

        <form className="mt-6 grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)_minmax(0,1fr)_auto]" onSubmit={handleCreateProvider}>
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
            <input
              required
              className={inputClass}
              value={apiKey}
              onChange={(event) => setApiKey(event.target.value)}
              placeholder="sk-example"
              type="password"
            />
          </label>
          <div className="flex flex-wrap items-center gap-3 xl:self-end xl:pb-0.5">
            <button type="submit" className={buttonClass("primary")} disabled={submitting}>
              {submitting
                ? t("common.saving")
                : editingId
                  ? t("providers.form.save")
                  : t("providers.form.create")}
            </button>
            {editingId ? (
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={resetForm}
                disabled={submitting}
              >
                {t("common.cancel")}
              </button>
            ) : null}
          </div>
        </form>
      </section>

      <section className={sectionCardClass}>
        <div className={sectionHeadClass}>
          <div className="space-y-1">
            <h2 className={sectionTitleClass}>{t("providers.list.title")}</h2>
            <p className={sectionMetaClass}>
              {t("providers.list.configured", { count: providers.length })}
            </p>
          </div>
        </div>

        {providers.length === 0 ? (
          <div className="mt-5">
            <div className={emptyStateClass}>
              <p>{t("providers.list.empty")}</p>
            </div>
          </div>
        ) : (
          <div className={`${listClass} mt-5`}>
            {providers.map((provider) => (
              <article
                key={provider.id}
                className={selectableItemClass(selectedProvider?.id === provider.id)}
              >
                <div className="flex flex-col gap-3">
                  <div className="flex items-start justify-between gap-3">
                    <button
                      type="button"
                      className="min-w-0 flex-1 text-left"
                      onClick={() => {
                        onSelectedProviderChange(provider);
                        setExpandedProviderId((current) =>
                          current === provider.id ? null : provider.id
                        );
                      }}
                    >
                      <div className="flex items-center gap-3">
                        <span className={iconBadgeClass}>
                          <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                            <path d="M4 7.5A2.5 2.5 0 0 1 6.5 5h11A2.5 2.5 0 0 1 20 7.5v9A2.5 2.5 0 0 1 17.5 19h-11A2.5 2.5 0 0 1 4 16.5zM6.5 7a.5.5 0 0 0-.5.5V10h12V7.5a.5.5 0 0 0-.5-.5zM18 12H6v4.5a.5.5 0 0 0 .5.5h11a.5.5 0 0 0 .5-.5z" />
                          </svg>
                        </span>
                        <div className="min-w-0">
                          <strong className="block truncate text-base font-semibold text-[color:var(--color-heading)]">
                            {provider.name}
                          </strong>
                          <p className="mt-1 text-xs text-[color:var(--color-muted)]">
                            {provider.api_key_masked}
                          </p>
                        </div>
                      </div>
                    </button>
                    <button
                      type="button"
                      className="inline-flex h-9 w-9 items-center justify-center rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-solid)] text-[color:var(--color-text)] transition hover:[border-color:var(--border-strong)] hover:[background:var(--panel-soft)]"
                      onClick={() => {
                        onSelectedProviderChange(provider);
                        setExpandedProviderId((current) =>
                          current === provider.id ? null : provider.id
                        );
                      }}
                      aria-label={expandedProviderId === provider.id ? "Collapse" : "Expand"}
                    >
                      <svg
                        className={`h-4 w-4 fill-current transition-transform ${expandedProviderId === provider.id ? "rotate-180" : ""}`}
                        viewBox="0 0 24 24"
                        aria-hidden="true"
                      >
                        <path d="m12 15.5-6-6 1.4-1.4 4.6 4.6 4.6-4.6L18 9.5z" />
                      </svg>
                    </button>
                  </div>

                  <div className="flex flex-wrap items-center gap-2">
                    {provider.status.is_active ? (
                      <span className={statusPillClass("success")}>
                        {t("providers.status.active")}
                      </span>
                    ) : null}
                    {!provider.status.is_active ? (
                      <button
                        type="button"
                        className={buttonClass("primary")}
                        onClick={() => void handleActivateProvider(provider)}
                      >
                        {t("providers.action.activate")}
                      </button>
                    ) : null}
                    <button
                      type="button"
                      className={buttonClass("secondary")}
                      onClick={() => {
                        onSelectedProviderChange(provider);
                        startEditing(provider);
                      }}
                    >
                      {t("common.edit")}
                    </button>
                      <button
                        type="button"
                        className={buttonClass("secondary")}
                        onClick={() => {
                          void handleHealthcheck(provider.id);
                        }}
                      >
                        {t("providers.action.test")}
                      </button>
                    <button
                      type="button"
                      className={buttonClass("danger")}
                      onClick={() => {
                        void handleDeleteProvider(provider.id);
                      }}
                    >
                      {t("common.delete")}
                    </button>
                  </div>

                  {expandedProviderId === provider.id ? (
                    <div className="grid gap-3 rounded-3xl border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-4">
                      <p className={metaClass}>
                        {t("providers.detail.baseUrl")} <span className={monoClass}>{provider.base_url}</span>
                      </p>
                      <p className={metaClass}>
                        {t("providers.detail.health")}{" "}
                        <span className={monoClass}>{provider.status.last_health_status}</span>
                      </p>
                      <p className={metaClass}>
                        {t("providers.detail.apiKey")}{" "}
                        <span className={monoClass}>
                          {provider.api_key_masked}
                        </span>
                      </p>
                      <p className={metaClass}>
                        {t("providers.detail.capabilities")}{" "}
                        <span className={monoClass}>
                          {provider.capabilities.supports_models_api
                            ? `${t("providers.detail.capability.models")} `
                            : ""}
                          {provider.capabilities.supports_balance_api
                            ? `${t("providers.detail.capability.balance")} `
                            : ""}
                          {provider.capabilities.supports_stream
                            ? t("providers.detail.capability.stream")
                            : ""}
                        </span>
                      </p>
                    </div>
                  ) : null}
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}
