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
