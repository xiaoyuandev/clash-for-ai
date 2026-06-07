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
SELECT p.id, p.version, p.manifest_path, p.scope, p.enabled, p.status, p.last_error, p.manifest_json,
       p.created_at, p.updated_at, i.source_type, i.source_url, i.install_dir, i.git_commit,
       i.installed_at, i.updated_at
FROM plugins p
LEFT JOIN plugin_installs i ON i.plugin_id = p.id
ORDER BY p.id ASC`)
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
SELECT p.id, p.version, p.manifest_path, p.scope, p.enabled, p.status, p.last_error, p.manifest_json,
       p.created_at, p.updated_at, i.source_type, i.source_url, i.install_dir, i.git_commit,
       i.installed_at, i.updated_at
FROM plugins p
LEFT JOIN plugin_installs i ON i.plugin_id = p.id
WHERE p.id = ?`, id)

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

func (r *SQLiteRepository) UpsertInstall(ctx context.Context, install PluginInstall) (PluginInstall, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if install.InstalledAt == "" {
		install.InstalledAt = now
	}
	install.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
INSERT INTO plugin_installs (
	plugin_id, source_type, source_url, install_dir, git_commit, installed_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(plugin_id) DO UPDATE SET
	source_type = excluded.source_type,
	source_url = excluded.source_url,
	install_dir = excluded.install_dir,
	git_commit = excluded.git_commit,
	updated_at = excluded.updated_at`,
		install.PluginID,
		install.SourceType,
		install.SourceURL,
		install.InstallDir,
		install.GitCommit,
		install.InstalledAt,
		install.UpdatedAt,
	)
	if err != nil {
		return PluginInstall{}, fmt.Errorf("upsert plugin install: %w", err)
	}
	return install, nil
}

func (r *SQLiteRepository) GetInstallByPluginID(ctx context.Context, pluginID string) (*PluginInstall, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT plugin_id, source_type, source_url, install_dir, git_commit, installed_at, updated_at
FROM plugin_installs
WHERE plugin_id = ?`, pluginID)
	return scanPluginInstall(row)
}

func (r *SQLiteRepository) GetInstallBySource(ctx context.Context, sourceType string, sourceURL string) (*PluginInstall, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT plugin_id, source_type, source_url, install_dir, git_commit, installed_at, updated_at
FROM plugin_installs
WHERE source_type = ? AND source_url = ?`, sourceType, sourceURL)
	return scanPluginInstall(row)
}

func (r *SQLiteRepository) DeletePlugin(ctx context.Context, pluginID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete plugin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, statement := range []string{
		`DELETE FROM plugin_settings WHERE plugin_id = ?`,
		`DELETE FROM plugin_grants WHERE plugin_id = ?`,
		`DELETE FROM plugin_audit_logs WHERE plugin_id = ?`,
		`DELETE FROM plugin_installs WHERE plugin_id = ?`,
		`DELETE FROM plugins WHERE id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, statement, pluginID); err != nil {
			return fmt.Errorf("delete plugin rows: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete plugin tx: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) GetSettings(ctx context.Context, pluginID string) (map[string]json.RawMessage, string, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT key, value_json, updated_at
FROM plugin_settings
WHERE plugin_id = ?
ORDER BY key ASC`, pluginID)
	if err != nil {
		return nil, "", fmt.Errorf("get plugin settings: %w", err)
	}
	defer rows.Close()

	values := map[string]json.RawMessage{}
	updatedAt := ""
	for rows.Next() {
		var (
			key          string
			valueJSON    string
			rowUpdatedAt string
		)
		if err := rows.Scan(&key, &valueJSON, &rowUpdatedAt); err != nil {
			return nil, "", fmt.Errorf("scan plugin setting: %w", err)
		}
		values[key] = json.RawMessage(valueJSON)
		if rowUpdatedAt > updatedAt {
			updatedAt = rowUpdatedAt
		}
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate plugin settings: %w", err)
	}

	return values, updatedAt, nil
}

func (r *SQLiteRepository) ReplaceSettings(ctx context.Context, pluginID string, values map[string]json.RawMessage) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin replace plugin settings tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `DELETE FROM plugin_settings WHERE plugin_id = ?`, pluginID); err != nil {
		return "", fmt.Errorf("delete plugin settings: %w", err)
	}

	for key, value := range values {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO plugin_settings (plugin_id, key, value_json, updated_at)
VALUES (?, ?, ?, ?)`,
			pluginID,
			key,
			string(value),
			now,
		); err != nil {
			return "", fmt.Errorf("insert plugin setting: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit replace plugin settings tx: %w", err)
	}

	return now, nil
}

func (r *SQLiteRepository) RecordAudit(ctx context.Context, entry AuditLogEntry) (AuditLogEntry, error) {
	now := time.Now().UTC()
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("audit-%d", now.UnixNano())
	}
	if entry.Timestamp == "" {
		entry.Timestamp = now.Format(time.RFC3339)
	}
	if entry.Metadata == nil {
		entry.Metadata = map[string]any{}
	}
	var latencyMs any
	if entry.LatencyMs != nil {
		latencyMs = *entry.LatencyMs
	}

	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		return AuditLogEntry{}, fmt.Errorf("marshal audit metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
INSERT INTO plugin_audit_logs (
	id, timestamp, plugin_id, plugin_version, capability, action, resource_type, resource_id,
	status, latency_ms, approval_source, error_message, metadata_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID,
		entry.Timestamp,
		entry.PluginID,
		entry.PluginVersion,
		entry.Capability,
		entry.Action,
		nullString(entry.ResourceType),
		nullString(entry.ResourceID),
		entry.Status,
		latencyMs,
		nullString(entry.ApprovalSource),
		nullString(entry.ErrorMessage),
		string(metadataJSON),
	)
	if err != nil {
		return AuditLogEntry{}, fmt.Errorf("record plugin audit log: %w", err)
	}

	return entry, nil
}

func (r *SQLiteRepository) ListAudit(ctx context.Context, pluginID string, limit int) ([]AuditLogEntry, error) {
	if limit <= 0 {
		limit = 100
	}

	var (
		rows *sql.Rows
		err  error
	)
	if pluginID == "" {
		rows, err = r.db.QueryContext(ctx, `
SELECT id, timestamp, plugin_id, plugin_version, capability, action, resource_type, resource_id,
       status, latency_ms, approval_source, error_message, metadata_json
FROM plugin_audit_logs
ORDER BY timestamp DESC, id DESC
LIMIT ?`, limit)
	} else {
		rows, err = r.db.QueryContext(ctx, `
SELECT id, timestamp, plugin_id, plugin_version, capability, action, resource_type, resource_id,
       status, latency_ms, approval_source, error_message, metadata_json
FROM plugin_audit_logs
WHERE plugin_id = ?
ORDER BY timestamp DESC, id DESC
LIMIT ?`, pluginID, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list plugin audit logs: %w", err)
	}
	defer rows.Close()

	items := []AuditLogEntry{}
	for rows.Next() {
		item, err := scanAuditLog(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plugin audit logs: %w", err)
	}

	return items, nil
}

type pluginScanner interface {
	Scan(dest ...any) error
}

func scanPlugin(scanner pluginScanner) (Plugin, error) {
	var (
		item             Plugin
		scope            string
		status           string
		enabled          int
		lastError        sql.NullString
		manifestJSON     string
		sourceType       sql.NullString
		sourceURL        sql.NullString
		installDir       sql.NullString
		gitCommit        sql.NullString
		installedAt      sql.NullString
		installUpdatedAt sql.NullString
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
		&sourceType,
		&sourceURL,
		&installDir,
		&gitCommit,
		&installedAt,
		&installUpdatedAt,
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
	if sourceType.Valid {
		item.Install = &PluginInstall{
			PluginID:    item.ID,
			SourceType:  sourceType.String,
			SourceURL:   nullStringValue(sourceURL),
			InstallDir:  nullStringValue(installDir),
			GitCommit:   nullStringValue(gitCommit),
			InstalledAt: nullStringValue(installedAt),
			UpdatedAt:   nullStringValue(installUpdatedAt),
		}
	}

	return normalizePluginFromManifest(item), nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullString(value string) sql.NullString {
	return sql.NullString{
		String: value,
		Valid:  value != "",
	}
}

func nullStringValue(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func scanPluginInstall(scanner pluginScanner) (*PluginInstall, error) {
	var item PluginInstall
	if err := scanner.Scan(
		&item.PluginID,
		&item.SourceType,
		&item.SourceURL,
		&item.InstallDir,
		&item.GitCommit,
		&item.InstalledAt,
		&item.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPluginNotFound
		}
		return nil, fmt.Errorf("scan plugin install: %w", err)
	}
	return &item, nil
}

func scanAuditLog(scanner pluginScanner) (AuditLogEntry, error) {
	var (
		item           AuditLogEntry
		resourceType   sql.NullString
		resourceID     sql.NullString
		latencyMs      sql.NullInt64
		approvalSource sql.NullString
		errorMessage   sql.NullString
		metadataJSON   sql.NullString
	)

	if err := scanner.Scan(
		&item.ID,
		&item.Timestamp,
		&item.PluginID,
		&item.PluginVersion,
		&item.Capability,
		&item.Action,
		&resourceType,
		&resourceID,
		&item.Status,
		&latencyMs,
		&approvalSource,
		&errorMessage,
		&metadataJSON,
	); err != nil {
		return AuditLogEntry{}, fmt.Errorf("scan plugin audit log: %w", err)
	}

	if resourceType.Valid {
		item.ResourceType = resourceType.String
	}
	if resourceID.Valid {
		item.ResourceID = resourceID.String
	}
	if latencyMs.Valid {
		value := latencyMs.Int64
		item.LatencyMs = &value
	}
	if approvalSource.Valid {
		item.ApprovalSource = approvalSource.String
	}
	if errorMessage.Valid {
		item.ErrorMessage = errorMessage.String
	}
	if metadataJSON.Valid && strings.TrimSpace(metadataJSON.String) != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &item.Metadata); err != nil {
			return AuditLogEntry{}, fmt.Errorf("decode audit metadata: %w", err)
		}
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}

	return item, nil
}
