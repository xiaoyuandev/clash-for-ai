package modelsource

import (
	"context"
	"database/sql"
	"fmt"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Source, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, name, base_url, provider_type, default_model_id, enabled, position, api_key_ref, api_key_masked
FROM model_sources
ORDER BY position ASC, name ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list model sources: %w", err)
	}
	defer rows.Close()

	items := []Source{}
	for rows.Next() {
		item, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model sources: %w", err)
	}

	return items, nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*Source, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, name, base_url, provider_type, default_model_id, enabled, position, api_key_ref, api_key_masked
FROM model_sources
WHERE id = ?`, id)

	item, err := scanSource(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrSourceNotFound
		}
		return nil, err
	}

	return &item, nil
}

func (r *SQLiteRepository) Create(ctx context.Context, source Source) (Source, error) {
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO model_sources (
	id, name, base_url, provider_type, default_model_id, enabled, position, api_key_ref, api_key_masked
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		source.ID,
		source.Name,
		source.BaseURL,
		source.ProviderType,
		source.DefaultModelID,
		boolToInt(source.Enabled),
		source.Position,
		source.APIKeyRef,
		source.APIKeyMasked,
	); err != nil {
		return Source{}, fmt.Errorf("insert model source: %w", err)
	}

	return source, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, source Source) (Source, error) {
	result, err := r.db.ExecContext(ctx, `
UPDATE model_sources
SET name = ?, base_url = ?, provider_type = ?, default_model_id = ?, enabled = ?, position = ?, api_key_ref = ?, api_key_masked = ?
WHERE id = ?`,
		source.Name,
		source.BaseURL,
		source.ProviderType,
		source.DefaultModelID,
		boolToInt(source.Enabled),
		source.Position,
		source.APIKeyRef,
		source.APIKeyMasked,
		source.ID,
	)
	if err != nil {
		return Source{}, fmt.Errorf("update model source: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Source{}, fmt.Errorf("rows affected for model source update: %w", err)
	}
	if affected == 0 {
		return Source{}, ErrSourceNotFound
	}

	return source, nil
}

func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM model_sources WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete model source: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for model source delete: %w", err)
	}
	if affected == 0 {
		return ErrSourceNotFound
	}

	return nil
}

func (r *SQLiteRepository) ReplaceOrder(ctx context.Context, sources []Source) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin model source order tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for index, source := range sources {
		if _, err := tx.ExecContext(ctx, `
UPDATE model_sources
SET position = ?
WHERE id = ?`,
			index,
			source.ID,
		); err != nil {
			return fmt.Errorf("update model source order: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit model source order tx: %w", err)
	}

	return nil
}

type sourceScanner interface {
	Scan(dest ...any) error
}

func scanSource(scanner sourceScanner) (Source, error) {
	var (
		item    Source
		enabled int
	)

	if err := scanner.Scan(
		&item.ID,
		&item.Name,
		&item.BaseURL,
		&item.ProviderType,
		&item.DefaultModelID,
		&enabled,
		&item.Position,
		&item.APIKeyRef,
		&item.APIKeyMasked,
	); err != nil {
		return Source{}, err
	}

	item.Enabled = enabled == 1
	return item, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
