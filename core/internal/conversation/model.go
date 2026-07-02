package conversation

import "time"

type SourceID string

const (
	SourceCodexCLI   SourceID = "codex"
	SourceClaudeCode SourceID = "claude-code"
)

type Catalog struct {
	Sources []SourceCatalog `json:"sources"`
}

type SourceCatalog struct {
	ID       SourceID  `json:"id"`
	Title    string    `json:"title"`
	Detected bool      `json:"detected"`
	Projects []Project `json:"projects"`
	Warnings []string  `json:"warnings,omitempty"`
}

type Project struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Path      string           `json:"path"`
	UpdatedAt time.Time        `json:"updated_at,omitempty"`
	Sessions  []SessionSummary `json:"sessions"`
}

type SessionSummary struct {
	ID           string    `json:"id"`
	Source       SourceID  `json:"source"`
	ProjectID    string    `json:"project_id"`
	ProjectName  string    `json:"project_name"`
	ProjectPath  string    `json:"project_path"`
	Title        string    `json:"title"`
	SourcePath   string    `json:"source_path"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
	MessageCount int       `json:"message_count"`
	BackedUp     bool      `json:"backed_up"`
}

type Session struct {
	Summary  SessionSummary `json:"summary"`
	Events   []Event        `json:"events"`
	Markdown string         `json:"markdown,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
}

type Event struct {
	ID        string         `json:"id"`
	Role      string         `json:"role"`
	Kind      string         `json:"kind"`
	Text      string         `json:"text,omitempty"`
	Language  string         `json:"language,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at,omitempty"`
}

type BackupConfig struct {
	Enabled         bool       `json:"enabled"`
	IntervalMinutes int        `json:"interval_minutes"`
	OutputDir       string     `json:"output_dir"`
	Sources         []SourceID `json:"sources"`
	IncludeMarkdown bool       `json:"include_markdown"`
	RedactSecrets   bool       `json:"redact_secrets"`
	Git             GitConfig  `json:"git"`
	UpdatedAt       time.Time  `json:"updated_at,omitempty"`
	LastRunAt       time.Time  `json:"last_run_at,omitempty"`
	LastError       string     `json:"last_error,omitempty"`
}

type GitConfig struct {
	Enabled    bool `json:"enabled"`
	AutoCommit bool `json:"auto_commit"`
	AutoPush   bool `json:"auto_push"`
}

type BackupResult struct {
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Exported   int       `json:"exported"`
	Skipped    int       `json:"skipped"`
	Warnings   []string  `json:"warnings"`
	Errors     []string  `json:"errors"`
}

type sessionRef struct {
	Source       SourceID
	ID           string
	ProjectID    string
	ProjectName  string
	ProjectPath  string
	Title        string
	SourcePath   string
	StartedAt    time.Time
	UpdatedAt    time.Time
	MessageCount int
}

type Adapter interface {
	Source() SourceID
	Title() string
	Discover() ([]sessionRef, []string)
	Load(ref sessionRef) (Session, error)
}
