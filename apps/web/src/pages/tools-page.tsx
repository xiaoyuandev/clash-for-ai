import { useEffect, useMemo, useState } from "react";
import { useI18n } from "../i18n/i18n-provider";
import {
  buttonClass,
  emptyStateClass,
  fieldLabelClass,
  heroClass,
  heroCopyClass,
  heroTitleClass,
  iconButtonClass,
  iconButtonSmallClass,
  infoCardClass,
  inputClass,
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

type DesktopState = {
  ok: boolean;
  runtime: string;
  platform: string;
  apiBase: string;
  config: {
    apiPort: number;
    apiPortSource: "default" | "config" | "env";
  };
  updates: {
    currentVersion: string;
    status:
      | "idle"
      | "checking"
      | "available"
      | "not-available"
      | "downloading"
      | "downloaded"
      | "error"
      | "unsupported";
    availableVersion?: string;
    downloadedVersion?: string;
    progressPercent?: number;
    message?: string;
  };
  core: {
    managed: boolean;
    running: boolean;
    apiBase: string;
    port: number;
    pid?: number;
    logRetentionDays: number;
    logMaxRecords: number;
    lastError?: string;
    command?: string;
  };
} | null;

type ToolPreset =
  | "codex-cli"
  | "claude-code"
  | "cursor"
  | "cherry-studio"
  | "open-code"
  | "openai-sdk";
type PlatformPreset = "unix" | "windows-cmd" | "powershell";
type ConnectMode = "command" | "manual";
type ToolCategory = "cli" | "desktop" | "sdk";

type ToolIntegrationState = {
  id: ToolPreset;
  detected: boolean;
  configured: boolean;
  supportsAdapter: boolean;
  configPath?: string;
  secondaryConfigPath?: string;
  executablePath?: string;
  backupPath?: string;
  message?: string;
};

interface ToolsPageProps {
  desktopState: DesktopState;
  onCopyText: (text: string) => Promise<void>;
}

export function ToolsPage({ desktopState, onCopyText }: ToolsPageProps) {
  const { t } = useI18n();
  const [toolPreset, setToolPreset] = useState<ToolPreset>("codex-cli");
  const [platformPreset, setPlatformPreset] = useState<PlatformPreset>("unix");
  const [connectMode, setConnectMode] = useState<ConnectMode>("command");
  const [copyFeedback, setCopyFeedback] = useState<string | null>(null);
  const [manualCopyFeedback, setManualCopyFeedback] = useState<string | null>(null);
  const [toolStates, setToolStates] = useState<Record<ToolPreset, ToolIntegrationState>>(
    {} as Record<ToolPreset, ToolIntegrationState>
  );
  const [configureBusy, setConfigureBusy] = useState<ToolPreset | null>(null);
  const [restoreBusy, setRestoreBusy] = useState<ToolPreset | null>(null);
  const [cherryBusy, setCherryBusy] = useState(false);
  const [actionFeedback, setActionFeedback] = useState<string | null>(null);

  const toolCatalog = useMemo(
    () =>
      [
        {
          id: "codex-cli" as const,
          title: "Codex CLI",
          category: "cli" as ToolCategory,
          recommended: true,
          supportsManual: false,
          supportsAdapter: true,
          fallbackSetupType: t("tools.setup.oneClick")
        },
        {
          id: "claude-code" as const,
          title: "Claude Code",
          category: "cli" as ToolCategory,
          recommended: true,
          supportsManual: false,
          supportsAdapter: true,
          fallbackSetupType: t("tools.setup.oneClick")
        },
        {
          id: "cursor" as const,
          title: "Cursor",
          category: "desktop" as ToolCategory,
          recommended: true,
          supportsManual: true,
          supportsAdapter: false,
          fallbackSetupType: t("tools.setup.manual")
        },
        {
          id: "cherry-studio" as const,
          title: "Cherry Studio",
          category: "desktop" as ToolCategory,
          recommended: true,
          supportsManual: true,
          supportsAdapter: false,
          fallbackSetupType: t("tools.setup.manual")
        },
        {
          id: "open-code" as const,
          title: "Open Code",
          category: "cli" as ToolCategory,
          recommended: false,
          supportsManual: false,
          supportsAdapter: false,
          fallbackSetupType: t("tools.setup.command")
        },
        {
          id: "openai-sdk" as const,
          title: "OpenAI SDK",
          category: "sdk" as ToolCategory,
          recommended: false,
          supportsManual: false,
          supportsAdapter: false,
          fallbackSetupType: t("tools.setup.command")
        }
      ] as const,
    [t]
  );

  useEffect(() => {
    setToolStates(
      toolCatalog.reduce(
        (accumulator, item) => {
          accumulator[item.id] = {
            id: item.id,
            detected: false,
            configured: false,
            supportsAdapter: item.supportsAdapter,
            message: item.supportsAdapter
              ? "Automatic configuration will be enabled after Go core tooling APIs are integrated."
              : "This preset currently provides guidance only."
          };
          return accumulator;
        },
        {} as Record<ToolPreset, ToolIntegrationState>
      )
    );
  }, []);

  const selectedTool = toolCatalog.find((tool) => tool.id === toolPreset) ?? toolCatalog[0];
  const selectedState = toolStates[toolPreset];
  const port = desktopState?.core.port ?? desktopState?.config.apiPort ?? 3456;
  const openAIBase = `http://127.0.0.1:${port}/v1`;
  const anthropicBase = `http://127.0.0.1:${port}`;

  const platformLabel =
    platformPreset === "unix"
      ? t("settings.platform.unix")
      : platformPreset === "windows-cmd"
        ? t("settings.platform.windowsCmd")
        : t("settings.platform.powershell");

  const formatEnvCommands = (pairs: Array<[string, string]>) => {
    switch (platformPreset) {
      case "windows-cmd":
        return pairs.map(([key, value]) => `set ${key}=${value}`).join("\n");
      case "powershell":
        return pairs.map(([key, value]) => `$env:${key}="${value}"`).join("\n");
      case "unix":
      default:
        return pairs.map(([key, value]) => `export ${key}="${value}"`).join("\n");
    }
  };

  const connectionGuide = useMemo(() => {
    const openAICommand = formatEnvCommands([
      ["OPENAI_BASE_URL", openAIBase],
      ["OPENAI_API_KEY", "dummy"]
    ]);
    const anthropicCommand = formatEnvCommands([
      ["ANTHROPIC_BASE_URL", anthropicBase],
      ["ANTHROPIC_AUTH_TOKEN", "dummy"]
    ]);

    const toolMetadata: Record<
      ToolPreset,
      {
        title: string;
        summary: string;
        command: string;
        note: string;
        supportsManual: boolean;
        manualTitle?: string;
        manualItems?: Array<{ label: string; value: string; hint?: string }>;
      }
    > = {
      "codex-cli": {
        title: t("settings.guide.codex.title", { platform: platformLabel }),
        summary: t("settings.guide.codex.summary"),
        command: openAICommand,
        note: t("settings.guide.codex.note"),
        supportsManual: false
      },
      "claude-code": {
        title: t("settings.guide.claude.title", { platform: platformLabel }),
        summary: t("settings.guide.claude.summary"),
        command: anthropicCommand,
        note: t("settings.guide.claude.note"),
        supportsManual: false
      },
      cursor: {
        title: t("settings.guide.cursor.title", { platform: platformLabel }),
        summary: t("settings.guide.cursor.summary"),
        command: openAICommand,
        note: t("settings.guide.cursor.note"),
        supportsManual: true,
        manualTitle: t("settings.guide.cursorManual"),
        manualItems: [
          {
            label: t("settings.guide.field.providerType"),
            value: t("settings.value.openaiCompatible")
          },
          { label: t("settings.guide.field.baseUrl"), value: openAIBase },
          {
            label: t("settings.guide.field.apiKey"),
            value: "dummy",
            hint: t("settings.guide.field.apiKeyHint")
          }
        ]
      },
      "cherry-studio": {
        title: t("settings.guide.cherry.title", { platform: platformLabel }),
        summary: t("settings.guide.cherry.summary"),
        command: openAICommand,
        note: t("settings.guide.cherry.note"),
        supportsManual: true,
        manualTitle: t("settings.guide.cherryManual"),
        manualItems: [
          { label: t("settings.guide.field.protocol"), value: t("settings.value.openaiCompatible") },
          { label: t("settings.guide.field.baseUrl"), value: openAIBase },
          {
            label: t("settings.guide.field.apiKey"),
            value: "dummy",
            hint: t("settings.guide.field.apiKeyHint")
          }
        ]
      },
      "open-code": {
        title: t("settings.guide.openCode.title", { platform: platformLabel }),
        summary: t("settings.guide.openCode.summary"),
        command: openAICommand,
        note: t("settings.guide.openCode.note"),
        supportsManual: false
      },
      "openai-sdk": {
        title: t("settings.guide.openaiSdk.title", { platform: platformLabel }),
        summary: t("settings.guide.openaiSdk.summary"),
        command: openAICommand,
        note: t("settings.guide.openaiSdk.note"),
        supportsManual: false
      }
    };

    return toolMetadata[toolPreset];
  }, [anthropicBase, openAIBase, platformLabel, t, toolPreset]);

  async function handleCopy(text: string, type: "command" | "value", label: string) {
    try {
      await onCopyText(text);
      if (type === "command") {
        setCopyFeedback(t("settings.copy.copied"));
        window.setTimeout(() => {
          setCopyFeedback((current) => (current === t("settings.copy.copied") ? null : current));
        }, 1500);
      } else {
        setManualCopyFeedback(label);
        window.setTimeout(() => {
          setManualCopyFeedback((current) => (current === label ? null : current));
        }, 1500);
      }
    } catch {
      if (type === "command") {
        setCopyFeedback(t("settings.copy.failed"));
      } else {
        setManualCopyFeedback(null);
      }
    }
  }

  async function handleConfigureTool(toolId: ToolPreset) {
    setConfigureBusy(toolId);
    setActionFeedback(null);

    try {
      setActionFeedback(
        "Web supplementary mode currently provides command guidance. One-click configuration will move to Go core in the next phase."
      );
    } catch (error) {
      setActionFeedback(error instanceof Error ? error.message : t("tools.action.failed"));
    } finally {
      setConfigureBusy(null);
    }
  }

  async function handleRestoreTool(toolId: ToolPreset) {
    setRestoreBusy(toolId);
    setActionFeedback(null);

    try {
      setActionFeedback(
        "Restore from backup is not available in web supplementary mode yet. It will be provided by Go core tooling APIs."
      );
    } catch (error) {
      setActionFeedback(error instanceof Error ? error.message : t("tools.action.restoreFailed"));
    } finally {
      setRestoreBusy(null);
    }
  }

  async function handleOpenCherryStudioImport() {
    setCherryBusy(true);
    setActionFeedback(null);

    try {
      setActionFeedback(
        "Deep link launch is only available in the desktop app. Use the manual fields below in web mode."
      );
    } catch (error) {
      setActionFeedback(
        error instanceof Error ? error.message : t("tools.action.cherryOpenFailed")
      );
    } finally {
      setCherryBusy(false);
    }
  }

  const recommendedTools = toolCatalog.filter((tool) => tool.recommended);
  const otherTools = toolCatalog.filter((tool) => !tool.recommended);

  function getCategoryLabel(category: ToolCategory) {
    switch (category) {
      case "desktop":
        return t("tools.category.desktop");
      case "sdk":
        return t("tools.category.sdk");
      case "cli":
      default:
        return t("tools.category.cli");
    }
  }

  function getSetupTypeLabel(tool: (typeof toolCatalog)[number]) {
    if (tool.supportsAdapter) {
      return t("tools.setup.oneClick");
    }

    return tool.fallbackSetupType;
  }

  function getStateLabel(tool: (typeof toolCatalog)[number], state?: ToolIntegrationState) {
    if (state?.configured) {
      return t("tools.state.configured");
    }

    if (state?.detected) {
      return tool.supportsAdapter ? t("tools.state.detected") : t("tools.state.readyGuide");
    }

    return tool.supportsAdapter ? t("tools.state.notDetected") : t("tools.state.guideOnly");
  }

  function getStateVariant(tool: (typeof toolCatalog)[number], state?: ToolIntegrationState) {
    if (state?.configured) {
      return "success" as const;
    }

    if (state?.detected) {
      return tool.supportsAdapter ? ("warning" as const) : ("default" as const);
    }

    return tool.supportsAdapter ? ("danger" as const) : ("default" as const);
  }

  return (
    <main className={pageShellClass}>
      <section className={heroClass}>
        <div className="space-y-4">
          <div>
            <p className={fieldLabelClass}>Clash for AI</p>
            <h1 className={heroTitleClass}>{t("tools.title")}</h1>
          </div>
          <p className={heroCopyClass}>{t("tools.subtitle")}</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <span className={statusPillClass(desktopState?.core.running ? "success" : "danger")}>
            {desktopState?.core.running ? t("app.coreRunning") : t("app.coreStopped")}
          </span>
          <button
            type="button"
            className={buttonClass("secondary")}
            onClick={() => {
              void handleCopy(openAIBase, "value", t("settings.guide.field.baseUrl"));
            }}
          >
            {t("tools.button.copyGateway")}
          </button>
        </div>
      </section>

      <section className="grid gap-4 xl:grid-cols-[332px_minmax(0,1fr)]">
        <div className={`${sectionCardClass} min-w-0`}>
          <div className={sectionHeadClass}>
            <div className="space-y-1">
              <h2 className={sectionTitleClass}>{t("tools.list.title")}</h2>
              <p className={sectionMetaClass}>{t("tools.list.subtitle")}</p>
            </div>
          </div>

          <div className="mt-4 space-y-4">
            <div>
              <p className={fieldLabelClass}>{t("tools.group.recommended")}</p>
              <div className="mt-2.5 grid gap-2.5">
                {recommendedTools.map((tool) => {
                  const state = toolStates[tool.id];
                  return (
                    <button
                      key={tool.id}
                      type="button"
                      className={selectableItemClass(tool.id === toolPreset)}
                      onClick={() => {
                        setToolPreset(tool.id);
                        setConnectMode("command");
                        setActionFeedback(null);
                      }}
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div>
                          <p className="text-base font-semibold text-[color:var(--color-heading)]">
                            {tool.title}
                          </p>
                          <p className={`${metaClass} mt-1`}>{getCategoryLabel(tool.category)}</p>
                        </div>
                        <span className={statusPillClass(getStateVariant(tool, state))}>
                          {getStateLabel(tool, state)}
                        </span>
                      </div>
                      <div className="mt-4 flex flex-wrap gap-2">
                        <span className={statusPillClass()}>{getSetupTypeLabel(tool)}</span>
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>

            <div>
              <p className={fieldLabelClass}>{t("tools.group.more")}</p>
              <div className="mt-2.5 grid gap-2.5">
                {otherTools.map((tool) => {
                  const state = toolStates[tool.id];
                  return (
                    <button
                      key={tool.id}
                      type="button"
                      className={selectableItemClass(tool.id === toolPreset)}
                      onClick={() => {
                        setToolPreset(tool.id);
                        setConnectMode("command");
                        setActionFeedback(null);
                      }}
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div>
                          <p className="text-base font-semibold text-[color:var(--color-heading)]">
                            {tool.title}
                          </p>
                          <p className={`${metaClass} mt-1`}>{getCategoryLabel(tool.category)}</p>
                        </div>
                        <span className={statusPillClass(getStateVariant(tool, state))}>
                          {getStateLabel(tool, state)}
                        </span>
                      </div>
                      <div className="mt-4 flex flex-wrap gap-2">
                        <span className={statusPillClass()}>{getSetupTypeLabel(tool)}</span>
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>
          </div>
        </div>

        <div className={`${sectionCardClass} min-w-0`}>
          <div className={sectionHeadClass}>
            <div className="space-y-1">
              <h2 className={sectionTitleClass}>{selectedTool.title}</h2>
              <p className={sectionMetaClass}>{connectionGuide.summary}</p>
            </div>
            <div className="flex flex-wrap gap-2">
              <span className={statusPillClass()}>{getCategoryLabel(selectedTool.category)}</span>
              <span className={statusPillClass(getStateVariant(selectedTool, selectedState))}>
                {getStateLabel(selectedTool, selectedState)}
              </span>
            </div>
          </div>

          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            <label className="flex flex-col gap-2">
              <span className={fieldLabelClass}>{t("settings.modal.tool")}</span>
              <select
                className={inputClass}
                value={toolPreset}
                onChange={(event) => {
                  setToolPreset(event.target.value as ToolPreset);
                  setConnectMode("command");
                  setActionFeedback(null);
                }}
              >
                {toolCatalog.map((tool) => (
                  <option key={tool.id} value={tool.id}>
                    {tool.title}
                  </option>
                ))}
              </select>
            </label>
            <label className="flex flex-col gap-2">
              <span className={fieldLabelClass}>{t("settings.modal.platform")}</span>
              <select
                className={inputClass}
                value={platformPreset}
                onChange={(event) => setPlatformPreset(event.target.value as PlatformPreset)}
              >
                <option value="unix">{t("settings.platform.unix")}</option>
                <option value="windows-cmd">{t("settings.platform.windowsCmd")}</option>
                <option value="powershell">{t("settings.platform.powershell")}</option>
              </select>
            </label>
          </div>

          {selectedTool.supportsAdapter ? (
            <div className={`${infoCardClass} mt-4 p-4`}>
              <div className="mb-4 flex items-start justify-between gap-3">
                <div>
                  <p className={fieldLabelClass}>{t("tools.detail.oneClickTitle")}</p>
                  <p className={`${metaClass} mt-2`}>{t("tools.detail.oneClickMeta")}</p>
                </div>
                {!selectedState?.configured ? (
                  <button
                    type="button"
                    className={buttonClass("primary")}
                    disabled={configureBusy === selectedTool.id}
                    onClick={() => {
                      void handleConfigureTool(selectedTool.id);
                    }}
                  >
                    {configureBusy === selectedTool.id
                      ? t("tools.button.configuring")
                      : t("tools.button.configure")}
                  </button>
                ) : (
                  <span className={statusPillClass("success")}>{t("tools.state.configured")}</span>
                )}
              </div>
              {selectedState?.backupPath ? (
                  <div className="mb-3 flex items-center justify-end">
                  <button
                    type="button"
                    className={buttonClass("secondary")}
                    disabled={restoreBusy === selectedTool.id}
                    onClick={() => {
                      void handleRestoreTool(selectedTool.id);
                    }}
                  >
                    {restoreBusy === selectedTool.id
                      ? t("tools.button.restoring")
                      : t("tools.button.restore")}
                  </button>
                </div>
              ) : null}
              <pre className="rounded-3xl border [border-color:var(--border-soft)] [background:var(--panel-input)] p-4 text-sm leading-7 text-[color:var(--color-text)]">
                <code>{connectionGuide.command}</code>
              </pre>
              <div className="mt-3 rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-input)] px-3 py-2.5">
                <p className={metaClass}>{connectionGuide.note}</p>
              </div>
              {selectedTool.category === "cli" ? (
                <div className="mt-3 rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-input)] px-3 py-2.5">
                  <p className={metaClass}>{t("tools.detail.cliReloadHint")}</p>
                </div>
              ) : null}
              {actionFeedback ? <p className={`${metaClass} mt-4`}>{actionFeedback}</p> : null}
            </div>
          ) : (
            <>
              {selectedTool.id === "cherry-studio" ? (
                <div className={`${infoCardClass} mt-4 p-4`}>
                  <div className="mb-4 flex items-start justify-between gap-3">
                    <div>
                      <p className={fieldLabelClass}>{t("tools.detail.cherryImportTitle")}</p>
                      <p className={`${metaClass} mt-2`}>{t("tools.detail.cherryImportMeta")}</p>
                    </div>
                    <button
                      type="button"
                      className={buttonClass("primary")}
                      disabled={cherryBusy}
                      onClick={() => {
                        void handleOpenCherryStudioImport();
                      }}
                    >
                      {cherryBusy ? t("tools.button.openingCherry") : t("tools.button.openCherry")}
                    </button>
                  </div>
                  {actionFeedback ? <p className={`${metaClass} mt-4`}>{actionFeedback}</p> : null}
                </div>
              ) : null}

              {connectionGuide.supportsManual ? (
                <div className="mt-4 grid grid-cols-2 rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-1">
                  <button
                    type="button"
                    className={connectMode === "command" ? buttonClass("primary") : buttonClass("ghost")}
                    onClick={() => setConnectMode("command")}
                  >
                    {t("settings.modal.mode.command")}
                  </button>
                  <button
                    type="button"
                    className={connectMode === "manual" ? buttonClass("primary") : buttonClass("ghost")}
                    onClick={() => setConnectMode("manual")}
                  >
                    {t("settings.modal.mode.manual")}
                  </button>
                </div>
              ) : null}

              {connectMode === "command" || !connectionGuide.supportsManual ? (
                <div className={`${infoCardClass} mt-4 p-4`}>
                  <div className="mb-4 flex items-start justify-between gap-3">
                    <div>
                      <p className={fieldLabelClass}>{connectionGuide.title}</p>
                      {copyFeedback ? <p className={`${metaClass} mt-2`}>{copyFeedback}</p> : null}
                    </div>
                    <button
                      type="button"
                      className={iconButtonClass}
                      aria-label={t("settings.copy.command")}
                      title={t("settings.copy.command")}
                      onClick={() => {
                        void handleCopy(connectionGuide.command, "command", connectionGuide.title);
                      }}
                    >
                      <svg className="h-5 w-5 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                        <path d="M9 9h10v10H9z" />
                        <path d="M5 5h10v2H7v8H5z" />
                      </svg>
                    </button>
                  </div>
                  <pre className="rounded-3xl border [border-color:var(--border-soft)] [background:var(--panel-input)] p-4 text-sm leading-7 text-[color:var(--color-text)]">
                    <code>{connectionGuide.command}</code>
                  </pre>
                  <div className="mt-3 rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-input)] px-3 py-2.5">
                    <p className={metaClass}>{connectionGuide.note}</p>
                  </div>
                </div>
              ) : (
                <div className={`${infoCardClass} mt-4 p-4`}>
                  <div className="mb-4">
                    <p className={fieldLabelClass}>{connectionGuide.manualTitle}</p>
                  </div>
                  {connectionGuide.manualItems?.length ? (
                    <div className="grid gap-2.5 sm:grid-cols-2 xl:grid-cols-3">
                      {connectionGuide.manualItems.map((item) => (
                        <div
                          key={item.label}
                          className="rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3"
                        >
                          <div className="flex items-start justify-between gap-3">
                            <p className={fieldLabelClass}>{item.label}</p>
                            <button
                              type="button"
                              className={iconButtonSmallClass}
                              aria-label={t("settings.copy.value", { label: item.label })}
                              title={t("settings.copy.value", { label: item.label })}
                              onClick={() => {
                                void handleCopy(item.value, "value", item.label);
                              }}
                            >
                              <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
                                <path d="M9 9h10v10H9z" />
                                <path d="M5 5h10v2H7v8H5z" />
                              </svg>
                            </button>
                          </div>
                          <p className={`${monoClass} mt-3`}>
                            {manualCopyFeedback === item.label
                              ? `${item.value} · ${t("settings.copy.copied")}`
                              : item.value}
                          </p>
                          {item.hint ? <p className={`${metaClass} mt-2`}>{item.hint}</p> : null}
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className={emptyStateClass}>{t("tools.empty.manual")}</div>
                  )}
                  <div className="mt-3 rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-input)] px-3 py-2.5">
                    <p className={metaClass}>{connectionGuide.note}</p>
                  </div>
                </div>
              )}
            </>
          )}

          <div className={`${infoCardClass} mt-4`}>
            <div className="grid gap-2.5 sm:grid-cols-2">
              <div>
                <p className={fieldLabelClass}>{t("tools.detail.configPath")}</p>
                <p className={`${monoClass} mt-2`}>
                  {selectedState?.configPath ?? t("tools.detail.notAvailable")}
                </p>
              </div>
              <div>
                <p className={fieldLabelClass}>{t("tools.detail.executable")}</p>
                <p className={`${monoClass} mt-2`}>
                  {selectedState?.executablePath ?? t("tools.detail.notDetected")}
                </p>
              </div>
            </div>
            {selectedState?.secondaryConfigPath ? (
              <div className="mt-4">
                <p className={fieldLabelClass}>{t("tools.detail.secondaryConfigPath")}</p>
                <p className={`${monoClass} mt-2`}>{selectedState.secondaryConfigPath}</p>
              </div>
            ) : null}
            {selectedState?.backupPath ? (
              <div className="mt-4">
                <p className={fieldLabelClass}>{t("tools.detail.lastBackup")}</p>
                <p className={`${monoClass} mt-2`}>{selectedState.backupPath}</p>
              </div>
            ) : null}
            {selectedState?.message ? <p className={`${metaClass} mt-4`}>{selectedState.message}</p> : null}
          </div>
        </div>
      </section>
    </main>
  );
}
