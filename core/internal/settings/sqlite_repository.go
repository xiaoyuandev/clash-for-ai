package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

const appSettingsKey = "app_settings"

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Get(ctx context.Context) (AppSettings, error) {
	row := r.db.QueryRowContext(ctx, `SELECT value_json FROM app_settings WHERE key = ?`, appSettingsKey)

	var raw string
	if err := row.Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return AppSettings{}, nil
		}
		return AppSettings{}, fmt.Errorf("load app settings: %w", err)
	}

	var settings AppSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return AppSettings{}, fmt.Errorf("decode app settings: %w", err)
	}

	return settings, nil
}

func (r *SQLiteRepository) Save(ctx context.Context, settings AppSettings) (AppSettings, error) {
	encoded, err := json.Marshal(settings)
	if err != nil {
		return AppSettings{}, fmt.Errorf("encode app settings: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, `
INSERT INTO app_settings (key, value_json)
VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json
`, appSettingsKey, string(encoded)); err != nil {
		return AppSettings{}, fmt.Errorf("save app settings: %w", err)
	}

	return settings, nil
}
