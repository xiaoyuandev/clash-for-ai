package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
)

type CreateInput struct {
	Name         string            `json:"name"`
	BaseURL      string            `json:"base_url"`
	AuthMode     AuthMode          `json:"auth_mode"`
	ExtraHeaders map[string]string `json:"extra_headers"`
	APIKey       string            `json:"api_key"`
}

type UpdateInput struct {
	Name         string            `json:"name"`
	BaseURL      string            `json:"base_url"`
	AuthMode     AuthMode          `json:"auth_mode"`
	ExtraHeaders map[string]string `json:"extra_headers"`
	APIKey       string            `json:"api_key"`
}

type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object,omitempty"`
	OwnedBy string `json:"owned_by,omitempty"`
}

type Service struct {
	repository  Repository
	credentials credential.Store
	client      *http.Client
}

func InferAuthMode(name string, baseURL string) AuthMode {
	source := strings.ToLower(strings.TrimSpace(name) + " " + strings.TrimSpace(baseURL))

	if strings.Contains(source, "anthropic") ||
		strings.Contains(source, "claude") ||
		strings.Contains(source, "x-api-key") {
		return AuthModeAPIKey
	}

	return AuthModeBearer
}

func ApplyCredentialHeaders(req *http.Request, item Provider, apiKey string, source http.Header) {
	mode := item.AuthMode
	if mode == "" {
		mode = InferAuthMode(item.Name, item.BaseURL)
	}

	switch mode {
	case AuthModeAPIKey:
		req.Header.Del("Authorization")
		req.Header.Set("x-api-key", apiKey)
		if source != nil {
			if strings.TrimSpace(source.Get("api-key")) != "" || strings.TrimSpace(source.Get("Api-Key")) != "" {
				req.Header.Set("api-key", apiKey)
			}
		}
		if req.Header.Get("anthropic-version") == "" {
			req.Header.Set("anthropic-version", "2023-06-01")
		}
	case AuthModeBoth:
		req.Header.Set("Authorization", rewriteAuthorizationValue(readAuthorizationValue(source), apiKey))
		req.Header.Set("x-api-key", apiKey)
		if source != nil {
			if strings.TrimSpace(source.Get("api-key")) != "" || strings.TrimSpace(source.Get("Api-Key")) != "" {
				req.Header.Set("api-key", apiKey)
			}
		}
	default:
		req.Header.Del("x-api-key")
		req.Header.Del("api-key")
		req.Header.Set("Authorization", rewriteAuthorizationValue(readAuthorizationValue(source), apiKey))
	}

	for key, value := range item.ExtraHeaders {
		req.Header.Set(key, value)
	}
}

func readAuthorizationValue(source http.Header) string {
	if source == nil {
		return ""
	}

	return source.Get("Authorization")
}

func rewriteAuthorizationValue(original string, apiKey string) string {
	trimmed := strings.TrimSpace(original)
	if trimmed == "" {
		return "Bearer " + apiKey
	}

	parts := strings.Fields(trimmed)
	if len(parts) >= 2 {
		return parts[0] + " " + apiKey
	}

	return apiKey
}

func NewService(repository Repository, credentials credential.Store) *Service {
	return &Service{
		repository:  repository,
		credentials: credentials,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *Service) List(ctx context.Context) ([]Provider, error) {
	return s.repository.List(ctx)
}

func (s *Service) GetActive(ctx context.Context) (*Provider, error) {
	return s.repository.GetActive(ctx)
}

func (s *Service) GetByID(ctx context.Context, id string) (*Provider, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Provider, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	id := fmt.Sprintf("provider-%d", time.Now().UnixNano())
	if input.AuthMode == "" {
		input.AuthMode = InferAuthMode(input.Name, input.BaseURL)
	}
	apiKeyRef, err := s.credentials.Save(ctx, fmt.Sprintf("provider/%s/api-key", id), input.APIKey)
	if err != nil {
		return Provider{}, err
	}

	item := Provider{
		ID:           id,
		Name:         input.Name,
		BaseURL:      input.BaseURL,
		APIKeyRef:    apiKeyRef,
		AuthMode:     input.AuthMode,
		ExtraHeaders: input.ExtraHeaders,
		Capabilities: Capabilities{
			SupportsOpenAICompatible:    true,
			SupportsAnthropicCompatible: true,
			SupportsModelsAPI:           true,
			SupportsBalanceAPI:          false,
			SupportsStream:              true,
		},
		Status: Status{
			IsActive:         false,
			LastHealthStatus: "pending",
		},
		APIKeyMasked: maskAPIKey(input.APIKey),
	}

	item.Status.LastHealthcheckAt = now

	return s.repository.Create(ctx, item)
}

func (s *Service) Activate(ctx context.Context, id string) (*Provider, error) {
	return s.repository.Activate(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Provider, error) {
	item, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return Provider{}, err
	}

	if input.AuthMode == "" {
		input.AuthMode = InferAuthMode(input.Name, input.BaseURL)
	}

	item.Name = strings.TrimSpace(input.Name)
	item.BaseURL = strings.TrimSpace(input.BaseURL)
	item.AuthMode = input.AuthMode
	item.ExtraHeaders = input.ExtraHeaders
	if item.ExtraHeaders == nil {
		item.ExtraHeaders = map[string]string{}
	}

	if strings.TrimSpace(input.APIKey) != "" {
		if err := s.credentials.Delete(ctx, item.APIKeyRef); err != nil {
			return Provider{}, err
		}

		apiKeyRef, err := s.credentials.Save(ctx, fmt.Sprintf("provider/%s/api-key", id), input.APIKey)
		if err != nil {
			return Provider{}, err
		}

		item.APIKeyRef = apiKeyRef
		item.APIKeyMasked = maskAPIKey(input.APIKey)
	}

	return s.repository.Update(ctx, *item)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	item, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.credentials.Delete(ctx, item.APIKeyRef); err != nil {
		return err
	}

	return s.repository.Delete(ctx, id)
}

func (s *Service) UpdateStatus(ctx context.Context, id string, status Status) (Provider, error) {
	item, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return Provider{}, err
	}

	item.Status = status

	return s.repository.Update(ctx, *item)
}

func (s *Service) FetchModels(ctx context.Context, id string) ([]ModelInfo, error) {
	item, err := s.repository.GetByID(ctx, id)
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
		return nil, fmt.Errorf("build models request: %w", err)
	}

	ApplyCredentialHeaders(req, *item, apiKey, nil)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request models: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read models response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("models request failed: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var openAIResponse struct {
		Data []ModelInfo `json:"data"`
	}
	if err := json.Unmarshal(body, &openAIResponse); err == nil && len(openAIResponse.Data) > 0 {
		return openAIResponse.Data, nil
	}

	var stringList []string
	if err := json.Unmarshal(body, &stringList); err == nil && len(stringList) > 0 {
		items := make([]ModelInfo, 0, len(stringList))
		for _, id := range stringList {
			items = append(items, ModelInfo{ID: id})
		}
		return items, nil
	}

	var wrappedStrings struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(body, &wrappedStrings); err == nil && len(wrappedStrings.Data) > 0 {
		items := make([]ModelInfo, 0, len(wrappedStrings.Data))
		for _, id := range wrappedStrings.Data {
			items = append(items, ModelInfo{ID: id})
		}
		return items, nil
	}

	return nil, fmt.Errorf("models response format not recognized")
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

func maskAPIKey(value string) string {
	if len(value) <= 4 {
		return "****"
	}

	return fmt.Sprintf("****%s", value[len(value)-4:])
}
