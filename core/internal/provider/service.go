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
	Name               string             `json:"name"`
	BaseURL            string             `json:"base_url"`
	AuthMode           AuthMode           `json:"auth_mode"`
	ExtraHeaders       map[string]string  `json:"extra_headers"`
	APIKey             string             `json:"api_key"`
	ClaudeCodeModelMap ClaudeCodeModelMap `json:"claude_code_model_map"`
}

type UpdateInput struct {
	Name               string             `json:"name"`
	BaseURL            string             `json:"base_url"`
	AuthMode           AuthMode           `json:"auth_mode"`
	ExtraHeaders       map[string]string  `json:"extra_headers"`
	APIKey             string             `json:"api_key"`
	ClaudeCodeModelMap ClaudeCodeModelMap `json:"claude_code_model_map"`
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
	items, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	for index := range items {
		items[index] = s.refreshMaskedKey(ctx, items[index])
	}

	return items, nil
}

func (s *Service) GetActive(ctx context.Context) (*Provider, error) {
	item, err := s.repository.GetActive(ctx)
	if err != nil || item == nil {
		return item, err
	}

	refreshed := s.refreshMaskedKey(ctx, *item)
	return &refreshed, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Provider, error) {
	item, err := s.repository.GetByID(ctx, id)
	if err != nil || item == nil {
		return item, err
	}

	refreshed := s.refreshMaskedKey(ctx, *item)
	return &refreshed, nil
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
		APIKey:       input.APIKey,
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
		APIKeyMasked:       maskAPIKey(input.APIKey),
		ClaudeCodeModelMap: normalizeClaudeCodeModelMap(input.ClaudeCodeModelMap),
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
	item.ClaudeCodeModelMap = normalizeClaudeCodeModelMap(input.ClaudeCodeModelMap)
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
		item.APIKey = input.APIKey
		item.APIKeyMasked = maskAPIKey(input.APIKey)
	}

	return s.repository.Update(ctx, *item)
}

func normalizeClaudeCodeModelMap(input ClaudeCodeModelMap) ClaudeCodeModelMap {
	return ClaudeCodeModelMap{
		Opus:   strings.TrimSpace(input.Opus),
		Sonnet: strings.TrimSpace(input.Sonnet),
		Haiku:  strings.TrimSpace(input.Haiku),
	}
}

func (s *Service) ListSelectedModels(ctx context.Context, id string) ([]SelectedModel, error) {
	if _, err := s.repository.GetByID(ctx, id); err != nil {
		return nil, err
	}

	return s.repository.ListSelectedModels(ctx, id)
}

func (s *Service) ReplaceSelectedModels(ctx context.Context, id string, items []SelectedModel) ([]SelectedModel, error) {
	if _, err := s.repository.GetByID(ctx, id); err != nil {
		return nil, err
	}

	normalized := make([]SelectedModel, 0, len(items))
	seen := make(map[string]struct{}, len(items))

	for _, item := range items {
		modelID := strings.TrimSpace(item.ModelID)
		if modelID == "" {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}

		seen[modelID] = struct{}{}
		normalized = append(normalized, SelectedModel{
			ModelID:  modelID,
			Position: len(normalized),
		})
	}

	if err := s.repository.ReplaceSelectedModels(ctx, id, normalized); err != nil {
		return nil, err
	}

	return normalized, nil
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
	target.Path = ResolveModelsPath(baseURL.Path)
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

func ResolveModelsPath(basePath string) string {
	trimmed := strings.TrimRight(basePath, "/")

	switch {
	case trimmed == "":
		return "/v1/models"
	case strings.HasSuffix(trimmed, "/v1"):
		return trimmed + "/models"
	default:
		return trimmed + "/v1/models"
	}
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
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 4 {
		return "****"
	}

	if len(trimmed) <= 12 {
		return fmt.Sprintf("%s****", trimmed[:len(trimmed)-4])
	}

	return fmt.Sprintf("%s••••%s", trimmed[:8], trimmed[len(trimmed)-4:])
}

func (s *Service) refreshMaskedKey(ctx context.Context, item Provider) Provider {
	if strings.TrimSpace(item.APIKeyRef) == "" {
		return item
	}

	apiKey, err := s.credentials.Get(ctx, item.APIKeyRef)
	if err != nil {
		return item
	}

	item.APIKey = apiKey
	item.APIKeyMasked = maskAPIKey(apiKey)
	return item
}
