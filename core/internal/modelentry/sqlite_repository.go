package modelentry

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

func (r *SQLiteRepository) List(ctx context.Context) ([]Entry, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, name, model_id, base_url, api_key, provider_type, protocol, enabled, position
FROM gateway_model_entries
ORDER BY position ASC, name ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list gateway model entries: %w", err)
	}
	defer rows.Close()

	items := []Entry{}
	for rows.Next() {
		item, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gateway model entries: %w", err)
	}

	return items, nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*Entry, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, name, model_id, base_url, api_key, provider_type, protocol, enabled, position
FROM gateway_model_entries
WHERE id = ?`, id)

	item, err := scanEntry(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrEntryNotFound
		}
		return nil, err
	}

	return &item, nil
}

func (r *SQLiteRepository) Create(ctx context.Context, entry Entry) (Entry, error) {
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO gateway_model_entries (
	id, name, model_id, base_url, api_key, provider_type, protocol, enabled, position
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID,
		entry.Name,
		entry.ModelID,
		entry.BaseURL,
		entry.APIKey,
		entry.ProviderType,
		entry.Protocol,
		boolToInt(entry.Enabled),
		entry.Position,
	); err != nil {
		return Entry{}, fmt.Errorf("insert gateway model entry: %w", err)
	}

	return entry, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, entry Entry) (Entry, error) {
	result, err := r.db.ExecContext(ctx, `
UPDATE gateway_model_entries
SET name = ?, model_id = ?, base_url = ?, api_key = ?, provider_type = ?, protocol = ?, enabled = ?, position = ?
WHERE id = ?`,
		entry.Name,
		entry.ModelID,
		entry.BaseURL,
		entry.APIKey,
		entry.ProviderType,
		entry.Protocol,
		boolToInt(entry.Enabled),
		entry.Position,
		entry.ID,
	)
	if err != nil {
		return Entry{}, fmt.Errorf("update gateway model entry: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Entry{}, fmt.Errorf("rows affected for gateway model entry update: %w", err)
	}
	if affected == 0 {
		return Entry{}, ErrEntryNotFound
	}

	return entry, nil
}

func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM gateway_model_entries WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete gateway model entry: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for gateway model entry delete: %w", err)
	}
	if affected == 0 {
		return ErrEntryNotFound
	}

	return nil
}

func (r *SQLiteRepository) ReplaceOrder(ctx context.Context, entries []Entry) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin gateway model order tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for index, item := range entries {
		if _, err := tx.ExecContext(ctx, `
UPDATE gateway_model_entries
SET position = ?
WHERE id = ?`,
			index,
			item.ID,
		); err != nil {
			return fmt.Errorf("update gateway model order: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit gateway model order tx: %w", err)
	}

	return nil
}

type entryScanner interface {
	Scan(dest ...any) error
}

func scanEntry(scanner entryScanner) (Entry, error) {
	var (
		item    Entry
		enabled int
	)

	if err := scanner.Scan(
		&item.ID,
		&item.Name,
		&item.ModelID,
		&item.BaseURL,
		&item.APIKey,
		&item.ProviderType,
		&item.Protocol,
		&enabled,
		&item.Position,
	); err != nil {
		return Entry{}, err
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
