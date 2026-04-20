package health

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type CheckResult struct {
	Status      string    `json:"status"`
	StatusCode  int       `json:"status_code"`
	LatencyMs   int64     `json:"latency_ms"`
	Summary     string    `json:"summary"`
	CheckedAt   time.Time `json:"checked_at"`
	ProviderID  string    `json:"provider_id"`
	ProviderURL string    `json:"provider_url"`
}

type ProviderReader interface {
	GetByID(ctx context.Context, id string) (*provider.Provider, error)
	UpdateStatus(ctx context.Context, id string, status provider.Status) (provider.Provider, error)
}

type Service struct {
	providers   ProviderReader
	credentials credential.Store
	client      *http.Client
}

func NewService(providers ProviderReader, credentials credential.Store) *Service {
	return &Service{
		providers:   providers,
		credentials: credentials,
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (s *Service) CheckProvider(ctx context.Context, id string) (*CheckResult, error) {
	item, err := s.providers.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(item.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid provider base_url: %w", err)
	}

	apiKey, err := s.credentials.Get(ctx, item.APIKeyRef)
	if err != nil {
		return nil, fmt.Errorf("load provider credential: %w", err)
	}

	target := *baseURL
	target.Path = joinURLPath(baseURL.Path, "/models")
	target.RawPath = target.Path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build healthcheck request: %w", err)
	}

	provider.ApplyCredentialHeaders(req, *item, apiKey, nil)

	startedAt := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		result := &CheckResult{
			Status:      "error",
			StatusCode:  0,
			LatencyMs:   time.Since(startedAt).Milliseconds(),
			Summary:     err.Error(),
			CheckedAt:   time.Now().UTC(),
			ProviderID:  item.ID,
			ProviderURL: item.BaseURL,
		}
		_, _ = s.providers.UpdateStatus(ctx, item.ID, provider.Status{
			IsActive:          item.Status.IsActive,
			LastHealthStatus:  "error",
			LastHealthcheckAt: result.CheckedAt.Format(time.RFC3339),
		})
		return result, nil
	}
	defer resp.Body.Close()

	bodySnippet := ""
	if body, readErr := io.ReadAll(io.LimitReader(resp.Body, 256)); readErr == nil {
		bodySnippet = string(body)
	}

	status := "ok"
	summary := fmt.Sprintf("HTTP %d", resp.StatusCode)
	if resp.StatusCode >= 400 {
		status = "error"
		if bodySnippet != "" {
			summary = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, bodySnippet)
		}
	}

	result := &CheckResult{
		Status:      status,
		StatusCode:  resp.StatusCode,
		LatencyMs:   time.Since(startedAt).Milliseconds(),
		Summary:     summary,
		CheckedAt:   time.Now().UTC(),
		ProviderID:  item.ID,
		ProviderURL: item.BaseURL,
	}

	_, updateErr := s.providers.UpdateStatus(ctx, item.ID, provider.Status{
		IsActive:          item.Status.IsActive,
		LastHealthStatus:  status,
		LastHealthcheckAt: result.CheckedAt.Format(time.RFC3339),
	})
	if updateErr != nil {
		return nil, updateErr
	}

	return result, nil
}

func joinURLPath(basePath string, requestPath string) string {
	switch {
	case basePath == "":
		if requestPath == "" {
			return "/"
		}
		return requestPath
	case requestPath == "":
		return basePath
	case strings.HasSuffix(basePath, "/") && strings.HasPrefix(requestPath, "/"):
		return basePath + strings.TrimPrefix(requestPath, "/")
	case !strings.HasSuffix(basePath, "/") && !strings.HasPrefix(requestPath, "/"):
		return basePath + "/" + requestPath
	default:
		return basePath + requestPath
	}
}
