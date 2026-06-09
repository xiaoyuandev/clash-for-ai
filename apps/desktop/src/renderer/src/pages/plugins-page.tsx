import { type FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useI18n } from "../i18n/i18n-provider";
import {
  disableExtension,
  enableExtension,
  executeExtensionCommand,
  getDeveloperMode,
  executeExtensionToolIntegrationAction,
  getExtensionAuditLogs,
  getExtensionCommands,
  getExtensions,
  getExtensionSettings,
  getExtensionToolIntegrations,
  installExtension,
  loadLocalExtension,
  rescanExtensions,
  uninstallExtension,
  updateExtension,
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
  iconButtonClass,
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

function isDirectorySetting(key: string, property: ExtensionSettingsProperty) {
  const text = `${key} ${property.title ?? ""} ${property.description ?? ""}`.toLowerCase();
  return text.includes("directory") || text.includes("目录");
}

function formatEntry(item: ExtensionPlugin) {
  const entry = item.manifest.entry;
  if (entry.type === "nodePackage") {
    return `${entry.package ?? ""}@${entry.version ?? ""} (${entry.bin ?? "bin"})`;
  }
  if (entry.type === "process") {
    return entry.command ?? "process";
  }
  return entry.type;
}

function shortCommit(value?: string) {
  return value ? value.slice(0, 12) : "";
}

function canSelectDirectory() {
  return Boolean(window.desktopBridge?.selectDirectory);
}

async function selectDirectory() {
  return window.desktopBridge?.selectDirectory?.() ?? null;
}

type PluginActionIconName = "enable" | "disable" | "update" | "uninstall";

function PluginActionIcon({ name, spinning = false }: { name: PluginActionIconName; spinning?: boolean }) {
  const className = `h-4 w-4 ${spinning ? "animate-spin" : ""}`;

  if (name === "enable") {
    return (
      <svg
        className={className}
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="m8 5 11 7-11 7V5Z" />
      </svg>
    );
  }

  if (name === "disable") {
    return (
      <svg
        className={className}
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M8 5v14" />
        <path d="M16 5v14" />
      </svg>
    );
  }

  if (name === "update") {
    return (
      <svg
        className={className}
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M21 12a9 9 0 0 1-15.2 6.5" />
        <path d="M3 12A9 9 0 0 1 18.2 5.5" />
        <path d="M18 2v4h4" />
        <path d="M6 22v-4H2" />
      </svg>
    );
  }

  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M3 6h18" />
      <path d="M8 6V4h8v2" />
      <path d="M19 6l-1 14H6L5 6" />
      <path d="M10 11v5" />
      <path d="M14 11v5" />
    </svg>
  );
}

function pluginActionButtonClass(variant: "primary" | "secondary" | "danger" = "secondary") {
  const base =
    "group relative inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--accent-strong)]/55 focus-visible:ring-offset-2 focus-visible:ring-offset-transparent disabled:cursor-not-allowed disabled:opacity-50";
  const variants = {
    primary:
      "[border-color:var(--accent-strong)]/25 [background:linear-gradient(135deg,var(--accent)_0%,var(--accent-strong)_100%)] text-[color:var(--accent-text)] shadow-[0_14px_32px_color-mix(in_srgb,var(--accent)_24%,transparent)] hover:brightness-105",
    secondary:
      "[border-color:var(--border-soft)] [background:var(--panel-solid)] text-[color:var(--color-text)] hover:[border-color:var(--border-strong)] hover:[background:var(--panel-soft)]",
    danger:
      "[border-color:var(--danger-border)] [background:var(--danger-soft)] text-[color:var(--danger-text)] hover:brightness-105"
  };

  return `${base} ${variants[variant]}`;
}

function PluginActionTooltip({ label }: { label: string }) {
  return (
    <span
      className="pointer-events-none absolute left-1/2 top-full z-30 mt-2 -translate-x-1/2 whitespace-nowrap rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-solid)] px-2 py-1 text-[11px] font-medium text-[color:var(--color-heading)] opacity-0 shadow-[var(--shadow-soft)] transition-opacity duration-150 group-hover:opacity-100 group-focus-visible:opacity-100"
      aria-hidden="true"
    >
      {label}
    </span>
  );
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
  const [busyAction, setBusyAction] = useState<
    "rescan" | "install" | "enable" | "disable" | "update" | "uninstall" | null
  >(null);
  const [settingsBusy, setSettingsBusy] = useState(false);
  const [commandBusy, setCommandBusy] = useState<string | null>(null);
  const [toolActionBusy, setToolActionBusy] = useState<string | null>(null);
  const [installUrl, setInstallUrl] = useState("");
  const [localPath, setLocalPath] = useState("");
  const [developerMode, setDeveloperMode] = useState(false);
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

  const directoryPickerAvailable = canSelectDirectory();

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
        const [nextItems, nextDeveloperMode] = await Promise.all([
          mode === "rescan" ? rescanExtensions(apiBase) : getExtensions(apiBase),
          getDeveloperMode(apiBase)
        ]);
        setItems(nextItems);
        setDeveloperMode(nextDeveloperMode.enabled);
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

  async function handleInstall(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const url = installUrl.trim();
    if (!url) {
      return;
    }

    setBusyAction("install");
    setError(null);
    setFeedback(null);
    try {
      const installed = await installExtension({ source: "github", url }, apiBase);
      setItems((current) =>
        [...current.filter((entry) => entry.id !== installed.id), installed].sort((left, right) =>
          left.id.localeCompare(right.id)
        )
      );
      setSelectedId(installed.id);
      setInstallUrl("");
      await syncDetail(installed.id);
      setFeedback(t("plugins.feedback.installed"));
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : t("common.unknownError"));
    } finally {
      setBusyAction(null);
    }
  }

  async function handlePickLocalDirectory() {
    setError(null);
    try {
      const path = await selectDirectory();
      if (path) {
        setLocalPath(path);
      }
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : t("common.unknownError"));
    }
  }

  async function handleLocalInstall(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const path = localPath.trim();
    if (!path) {
      return;
    }

    setBusyAction("install");
    setError(null);
    setFeedback(null);
    try {
      const installed = await loadLocalExtension({ path }, apiBase);
      setItems((current) =>
        [...current.filter((entry) => entry.id !== installed.id), installed].sort((left, right) =>
          left.id.localeCompare(right.id)
        )
      );
      setSelectedId(installed.id);
      await syncDetail(installed.id);
      setFeedback(t("plugins.feedback.localLoaded"));
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : t("common.unknownError"));
    } finally {
      setBusyAction(null);
    }
  }

  async function handleUpdate(item: ExtensionPlugin) {
    setBusyAction("update");
    setError(null);
    setFeedback(null);
    try {
      const updated = await updateExtension(item.id, apiBase);
      setItems((current) => current.map((entry) => (entry.id === updated.id ? updated : entry)));
      await syncDetail(updated.id);
      setFeedback(
        item.install?.source_type === "localDirectory"
          ? t("plugins.feedback.reloaded")
          : t("plugins.feedback.updated")
      );
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : t("common.unknownError"));
    } finally {
      setBusyAction(null);
    }
  }

  async function handleUninstall(item: ExtensionPlugin) {
    if (!window.confirm(t("plugins.uninstall.confirm", { name: item.name }))) {
      return;
    }

    setBusyAction("uninstall");
    setError(null);
    setFeedback(null);
    try {
      await uninstallExtension(item.id, apiBase);
      const nextItems = items.filter((entry) => entry.id !== item.id);
      setItems(nextItems);
      setSelectedId(nextItems[0]?.id ?? null);
      setFeedback(t("plugins.feedback.uninstalled"));
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

  async function handlePickSettingDirectory(key: string) {
    setError(null);
    try {
      const path = await selectDirectory();
      if (path) {
        updateSettingValue(key, path);
      }
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : t("common.unknownError"));
    }
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
          {t("settings.runtime.yes")}
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

    if (property.type === "string" && isDirectorySetting(key, property)) {
      return (
        <div className="grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]">
          <input
            className={inputClass}
            type="text"
            value={formatValue(value)}
            onChange={(event) => updateSettingValue(key, event.target.value)}
          />
          {directoryPickerAvailable ? (
            <button
              type="button"
              className={buttonClass("secondary")}
              onClick={() => void handlePickSettingDirectory(key)}
            >
              {t("plugins.install.chooseDirectory")}
            </button>
          ) : null}
        </div>
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

      <form
        className={`${sectionCardClass} grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end`}
        onSubmit={handleInstall}
      >
        <label className="grid gap-2">
          <span className={fieldLabelClass}>{t("plugins.install.githubUrl")}</span>
          <input
            className={inputClass}
            type="url"
            value={installUrl}
            placeholder="https://github.com/owner/repo"
            onChange={(event) => setInstallUrl(event.target.value)}
          />
        </label>
        <button
          type="submit"
          className={`${buttonClass("primary")} sm:min-w-[132px]`}
          disabled={busyAction === "install" || installUrl.trim() === ""}
        >
          {busyAction === "install" ? t("plugins.install.installing") : t("plugins.install.button")}
        </button>
      </form>

      {developerMode ? (
        <form
          className={`${sectionCardClass} grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end`}
          onSubmit={handleLocalInstall}
        >
          <label className="grid gap-2">
            <span className={fieldLabelClass}>{t("plugins.install.localDirectory")}</span>
            <div className="grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]">
              <input
                className={inputClass}
                type="text"
                value={localPath}
                placeholder="/absolute/path/to/plugin"
                onChange={(event) => setLocalPath(event.target.value)}
              />
              {directoryPickerAvailable ? (
                <button
                  type="button"
                  className={buttonClass("secondary")}
                  onClick={() => void handlePickLocalDirectory()}
                >
                  {t("plugins.install.chooseDirectory")}
                </button>
              ) : null}
            </div>
          </label>
          <button
            type="submit"
            className={`${buttonClass("secondary")} sm:min-w-[132px]`}
            disabled={busyAction === "install" || localPath.trim() === ""}
          >
            {busyAction === "install" ? t("plugins.install.loadingLocal") : t("plugins.install.localButton")}
          </button>
        </form>
      ) : null}

      <section className="grid min-h-0 gap-4 xl:grid-cols-[360px_minmax(0,1fr)]">
        <div className={sectionCardClass}>
          <div className={sectionHeadClass}>
            <div className="space-y-1">
              <h2 className={sectionTitleClass}>{t("plugins.list.title")}</h2>
            </div>
            <button
              type="button"
              className={iconButtonClass}
              disabled={busyAction === "rescan"}
              aria-label={busyAction === "rescan" ? t("plugins.button.rescanning") : t("plugins.button.rescan")}
              title={busyAction === "rescan" ? t("plugins.button.rescanning") : t("plugins.button.rescan")}
              onClick={() => void syncItems("rescan")}
            >
              <svg
                className={`h-4 w-4 ${busyAction === "rescan" ? "animate-spin" : ""}`}
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <path d="M21 12a9 9 0 0 1-15.2 6.5" />
                <path d="M3 12A9 9 0 0 1 18.2 5.5" />
                <path d="M18 2v4h4" />
                <path d="M6 22v-4H2" />
              </svg>
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
                  <div className="flex shrink-0 flex-nowrap items-center gap-2">
                    <button
                      type="button"
                      className={pluginActionButtonClass(selected.enabled ? "secondary" : "primary")}
                      disabled={
                        selected.status === "invalid" ||
                        selected.status === "incompatible" ||
                        busyAction === "enable" ||
                        busyAction === "disable"
                      }
                      aria-label={selected.enabled ? t("plugins.button.disable") : t("plugins.button.enable")}
                      onClick={() => void handleToggle(selected)}
                    >
                      <PluginActionIcon
                        name={
                          busyAction === "enable" || busyAction === "disable"
                            ? "update"
                            : selected.enabled
                              ? "disable"
                              : "enable"
                        }
                        spinning={busyAction === "enable" || busyAction === "disable"}
                      />
                      <PluginActionTooltip
                        label={selected.enabled ? t("plugins.button.disable") : t("plugins.button.enable")}
                      />
                    </button>
                    {selected.install ? (
                      <>
                        <button
                          type="button"
                          className={pluginActionButtonClass("secondary")}
                          disabled={busyAction === "update"}
                          aria-label={
                            selected.install.source_type === "localDirectory"
                              ? t("plugins.button.reload")
                              : t("plugins.button.update")
                          }
                          onClick={() => void handleUpdate(selected)}
                        >
                          <PluginActionIcon name="update" spinning={busyAction === "update"} />
                          <PluginActionTooltip
                            label={
                              selected.install.source_type === "localDirectory"
                                ? t("plugins.button.reload")
                                : t("plugins.button.update")
                            }
                          />
                        </button>
                        <button
                          type="button"
                          className={pluginActionButtonClass("danger")}
                          disabled={busyAction === "uninstall"}
                          aria-label={t("plugins.button.uninstall")}
                          onClick={() => void handleUninstall(selected)}
                        >
                          <PluginActionIcon
                            name={busyAction === "uninstall" ? "update" : "uninstall"}
                            spinning={busyAction === "uninstall"}
                          />
                          <PluginActionTooltip label={t("plugins.button.uninstall")} />
                        </button>
                      </>
                    ) : null}
                  </div>
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
                    {formatEntry(selected)}
                  </p>
                </div>
                <div>
                  <p className={fieldLabelClass}>{t("plugins.detail.runtime")}</p>
                  <p className="mt-1 text-sm font-medium text-[color:var(--color-heading)]">
                    {selected.runtime.state}
                  </p>
                </div>
              </div>

              {selected.install ? (
                <div className="grid gap-3 border-t pt-5 [border-color:var(--border-soft)] sm:grid-cols-2">
                  <div className="min-w-0">
                    <p className={fieldLabelClass}>{t("plugins.detail.source")}</p>
                    <p className={`${monoClass} mt-1 truncate text-[11px]`}>{selected.install.source_url}</p>
                  </div>
                  <div>
                    <p className={fieldLabelClass}>{t("plugins.detail.gitCommit")}</p>
                    <p className={`${monoClass} mt-1 text-[11px]`}>
                      {shortCommit(selected.install.git_commit) || t("settings.value.unknown")}
                    </p>
                  </div>
                </div>
              ) : null}

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

              {selected.runtime.command ? (
                <p className={hintClass}>
                  {t("plugins.detail.runtimeCommand", {
                    command: [selected.runtime.command, ...(selected.runtime.args ?? [])].join(" ")
                  })}
                </p>
              ) : null}
            </div>
          ) : (
            <div className={emptyStateClass}>{t("plugins.empty")}</div>
          )}
        </div>
      </section>
    </main>
  );
}
