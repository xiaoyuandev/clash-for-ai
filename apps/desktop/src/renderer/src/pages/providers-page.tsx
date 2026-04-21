import { useEffect, useState } from "react";
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
  actionRowClass,
  buttonClass,
  dangerNoticeClass,
  emptyStateClass,
  eyebrowClass,
  fieldLabelClass,
  gridStatsClass,
  heroClass,
  heroCopyClass,
  heroPillsClass,
  heroTitleClass,
  hintClass,
  iconBadgeClass,
  infoCardClass,
  inputClass,
  labelClass,
  listClass,
  metaClass,
  metricValueClass,
  monoClass,
  nestedCardClass,
  pageShellClass,
  sectionCardClass,
  sectionHeadClass,
  sectionMetaClass,
  sectionTitleClass,
  selectableItemClass,
  statusDotClass,
  splitLayoutClass,
  successNoticeClass,
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
          <div className={statusPillClass(desktopState?.ok ? "success" : "danger")}>
            {t("providers.desktopRuntime", {
              runtime: desktopState?.runtime ?? t("settings.value.browser")
            })}
          </div>
        </div>
      </section>

      {error ? (
        <p className={dangerNoticeClass}>
          {error}
        </p>
      ) : null}
      {feedback ? (
        <p className={successNoticeClass}>
          {feedback}
        </p>
      ) : null}

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

        <form className="mt-6 grid gap-4 xl:grid-cols-4" onSubmit={handleCreateProvider}>
          <label className={labelClass}>
            <span className={fieldLabelClass}>{t("providers.form.name")}</span>
            <input
              required
              className={inputClass}
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
            <span className={hintClass}>{t("providers.form.nameHint")}</span>
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
            <span className={hintClass}>{t("providers.form.baseUrlHint")}</span>
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
            <span className={hintClass}>{t("providers.form.apiKeyHint")}</span>
          </label>
          <div className="flex flex-wrap items-end gap-3 xl:justify-end">
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

      <section className={splitLayoutClass}>
        <aside className={sectionCardClass}>
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
                <button
                  key={provider.id}
                  type="button"
                  className={selectableItemClass(selectedProvider?.id === provider.id)}
                  onClick={() => {
                    onSelectedProviderChange(provider);
                  }}
                >
                  <div className="mb-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <div className="flex items-center gap-3">
                      <span className={iconBadgeClass}>
                        <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                          <path d="M4 7.5A2.5 2.5 0 0 1 6.5 5h11A2.5 2.5 0 0 1 20 7.5v9A2.5 2.5 0 0 1 17.5 19h-11A2.5 2.5 0 0 1 4 16.5zM6.5 7a.5.5 0 0 0-.5.5V10h12V7.5a.5.5 0 0 0-.5-.5zM18 12H6v4.5a.5.5 0 0 0 .5.5h11a.5.5 0 0 0 .5-.5z" />
                        </svg>
                      </span>
                      <div>
                        <strong className="text-base font-semibold text-[color:var(--color-heading)]">
                          {provider.name}
                        </strong>
                        <p className="text-xs text-[color:var(--color-muted)]">
                          {provider.status.last_health_status}
                        </p>
                      </div>
                    </div>
                    <span
                      className={statusPillClass(
                        provider.status.is_active ? "success" : "default"
                      )}
                    >
                      {provider.status.is_active
                        ? t("providers.status.active")
                        : t("providers.status.standby")}
                    </span>
                  </div>
                  <p className={monoClass}>{provider.base_url}</p>
                </button>
              ))}
            </div>
          )}
        </aside>

        <section className="grid gap-4">
          <section className={`${sectionCardClass} min-h-[460px]`}>
            <div className={sectionHeadClass}>
              <div className="space-y-1">
                <h2 className={sectionTitleClass}>
                  {selectedProvider
                    ? t("providers.detail.title", { name: selectedProvider.name })
                    : t("providers.detail.fallbackTitle")}
                </h2>
                <p className={sectionMetaClass}>
                  {selectedProvider?.status.is_active
                    ? t("providers.status.active")
                    : t("providers.status.standby")}
                </p>
              </div>
            </div>

            {!selectedProvider ? (
              <div className="mt-6">
                <div className={emptyStateClass}>
                  <p>{t("providers.detail.inspectHint")}</p>
                </div>
              </div>
            ) : (
              <div className="mt-6 space-y-5">
                <div className={gridStatsClass}>
                  <div className={infoCardClass}>
                    <div className="flex items-center gap-3">
                      <span className={iconBadgeClass}>
                        <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                          <path d="M12 4a8 8 0 1 0 8 8h-2a6 6 0 1 1-1.8-4.2L13 11h7V4l-2.4 2.4A7.9 7.9 0 0 0 12 4" />
                        </svg>
                      </span>
                      <p className={fieldLabelClass}>{t("providers.detail.baseUrl")}</p>
                    </div>
                    <p className={`${monoClass} mt-3`}>{selectedProvider.base_url}</p>
                  </div>
                  <div className={infoCardClass}>
                    <div className="flex items-center gap-3">
                      <span className={iconBadgeClass}>
                        <span
                          className={statusDotClass(
                            selectedProvider.status.last_health_status === "ok"
                              ? "success"
                              : selectedProvider.status.last_health_status === "offline"
                                ? "danger"
                                : "warning"
                          )}
                        />
                      </span>
                      <p className={fieldLabelClass}>{t("providers.detail.health")}</p>
                    </div>
                    <p className={metricValueClass}>
                      {selectedProvider.status.last_health_status}
                    </p>
                  </div>
                  <div className={infoCardClass}>
                    <div className="flex items-center gap-3">
                      <span className={iconBadgeClass}>
                        <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                          <path d="M7 10V8a5 5 0 0 1 10 0v2h1a2 2 0 0 1 2 2v7H4v-7a2 2 0 0 1 2-2zm2 0h6V8a3 3 0 1 0-6 0z" />
                        </svg>
                      </span>
                      <p className={fieldLabelClass}>{t("providers.detail.apiKey")}</p>
                    </div>
                    <p className={`${monoClass} mt-3`}>{selectedProvider.api_key_masked}</p>
                  </div>
                  <div className={infoCardClass}>
                    <div className="flex items-center gap-3">
                      <span className={iconBadgeClass}>
                        <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                          <path d="M5 5h14v4H5zm0 5h6v9H5zm7 0h7v9h-7z" />
                        </svg>
                      </span>
                      <p className={fieldLabelClass}>{t("providers.detail.capabilities")}</p>
                    </div>
                    <p className={`${hintClass} mt-3`}>
                      {selectedProvider.capabilities.supports_models_api
                        ? `${t("providers.detail.capability.models")} `
                        : ""}
                      {selectedProvider.capabilities.supports_balance_api
                        ? `${t("providers.detail.capability.balance")} `
                        : ""}
                      {selectedProvider.capabilities.supports_stream
                        ? t("providers.detail.capability.stream")
                        : ""}
                    </p>
                  </div>
                </div>

                <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                  <div className={nestedCardClass}>
                    <p className="flex items-center gap-2">
                      <span
                        className={statusDotClass(
                          selectedProvider.status.is_active ? "success" : "default"
                        )}
                      />
                      <span className={fieldLabelClass}>Provider State</span>
                    </p>
                    <p className={`${metricValueClass} mt-2`}>
                      {selectedProvider.status.is_active
                        ? t("providers.status.active")
                        : t("providers.status.standby")}
                    </p>
                  </div>
                  <div className={actionRowClass}>
                    {!selectedProvider.status.is_active ? (
                      <button
                        type="button"
                        className={buttonClass("primary")}
                        onClick={() => void handleActivateProvider(selectedProvider)}
                      >
                        {t("providers.action.activate")}
                      </button>
                    ) : null}
                    <button
                      type="button"
                      className={buttonClass("secondary")}
                      onClick={() => {
                        startEditing(selectedProvider);
                      }}
                    >
                      {t("common.edit")}
                    </button>
                    <button
                      type="button"
                      className={buttonClass("secondary")}
                      onClick={() => {
                        void handleHealthcheck(selectedProvider.id);
                      }}
                    >
                      {t("common.check")}
                    </button>
                    <button
                      type="button"
                      className={buttonClass("danger")}
                      onClick={() => {
                        void handleDeleteProvider(selectedProvider.id);
                      }}
                    >
                      {t("common.delete")}
                    </button>
                  </div>
                </div>
              </div>
            )}
          </section>
        </section>
      </section>
    </main>
  );
}
