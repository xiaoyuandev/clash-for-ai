package runtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

const runtimeConfigKey = "runtime_config"
const localGatewayClaudeMapKey = "local_gateway_claude_map"

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) GetConfig(ctx context.Context) (Config, error) {
	row := r.db.QueryRowContext(ctx, `SELECT value_json FROM app_settings WHERE key = ?`, runtimeConfigKey)

	var raw string
	if err := row.Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("load runtime config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return Config{}, fmt.Errorf("decode runtime config: %w", err)
	}

	return cfg, nil
}

func (r *SQLiteRepository) SaveConfig(ctx context.Context, cfg Config) (Config, error) {
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return Config{}, fmt.Errorf("encode runtime config: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, `
INSERT INTO app_settings (key, value_json)
VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json
`, runtimeConfigKey, string(encoded)); err != nil {
		return Config{}, fmt.Errorf("save runtime config: %w", err)
	}

	return cfg, nil
}

func (r *SQLiteRepository) GetLocalGatewayClaudeMap(ctx context.Context) (ClaudeCodeModelMap, error) {
	row := r.db.QueryRowContext(ctx, `SELECT value_json FROM app_settings WHERE key = ?`, localGatewayClaudeMapKey)

	var raw string
	if err := row.Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return ClaudeCodeModelMap{}, nil
		}
		return ClaudeCodeModelMap{}, fmt.Errorf("load local gateway claude map: %w", err)
	}

	var cfg ClaudeCodeModelMap
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ClaudeCodeModelMap{}, fmt.Errorf("decode local gateway claude map: %w", err)
	}

	return cfg, nil
}

func (r *SQLiteRepository) SaveLocalGatewayClaudeMap(ctx context.Context, cfg ClaudeCodeModelMap) (ClaudeCodeModelMap, error) {
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return ClaudeCodeModelMap{}, fmt.Errorf("encode local gateway claude map: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, `
INSERT INTO app_settings (key, value_json)
VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json
`, localGatewayClaudeMapKey, string(encoded)); err != nil {
		return ClaudeCodeModelMap{}, fmt.Errorf("save local gateway claude map: %w", err)
	}

	return cfg, nil
}
