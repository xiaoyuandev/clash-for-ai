package localgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xiaoyuandev/relay-switch/core/internal/credential"
	upstreamprovider "github.com/xiaoyuandev/relay-switch/core/internal/provider"
)

type CreateModelSourceInput struct {
	Name            string   `json:"name"`
	BaseURL         string   `json:"base_url"`
	APIKey          string   `json:"api_key"`
	ProviderType    string   `json:"provider_type"`
	DefaultModelID  string   `json:"default_model_id"`
	ExposedModelIDs []string `json:"exposed_model_ids"`
	Enabled         bool     `json:"enabled"`
	Position        int      `json:"position"`
}

type UpdateModelSourceInput struct {
	Name            string   `json:"name"`
	BaseURL         string   `json:"base_url"`
	APIKey          string   `json:"api_key"`
	ProviderType    string   `json:"provider_type"`
	DefaultModelID  string   `json:"default_model_id"`
	ExposedModelIDs []string `json:"exposed_model_ids"`
	Enabled         bool     `json:"enabled"`
	Position        int      `json:"position"`
}

type PreviewModelSourceInput struct {
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	ProviderType string `json:"provider_type"`
}

type SourceModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object,omitempty"`
	OwnedBy string `json:"owned_by,omitempty"`
}

type Service struct {
	repository  Repository
	credentials credential.Store
	client      *http.Client
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

func (s *Service) ListSources(ctx context.Context) ([]ModelSource, error) {
	items, err := s.repository.ListSources(ctx)
	if err != nil {
		return nil, err
	}

	for index := range items {
		items[index] = s.refreshMaskedKey(ctx, items[index])
	}

	return items, nil
}

func (s *Service) GetSourceByID(ctx context.Context, id string) (*ModelSource, error) {
	item, err := s.repository.GetSourceByID(ctx, id)
	if err != nil || item == nil {
		return item, err
	}

	refreshed := s.refreshMaskedKey(ctx, *item)
	return &refreshed, nil
}

func (s *Service) CreateSource(ctx context.Context, input CreateModelSourceInput) (ModelSource, error) {
	normalizedInput, err := normalizeCreateSourceInput(input)
	if err != nil {
		return ModelSource{}, err
	}

	sources, err := s.repository.ListSources(ctx)
	if err != nil {
		return ModelSource{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	id := fmt.Sprintf("local-source-%d", time.Now().UnixNano())
	apiKeyRef, err := s.credentials.Save(ctx, fmt.Sprintf("local-gateway/%s/api-key", id), normalizedInput.APIKey)
	if err != nil {
		return ModelSource{}, err
	}

	item := ModelSource{
		ID:              id,
		Name:            normalizedInput.Name,
		BaseURL:         normalizedInput.BaseURL,
		APIKeyRef:       apiKeyRef,
		APIKey:          normalizedInput.APIKey,
		ProviderType:    normalizedInput.ProviderType,
		DefaultModelID:  normalizedInput.DefaultModelID,
		ExposedModelIDs: normalizedInput.ExposedModelIDs,
		Enabled:         normalizedInput.Enabled,
		Position:        len(sources),
		APIKeyMasked:    maskAPIKey(normalizedInput.APIKey),
		LastSyncStatus:  SourceSyncStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	return s.repository.CreateSource(ctx, item)
}

func (s *Service) UpdateSource(ctx context.Context, id string, input UpdateModelSourceInput) (ModelSource, error) {
	item, err := s.repository.GetSourceByID(ctx, id)
	if err != nil {
		return ModelSource{}, err
	}

	normalizedInput, err := normalizeUpdateSourceInput(input)
	if err != nil {
		return ModelSource{}, err
	}

	item.Name = normalizedInput.Name
	item.BaseURL = normalizedInput.BaseURL
	item.ProviderType = normalizedInput.ProviderType
	item.DefaultModelID = normalizedInput.DefaultModelID
	item.ExposedModelIDs = normalizedInput.ExposedModelIDs
	item.Enabled = normalizedInput.Enabled
	item.LastSyncStatus = SourceSyncStatusPending
	item.LastSyncError = ""
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if normalizedInput.APIKey != "" {
		if err := s.credentials.Delete(ctx, item.APIKeyRef); err != nil {
			return ModelSource{}, err
		}

		apiKeyRef, err := s.credentials.Save(ctx, fmt.Sprintf("local-gateway/%s/api-key", id), normalizedInput.APIKey)
		if err != nil {
			return ModelSource{}, err
		}

		item.APIKeyRef = apiKeyRef
		item.APIKey = normalizedInput.APIKey
		item.APIKeyMasked = maskAPIKey(normalizedInput.APIKey)
	}

	updated, err := s.repository.UpdateSource(ctx, *item)
	if err != nil {
		return ModelSource{}, err
	}

	if _, err := s.removeInvalidSelectedModels(ctx); err != nil {
		return ModelSource{}, err
	}

	return updated, nil
}

func (s *Service) DeleteSource(ctx context.Context, id string) error {
	item, err := s.repository.GetSourceByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.credentials.Delete(ctx, item.APIKeyRef); err != nil {
		return err
	}

	if err := s.repository.DeleteSource(ctx, id); err != nil {
		return err
	}
	if err := s.repository.NormalizeSourcePositions(ctx); err != nil {
		return err
	}

	_, err = s.removeInvalidSelectedModels(ctx)
	return err
}

func (s *Service) ListSelectedModels(ctx context.Context) ([]SelectedModel, error) {
	return s.repository.ListSelectedModels(ctx)
}

func (s *Service) BuildSyncInput(ctx context.Context) (SyncInput, error) {
	sources, err := s.repository.ListSources(ctx)
	if err != nil {
		return SyncInput{}, err
	}

	resolvedSources := make([]SyncModelSource, 0, len(sources))
	for _, source := range sources {
		apiKey, err := s.credentials.Get(ctx, source.APIKeyRef)
		if err != nil {
			return SyncInput{}, err
		}

		resolvedSources = append(resolvedSources, SyncModelSource{
			ID:              source.ID,
			ExternalID:      source.ID,
			Name:            source.Name,
			BaseURL:         source.BaseURL,
			APIKey:          apiKey,
			ProviderType:    source.ProviderType,
			DefaultModelID:  source.DefaultModelID,
			ExposedModelIDs: append([]string(nil), source.ExposedModelIDs...),
			Enabled:         source.Enabled,
			Position:        source.Position,
		})
	}

	input := SyncInput{
		Sources:        resolvedSources,
		SelectedModels: []SelectedModel{},
	}

	if err := s.ValidateSyncInput(input); err != nil {
		return SyncInput{}, err
	}

	return input, nil
}

func (s *Service) PreviewSourceModels(ctx context.Context, input PreviewModelSourceInput) ([]SourceModelInfo, error) {
	normalizedInput, err := normalizePreviewSourceInput(input)
	if err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(normalizedInput.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid source base_url: %w", err)
	}

	target := *baseURL
	target.Path = upstreamprovider.ResolveModelsPath(baseURL.Path)
	target.RawPath = target.Path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build source models request: %w", err)
	}

	upstreamprovider.ApplyCredentialHeaders(req, upstreamprovider.Provider{
		Name:         normalizedInput.ProviderType,
		BaseURL:      normalizedInput.BaseURL,
		AuthMode:     authModeForSourceProviderType(normalizedInput.ProviderType),
		ExtraHeaders: map[string]string{},
	}, normalizedInput.APIKey, nil)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request source models: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read source models response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("source models request failed: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	items, err := parseSourceModelsResponse(body, normalizedInput.ProviderType)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) ValidateSyncInput(input SyncInput) error {
	availableModels := make(map[string]struct{})
	for _, source := range input.Sources {
		if err := validateSyncSource(source); err != nil {
			return err
		}
		if !source.Enabled {
			continue
		}

		if modelID := strings.TrimSpace(source.DefaultModelID); modelID != "" {
			availableModels[modelID] = struct{}{}
		}
		for _, modelID := range normalizeModelIDs(source.ExposedModelIDs) {
			availableModels[modelID] = struct{}{}
		}
	}

	for _, item := range input.SelectedModels {
		modelID := strings.TrimSpace(item.ModelID)
		if modelID == "" {
			return fmt.Errorf("selected model id is required")
		}
		if _, ok := availableModels[modelID]; !ok {
			return fmt.Errorf("selected model %q is not available from enabled sources", modelID)
		}
	}

	return nil
}

func (s *Service) ReplaceSelectedModels(ctx context.Context, items []SelectedModel) ([]SelectedModel, error) {
	availableModels, err := s.availableModelSet(ctx)
	if err != nil {
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
		if _, ok := availableModels[modelID]; !ok {
			return nil, fmt.Errorf("selected model %q is not available from enabled sources", modelID)
		}
		seen[modelID] = struct{}{}
		normalized = append(normalized, SelectedModel{
			ModelID:  modelID,
			Position: len(normalized),
		})
	}

	if err := s.repository.ReplaceSelectedModels(ctx, normalized); err != nil {
		return nil, err
	}

	return s.repository.ListSelectedModels(ctx)
}

func (s *Service) UpdateAllSourcesSyncState(ctx context.Context, status string, syncError string) error {
	items, err := s.repository.ListSources(ctx)
	if err != nil {
		return err
	}

	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return s.repository.UpdateSourcesSyncState(ctx, ids, status, syncError)
}

func (s *Service) refreshMaskedKey(ctx context.Context, item ModelSource) ModelSource {
	if item.APIKeyRef == "" {
		return item
	}

	value, err := s.credentials.Get(ctx, item.APIKeyRef)
	if err != nil {
		return item
	}

	item.APIKey = value
	if item.APIKeyMasked == "" {
		item.APIKeyMasked = maskAPIKey(value)
	}
	return item
}

func normalizeModelIDs(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}

	normalized := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	return normalized
}

func maskAPIKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 4 {
		return "****"
	}
	if len(trimmed) <= 12 {
		return trimmed[:len(trimmed)-4] + "****"
	}

	return trimmed[:8] + "****" + trimmed[len(trimmed)-4:]
}

func validateSyncSource(source SyncModelSource) error {
	if strings.TrimSpace(source.Name) == "" {
		return fmt.Errorf("source name is required")
	}
	if strings.TrimSpace(source.ProviderType) == "" {
		return fmt.Errorf("source provider_type is required")
	}
	if strings.TrimSpace(source.DefaultModelID) == "" {
		return fmt.Errorf("source default_model_id is required")
	}

	parsed, err := url.Parse(strings.TrimSpace(source.BaseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("source base_url must be a valid absolute URL")
	}

	switch strings.TrimSpace(source.ProviderType) {
	case "openai-compatible", "anthropic-compatible":
		return nil
	default:
		return fmt.Errorf("source provider_type %q is not supported", source.ProviderType)
	}
}

func normalizePreviewSourceInput(input PreviewModelSourceInput) (PreviewModelSourceInput, error) {
	normalized := PreviewModelSourceInput{
		BaseURL:      strings.TrimSpace(input.BaseURL),
		APIKey:       strings.TrimSpace(input.APIKey),
		ProviderType: strings.TrimSpace(input.ProviderType),
	}
	if normalized.APIKey == "" {
		return PreviewModelSourceInput{}, fmt.Errorf("source api_key is required")
	}
	if err := validateSourceFields("preview", normalized.BaseURL, normalized.ProviderType, "preview"); err != nil {
		return PreviewModelSourceInput{}, err
	}
	return normalized, nil
}

func normalizeCreateSourceInput(input CreateModelSourceInput) (CreateModelSourceInput, error) {
	normalized := CreateModelSourceInput{
		Name:            strings.TrimSpace(input.Name),
		BaseURL:         strings.TrimSpace(input.BaseURL),
		APIKey:          strings.TrimSpace(input.APIKey),
		ProviderType:    strings.TrimSpace(input.ProviderType),
		DefaultModelID:  strings.TrimSpace(input.DefaultModelID),
		ExposedModelIDs: normalizeModelIDs(input.ExposedModelIDs),
		Enabled:         input.Enabled,
	}
	if err := validateSourceFields(normalized.Name, normalized.BaseURL, normalized.ProviderType, normalized.DefaultModelID); err != nil {
		return CreateModelSourceInput{}, err
	}
	return normalized, nil
}

func normalizeUpdateSourceInput(input UpdateModelSourceInput) (UpdateModelSourceInput, error) {
	normalized := UpdateModelSourceInput{
		Name:            strings.TrimSpace(input.Name),
		BaseURL:         strings.TrimSpace(input.BaseURL),
		APIKey:          strings.TrimSpace(input.APIKey),
		ProviderType:    strings.TrimSpace(input.ProviderType),
		DefaultModelID:  strings.TrimSpace(input.DefaultModelID),
		ExposedModelIDs: normalizeModelIDs(input.ExposedModelIDs),
		Enabled:         input.Enabled,
	}
	if err := validateSourceFields(normalized.Name, normalized.BaseURL, normalized.ProviderType, normalized.DefaultModelID); err != nil {
		return UpdateModelSourceInput{}, err
	}
	return normalized, nil
}

func validateSourceFields(name string, baseURL string, providerType string, defaultModelID string) error {
	if name == "" {
		return fmt.Errorf("source name is required")
	}
	if providerType == "" {
		return fmt.Errorf("source provider_type is required")
	}
	if defaultModelID == "" {
		return fmt.Errorf("source default_model_id is required")
	}

	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("source base_url must be a valid absolute URL")
	}

	switch providerType {
	case "openai-compatible", "anthropic-compatible":
		return nil
	default:
		return fmt.Errorf("source provider_type %q is not supported", providerType)
	}
}

func authModeForSourceProviderType(providerType string) upstreamprovider.AuthMode {
	switch strings.TrimSpace(providerType) {
	case "anthropic-compatible":
		return upstreamprovider.AuthModeAPIKey
	default:
		return upstreamprovider.AuthModeBearer
	}
}

func parseSourceModelsResponse(body []byte, providerType string) ([]SourceModelInfo, error) {
	var openAIResponse struct {
		Data []SourceModelInfo `json:"data"`
	}
	if err := json.Unmarshal(body, &openAIResponse); err == nil && len(openAIResponse.Data) > 0 {
		for index := range openAIResponse.Data {
			if openAIResponse.Data[index].OwnedBy == "" {
				openAIResponse.Data[index].OwnedBy = providerType
			}
		}
		return openAIResponse.Data, nil
	}

	var stringList []string
	if err := json.Unmarshal(body, &stringList); err == nil && len(stringList) > 0 {
		items := make([]SourceModelInfo, 0, len(stringList))
		for _, id := range stringList {
			items = append(items, SourceModelInfo{
				ID:      id,
				Object:  "model",
				OwnedBy: providerType,
			})
		}
		return items, nil
	}

	var wrappedStrings struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(body, &wrappedStrings); err == nil && len(wrappedStrings.Data) > 0 {
		items := make([]SourceModelInfo, 0, len(wrappedStrings.Data))
		for _, id := range wrappedStrings.Data {
			items = append(items, SourceModelInfo{
				ID:      id,
				Object:  "model",
				OwnedBy: providerType,
			})
		}
		return items, nil
	}

	return nil, fmt.Errorf("source models response format not recognized")
}

func (s *Service) removeInvalidSelectedModels(ctx context.Context) ([]SelectedModel, error) {
	availableModels, err := s.availableModelSet(ctx)
	if err != nil {
		return nil, err
	}

	selectedModels, err := s.repository.ListSelectedModels(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]SelectedModel, 0, len(selectedModels))
	for _, item := range selectedModels {
		if _, ok := availableModels[item.ModelID]; !ok {
			continue
		}
		filtered = append(filtered, SelectedModel{
			ModelID:  item.ModelID,
			Position: len(filtered),
		})
	}

	if len(filtered) == len(selectedModels) {
		return selectedModels, nil
	}
	if err := s.repository.ReplaceSelectedModels(ctx, filtered); err != nil {
		return nil, err
	}
	return filtered, nil
}

func (s *Service) availableModelSet(ctx context.Context) (map[string]struct{}, error) {
	sources, err := s.repository.ListSources(ctx)
	if err != nil {
		return nil, err
	}

	availableModels := make(map[string]struct{})
	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		if modelID := strings.TrimSpace(source.DefaultModelID); modelID != "" {
			availableModels[modelID] = struct{}{}
		}
		for _, modelID := range normalizeModelIDs(source.ExposedModelIDs) {
			availableModels[modelID] = struct{}{}
		}
	}

	return availableModels, nil
}
