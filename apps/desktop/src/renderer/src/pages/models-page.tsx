import { useEffect, useMemo, useState } from "react";
import { ToastRegion, type ToastItem } from "../components/toast-region";
import { useI18n } from "../i18n/i18n-provider";
import { getProviderModels, getProviders } from "../services/api";
import type { Provider } from "../types/provider";
import type { ProviderModel } from "../types/provider-model";
import {
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
  statusPillClass
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
  const [models, setModels] = useState<ProviderModel[]>([]);
  const [loadingProviders, setLoadingProviders] = useState(true);
  const [loadingModels, setLoadingModels] = useState(false);
  const [search, setSearch] = useState("");
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const activeProvider = useMemo(
    () => providers.find((provider) => provider.status.is_active) ?? null,
    [providers]
  );

  const filteredModels = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    if (!keyword) {
      return models;
    }

    return models.filter((model) => {
      const owner = model.owned_by?.toLowerCase() ?? "";
      return model.id.toLowerCase().includes(keyword) || owner.includes(keyword);
    });
  }, [models, search]);

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
            </div>

            <div className="mt-4">
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

            <div className={`${scrollListClass} mt-4`}>
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
                    </div>
                  </article>
                ))
              )}
            </div>
          </>
        )}
      </section>
    </main>
  );
}
