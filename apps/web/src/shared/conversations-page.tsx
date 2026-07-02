import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ComponentType } from "react";
import ReactMarkdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import rehypeSanitize, { defaultSchema } from "rehype-sanitize";
import remarkGfm from "remark-gfm";

export type ConversationSourceID = "codex" | "claude-code";

export interface ConversationCatalog {
  sources: ConversationSource[];
}

export interface ConversationSource {
  id: ConversationSourceID;
  title: string;
  detected: boolean;
  projects: ConversationProject[];
  warnings?: string[];
}

export interface ConversationProject {
  id: string;
  name: string;
  path: string;
  updated_at?: string;
  sessions: ConversationSessionSummary[];
}

export interface ConversationSessionSummary {
  id: string;
  source: ConversationSourceID;
  project_id: string;
  project_name: string;
  project_path: string;
  title: string;
  source_path: string;
  started_at?: string;
  updated_at?: string;
  message_count: number;
  backed_up: boolean;
}

export interface ConversationSession {
  summary: ConversationSessionSummary;
  events: ConversationEvent[];
  markdown?: string;
  warnings?: string[];
}

export interface ConversationEvent {
  id: string;
  role: string;
  kind: string;
  text?: string;
  language?: string;
  metadata?: Record<string, unknown>;
  created_at?: string;
}

export interface ConversationBackupConfig {
  enabled: boolean;
  interval_minutes: number;
  output_dir: string;
  sources: ConversationSourceID[];
  include_markdown: boolean;
  redact_secrets: boolean;
  git: {
    enabled: boolean;
    auto_commit: boolean;
    auto_push: boolean;
  };
  updated_at?: string;
  last_run_at?: string;
  last_error?: string;
}

export interface ConversationBackupResult {
  started_at: string;
  finished_at: string;
  exported: number;
  skipped: number;
  warnings: string[];
  errors: string[];
}

export interface ConversationToastItem {
  id: string;
  message: string;
  tone: "success" | "error" | "default";
}

type Translate = (key: string, values?: Record<string, string | number>) => string;

interface ConversationUiClasses {
  buttonClass: (tone?: "primary" | "secondary" | "ghost") => string;
  emptyStateClass: string;
  fieldLabelClass: string;
  heroTitleClass: string;
  iconButtonClass: string;
  inputClass: string;
  metaClass: string;
  modalBackdropClass: string;
  modalPanelClass: string;
  monoClass: string;
}

interface ConversationApi {
  getConversationBackupConfig: (apiBase?: string) => Promise<ConversationBackupConfig>;
  getConversationCatalog: (apiBase?: string) => Promise<ConversationCatalog>;
  getConversationSession: (
    source: ConversationSourceID,
    sessionId: string,
    apiBase?: string
  ) => Promise<ConversationSession>;
  runConversationBackupNow: (apiBase?: string) => Promise<ConversationBackupResult>;
  updateConversationBackupConfig: (
    input: ConversationBackupConfig,
    apiBase?: string
  ) => Promise<ConversationBackupConfig>;
}

export interface ConversationsPageDependencies {
  api: ConversationApi;
  locale: string;
  logos: Record<ConversationSourceID, string>;
  t: Translate;
  ToastRegion: ComponentType<{
    items: ConversationToastItem[];
    onDismiss: (id: string) => void;
  }>;
  ui: ConversationUiClasses;
}

const ConversationDependenciesContext = createContext<ConversationsPageDependencies | null>(null);

function useConversationDependencies() {
  const dependencies = useContext(ConversationDependenciesContext);
  if (!dependencies) {
    throw new Error("ConversationsPage dependencies are missing.");
  }
  return dependencies;
}

const markdownSanitizeSchema = {
  ...defaultSchema,
  tagNames: [...(defaultSchema.tagNames ?? []), "details", "summary"],
  attributes: {
    ...defaultSchema.attributes,
    code: [...(defaultSchema.attributes?.code ?? []), ["className"]],
    details: [["open"]],
    summary: []
  }
};

interface ConversationsPageProps {
  dependencies: ConversationsPageDependencies;
  apiBase?: string;
  onCopyText?: (text: string) => Promise<void>;
  onSelectBackupDirectory?: () => Promise<string | null>;
  onOpenBackupDirectory?: (path: string) => Promise<void>;
}

const DEFAULT_CONFIG: ConversationBackupConfig = {
  enabled: false,
  interval_minutes: 60,
  output_dir: "",
  sources: ["codex", "claude-code"],
  include_markdown: false,
  redact_secrets: true,
  git: {
    enabled: false,
    auto_commit: false,
    auto_push: false
  }
};

export function ConversationsPage({
  dependencies,
  apiBase,
  onCopyText,
  onSelectBackupDirectory,
  onOpenBackupDirectory
}: ConversationsPageProps) {
  return (
    <ConversationDependenciesContext.Provider value={dependencies}>
      <ConversationsPageContent
        apiBase={apiBase}
        onCopyText={onCopyText}
        onSelectBackupDirectory={onSelectBackupDirectory}
        onOpenBackupDirectory={onOpenBackupDirectory}
      />
    </ConversationDependenciesContext.Provider>
  );
}

function ConversationsPageContent({
  apiBase,
  onCopyText,
  onSelectBackupDirectory,
  onOpenBackupDirectory
}: Omit<ConversationsPageProps, "dependencies">) {
  const { api, locale, t, ToastRegion, ui } = useConversationDependencies();
  const [catalog, setCatalog] = useState<ConversationCatalog | null>(null);
  const [config, setConfig] = useState<ConversationBackupConfig>(DEFAULT_CONFIG);
  const [draftConfig, setDraftConfig] = useState<ConversationBackupConfig>(DEFAULT_CONFIG);
  const [selected, setSelected] = useState<{ source: ConversationSourceID; sessionId: string } | null>(null);
  const [session, setSession] = useState<ConversationSession | null>(null);
  const [loadingCatalog, setLoadingCatalog] = useState(true);
  const [loadingSession, setLoadingSession] = useState(false);
  const [backupRunning, setBackupRunning] = useState(false);
  const [savingConfig, setSavingConfig] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [mobileReaderOpen, setMobileReaderOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [toasts, setToasts] = useState<ConversationToastItem[]>([]);

  const pushToast = useCallback((message: string, tone: ConversationToastItem["tone"]) => {
    setToasts((current) => [
      ...current,
      { id: `${Date.now()}-${Math.random().toString(16).slice(2)}`, message, tone }
    ]);
  }, []);

  const dismissToast = useCallback((id: string) => {
    setToasts((current) => current.filter((item) => item.id !== id));
  }, []);

  const loadCatalog = useCallback(async () => {
    setLoadingCatalog(true);
    try {
      const [nextCatalog, nextConfig] = await Promise.all([
        api.getConversationCatalog(apiBase),
        api.getConversationBackupConfig(apiBase).catch(() => DEFAULT_CONFIG)
      ]);
      setCatalog(nextCatalog);
      setConfig(nextConfig);
      setDraftConfig(nextConfig);
      const allSessions = flattenSessions(nextCatalog);
      setSelected((current) => {
        if (current && allSessions.some((item) => item.source === current.source && item.id === current.sessionId)) {
          return current;
        }
        const first = allSessions[0];
        return first ? { source: first.source, sessionId: first.id } : null;
      });
    } catch (error) {
      pushToast(error instanceof Error ? error.message : t("common.unknownError"), "error");
    } finally {
      setLoadingCatalog(false);
    }
  }, [api, apiBase, pushToast, t]);

  useEffect(() => {
    void loadCatalog();
  }, [loadCatalog]);

  useEffect(() => {
    if (!selected) {
      setSession(null);
      return;
    }
    let cancelled = false;
    setLoadingSession(true);
    void api.getConversationSession(selected.source, selected.sessionId, apiBase)
      .then((nextSession) => {
        if (!cancelled) {
          setSession(nextSession);
        }
      })
      .catch((error) => {
        if (!cancelled) {
          setSession(null);
          pushToast(error instanceof Error ? error.message : t("common.unknownError"), "error");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoadingSession(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [api, apiBase, pushToast, selected, t]);

  const filteredCatalog = useMemo(() => filterCatalog(catalog, search), [catalog, search]);
  const totalProjects = catalog?.sources.reduce((sum, source) => sum + source.projects.length, 0) ?? 0;
  const totalSessions = flattenSessions(catalog).length;

  async function handleBackupNow() {
    setBackupRunning(true);
    try {
      const result = await api.runConversationBackupNow(apiBase);
      if (result.errors.length > 0) {
        pushToast(result.errors.join("; "), "error");
      } else {
        pushToast(t("conversations.backup.completed", { count: result.exported }), "success");
      }
      await loadCatalog();
    } catch (error) {
      pushToast(error instanceof Error ? error.message : t("common.unknownError"), "error");
    } finally {
      setBackupRunning(false);
    }
  }

  async function handleSaveConfig() {
    setSavingConfig(true);
    try {
      const nextConfig = await api.updateConversationBackupConfig(draftConfig, apiBase);
      setConfig(nextConfig);
      setDraftConfig(nextConfig);
      setSettingsOpen(false);
      pushToast(t("conversations.backup.saved"), "success");
    } catch (error) {
      pushToast(error instanceof Error ? error.message : t("common.unknownError"), "error");
    } finally {
      setSavingConfig(false);
    }
  }

  return (
    <main className="flex h-full min-h-0 flex-col overflow-hidden rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-glass)]">
      <ToastRegion items={toasts} onDismiss={dismissToast} />

      <header className="flex min-h-14 items-center justify-between gap-3 border-b px-4 [border-color:var(--border-soft)]">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-soft)] text-[color:var(--accent)]">
            <ConversationIcon />
          </div>
          <div className="min-w-0">
            <h1 className="truncate text-base font-semibold text-[color:var(--color-heading)]">{t("conversations.title")}</h1>
            <p className="hidden truncate text-xs text-[color:var(--color-muted)] md:block">
              {t("conversations.stats.projects", { count: totalProjects })} · {t("conversations.stats.sessions", { count: totalSessions })} ·{" "}
              {config.enabled ? t("conversations.backup.enabled") : t("conversations.backup.disabled")}
            </p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <button
            type="button"
            className={ui.iconButtonClass}
            aria-label={loadingCatalog ? t("common.loading") : t("common.refresh")}
            title={loadingCatalog ? t("common.loading") : t("common.refresh")}
            onClick={() => void loadCatalog()}
          >
            <RefreshIcon />
          </button>
          <button
            type="button"
            className="hidden min-h-9 cursor-pointer items-center rounded-lg border px-3 text-sm font-medium transition [border-color:var(--border-soft)] [background:var(--panel-solid)] text-[color:var(--color-text)] hover:[background:var(--panel-soft)] sm:inline-flex"
            disabled={backupRunning || !config.output_dir}
            onClick={() => void handleBackupNow()}
          >
            {backupRunning ? t("conversations.backup.running") : t("conversations.backup.now")}
          </button>
          <button type="button" className={ui.buttonClass("primary")} onClick={() => setSettingsOpen(true)}>
            {t("conversations.backup.settings")}
          </button>
        </div>
      </header>

      <section className="grid min-h-0 flex-1 lg:grid-cols-[300px_minmax(0,1fr)] 2xl:grid-cols-[340px_minmax(0,1fr)]">
        <ConversationSidebar
          catalog={filteredCatalog}
          loading={loadingCatalog}
          search={search}
          selected={selected}
          onSearchChange={setSearch}
          onSelect={(next) => {
            setSelected(next);
            setMobileReaderOpen(true);
          }}
        />
        <ConversationReader
          session={session}
          loading={loadingSession}
          locale={locale}
          mobileOpen={mobileReaderOpen}
          onBack={() => setMobileReaderOpen(false)}
          onCopyText={onCopyText}
        />
      </section>

      {settingsOpen ? (
        <ConversationBackupDialog
          config={draftConfig}
          saving={savingConfig}
          onChange={setDraftConfig}
          onClose={() => {
            setDraftConfig(config);
            setSettingsOpen(false);
          }}
          onSave={() => void handleSaveConfig()}
          onSelectDirectory={onSelectBackupDirectory}
          onOpenDirectory={onOpenBackupDirectory}
        />
      ) : null}
    </main>
  );
}

function ConversationSidebar({
  catalog,
  loading,
  search,
  selected,
  onSearchChange,
  onSelect
}: {
  catalog: ConversationCatalog | null;
  loading: boolean;
  search: string;
  selected: { source: ConversationSourceID; sessionId: string } | null;
  onSearchChange: (value: string) => void;
  onSelect: (value: { source: ConversationSourceID; sessionId: string }) => void;
}) {
  const { t, ui } = useConversationDependencies();
  const hasSessions = flattenSessions(catalog).length > 0;

  return (
    <aside className="min-h-0 border-r [border-color:var(--border-soft)] [background:var(--panel-soft)] lg:flex lg:flex-col">
      <label className="flex flex-col gap-2 border-b p-3 [border-color:var(--border-soft)]">
        <span className="text-xs font-semibold text-[color:var(--color-subtle)]">{t("conversations.search")}</span>
        <input
          className={`${ui.inputClass} rounded-xl`}
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder={t("conversations.searchPlaceholder")}
        />
      </label>

      <div className="min-h-0 flex-1 space-y-2 overflow-y-auto px-2 py-3">
        {loading ? (
          <div className={ui.emptyStateClass}>{t("common.loading")}</div>
        ) : !hasSessions ? (
          <div className={ui.emptyStateClass}>{t("conversations.empty")}</div>
        ) : (
          catalog?.sources.map((source) => (
            <SourceGroup key={source.id} source={source} selected={selected} onSelect={onSelect} />
          ))
        )}
      </div>
    </aside>
  );
}

function SourceGroup({
  source,
  selected,
  onSelect
}: {
  source: ConversationSource;
  selected: { source: ConversationSourceID; sessionId: string } | null;
  onSelect: (value: { source: ConversationSourceID; sessionId: string }) => void;
}) {
  if (source.projects.length === 0) {
    return null;
  }
  return (
    <section className="space-y-1">
      {source.projects.map((project) => (
        <details key={project.id} open className="group/project">
          <summary className="flex cursor-pointer list-none items-center gap-2 rounded-lg px-2 py-1.5 transition hover:[background:var(--panel-solid)]">
            <span className="shrink-0 text-[color:var(--color-muted)]">
              <SourceIcon source={source.id} />
            </span>
            <div className="min-w-0 flex-1">
              <p className="truncate text-[15px] font-medium text-[color:var(--color-heading)]">{project.name}</p>
            </div>
            <span className="text-[color:var(--color-subtle)] transition group-open/project:rotate-90">
              <ChevronRightIcon />
            </span>
          </summary>
          <div className="mt-0.5 grid gap-0.5 pl-8">
            {project.sessions.map((item) => {
              const active = selected?.source === item.source && selected.sessionId === item.id;
              return (
                <button
                  key={`${item.source}-${item.id}`}
                  type="button"
                  className={[
                    "grid h-8 w-full cursor-pointer grid-cols-[minmax(0,1fr)_auto] items-center gap-2 rounded-lg px-2 text-left transition",
                    active
                      ? "[background:var(--panel-solid)] text-[color:var(--color-heading)]"
                      : "text-[color:var(--color-muted)] hover:[background:var(--panel-soft)] hover:text-[color:var(--color-text)]"
                  ].join(" ")}
                  aria-current={active ? "page" : undefined}
                  title={cleanConversationTitle(item.title)}
                  onClick={() => onSelect({ source: item.source, sessionId: item.id })}
                >
                  <span className="truncate text-sm font-medium">{cleanConversationTitle(item.title)}</span>
                  <span className="shrink-0 text-xs text-[color:var(--color-subtle)]">{formatCompactRelative(item.updated_at)}</span>
                </button>
              );
            })}
          </div>
        </details>
      ))}
    </section>
  );
}

function ConversationReader({
  session,
  loading,
  locale,
  mobileOpen,
  onBack,
  onCopyText
}: {
  session: ConversationSession | null;
  loading: boolean;
  locale: string;
  mobileOpen: boolean;
  onBack: () => void;
  onCopyText?: (text: string) => Promise<void>;
}) {
  const { t, ui } = useConversationDependencies();

  return (
    <article className={`${mobileOpen ? "flex" : "hidden"} min-h-0 flex-col [background:var(--panel-glass)] lg:flex`}>
      <div className="flex min-h-14 items-center justify-between gap-3 border-b px-4 [border-color:var(--border-soft)]">
        <div className="min-w-0">
          <button type="button" className={`${ui.buttonClass("ghost")} mb-2 lg:hidden`} onClick={onBack}>
            {t("conversations.reader.back")}
          </button>
          <h2 className="truncate text-base font-semibold text-[color:var(--color-heading)]">
            {cleanConversationTitle(session?.summary.title ?? t("conversations.reader.emptyTitle"))}
          </h2>
          {session ? (
            <p className="truncate text-xs text-[color:var(--color-muted)]">
              {session.summary.project_name} · {session.summary.source} ·{" "}
              {session.summary.updated_at ? new Date(session.summary.updated_at).toLocaleString(locale === "zh" ? "zh-CN" : "en-US") : "-"}
            </p>
          ) : null}
        </div>
        {session ? (
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              className={ui.iconButtonClass}
              aria-label={t("conversations.reader.copySession")}
              title={t("conversations.reader.copySession")}
              onClick={() => void onCopyText?.(session.summary.id)}
            >
              <CopyIcon />
            </button>
          </div>
        ) : null}
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-6">
        {loading ? (
          <div className={ui.emptyStateClass}>{t("common.loading")}</div>
        ) : !session ? (
          <div className={ui.emptyStateClass}>{t("conversations.reader.empty")}</div>
        ) : (
          <div className="mx-auto grid max-w-4xl gap-6">
            {session.warnings?.length ? (
              <div role="alert" className="rounded-lg border [border-color:var(--warning-border)] [background:var(--warning-soft)] p-3 text-sm text-[color:var(--warning-text)]">
                {session.warnings.join("; ")}
              </div>
            ) : null}
            {session.markdown?.trim() ? (
              <ConversationMarkdownViewer markdown={session.markdown} />
            ) : (
              compactEvents(session.events).map((event, index) => (
                <ConversationEventBlock key={event.id || `${event.kind}-${index}`} event={event} />
              ))
            )}
          </div>
        )}
      </div>
    </article>
  );
}

function ConversationMarkdownViewer({ markdown }: { markdown: string }) {
  const { ui } = useConversationDependencies();
  return (
    <div className="conversation-markdown text-[15px] leading-7 text-[color:var(--color-text)]">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeRaw, [rehypeSanitize, markdownSanitizeSchema]]}
        components={{
          h1: ({ children }) => <h1 className="mt-8 text-xl font-semibold text-[color:var(--color-heading)] first:mt-0">{children}</h1>,
          h2: ({ children }) => <h2 className="mt-7 text-lg font-semibold text-[color:var(--color-heading)] first:mt-0">{children}</h2>,
          h3: ({ children }) => <h3 className="mt-5 text-base font-semibold text-[color:var(--color-heading)]">{children}</h3>,
          p: ({ children }) => <p className="my-3">{children}</p>,
          ul: ({ children }) => <ul className="my-3 list-disc space-y-1 pl-6">{children}</ul>,
          ol: ({ children }) => <ol className="my-3 list-decimal space-y-1 pl-6">{children}</ol>,
          blockquote: ({ children }) => (
            <blockquote className="my-5 ml-auto max-w-[72ch] rounded-2xl border-0 px-4 py-3 text-sm leading-6 [background:var(--panel-solid)] text-[color:var(--color-text)] shadow-[var(--shadow-soft)]">
              {children}
            </blockquote>
          ),
          code: ({ children, className }) => {
            const inline = !className;
            if (inline) {
              return <code className={`${ui.monoClass} rounded-md [background:var(--panel-input)] px-1.5 py-0.5 text-[0.9em]`}>{children}</code>;
            }
            return <code className={`${ui.monoClass} text-xs leading-5 ${className ?? ""}`}>{children}</code>;
          },
          pre: ({ children }) => (
            <pre className={`${ui.monoClass} my-4 max-h-[520px] overflow-auto rounded-lg [background:var(--panel-input)] p-3 text-xs leading-5 text-[color:var(--color-text)]`}>
              {children}
            </pre>
          ),
          details: ({ children }) => (
            <details className="my-3 rounded-lg border px-3 py-2 [border-color:var(--border-soft)] [background:color-mix(in_srgb,var(--panel-soft)_72%,transparent)]">
              {children}
            </details>
          ),
          summary: ({ children }) => (
            <summary className="cursor-pointer select-none text-sm font-medium text-[color:var(--color-muted)]">{children}</summary>
          ),
          table: ({ children }) => (
            <div className="my-4 overflow-x-auto">
              <table className="min-w-full border-collapse text-sm">{children}</table>
            </div>
          ),
          th: ({ children }) => <th className="border px-3 py-2 text-left [border-color:var(--border-soft)] [background:var(--panel-soft)]">{children}</th>,
          td: ({ children }) => <td className="border px-3 py-2 [border-color:var(--border-soft)]">{children}</td>
        }}
      >
        {stripGeneratedMarkdownTitle(markdown)}
      </ReactMarkdown>
    </div>
  );
}

function ConversationEventBlock({ event }: { event: ConversationEvent }) {
  const { ui } = useConversationDependencies();
  const title = event.kind === "tool_call" ? "Tool Call" : event.kind === "tool" ? "Tool Output" : event.kind === "raw" ? "Raw Event" : titleCase(event.role);
  const isCode = event.kind === "tool" || event.kind === "tool_call" || event.kind === "raw";
  const isUser = event.role === "user" && event.kind === "message";
  const isAssistant = event.role === "assistant" && event.kind === "message";

  if (isUser) {
    return (
      <section className="flex justify-end">
        <div className="max-w-[72ch] rounded-2xl px-4 py-3 text-sm leading-6 [background:var(--panel-solid)] text-[color:var(--color-text)] shadow-[var(--shadow-soft)]">
          <p className="whitespace-pre-wrap">{cleanEventText(event.text || "")}</p>
        </div>
      </section>
    );
  }

  if (isAssistant) {
    return (
      <article className="max-w-none text-[color:var(--color-text)]">
        <div className="mb-2 flex items-center gap-2 text-xs text-[color:var(--color-subtle)]">
          <span>{event.created_at ? formatRelative(event.created_at) : ""}</span>
        </div>
        <MarkdownContent text={cleanEventText(event.text || "")} />
      </article>
    );
  }

  if (event.kind === "raw" || event.kind === "tool" || event.kind === "tool_call") {
    return (
      <details className="rounded-lg border px-3 py-2 [border-color:var(--border-soft)] [background:color-mix(in_srgb,var(--panel-soft)_72%,transparent)]">
        <summary className="cursor-pointer select-none text-xs font-semibold text-[color:var(--color-muted)]">
          {title}
          {event.created_at ? <span className="ml-2 font-normal text-[color:var(--color-subtle)]">{formatRelative(event.created_at)}</span> : null}
        </summary>
        <pre className={`${ui.monoClass} mt-2 max-h-56 overflow-auto rounded-md [background:var(--panel-input)] p-3`}>
          {event.text || ""}
        </pre>
      </details>
    );
  }

  return (
    <section className="rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-soft)] p-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <h3 className={ui.fieldLabelClass}>{title}</h3>
        {event.created_at ? <span className="text-xs text-[color:var(--color-subtle)]">{formatRelative(event.created_at)}</span> : null}
      </div>
      {isCode ? (
        <pre className={`${ui.monoClass} max-h-[420px] overflow-auto rounded-md [background:var(--panel-input)] p-3`}>
          {event.text || ""}
        </pre>
      ) : (
        <p className="whitespace-pre-wrap text-sm leading-6 text-[color:var(--color-text)]">{cleanEventText(event.text || "")}</p>
      )}
    </section>
  );
}

function ConversationBackupDialog({
  config,
  saving,
  onChange,
  onClose,
  onSave,
  onSelectDirectory,
  onOpenDirectory
}: {
  config: ConversationBackupConfig;
  saving: boolean;
  onChange: (config: ConversationBackupConfig) => void;
  onClose: () => void;
  onSave: () => void;
  onSelectDirectory?: () => Promise<string | null>;
  onOpenDirectory?: (path: string) => Promise<void>;
}) {
  const { t, ui } = useConversationDependencies();
  const outputRequired = config.enabled && !config.output_dir.trim();
  return (
    <div className={ui.modalBackdropClass} role="dialog" aria-modal="true" aria-labelledby="conversation-backup-title">
      <section className={`${ui.modalPanelClass} max-w-xl`}>
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 id="conversation-backup-title" className={ui.heroTitleClass}>
              {t("conversations.backup.settings")}
            </h2>
            <p className={`${ui.metaClass} mt-1`}>{t("conversations.backup.runtimeHint")}</p>
          </div>
          <button type="button" className={ui.iconButtonClass} onClick={onClose} aria-label={t("common.close")} title={t("common.close")}>
            ×
          </button>
        </div>

        <div className="mt-4 grid gap-4">
          <label className="flex items-center justify-between gap-3 rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3">
            <span>
              <span className="block text-sm font-semibold text-[color:var(--color-heading)]">{t("conversations.backup.enableAuto")}</span>
              <span className={ui.metaClass}>{t("conversations.backup.enableAutoHint")}</span>
            </span>
            <input
              type="checkbox"
              className="h-5 w-5 accent-[var(--accent)]"
              checked={config.enabled}
              onChange={(event) => onChange({ ...config, enabled: event.target.checked })}
            />
          </label>

          <label className="flex flex-col gap-2">
            <span className={ui.fieldLabelClass}>{t("conversations.backup.outputDir")}</span>
            <div className="flex flex-col gap-2 sm:flex-row">
              <input
                className={ui.inputClass}
                value={config.output_dir}
                onChange={(event) => onChange({ ...config, output_dir: event.target.value })}
                placeholder="/Users/name/Dropbox/AI-Conversations"
              />
              {onSelectDirectory ? (
                <button
                  type="button"
                  className={ui.buttonClass("secondary")}
                  onClick={() => {
                    void onSelectDirectory().then((path) => {
                      if (path) {
                        onChange({ ...config, output_dir: path });
                      }
                    });
                  }}
                >
                  {t("conversations.backup.chooseDir")}
                </button>
              ) : null}
              {onOpenDirectory ? (
                <button
                  type="button"
                  className={ui.buttonClass("secondary")}
                  disabled={!config.output_dir.trim()}
                  onClick={() => void onOpenDirectory(config.output_dir)}
                >
                  {t("conversations.backup.openDir")}
                </button>
              ) : null}
            </div>
            {outputRequired ? <span role="alert" className="text-sm text-[color:var(--danger-text)]">{t("conversations.backup.outputRequired")}</span> : null}
          </label>

          <label className="flex flex-col gap-2">
            <span className={ui.fieldLabelClass}>{t("conversations.backup.interval")}</span>
            <select
              className={ui.inputClass}
              value={config.interval_minutes}
              onChange={(event) => onChange({ ...config, interval_minutes: Number(event.target.value) })}
            >
              <option value={15}>15 minutes</option>
              <option value={60}>1 hour</option>
              <option value={360}>6 hours</option>
              <option value={1440}>1 day</option>
            </select>
          </label>

          <fieldset className="grid gap-2 rounded-lg border [border-color:var(--border-soft)] p-3">
            <legend className={ui.fieldLabelClass}>{t("conversations.backup.sources")}</legend>
            {(["codex", "claude-code"] as ConversationSourceID[]).map((source) => (
              <label key={source} className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  className="h-4 w-4 accent-[var(--accent)]"
                  checked={config.sources.includes(source)}
                  onChange={(event) => {
                    const nextSources = event.target.checked
                      ? Array.from(new Set([...config.sources, source]))
                      : config.sources.filter((item) => item !== source);
                    onChange({ ...config, sources: nextSources });
                  }}
                />
                {source === "codex" ? "Codex CLI" : "Claude Code"}
              </label>
            ))}
          </fieldset>

          <label className="flex items-center justify-between gap-3">
            <span className="text-sm text-[color:var(--color-text)]">{t("conversations.backup.markdown")}</span>
            <input type="checkbox" className="h-5 w-5 accent-[var(--accent)]" checked={config.include_markdown} onChange={(event) => onChange({ ...config, include_markdown: event.target.checked })} />
          </label>
          <label className="flex items-center justify-between gap-3">
            <span className="text-sm text-[color:var(--color-text)]">{t("conversations.backup.redact")}</span>
            <input type="checkbox" className="h-5 w-5 accent-[var(--accent)]" checked={config.redact_secrets} onChange={(event) => onChange({ ...config, redact_secrets: event.target.checked })} />
          </label>
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <button type="button" className={ui.buttonClass("secondary")} onClick={onClose}>
            {t("common.cancel")}
          </button>
          <button type="button" className={ui.buttonClass("primary")} disabled={saving || outputRequired} onClick={onSave}>
            {saving ? t("common.saving") : t("conversations.backup.save")}
          </button>
        </div>
      </section>
    </div>
  );
}

function flattenSessions(catalog: ConversationCatalog | null | undefined) {
  return (
    catalog?.sources.flatMap((source) =>
      source.projects.flatMap((project) => project.sessions)
    ) ?? []
  );
}

function filterCatalog(catalog: ConversationCatalog | null, search: string): ConversationCatalog | null {
  if (!catalog || !search.trim()) {
    return catalog;
  }
  const needle = search.trim().toLowerCase();
  return {
    sources: catalog.sources
      .map((source) => ({
        ...source,
        projects: source.projects
          .map((project) => ({
            ...project,
            sessions: project.sessions.filter((session) =>
              `${project.name} ${project.path} ${session.title} ${session.id}`.toLowerCase().includes(needle)
            )
          }))
          .filter((project) => project.sessions.length > 0)
      }))
      .filter((source) => source.projects.length > 0)
  };
}

function formatRelative(value?: string) {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function formatCompactRelative(value?: string) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const diffMs = Date.now() - date.getTime();
  const minute = 60 * 1000;
  const hour = 60 * minute;
  const day = 24 * hour;
  if (diffMs < hour) {
    return `${Math.max(1, Math.floor(diffMs / minute))} 分`;
  }
  if (diffMs < day) {
    return `${Math.floor(diffMs / hour)} 小时`;
  }
  if (diffMs < 7 * day) {
    return `${Math.floor(diffMs / day)} 天`;
  }
  return `${Math.floor(diffMs / (7 * day))} 周`;
}

function cleanConversationTitle(value: string) {
  return cleanEventText(value).replace(/\s+/g, " ").trim();
}

function cleanEventText(value: string) {
  return value
    .replace(/<ide_selection>/g, "")
    .replace(/<ide_opened_file>/g, "")
    .replace(/<local-command-caveat>/g, "")
    .replace(/<\/?[^>\s]+>/g, "")
    .trim();
}

function compactEvents(events: ConversationEvent[]) {
  const meaningful = events.filter((event) => {
    if (event.kind === "meta") {
      return false;
    }
    if (event.kind === "raw") {
      return false;
    }
    return Boolean(cleanEventText(event.text || ""));
  });
  return meaningful.length > 0 ? meaningful : events;
}

function MarkdownContent({ text }: { text: string }) {
  const { ui } = useConversationDependencies();
  const blocks = parseMarkdownBlocks(text);
  return (
    <div className="space-y-3 text-[15px] leading-7 text-[color:var(--color-text)]">
      {blocks.map((block, index) => {
        if (block.type === "heading") {
          const Tag = block.level === 1 ? "h2" : block.level === 2 ? "h3" : "h4";
          return (
            <Tag key={index} className="mt-6 text-base font-semibold text-[color:var(--color-heading)] first:mt-0">
              {renderInlineMarkdown(block.text, ui.monoClass)}
            </Tag>
          );
        }
        if (block.type === "code") {
          return (
            <pre key={index} className={`${ui.monoClass} overflow-auto rounded-lg [background:var(--panel-input)] p-3 text-xs leading-5 text-[color:var(--color-text)]`}>
              {block.text}
            </pre>
          );
        }
        if (block.type === "bullet") {
          return (
            <ul key={index} className="list-disc space-y-1 pl-6">
              {block.items.map((item, itemIndex) => (
                <li key={itemIndex}>{renderInlineMarkdown(item, ui.monoClass)}</li>
              ))}
            </ul>
          );
        }
        if (block.type === "ordered") {
          return (
            <ol key={index} className="list-decimal space-y-1 pl-6">
              {block.items.map((item, itemIndex) => (
                <li key={itemIndex}>{renderInlineMarkdown(item, ui.monoClass)}</li>
              ))}
            </ol>
          );
        }
        return (
          <p key={index} className="whitespace-pre-wrap">
            {renderInlineMarkdown(block.text, ui.monoClass)}
          </p>
        );
      })}
    </div>
  );
}

type MarkdownBlock =
  | { type: "paragraph"; text: string }
  | { type: "heading"; level: number; text: string }
  | { type: "code"; text: string }
  | { type: "bullet"; items: string[] }
  | { type: "ordered"; items: string[] };

function parseMarkdownBlocks(text: string): MarkdownBlock[] {
  const lines = text.replace(/\r\n/g, "\n").split("\n");
  const blocks: MarkdownBlock[] = [];
  let paragraph: string[] = [];
  let code: string[] | null = null;

  const flushParagraph = () => {
    if (paragraph.length > 0) {
      blocks.push({ type: "paragraph", text: paragraph.join("\n") });
      paragraph = [];
    }
  };

  for (const line of lines) {
    if (line.trim().startsWith("```")) {
      if (code) {
        blocks.push({ type: "code", text: code.join("\n") });
        code = null;
      } else {
        flushParagraph();
        code = [];
      }
      continue;
    }
    if (code) {
      code.push(line);
      continue;
    }
    if (!line.trim()) {
      flushParagraph();
      continue;
    }
    const heading = /^(#{1,3})\s+(.+)$/.exec(line);
    if (heading) {
      flushParagraph();
      blocks.push({ type: "heading", level: heading[1].length, text: heading[2] });
      continue;
    }
    const bullet = /^\s*[-*]\s+(.+)$/.exec(line);
    if (bullet) {
      flushParagraph();
      const previous = blocks[blocks.length - 1];
      if (previous?.type === "bullet") {
        previous.items.push(bullet[1]);
      } else {
        blocks.push({ type: "bullet", items: [bullet[1]] });
      }
      continue;
    }
    const ordered = /^\s*\d+\.\s+(.+)$/.exec(line);
    if (ordered) {
      flushParagraph();
      const previous = blocks[blocks.length - 1];
      if (previous?.type === "ordered") {
        previous.items.push(ordered[1]);
      } else {
        blocks.push({ type: "ordered", items: [ordered[1]] });
      }
      continue;
    }
    paragraph.push(line);
  }

  if (code) {
    blocks.push({ type: "code", text: code.join("\n") });
  }
  flushParagraph();
  return blocks.length > 0 ? blocks : [{ type: "paragraph", text }];
}

function stripGeneratedMarkdownTitle(markdown: string) {
  return markdown.replace(/^\s*#\s+[^\n]+(?:\n+|$)/, "");
}

function renderInlineMarkdown(text: string, monoClassName: string) {
  const parts = text.split(/(`[^`]+`)/g);
  return parts.map((part, index) => {
    if (part.startsWith("`") && part.endsWith("`")) {
      return (
        <code key={index} className={`${monoClassName} rounded-md [background:var(--panel-input)] px-1.5 py-0.5 text-[0.9em]`}>
          {part.slice(1, -1)}
        </code>
      );
    }
    return <span key={index}>{part}</span>;
  });
}

function titleCase(value: string) {
  if (!value) {
    return "Message";
  }
  return value.slice(0, 1).toUpperCase() + value.slice(1).replace(/_/g, " ");
}

function SourceIcon({ source }: { source: ConversationSourceID }) {
  const { logos } = useConversationDependencies();
  const logo = logos[source];
  const label = source === "claude-code" ? "Claude Code" : "Codex";
  return (
    <img
      src={logo}
      alt={label}
      className="h-4 w-4 shrink-0 rounded-[4px]"
      draggable={false}
    />
  );
}

function ChevronRightIcon() {
  return (
    <svg className="h-3.5 w-3.5 fill-current" viewBox="0 0 20 20" aria-hidden="true">
      <path d="M7.2 4.3 12.9 10l-5.7 5.7-1.4-1.4 4.3-4.3-4.3-4.3z" />
    </svg>
  );
}

function CopyIcon() {
  return (
    <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M8 7a3 3 0 0 1 3-3h6a3 3 0 0 1 3 3v6a3 3 0 0 1-3 3h-1v1a3 3 0 0 1-3 3H7a3 3 0 0 1-3-3v-6a3 3 0 0 1 3-3h1zm2 1h3a3 3 0 0 1 3 3v3h1a1 1 0 0 0 1-1V7a1 1 0 0 0-1-1h-6a1 1 0 0 0-1 1zM7 10a1 1 0 0 0-1 1v6a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1v-6a1 1 0 0 0-1-1z" />
    </svg>
  );
}

function RefreshIcon() {
  return (
    <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M17.7 6.3A8 8 0 1 0 20 12h-2a6 6 0 1 1-1.8-4.3L13 11h8V3z" />
    </svg>
  );
}

function ConversationIcon() {
  return (
    <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M5 4h14a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H9.8L5 20.6V17a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2m0 2v9h2v1.6l2.9-2.1H19V6zm3 2h8v2H8zm0 4h6v2H8z" />
    </svg>
  );
}
