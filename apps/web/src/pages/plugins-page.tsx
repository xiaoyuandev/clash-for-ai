import { useCallback, useEffect, useMemo, useState } from "react";
import { useI18n } from "../i18n/i18n-provider";
import {
  disableExtension,
  enableExtension,
  executeExtensionCommand,
  executeExtensionToolIntegrationAction,
  getExtensionAuditLogs,
  getExtensionCommands,
  getExtensions,
  getExtensionSettings,
  getExtensionToolIntegrations,
  rescanExtensions,
  updateExtensionSettings
} from "../services/extensions";
import type {
  ExtensionAuditLog,
  ExtensionCommand,
  ExtensionPlugin,
  ExtensionSettings,
  ExtensionSettingsProperty,
  ExtensionToolIntegration,
  ExtensionToolIntegrationAction,
  PluginStatus
} from "../types/extension";
import {
  actionRowClass,
  buttonClass,
  dangerNoticeClass,
  emptyStateClass,
  eyebrowClass,
  fieldLabelClass,
  heroClass,
  heroCopyClass,
  heroTitleClass,
  hintClass,
  inputClass,
  metaClass,
  monoClass,
  pageShellClass,
  sectionCardClass,
  sectionHeadClass,
  sectionMetaClass,
  sectionTitleClass,
  selectableItemClass,
  statusDotClass,
  statusPillClass,
  successNoticeClass
} from "../ui";

interface PluginsPageProps {
  apiBase?: string;
}

function statusTone(status: PluginStatus, enabled: boolean) {
  if (status === "invalid" || status === "incompatible") {
    return "danger" as const;
  }
  if (enabled || status === "enabled") {
    return "success" as const;
  }
  if (status === "disabled") {
    return "warning" as const;
  }
  return "default" as const;
}

function formatContributionSummary(item: ExtensionPlugin) {
  return Object.entries(item.contributes)
    .filter(([, count]) => count > 0)
    .sort(([left], [right]) => left.localeCompare(right));
}

function formatValue(value: unknown) {
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  if (Array.isArray(value)) {
    return value.join("\n");
  }
  return "";
}

function enumOptionValue(value: unknown) {
  return JSON.stringify(value);
}

function parseEnumOptionValue(value: string) {
  try {
    return JSON.parse(value) as unknown;
  } catch {
    return value;
  }
}

function serializeSettingsValues(values: Record<string, unknown>) {
  return Object.fromEntries(Object.entries(values).filter(([, value]) => value !== undefined));
}

function formatAuditTime(value: string) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function settingInputType(property: ExtensionSettingsProperty) {
  return property.type === "integer" || property.type === "number" ? "number" : "text";
}

export function PluginsPage({ apiBase }: PluginsPageProps) {
  const { t } = useI18n();
  const [items, setItems] = useState<ExtensionPlugin[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [commands, setCommands] = useState<ExtensionCommand[]>([]);
  const [toolIntegrations, setToolIntegrations] = useState<ExtensionToolIntegration[]>([]);
  const [settings, setSettings] = useState<ExtensionSettings | null>(null);
  const [settingsValues, setSettingsValues] = useState<Record<string, unknown>>({});
  const [auditLogs, setAuditLogs] = useState<ExtensionAuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [busyAction, setBusyAction] = useState<"rescan" | "enable" | "disable" | null>(null);
  const [settingsBusy, setSettingsBusy] = useState(false);
  const [commandBusy, setCommandBusy] = useState<string | null>(null);
  const [toolActionBusy, setToolActionBusy] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const selected = useMemo(
    () => items.find((item) => item.id === selectedId) ?? items[0] ?? null,
    [items, selectedId]
  );

  const stats = useMemo(
    () => ({
      total: items.length,
      enabled: items.filter((item) => item.enabled).length,
      invalid: items.filter((item) => item.status === "invalid").length
    }),
    [items]
  );

  const selectedCommands = useMemo(
    () => commands.filter((command) => command.plugin_id === selected?.id),
    [commands, selected?.id]
  );

  const selectedToolIntegrations = useMemo(
    () => toolIntegrations.filter((integration) => integration.plugin_id === selected?.id),
    [toolIntegrations, selected?.id]
  );

  const settingsEntries = useMemo(
    () => Object.entries(settings?.schema.properties ?? {}),
    [settings]
  );

  const requiredSettings = useMemo(
    () => new Set(settings?.schema.required ?? []),
    [settings]
  );

  const syncDetail = useCallback(
    async (pluginId: string | null) => {
      if (!pluginId) {
        setCommands([]);
        setToolIntegrations([]);
        setSettings(null);
        setSettingsValues({});
        setAuditLogs([]);
        return;
      }

      setDetailLoading(true);
      try {
        const [nextCommands, nextToolIntegrations, nextSettings, nextAuditLogs] = await Promise.all([
          getExtensionCommands(apiBase),
          getExtensionToolIntegrations(apiBase),
          getExtensionSettings(pluginId, apiBase),
          getExtensionAuditLogs(pluginId, 20, apiBase)
        ]);
        setCommands(nextCommands);
        setToolIntegrations(nextToolIntegrations);
        setSettings(nextSettings);
        setSettingsValues(nextSettings.effective_values ?? {});
        setAuditLogs(nextAuditLogs);
      } catch (nextError) {
        setError(nextError instanceof Error ? nextError.message : t("common.unknownError"));
      } finally {
        setDetailLoading(false);
      }
    },
    [apiBase, t]
  );

  const syncItems = useCallback(
    async (mode: "load" | "rescan" = "load") => {
      setError(null);
      if (mode === "load") {
        setLoading(true);
      } else {
        setBusyAction("rescan");
      }

      try {
        const nextItems = mode === "rescan" ? await rescanExtensions(apiBase) : await getExtensions(apiBase);
        setItems(nextItems);
        setSelectedId((current) =>
          current && nextItems.some((item) => item.id === current)
            ? current
            : nextItems[0]?.id ?? null
        );
        if (mode === "rescan") {
          setFeedback(t("plugins.feedback.rescanned"));
        }
      } catch (nextError) {
        setError(nextError instanceof Error ? nextError.message : t("common.unknownError"));
      } finally {
        setLoading(false);
        setBusyAction(null);
      }
    },
    [apiBase, t]
  );

  useEffect(() => {
    void syncItems();
  }, [syncItems]);

  useEffect(() => {
    void syncDetail(selected?.id ?? null);
  }, [selected?.id, syncDetail]);

  async function handleToggle(item: ExtensionPlugin) {
    setBusyAction(item.enabled ? "disable" : "enable");
    setError(null);
    setFeedback(null);

    try {
      const updated = item.enabled
        ? await disableExtension(item.id, apiBase)
        : await enableExtension(item.id, apiBase);
      setItems((current) => current.map((entry) => (entry.id === updated.id ? updated : entry)));
      await syncDetail(updated.id);
      setFeedback(item.enabled ? t("plugins.feedback.disabled") : t("plugins.feedback.enabled"));
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : t("common.unknownError"));
    } finally {
      setBusyAction(null);
    }
  }

  async function handleSaveSettings() {
    if (!selected) {
      return;
    }

    setSettingsBusy(true);
    setError(null);
    setFeedback(null);
    try {
      const nextSettings = await updateExtensionSettings(
        selected.id,
        serializeSettingsValues(settingsValues),
        apiBase
      );
      setSettings(nextSettings);
      setSettingsValues(nextSettings.effective_values ?? {});
      setAuditLogs(await getExtensionAuditLogs(selected.id, 20, apiBase));
      setFeedback(t("plugins.feedback.settingsSaved"));
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : t("common.unknownError"));
    } finally {
      setSettingsBusy(false);
    }
  }

  async function handleExecuteCommand(command: ExtensionCommand) {
    if (!selected) {
      return;
    }

    setCommandBusy(command.id);
    setError(null);
    setFeedback(null);
    try {
      const result = await executeExtensionCommand(command.id, apiBase);
      setAuditLogs(await getExtensionAuditLogs(selected.id, 20, apiBase));
      setFeedback(t("plugins.feedback.commandExecuted", { status: result.status }));
    } catch (nextError) {
      setAuditLogs(await getExtensionAuditLogs(selected.id, 20, apiBase).catch(() => auditLogs));
      setError(nextError instanceof Error ? nextError.message : t("common.unknownError"));
    } finally {
      setCommandBusy(null);
    }
  }

  async function handleToolIntegrationAction(
    integration: ExtensionToolIntegration,
    action: ExtensionToolIntegrationAction
  ) {
    if (!selected) {
      return;
    }

    const busyKey = `${integration.id}:${action}`;
    setToolActionBusy(busyKey);
    setError(null);
    setFeedback(null);
    try {
      const result = await executeExtensionToolIntegrationAction(integration.id, action, apiBase);
      setAuditLogs(await getExtensionAuditLogs(selected.id, 20, apiBase));
      setFeedback(t("plugins.feedback.toolActionRecorded", { status: result.status }));
    } catch (nextError) {
      setAuditLogs(await getExtensionAuditLogs(selected.id, 20, apiBase).catch(() => auditLogs));
      setError(nextError instanceof Error ? nextError.message : t("common.unknownError"));
    } finally {
      setToolActionBusy(null);
    }
  }

  function updateSettingValue(key: string, value: unknown) {
    setSettingsValues((current) => ({
      ...current,
      [key]: value
    }));
  }

  function renderSettingControl(key: string, property: ExtensionSettingsProperty) {
    const value = settingsValues[key] ?? "";
    if (property.enum && property.enum.length > 0) {
      const currentValue = enumOptionValue(value);
      return (
        <select
          className={inputClass}
          value={currentValue}
          onChange={(event) => updateSettingValue(key, parseEnumOptionValue(event.target.value))}
        >
          {property.enum.map((item) => (
            <option key={enumOptionValue(item)} value={enumOptionValue(item)}>
              {formatValue(item)}
            </option>
          ))}
        </select>
      );
    }

    if (property.type === "boolean") {
      return (
        <label className="inline-flex items-center gap-2 text-sm text-[color:var(--color-text)]">
          <input
            type="checkbox"
            checked={Boolean(value)}
            onChange={(event) => updateSettingValue(key, event.target.checked)}
          />
          {Boolean(value) ? t("settings.runtime.yes") : t("settings.runtime.no")}
        </label>
      );
    }

    if (property.type === "array") {
      return (
        <textarea
          className={`${inputClass} min-h-24 resize-y`}
          value={formatValue(value)}
          onChange={(event) =>
            updateSettingValue(
              key,
              event.target.value
                .split("\n")
                .map((item) => item.trim())
                .filter(Boolean)
            )
          }
        />
      );
    }

    return (
      <input
        className={inputClass}
        type={settingInputType(property)}
        value={formatValue(value)}
        onChange={(event) => {
          if (property.type === "integer" || property.type === "number") {
            const nextValue = event.target.value === "" ? undefined : Number(event.target.value);
            updateSettingValue(key, nextValue);
            return;
          }
          updateSettingValue(key, event.target.value);
        }}
      />
    );
  }

  const selectedContributions = selected ? formatContributionSummary(selected) : [];

  return (
    <main className={pageShellClass}>
      <section className={heroClass}>
        <div className="space-y-4">
          <div>
            <p className={eyebrowClass}>Relay Switch</p>
            <h1 className={heroTitleClass}>{t("plugins.title")}</h1>
          </div>
          <p className={heroCopyClass}>{t("plugins.subtitle")}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className={statusPillClass()}>{t("plugins.stats.total", { count: stats.total })}</span>
          <span className={statusPillClass("success")}>
            {t("plugins.stats.enabled", { count: stats.enabled })}
          </span>
          {stats.invalid > 0 ? (
            <span className={statusPillClass("danger")}>
              {t("plugins.stats.invalid", { count: stats.invalid })}
            </span>
          ) : null}
        </div>
      </section>

      {feedback ? <div className={successNoticeClass}>{feedback}</div> : null}
      {error ? <div className={dangerNoticeClass}>{error}</div> : null}

      <section className="grid min-h-0 gap-4 xl:grid-cols-[360px_minmax(0,1fr)]">
        <div className={sectionCardClass}>
          <div className={sectionHeadClass}>
            <div className="space-y-1">
              <h2 className={sectionTitleClass}>{t("plugins.list.title")}</h2>
              <p className={sectionMetaClass}>{t("plugins.list.subtitle")}</p>
            </div>
            <button
              type="button"
              className={buttonClass("secondary")}
              disabled={busyAction === "rescan"}
              onClick={() => void syncItems("rescan")}
            >
              {busyAction === "rescan" ? t("plugins.button.rescanning") : t("plugins.button.rescan")}
            </button>
          </div>

          <div className="mt-4 grid gap-3">
            {loading ? (
              <div className={emptyStateClass}>{t("common.loading")}</div>
            ) : items.length === 0 ? (
              <div className={emptyStateClass}>{t("plugins.empty")}</div>
            ) : (
              items.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  className={selectableItemClass(selected?.id === item.id)}
                  onClick={() => setSelectedId(item.id)}
                >
                  <span className="flex items-start justify-between gap-3">
                    <span className="min-w-0">
                      <span className="block truncate text-sm font-semibold text-[color:var(--color-heading)]">
                        {item.name}
                      </span>
                      <span className={`${monoClass} mt-1 block text-[11px]`}>{item.id}</span>
                    </span>
                    <span className={statusPillClass(statusTone(item.status, item.enabled))}>
                      {item.status}
                    </span>
                  </span>
                </button>
              ))
            )}
          </div>
        </div>

        <div className={sectionCardClass}>
          {selected ? (
            <div className="space-y-6">
              <div className={sectionHeadClass}>
                <div className="min-w-0 space-y-1">
                  <h2 className={sectionTitleClass}>{selected.name}</h2>
                  <p className={sectionMetaClass}>{selected.description || t("plugins.detail.noDescription")}</p>
                </div>
                <div className={actionRowClass}>
                  <span className={statusPillClass(statusTone(selected.status, selected.enabled))}>
                    {selected.status}
                  </span>
                  <button
                    type="button"
                    className={buttonClass(selected.enabled ? "danger" : "primary")}
                    disabled={
                      selected.status === "invalid" ||
                      selected.status === "incompatible" ||
                      busyAction === "enable" ||
                      busyAction === "disable"
                    }
                    onClick={() => void handleToggle(selected)}
                  >
                    {busyAction === "enable" || busyAction === "disable"
                      ? t("common.saving")
                      : selected.enabled
                        ? t("plugins.button.disable")
                        : t("plugins.button.enable")}
                  </button>
                </div>
              </div>

              <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                <div>
                  <p className={fieldLabelClass}>{t("plugins.detail.version")}</p>
                  <p className="mt-1 text-sm font-medium text-[color:var(--color-heading)]">{selected.version}</p>
                </div>
                <div>
                  <p className={fieldLabelClass}>{t("plugins.detail.scope")}</p>
                  <p className="mt-1 text-sm font-medium text-[color:var(--color-heading)]">{selected.scope}</p>
                </div>
                <div>
                  <p className={fieldLabelClass}>{t("plugins.detail.publisher")}</p>
                  <p className="mt-1 text-sm font-medium text-[color:var(--color-heading)]">
                    {selected.publisher || t("settings.value.unknown")}
                  </p>
                </div>
                <div>
                  <p className={fieldLabelClass}>{t("plugins.detail.entry")}</p>
                  <p className="mt-1 text-sm font-medium text-[color:var(--color-heading)]">
                    {selected.manifest.entry.type}
                  </p>
                </div>
              </div>

              <div className="space-y-2">
                <p className={fieldLabelClass}>{t("plugins.detail.manifestPath")}</p>
                <code className={`${monoClass} block rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-input)] p-3`}>
                  {selected.manifest_path}
                </code>
              </div>

              {selected.last_error ? (
                <div className={dangerNoticeClass}>{selected.last_error}</div>
              ) : null}

              {selected.warnings.length > 0 ? (
                <div className="rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3 text-sm text-[color:var(--color-muted)]">
                  <p className="mb-2 font-medium text-[color:var(--color-heading)]">
                    {t("plugins.detail.warnings")}
                  </p>
                  <ul className="grid gap-1.5">
                    {selected.warnings.map((warning) => (
                      <li key={warning} className="flex items-start gap-2">
                        <span className={`${statusDotClass("warning")} mt-1.5`} />
                        <span>{warning}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              ) : null}

              <div className="grid gap-4 xl:grid-cols-2">
                <div className="space-y-2">
                  <p className={fieldLabelClass}>{t("plugins.detail.permissions")}</p>
                  {selected.permissions.length > 0 ? (
                    <div className="flex flex-wrap gap-2">
                      {selected.permissions.map((permission) => (
                        <span key={permission} className={statusPillClass("warning")}>
                          {permission}
                        </span>
                      ))}
                    </div>
                  ) : (
                    <p className={metaClass}>{t("plugins.detail.noPermissions")}</p>
                  )}
                </div>

                <div className="space-y-2">
                  <p className={fieldLabelClass}>{t("plugins.detail.contributions")}</p>
                  {selectedContributions.length > 0 ? (
                    <div className="flex flex-wrap gap-2">
                      {selectedContributions.map(([name, count]) => (
                        <span key={name} className={statusPillClass()}>
                          {name}: {count}
                        </span>
                      ))}
                    </div>
                  ) : (
                    <p className={metaClass}>{t("plugins.detail.noContributions")}</p>
                  )}
                </div>
              </div>

              <div className="space-y-3 border-t pt-5 [border-color:var(--border-soft)]">
                <div className={sectionHeadClass}>
                  <div className="space-y-1">
                    <h3 className={sectionTitleClass}>{t("plugins.settings.title")}</h3>
                    <p className={sectionMetaClass}>{t("plugins.settings.subtitle")}</p>
                  </div>
                  <button
                    type="button"
                    className={buttonClass("primary")}
                    disabled={settingsBusy || selected.status === "invalid" || detailLoading}
                    onClick={() => void handleSaveSettings()}
                  >
                    {settingsBusy ? t("common.saving") : t("plugins.settings.save")}
                  </button>
                </div>

                {detailLoading ? (
                  <div className={emptyStateClass}>{t("common.loading")}</div>
                ) : settingsEntries.length === 0 ? (
                  <div className={emptyStateClass}>{t("plugins.settings.empty")}</div>
                ) : (
                  <div className="grid gap-3">
                    {settingsEntries.map(([key, property]) => (
                      <label key={key} className="grid gap-2 rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3">
                        <span className="flex flex-wrap items-center gap-2">
                          <span className={fieldLabelClass}>{property.title || key}</span>
                          {requiredSettings.has(key) ? (
                            <span className={statusPillClass("warning")}>{t("plugins.settings.required")}</span>
                          ) : null}
                        </span>
                        {renderSettingControl(key, property)}
                        {property.description ? <span className={hintClass}>{property.description}</span> : null}
                      </label>
                    ))}
                  </div>
                )}
              </div>

              <div className="space-y-3 border-t pt-5 [border-color:var(--border-soft)]">
                <div className={sectionHeadClass}>
                  <div className="space-y-1">
                    <h3 className={sectionTitleClass}>{t("plugins.toolIntegrations.title")}</h3>
                    <p className={sectionMetaClass}>{t("plugins.toolIntegrations.subtitle")}</p>
                  </div>
                </div>
                {selectedToolIntegrations.length === 0 ? (
                  <div className={emptyStateClass}>{t("plugins.toolIntegrations.empty")}</div>
                ) : (
                  <div className="grid gap-2">
                    {selectedToolIntegrations.map((integration) => {
                      const actions = ([
                        {
                          action: "detect",
                          label: t("plugins.toolIntegrations.detect"),
                          supported: integration.supports_detect
                        },
                        {
                          action: "configure",
                          label: t("plugins.toolIntegrations.configure"),
                          supported: integration.supports_configure
                        },
                        {
                          action: "restore",
                          label: t("plugins.toolIntegrations.restore"),
                          supported: integration.supports_restore
                        }
                      ] as Array<{
                        action: ExtensionToolIntegrationAction;
                        label: string;
                        supported: boolean;
                      }>).filter((item) => item.supported);

                      return (
                        <div
                          key={integration.id}
                          className="flex flex-col gap-3 rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3"
                        >
                          <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                            <div className="min-w-0">
                              <p className="text-sm font-semibold text-[color:var(--color-heading)]">
                                {integration.title}
                              </p>
                              <p className={`${monoClass} mt-1 text-[11px]`}>{integration.id}</p>
                            </div>
                            <div className="flex flex-wrap gap-2">
                              {actions.map((item) => (
                                <span key={item.action} className={statusPillClass()}>
                                  {item.action}
                                </span>
                              ))}
                            </div>
                          </div>
                          <div className={actionRowClass}>
                            {actions.map((item) => {
                              const busyKey = `${integration.id}:${item.action}`;
                              return (
                                <button
                                  key={item.action}
                                  type="button"
                                  className={buttonClass("secondary")}
                                  disabled={!integration.enabled || toolActionBusy === busyKey}
                                  onClick={() => void handleToolIntegrationAction(integration, item.action)}
                                >
                                  {toolActionBusy === busyKey
                                    ? t("plugins.toolIntegrations.running")
                                    : item.label}
                                </button>
                              );
                            })}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>

              <div className="space-y-3 border-t pt-5 [border-color:var(--border-soft)]">
                <div className={sectionHeadClass}>
                  <div className="space-y-1">
                    <h3 className={sectionTitleClass}>{t("plugins.commands.title")}</h3>
                    <p className={sectionMetaClass}>{t("plugins.commands.subtitle")}</p>
                  </div>
                </div>
                {selectedCommands.length === 0 ? (
                  <div className={emptyStateClass}>{t("plugins.commands.empty")}</div>
                ) : (
                  <div className="grid gap-2">
                    {selectedCommands.map((command) => (
                      <div
                        key={command.id}
                        className="flex flex-col gap-3 rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3 sm:flex-row sm:items-center sm:justify-between"
                      >
                        <div className="min-w-0">
                          <p className="text-sm font-semibold text-[color:var(--color-heading)]">{command.title}</p>
                          <p className={`${monoClass} mt-1 text-[11px]`}>{command.id}</p>
                          {command.category ? <p className={hintClass}>{command.category}</p> : null}
                        </div>
                        <button
                          type="button"
                          className={buttonClass("secondary")}
                          disabled={!command.enabled || commandBusy === command.id}
                          onClick={() => void handleExecuteCommand(command)}
                        >
                          {commandBusy === command.id ? t("plugins.commands.executing") : t("plugins.commands.execute")}
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="space-y-3 border-t pt-5 [border-color:var(--border-soft)]">
                <div className={sectionHeadClass}>
                  <div className="space-y-1">
                    <h3 className={sectionTitleClass}>{t("plugins.audit.title")}</h3>
                    <p className={sectionMetaClass}>{t("plugins.audit.subtitle")}</p>
                  </div>
                  <button
                    type="button"
                    className={buttonClass("secondary")}
                    disabled={detailLoading}
                    onClick={() => void syncDetail(selected.id)}
                  >
                    {t("common.refresh")}
                  </button>
                </div>
                {auditLogs.length === 0 ? (
                  <div className={emptyStateClass}>{t("plugins.audit.empty")}</div>
                ) : (
                  <div className="grid gap-2">
                    {auditLogs.map((entry) => (
                      <div
                        key={entry.id}
                        className="rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3"
                      >
                        <div className="flex flex-wrap items-center justify-between gap-2">
                          <span className="text-sm font-semibold text-[color:var(--color-heading)]">
                            {entry.action}
                          </span>
                          <span className={statusPillClass(entry.status === "failed" ? "danger" : "default")}>
                            {entry.status}
                          </span>
                        </div>
                        <p className={`${monoClass} mt-1 text-[11px]`}>{entry.capability}</p>
                        <p className={hintClass}>{formatAuditTime(entry.timestamp)}</p>
                        {entry.error_message ? (
                          <p className="mt-2 text-sm text-[color:var(--danger-text)]">{entry.error_message}</p>
                        ) : null}
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <p className={hintClass}>{t("plugins.detail.runtimeHint")}</p>
            </div>
          ) : (
            <div className={emptyStateClass}>{t("plugins.empty")}</div>
          )}
        </div>
      </section>
    </main>
  );
}
