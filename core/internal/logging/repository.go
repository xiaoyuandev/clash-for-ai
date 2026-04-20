package logging

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository interface {
	Create(ctx context.Context, item RequestLog) error
	List(ctx context.Context, limit int) ([]RequestLog, error)
	Prune(ctx context.Context, cutoffTimestamp string, keepLatest int) error
}

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Create(ctx context.Context, item RequestLog) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO request_logs (
	id, timestamp, provider_id, provider_name, method, path, model, status_code,
	is_stream, upstream_host, latency_ms, first_byte_ms, first_token_ms,
	error_type, error_message, error_snippet
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID,
		item.Timestamp,
		item.ProviderID,
		item.ProviderName,
		item.Method,
		item.Path,
		item.Model,
		item.StatusCode,
		boolToInt(item.IsStream),
		item.UpstreamHost,
		item.LatencyMs,
		item.FirstByteMs,
		item.FirstTokenMs,
		item.ErrorType,
		item.ErrorMessage,
		item.ErrorSnippet,
	)
	if err != nil {
		return fmt.Errorf("insert request log: %w", err)
	}

	return nil
}

func (r *SQLiteRepository) List(ctx context.Context, limit int) ([]RequestLog, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT id, timestamp, provider_id, provider_name, method, path, model, status_code,
       is_stream, upstream_host, latency_ms, first_byte_ms, first_token_ms,
       error_type, error_message, error_snippet
FROM request_logs
ORDER BY timestamp DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list request logs: %w", err)
	}
	defer rows.Close()

	items := []RequestLog{}
	for rows.Next() {
		var (
			item     RequestLog
			isStream int
		)

		if err := rows.Scan(
			&item.ID,
			&item.Timestamp,
			&item.ProviderID,
			&item.ProviderName,
			&item.Method,
			&item.Path,
			&item.Model,
			&item.StatusCode,
			&isStream,
			&item.UpstreamHost,
			&item.LatencyMs,
			&item.FirstByteMs,
			&item.FirstTokenMs,
			&item.ErrorType,
			&item.ErrorMessage,
			&item.ErrorSnippet,
		); err != nil {
			return nil, fmt.Errorf("scan request log: %w", err)
		}

		item.IsStream = isStream == 1
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate request logs: %w", err)
	}

	return items, nil
}

func (r *SQLiteRepository) Prune(ctx context.Context, cutoffTimestamp string, keepLatest int) error {
	if cutoffTimestamp != "" {
		if _, err := r.db.ExecContext(ctx, `
DELETE FROM request_logs
WHERE timestamp < ?`, cutoffTimestamp); err != nil {
			return fmt.Errorf("prune request logs by age: %w", err)
		}
	}

	if keepLatest > 0 {
		if _, err := r.db.ExecContext(ctx, `
DELETE FROM request_logs
WHERE id IN (
	SELECT id
	FROM request_logs
	ORDER BY timestamp DESC
	LIMIT -1 OFFSET ?
)`, keepLatest); err != nil {
			return fmt.Errorf("prune request logs by count: %w", err)
		}
	}

	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
