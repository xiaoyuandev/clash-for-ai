package provider

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Provider, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, name, base_url, api_key_ref, auth_mode, extra_headers_json, capabilities_json,
       is_active, last_health_status, last_healthcheck_at, api_key_masked
FROM providers
ORDER BY name ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	defer rows.Close()

	items := []Provider{}
	for rows.Next() {
		item, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate providers: %w", err)
	}

	return items, nil
}

func (r *SQLiteRepository) GetActive(ctx context.Context) (*Provider, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, name, base_url, api_key_ref, auth_mode, extra_headers_json, capabilities_json,
       is_active, last_health_status, last_healthcheck_at, api_key_masked
FROM providers
WHERE is_active = 1
LIMIT 1`)

	item, err := scanProvider(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &item, nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*Provider, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, name, base_url, api_key_ref, auth_mode, extra_headers_json, capabilities_json,
       is_active, last_health_status, last_healthcheck_at, api_key_masked
FROM providers
WHERE id = ?`, id)

	item, err := scanProvider(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}

	return &item, nil
}

func (r *SQLiteRepository) Create(ctx context.Context, item Provider) (Provider, error) {
	extraHeadersJSON, err := json.Marshal(item.ExtraHeaders)
	if err != nil {
		return Provider{}, fmt.Errorf("marshal extra headers: %w", err)
	}

	capabilitiesJSON, err := json.Marshal(item.Capabilities)
	if err != nil {
		return Provider{}, fmt.Errorf("marshal capabilities: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
INSERT INTO providers (
	id, name, base_url, api_key_ref, auth_mode, extra_headers_json, capabilities_json,
	is_active, last_health_status, last_healthcheck_at, api_key_masked
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID,
		item.Name,
		item.BaseURL,
		item.APIKeyRef,
		string(item.AuthMode),
		string(extraHeadersJSON),
		string(capabilitiesJSON),
		boolToInt(item.Status.IsActive),
		item.Status.LastHealthStatus,
		item.Status.LastHealthcheckAt,
		item.APIKeyMasked,
	)
	if err != nil {
		return Provider{}, fmt.Errorf("insert provider: %w", err)
	}

	return item, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, item Provider) (Provider, error) {
	extraHeadersJSON, err := json.Marshal(item.ExtraHeaders)
	if err != nil {
		return Provider{}, fmt.Errorf("marshal extra headers: %w", err)
	}

	capabilitiesJSON, err := json.Marshal(item.Capabilities)
	if err != nil {
		return Provider{}, fmt.Errorf("marshal capabilities: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `
UPDATE providers
SET name = ?, base_url = ?, api_key_ref = ?, auth_mode = ?, extra_headers_json = ?,
    capabilities_json = ?, is_active = ?, last_health_status = ?, last_healthcheck_at = ?, api_key_masked = ?
WHERE id = ?`,
		item.Name,
		item.BaseURL,
		item.APIKeyRef,
		string(item.AuthMode),
		string(extraHeadersJSON),
		string(capabilitiesJSON),
		boolToInt(item.Status.IsActive),
		item.Status.LastHealthStatus,
		item.Status.LastHealthcheckAt,
		item.APIKeyMasked,
		item.ID,
	)
	if err != nil {
		return Provider{}, fmt.Errorf("update provider: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Provider{}, fmt.Errorf("rows affected for update: %w", err)
	}
	if affected == 0 {
		return Provider{}, ErrProviderNotFound
	}

	return item, nil
}

func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM providers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete provider: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for delete: %w", err)
	}
	if affected == 0 {
		return ErrProviderNotFound
	}

	return nil
}

func (r *SQLiteRepository) Activate(ctx context.Context, id string) (*Provider, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin activate tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	result, err := tx.ExecContext(ctx, `UPDATE providers SET is_active = 0`)
	if err != nil {
		return nil, fmt.Errorf("reset active provider: %w", err)
	}
	_ = result

	updateResult, err := tx.ExecContext(ctx, `UPDATE providers SET is_active = 1 WHERE id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("activate provider: %w", err)
	}

	affected, err := updateResult.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected for activate: %w", err)
	}
	if affected == 0 {
		return nil, ErrProviderNotFound
	}

	row := tx.QueryRowContext(ctx, `
SELECT id, name, base_url, api_key_ref, auth_mode, extra_headers_json, capabilities_json,
       is_active, last_health_status, last_healthcheck_at, api_key_masked
FROM providers
WHERE id = ?`, id)

	item, err := scanProvider(row)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit activate tx: %w", err)
	}

	return &item, nil
}

type providerScanner interface {
	Scan(dest ...any) error
}

func scanProvider(scanner providerScanner) (Provider, error) {
	var (
		item             Provider
		authMode         string
		extraHeadersJSON string
		capabilitiesJSON string
		isActive         int
	)

	if err := scanner.Scan(
		&item.ID,
		&item.Name,
		&item.BaseURL,
		&item.APIKeyRef,
		&authMode,
		&extraHeadersJSON,
		&capabilitiesJSON,
		&isActive,
		&item.Status.LastHealthStatus,
		&item.Status.LastHealthcheckAt,
		&item.APIKeyMasked,
	); err != nil {
		return Provider{}, err
	}

	item.AuthMode = AuthMode(authMode)
	item.Status.IsActive = isActive == 1

	if extraHeadersJSON == "" {
		item.ExtraHeaders = map[string]string{}
	} else if err := json.Unmarshal([]byte(extraHeadersJSON), &item.ExtraHeaders); err != nil {
		return Provider{}, fmt.Errorf("decode extra headers: %w", err)
	}

	if capabilitiesJSON == "" {
		item.Capabilities = Capabilities{}
	} else if err := json.Unmarshal([]byte(capabilitiesJSON), &item.Capabilities); err != nil {
		return Provider{}, fmt.Errorf("decode capabilities: %w", err)
	}

	return item, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
