package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type SQLite struct {
	Path string
	DB   *sql.DB
}

func NewSQLite(path string) (*SQLite, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	store := &SQLite{
		Path: path,
		DB:   db,
	}

	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *SQLite) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}

	return s.DB.Close()
}

func (s *SQLite) migrate() error {
	const providersTable = `
CREATE TABLE IF NOT EXISTS providers (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	base_url TEXT NOT NULL,
	api_key_ref TEXT NOT NULL,
	auth_mode TEXT NOT NULL,
	extra_headers_json TEXT NOT NULL,
	capabilities_json TEXT NOT NULL,
	is_active INTEGER NOT NULL DEFAULT 0,
	last_health_status TEXT NOT NULL DEFAULT 'pending',
	last_healthcheck_at TEXT NOT NULL DEFAULT '',
	api_key_masked TEXT NOT NULL DEFAULT ''
);`

	if _, err := s.DB.Exec(providersTable); err != nil {
		return fmt.Errorf("migrate providers table: %w", err)
	}

	const requestLogsTable = `
CREATE TABLE IF NOT EXISTS request_logs (
	id TEXT PRIMARY KEY,
	timestamp TEXT NOT NULL,
	provider_id TEXT NOT NULL,
	provider_name TEXT NOT NULL,
	method TEXT NOT NULL,
	path TEXT NOT NULL,
	model TEXT,
	status_code INTEGER,
	is_stream INTEGER NOT NULL DEFAULT 0,
	upstream_host TEXT NOT NULL DEFAULT '',
	latency_ms INTEGER NOT NULL DEFAULT 0,
	first_byte_ms INTEGER,
	first_token_ms INTEGER,
	error_type TEXT,
	error_message TEXT,
	error_snippet TEXT
);`

	if _, err := s.DB.Exec(requestLogsTable); err != nil {
		return fmt.Errorf("migrate request_logs table: %w", err)
	}

	const requestLogsTimestampIndex = `
CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp
ON request_logs (timestamp DESC);`

	if _, err := s.DB.Exec(requestLogsTimestampIndex); err != nil {
		return fmt.Errorf("migrate request_logs timestamp index: %w", err)
	}

	return nil
}
