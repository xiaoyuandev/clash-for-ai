package extension

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Plugin, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, version, manifest_path, scope, enabled, status, last_error, manifest_json, created_at, updated_at
FROM plugins
ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list plugins: %w", err)
	}
	defer rows.Close()

	items := []Plugin{}
	for rows.Next() {
		item, err := scanPlugin(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plugins: %w", err)
	}

	return items, nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*Plugin, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, version, manifest_path, scope, enabled, status, last_error, manifest_json, created_at, updated_at
FROM plugins
WHERE id = ?`, id)

	item, err := scanPlugin(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPluginNotFound
		}
		return nil, err
	}

	return &item, nil
}

func (r *SQLiteRepository) Upsert(ctx context.Context, item Plugin) (Plugin, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if item.CreatedAt == "" {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	item = normalizePluginFromManifest(item)

	manifestJSON, err := json.Marshal(item.Manifest)
	if err != nil {
		return Plugin{}, fmt.Errorf("marshal plugin manifest: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
INSERT INTO plugins (
	id, version, manifest_path, scope, enabled, status, last_error, manifest_json, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	version = excluded.version,
	manifest_path = excluded.manifest_path,
	scope = excluded.scope,
	enabled = excluded.enabled,
	status = excluded.status,
	last_error = excluded.last_error,
	manifest_json = excluded.manifest_json,
	updated_at = excluded.updated_at`,
		item.ID,
		item.Version,
		item.ManifestPath,
		string(item.Scope),
		boolToInt(item.Enabled),
		string(item.Status),
		item.LastError,
		string(manifestJSON),
		item.CreatedAt,
		item.UpdatedAt,
	)
	if err != nil {
		return Plugin{}, fmt.Errorf("upsert plugin: %w", err)
	}

	stored, err := r.GetByID(ctx, item.ID)
	if err != nil {
		return Plugin{}, err
	}
	return *stored, nil
}

func (r *SQLiteRepository) SetEnabled(ctx context.Context, id string, enabled bool) (*Plugin, error) {
	status := PluginStatusDisabled
	if enabled {
		status = PluginStatusEnabled
	}

	result, err := r.db.ExecContext(ctx, `
UPDATE plugins
SET enabled = ?, status = ?, last_error = '', updated_at = ?
WHERE id = ? AND status != ?`,
		boolToInt(enabled),
		string(status),
		time.Now().UTC().Format(time.RFC3339),
		id,
		string(PluginStatusInvalid),
	)
	if err != nil {
		return nil, fmt.Errorf("set plugin enabled: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("set plugin enabled rows affected: %w", err)
	}
	if affected == 0 {
		if _, err := r.GetByID(ctx, id); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("cannot enable or disable invalid plugin")
	}

	return r.GetByID(ctx, id)
}

func (r *SQLiteRepository) MarkMissing(ctx context.Context, scope PluginScope, seenIDs []string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if len(seenIDs) == 0 {
		_, err := r.db.ExecContext(ctx, `
UPDATE plugins
SET enabled = 0, status = ?, last_error = ?, updated_at = ?
WHERE scope = ?`,
			string(PluginStatusInvalid),
			"manifest not found",
			now,
			string(scope),
		)
		if err != nil {
			return fmt.Errorf("mark missing plugins: %w", err)
		}
		return nil
	}

	placeholders := make([]string, len(seenIDs))
	args := []any{
		string(PluginStatusInvalid),
		"manifest not found",
		now,
		string(scope),
	}
	for index, id := range seenIDs {
		placeholders[index] = "?"
		args = append(args, id)
	}

	_, err := r.db.ExecContext(ctx, `
UPDATE plugins
SET enabled = 0, status = ?, last_error = ?, updated_at = ?
WHERE scope = ? AND id NOT IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	)
	if err != nil {
		return fmt.Errorf("mark missing plugins: %w", err)
	}
	return nil
}

type pluginScanner interface {
	Scan(dest ...any) error
}

func scanPlugin(scanner pluginScanner) (Plugin, error) {
	var (
		item         Plugin
		scope        string
		status       string
		enabled      int
		lastError    sql.NullString
		manifestJSON string
	)

	if err := scanner.Scan(
		&item.ID,
		&item.Version,
		&item.ManifestPath,
		&scope,
		&enabled,
		&status,
		&lastError,
		&manifestJSON,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Plugin{}, err
	}

	if err := json.Unmarshal([]byte(manifestJSON), &item.Manifest); err != nil {
		return Plugin{}, fmt.Errorf("decode plugin manifest: %w", err)
	}

	item.Scope = PluginScope(scope)
	item.Enabled = enabled != 0
	item.Status = PluginStatus(status)
	if lastError.Valid {
		item.LastError = lastError.String
	}

	return normalizePluginFromManifest(item), nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
