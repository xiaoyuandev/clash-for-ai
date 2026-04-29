package localgatewaystate

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

func (r *SQLiteRepository) ListSelectedModels(ctx context.Context) ([]SelectedModel, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT model_id, position
FROM local_gateway_selected_models
ORDER BY position ASC, model_id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list local gateway selected models: %w", err)
	}
	defer rows.Close()

	items := []SelectedModel{}
	for rows.Next() {
		var item SelectedModel
		if err := rows.Scan(&item.ModelID, &item.Position); err != nil {
			return nil, fmt.Errorf("scan local gateway selected model: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate local gateway selected models: %w", err)
	}

	return items, nil
}

func (r *SQLiteRepository) ReplaceSelectedModels(ctx context.Context, items []SelectedModel) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace local gateway selected models tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `DELETE FROM local_gateway_selected_models`); err != nil {
		return fmt.Errorf("delete local gateway selected models: %w", err)
	}

	for index, item := range items {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO local_gateway_selected_models (model_id, position)
VALUES (?, ?)`,
			item.ModelID,
			index,
		); err != nil {
			return fmt.Errorf("insert local gateway selected model: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace local gateway selected models tx: %w", err)
	}

	return nil
}
