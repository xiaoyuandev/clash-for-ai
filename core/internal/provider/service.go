package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/xiaoyuandev/relay-switch/core/internal/credential"
)

type CreateInput struct {
	Name               string             `json:"name"`
	BaseURL            string             `json:"base_url"`
	ModelsPath         string             `json:"models_path"`
	AuthMode           AuthMode           `json:"auth_mode"`
	ExtraHeaders       map[string]string  `json:"extra_headers"`
	APIKey             string             `json:"api_key"`
	ClaudeCodeModelMap ClaudeCodeModelMap `json:"claude_code_model_map"`
}

type UpdateInput struct {
	Name               string             `json:"name"`
	BaseURL            string             `json:"base_url"`
	ModelsPath         string             `json:"models_path"`
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

type TestModelInput struct {
	ModelID string `json:"model_id"`
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

func normalizeProviderManagement(item Provider) Provider {
	if item.RuntimeKind == "" {
		item.RuntimeKind = RuntimeKindExternal
	}
	if !item.IsSystemManaged {
		item.IsEditable = true
		item.IsDeletable = true
		return item
	}
	if !item.IsEditable && !item.IsDeletable {
		return item
	}
	return item
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
		Name:         strings.TrimSpace(input.Name),
		BaseURL:      strings.TrimSpace(input.BaseURL),
		ModelsPath:   normalizeModelsPath(input.ModelsPath),
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
	item = normalizeProviderManagement(item)

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
	if item.IsSystemManaged && !item.IsEditable {
		if !s.canUpdateManagedProviderClaudeSlots(ctx, *item, input) {
			return Provider{}, ErrProviderNotEditable
		}

		item.ClaudeCodeModelMap = normalizeClaudeCodeModelMap(input.ClaudeCodeModelMap)
		return s.repository.Update(ctx, normalizeProviderManagement(*item))
	}

	if input.AuthMode == "" {
		input.AuthMode = InferAuthMode(input.Name, input.BaseURL)
	}

	item.Name = strings.TrimSpace(input.Name)
	item.BaseURL = strings.TrimSpace(input.BaseURL)
	item.ModelsPath = normalizeModelsPath(input.ModelsPath)
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

	normalized := normalizeProviderManagement(*item)

	return s.repository.Update(ctx, normalized)
}

func (s *Service) canUpdateManagedProviderClaudeSlots(ctx context.Context, item Provider, input UpdateInput) bool {
	if strings.TrimSpace(input.Name) != strings.TrimSpace(item.Name) {
		return false
	}
	if strings.TrimSpace(input.BaseURL) != strings.TrimSpace(item.BaseURL) {
		return false
	}

	expectedAuthMode := item.AuthMode
	if expectedAuthMode == "" {
		expectedAuthMode = InferAuthMode(item.Name, item.BaseURL)
	}
	inputAuthMode := input.AuthMode
	if inputAuthMode == "" {
		inputAuthMode = InferAuthMode(input.Name, input.BaseURL)
	}
	if inputAuthMode != expectedAuthMode {
		return false
	}

	if !reflect.DeepEqual(normalizeExtraHeaders(input.ExtraHeaders), normalizeExtraHeaders(item.ExtraHeaders)) {
		return false
	}

	trimmedAPIKey := strings.TrimSpace(input.APIKey)
	if trimmedAPIKey == "" {
		return true
	}

	currentAPIKey, err := s.credentials.Get(ctx, item.APIKeyRef)
	if err != nil {
		return false
	}
	return trimmedAPIKey == currentAPIKey
}

func normalizeClaudeCodeModelMap(input ClaudeCodeModelMap) ClaudeCodeModelMap {
	return ClaudeCodeModelMap{
		Opus:   strings.TrimSpace(input.Opus),
		Sonnet: strings.TrimSpace(input.Sonnet),
		Haiku:  strings.TrimSpace(input.Haiku),
	}
}

func normalizeExtraHeaders(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}

	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
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

func (s *Service) ListCodexModels(ctx context.Context, id string) ([]CodexModel, error) {
	if _, err := s.repository.GetByID(ctx, id); err != nil {
		return nil, err
	}

	return s.repository.ListCodexModels(ctx, id)
}

func (s *Service) ReplaceCodexModels(ctx context.Context, id string, items []CodexModel) ([]CodexModel, error) {
	if _, err := s.repository.GetByID(ctx, id); err != nil {
		return nil, err
	}

	normalized := make([]CodexModel, 0, len(items))
	seen := make(map[string]struct{}, len(items))

	for _, item := range items {
		modelID := strings.TrimSpace(item.ModelID)
		if modelID == "" {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}

		displayName := strings.TrimSpace(item.DisplayName)
		if displayName == "" {
			displayName = modelID
		}

		var contextWindow *int
		if item.ContextWindow != nil && *item.ContextWindow > 0 {
			value := *item.ContextWindow
			contextWindow = &value
		}

		seen[modelID] = struct{}{}
		normalized = append(normalized, CodexModel{
			ProviderID:    id,
			ModelID:       modelID,
			DisplayName:   displayName,
			Enabled:       item.Enabled,
			Position:      len(normalized),
			ContextWindow: contextWindow,
		})
	}

	if err := s.repository.ReplaceCodexModels(ctx, id, normalized); err != nil {
		return nil, err
	}

	return normalized, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	item, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if item.IsSystemManaged && !item.IsDeletable {
		return ErrProviderNotDeletable
	}

	if err := s.credentials.Delete(ctx, item.APIKeyRef); err != nil {
		return err
	}

	return s.repository.Delete(ctx, id)
}

func (s *Service) EnsureManagedLocalGateway(ctx context.Context, name string, baseURL string, apiKey string) (Provider, error) {
	items, err := s.repository.List(ctx)
	if err != nil {
		return Provider{}, err
	}

	for _, item := range items {
		if !item.IsSystemManaged || item.RuntimeKind != RuntimeKindManagedLocalGate {
			continue
		}

		item.Name = strings.TrimSpace(name)
		item.BaseURL = strings.TrimSpace(baseURL)
		item.AuthMode = AuthModeBearer
		item.ExtraHeaders = map[string]string{}
		item.IsSystemManaged = true
		item.IsEditable = false
		item.IsDeletable = false
		item.RuntimeKind = RuntimeKindManagedLocalGate
		item.Capabilities = Capabilities{
			SupportsOpenAICompatible:    true,
			SupportsAnthropicCompatible: true,
			SupportsModelsAPI:           true,
			SupportsBalanceAPI:          false,
			SupportsStream:              true,
		}

		if strings.TrimSpace(apiKey) != "" {
			if err := s.credentials.Delete(ctx, item.APIKeyRef); err != nil {
				return Provider{}, err
			}

			apiKeyRef, err := s.credentials.Save(ctx, fmt.Sprintf("provider/%s/api-key", item.ID), apiKey)
			if err != nil {
				return Provider{}, err
			}

			item.APIKeyRef = apiKeyRef
			item.APIKeyMasked = maskAPIKey(apiKey)
		}

		return s.repository.Update(ctx, normalizeProviderManagement(item))
	}

	id := "provider-local-gateway"
	apiKeyRef, err := s.credentials.Save(ctx, fmt.Sprintf("provider/%s/api-key", id), apiKey)
	if err != nil {
		return Provider{}, err
	}

	item := normalizeProviderManagement(Provider{
		ID:           id,
		Name:         strings.TrimSpace(name),
		BaseURL:      strings.TrimSpace(baseURL),
		APIKeyRef:    apiKeyRef,
		APIKey:       apiKey,
		AuthMode:     AuthModeBearer,
		ExtraHeaders: map[string]string{},
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
		APIKeyMasked:    maskAPIKey(apiKey),
		IsSystemManaged: true,
		IsEditable:      false,
		IsDeletable:     false,
		RuntimeKind:     RuntimeKindManagedLocalGate,
	})
	item.Status.LastHealthcheckAt = time.Now().UTC().Format(time.RFC3339)

	return s.repository.Create(ctx, item)
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
	target.Path = ResolveModelsPath(baseURL.Path, item.ModelsPath)
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
	if err := json.Unmarshal(body, &openAIResponse); err == nil && openAIResponse.Data != nil {
		return openAIResponse.Data, nil
	}

	var stringList []string
	if err := json.Unmarshal(body, &stringList); err == nil && stringList != nil {
		items := make([]ModelInfo, 0, len(stringList))
		for _, id := range stringList {
			items = append(items, ModelInfo{ID: id})
		}
		return items, nil
	}

	var wrappedStrings struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(body, &wrappedStrings); err == nil && wrappedStrings.Data != nil {
		items := make([]ModelInfo, 0, len(wrappedStrings.Data))
		for _, id := range wrappedStrings.Data {
			items = append(items, ModelInfo{ID: id})
		}
		return items, nil
	}

	return nil, fmt.Errorf("models response format not recognized")
}

func (s *Service) TestModelAvailability(ctx context.Context, id string, modelID string) (*ModelTestResult, error) {
	item, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	trimmedModelID := strings.TrimSpace(modelID)
	if trimmedModelID == "" {
		return nil, fmt.Errorf("model_id is required")
	}

	baseURL, err := url.Parse(item.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid provider base_url: %w", err)
	}

	apiKey, err := s.credentials.Get(ctx, item.APIKeyRef)
	if err != nil {
		return nil, fmt.Errorf("load provider credential: %w", err)
	}

	protocols := modelTestProtocolOrder(*item, trimmedModelID)
	var lastResult *ModelTestResult
	for index, protocol := range protocols {
		result := s.runModelAvailabilityRequest(ctx, *item, *baseURL, apiKey, trimmedModelID, protocol)
		if result.Status == "ok" {
			return result, nil
		}
		lastResult = result
		if index < len(protocols)-1 && shouldRetryModelTestWithAlternateProtocol(result.StatusCode) {
			continue
		}
		break
	}

	return lastResult, nil
}

func (s *Service) runModelAvailabilityRequest(
	ctx context.Context,
	item Provider,
	baseURL url.URL,
	apiKey string,
	modelID string,
	protocol string,
) *ModelTestResult {
	target := baseURL
	target.Path = resolveModelTestPath(baseURL.Path, protocol)
	target.RawPath = target.Path

	payload := map[string]any{
		"model":      modelID,
		"max_tokens": 1,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": "hi",
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return modelTestErrorResult(item, modelID, protocol, target.Path, 0, 0, err.Error())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		return modelTestErrorResult(item, modelID, protocol, target.Path, 0, 0, err.Error())
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(body))
	ApplyCredentialHeaders(req, item, apiKey, nil)
	if protocol == "anthropic-compatible" && req.Header.Get("anthropic-version") == "" {
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	startedAt := time.Now()
	resp, err := s.client.Do(req)
	latencyMs := time.Since(startedAt).Milliseconds()
	if err != nil {
		return modelTestErrorResult(item, modelID, protocol, target.Path, 0, latencyMs, err.Error())
	}
	defer resp.Body.Close()

	bodySnippet := ""
	if responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 512)); readErr == nil {
		bodySnippet = strings.TrimSpace(string(responseBody))
	}

	status := "ok"
	summary := fmt.Sprintf("HTTP %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		status = "error"
		if bodySnippet != "" {
			summary = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, bodySnippet)
		}
	}

	return &ModelTestResult{
		ModelID:     modelID,
		Status:      status,
		StatusCode:  resp.StatusCode,
		LatencyMs:   latencyMs,
		Summary:     summary,
		CheckedAt:   time.Now().UTC().Format(time.RFC3339),
		ProviderID:  item.ID,
		ProviderURL: item.BaseURL,
		Protocol:    protocol,
		RequestPath: target.Path,
	}
}

func modelTestErrorResult(
	item Provider,
	modelID string,
	protocol string,
	requestPath string,
	statusCode int,
	latencyMs int64,
	summary string,
) *ModelTestResult {
	return &ModelTestResult{
		ModelID:     modelID,
		Status:      "error",
		StatusCode:  statusCode,
		LatencyMs:   latencyMs,
		Summary:     summary,
		CheckedAt:   time.Now().UTC().Format(time.RFC3339),
		ProviderID:  item.ID,
		ProviderURL: item.BaseURL,
		Protocol:    protocol,
		RequestPath: requestPath,
	}
}

func modelTestProtocolOrder(item Provider, modelID string) []string {
	openAI := item.Capabilities.SupportsOpenAICompatible
	anthropic := item.Capabilities.SupportsAnthropicCompatible

	switch {
	case openAI && !anthropic:
		return []string{"openai-compatible"}
	case anthropic && !openAI:
		return []string{"anthropic-compatible"}
	case item.AuthMode == AuthModeBearer:
		return []string{"openai-compatible", "anthropic-compatible"}
	case item.AuthMode == AuthModeAPIKey:
		return []string{"anthropic-compatible", "openai-compatible"}
	case looksLikeAnthropicModel(modelID):
		return []string{"anthropic-compatible", "openai-compatible"}
	default:
		return []string{"openai-compatible", "anthropic-compatible"}
	}
}

func looksLikeAnthropicModel(modelID string) bool {
	normalized := strings.ToLower(strings.TrimSpace(modelID))
	return strings.Contains(normalized, "claude") ||
		strings.Contains(normalized, "anthropic") ||
		strings.Contains(normalized, "sonnet") ||
		strings.Contains(normalized, "opus") ||
		strings.Contains(normalized, "haiku")
}

func shouldRetryModelTestWithAlternateProtocol(statusCode int) bool {
	switch statusCode {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusUnsupportedMediaType, http.StatusUnprocessableEntity:
		return true
	default:
		return false
	}
}

func resolveModelTestPath(basePath string, protocol string) string {
	switch protocol {
	case "anthropic-compatible":
		return resolveProviderV1Path(basePath, "/messages")
	default:
		return resolveProviderV1Path(basePath, "/chat/completions")
	}
}

func resolveProviderV1Path(basePath string, requestPath string) string {
	trimmed := strings.TrimRight(basePath, "/")
	suffix := strings.TrimPrefix(requestPath, "/")

	switch {
	case trimmed == "":
		return "/v1/" + suffix
	case strings.HasSuffix(trimmed, "/v1"):
		return trimmed + "/" + suffix
	default:
		return trimmed + "/v1/" + suffix
	}
}

func ResolveModelsPath(basePath string, modelsPath ...string) string {
	trimmed := strings.TrimRight(basePath, "/")
	if len(modelsPath) > 0 {
		override := normalizeModelsPath(modelsPath[0])
		if override != "" {
			if trimmed != "" && (override == trimmed || strings.HasPrefix(override, trimmed+"/")) {
				return override
			}
			return joinURLPath(trimmed, override)
		}
	}

	switch {
	case trimmed == "":
		return "/v1/models"
	case strings.HasSuffix(trimmed, "/v1"):
		return trimmed + "/models"
	default:
		return trimmed + "/v1/models"
	}
}

func normalizeModelsPath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	return "/" + trimmed
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
