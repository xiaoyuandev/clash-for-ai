package logging

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	repository       Repository
	retentionDays    int
	maxRecordsToKeep int
}

func NewService(repository Repository, retentionDays int, maxRecordsToKeep int) *Service {
	return &Service{
		repository:       repository,
		retentionDays:    retentionDays,
		maxRecordsToKeep: maxRecordsToKeep,
	}
}

func (s *Service) Record(ctx context.Context, entry Entry) error {
	item := RequestLog{
		ID:           fmt.Sprintf("log-%d", time.Now().UnixNano()),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		ProviderID:   entry.ProviderID,
		ProviderName: entry.ProviderName,
		Method:       entry.Method,
		Path:         entry.Path,
		Model:        entry.Model,
		StatusCode:   entry.StatusCode,
		IsStream:     entry.IsStream,
		UpstreamHost: entry.UpstreamHost,
		LatencyMs:    entry.LatencyMs,
		FirstByteMs:  entry.FirstByteMs,
		FirstTokenMs: entry.FirstTokenMs,
		ErrorType:    classifyError(entry.StatusCode, entry.ErrorMessage),
		ErrorMessage: entry.ErrorMessage,
		ErrorSnippet: entry.ErrorSnippet,
	}

	if err := s.repository.Create(ctx, item); err != nil {
		return err
	}

	cutoffTimestamp := ""
	if s.retentionDays > 0 {
		cutoffTimestamp = time.Now().UTC().AddDate(0, 0, -s.retentionDays).Format(time.RFC3339)
	}

	return s.repository.Prune(ctx, cutoffTimestamp, s.maxRecordsToKeep)
}

func (s *Service) List(ctx context.Context, limit int) ([]RequestLog, error) {
	return s.repository.List(ctx, limit)
}

func (s *Service) Clear(ctx context.Context) error {
	return s.repository.Clear(ctx)
}

type Entry struct {
	ProviderID   string
	ProviderName string
	Method       string
	Path         string
	Model        *string
	StatusCode   *int
	IsStream     bool
	UpstreamHost string
	LatencyMs    int64
	FirstByteMs  *int64
	FirstTokenMs *int64
	ErrorMessage *string
	ErrorSnippet *string
}

func classifyError(statusCode *int, errorMessage *string) *string {
	if errorMessage != nil {
		normalized := strings.ToLower(strings.TrimSpace(*errorMessage))
		switch {
		case strings.Contains(normalized, "connection refused"),
			strings.Contains(normalized, "dial tcp"),
			strings.Contains(normalized, "no such host"),
			strings.Contains(normalized, "tls"),
			strings.Contains(normalized, "timeout"):
			value := "network_error"
			return &value
		}
	}

	if statusCode == nil {
		if errorMessage == nil {
			return nil
		}

		value := "network_error"
		return &value
	}

	switch {
	case *statusCode == 401 || *statusCode == 403:
		value := "auth_error"
		return &value
	case *statusCode == 429:
		value := "rate_limited"
		return &value
	case *statusCode >= 500:
		value := "upstream_error"
		return &value
	case *statusCode >= 400:
		value := "client_error"
		return &value
	default:
		if errorMessage != nil && strings.TrimSpace(*errorMessage) != "" {
			value := "unknown_error"
			return &value
		}
		return nil
	}
}
