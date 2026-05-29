package modelpreset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Catalog struct {
	SchemaVersion int      `json:"schema_version"`
	UpdatedAt     string   `json:"updated_at,omitempty"`
	Presets       []Preset `json:"presets"`
}

type Preset struct {
	ID          string           `json:"id"`
	Label       string           `json:"label"`
	Description string           `json:"description,omitempty"`
	Aliases     []string         `json:"aliases,omitempty"`
	Providers   []PresetProvider `json:"providers"`
	Tags        []string         `json:"tags,omitempty"`
	Deprecated  bool             `json:"deprecated,omitempty"`
	Disabled    bool             `json:"disabled,omitempty"`
}

type PresetProvider struct {
	ID           string   `json:"id,omitempty"`
	Label        string   `json:"label,omitempty"`
	ProviderType string   `json:"provider_type"`
	BaseURL      string   `json:"base_url"`
	ModelsAPI    string   `json:"models_api,omitempty"`
	ModelIDs     []string `json:"model_ids"`
}

type CatalogResponse struct {
	SchemaVersion    int      `json:"schema_version"`
	UpdatedAt        string   `json:"updated_at,omitempty"`
	Presets          []Preset `json:"presets"`
	SourceURL        string   `json:"source_url"`
	CachePath        string   `json:"cache_path"`
	CachedAt         string   `json:"cached_at,omitempty"`
	LastRefreshError string   `json:"last_refresh_error,omitempty"`
}

type Service struct {
	cachePath string
	sourceURL string
	client    *http.Client

	mu               sync.Mutex
	lastRefreshError string
}

func NewService(cachePath string, sourceURL string) *Service {
	return &Service{
		cachePath: cachePath,
		sourceURL: strings.TrimSpace(sourceURL),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (s *Service) Get(_ context.Context) CatalogResponse {
	catalog, cachedAt, err := s.readCache()
	response := s.responseFromCatalog(catalog, cachedAt)
	if err != nil {
		response.LastRefreshError = err.Error()
	}
	if lastErr := s.getLastRefreshError(); lastErr != "" {
		response.LastRefreshError = lastErr
	}
	return response
}

func (s *Service) Refresh(ctx context.Context) (CatalogResponse, error) {
	if s.sourceURL == "" {
		err := errors.New("model presets source url is not configured")
		s.setLastRefreshError(err)
		return s.Get(ctx), err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.sourceURL, nil)
	if err != nil {
		s.setLastRefreshError(err)
		return s.Get(ctx), err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "relay-switch-core")

	resp, err := s.client.Do(req)
	if err != nil {
		s.setLastRefreshError(err)
		return s.Get(ctx), err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		s.setLastRefreshError(err)
		return s.Get(ctx), err
	}
	if resp.StatusCode >= 400 {
		err := fmt.Errorf("model presets request failed: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		s.setLastRefreshError(err)
		return s.Get(ctx), err
	}

	catalog, err := DecodeCatalog(body)
	if err != nil {
		s.setLastRefreshError(err)
		return s.Get(ctx), err
	}

	if err := s.writeCache(catalog); err != nil {
		s.setLastRefreshError(err)
		return s.Get(ctx), err
	}

	s.setLastRefreshError(nil)
	return s.Get(ctx), nil
}

func DecodeCatalog(data []byte) (Catalog, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var catalog Catalog
	if err := decoder.Decode(&catalog); err != nil {
		return Catalog{}, fmt.Errorf("decode model presets: %w", err)
	}

	return normalizeCatalog(catalog)
}

func normalizeCatalog(catalog Catalog) (Catalog, error) {
	if catalog.SchemaVersion != 1 {
		return Catalog{}, fmt.Errorf("model presets schema_version %d is not supported", catalog.SchemaVersion)
	}

	seen := make(map[string]struct{}, len(catalog.Presets))
	normalized := make([]Preset, 0, len(catalog.Presets))
	for _, preset := range catalog.Presets {
		item := Preset{
			ID:          strings.TrimSpace(preset.ID),
			Label:       strings.TrimSpace(preset.Label),
			Description: strings.TrimSpace(preset.Description),
			Aliases:     normalizeStrings(preset.Aliases),
			Providers:   normalizeProviders(preset.Providers),
			Tags:        normalizeStrings(preset.Tags),
			Deprecated:  preset.Deprecated,
			Disabled:    preset.Disabled,
		}

		if item.ID == "" {
			return Catalog{}, errors.New("model preset id is required")
		}
		if _, ok := seen[item.ID]; ok {
			return Catalog{}, fmt.Errorf("model preset id %q is duplicated", item.ID)
		}
		seen[item.ID] = struct{}{}

		if item.Label == "" {
			return Catalog{}, fmt.Errorf("model preset %q label is required", item.ID)
		}
		if len(item.Providers) == 0 {
			return Catalog{}, fmt.Errorf("model preset %q requires at least one provider", item.ID)
		}
		if err := validateProviders(item); err != nil {
			return Catalog{}, err
		}

		normalized = append(normalized, item)
	}

	return Catalog{
		SchemaVersion: catalog.SchemaVersion,
		UpdatedAt:     strings.TrimSpace(catalog.UpdatedAt),
		Presets:       normalized,
	}, nil
}

func normalizeStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(items))
	normalized := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func normalizeProviders(providers []PresetProvider) []PresetProvider {
	if len(providers) == 0 {
		return []PresetProvider{}
	}

	normalized := make([]PresetProvider, 0, len(providers))
	for _, provider := range providers {
		modelsAPI := strings.TrimSpace(provider.ModelsAPI)
		if modelsAPI == "" {
			modelsAPI = "auto"
		}
		normalized = append(normalized, PresetProvider{
			ID:           strings.TrimSpace(provider.ID),
			Label:        strings.TrimSpace(provider.Label),
			ProviderType: strings.TrimSpace(provider.ProviderType),
			BaseURL:      strings.TrimSpace(provider.BaseURL),
			ModelsAPI:    modelsAPI,
			ModelIDs:     normalizeStrings(provider.ModelIDs),
		})
	}

	return normalized
}

func validateProviders(preset Preset) error {
	seen := make(map[string]struct{}, len(preset.Providers))
	for index, provider := range preset.Providers {
		key := provider.ID
		if key == "" {
			key = fmt.Sprintf("%s:%s", provider.ProviderType, provider.BaseURL)
		}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("model preset %q provider %q is duplicated", preset.ID, key)
		}
		seen[key] = struct{}{}

		switch provider.ProviderType {
		case "openai-compatible", "anthropic-compatible":
		default:
			return fmt.Errorf("model preset %q provider %d provider_type %q is not supported", preset.ID, index, provider.ProviderType)
		}
		parsed, err := url.Parse(provider.BaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("model preset %q provider %d base_url must be a valid absolute URL", preset.ID, index)
		}
		switch provider.ModelsAPI {
		case "auto", "supported", "unsupported":
		default:
			return fmt.Errorf("model preset %q provider %d models_api %q is not supported", preset.ID, index, provider.ModelsAPI)
		}
		if provider.ModelsAPI == "unsupported" && len(provider.ModelIDs) == 0 {
			return fmt.Errorf("model preset %q provider %d requires model_ids when models_api is unsupported", preset.ID, index)
		}
	}
	return nil
}

func (s *Service) readCache() (Catalog, string, error) {
	content, err := os.ReadFile(s.cachePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return emptyCatalog(), "", nil
		}
		return emptyCatalog(), "", fmt.Errorf("read model presets cache: %w", err)
	}

	catalog, err := DecodeCatalog(content)
	if err != nil {
		return emptyCatalog(), "", err
	}

	info, err := os.Stat(s.cachePath)
	if err != nil {
		return catalog, "", nil
	}
	return catalog, info.ModTime().UTC().Format(time.RFC3339), nil
}

func (s *Service) writeCache(catalog Catalog) error {
	if err := os.MkdirAll(filepath.Dir(s.cachePath), 0o755); err != nil {
		return fmt.Errorf("create model presets cache directory: %w", err)
	}

	content, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("encode model presets cache: %w", err)
	}
	content = append(content, '\n')

	tempFile, err := os.CreateTemp(filepath.Dir(s.cachePath), ".model-presets-*.json")
	if err != nil {
		return fmt.Errorf("create model presets cache temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.Write(content); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write model presets cache temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close model presets cache temp file: %w", err)
	}
	if err := os.Rename(tempPath, s.cachePath); err != nil {
		return fmt.Errorf("replace model presets cache: %w", err)
	}
	return nil
}

func (s *Service) responseFromCatalog(catalog Catalog, cachedAt string) CatalogResponse {
	return CatalogResponse{
		SchemaVersion: catalog.SchemaVersion,
		UpdatedAt:     catalog.UpdatedAt,
		Presets:       catalog.Presets,
		SourceURL:     s.sourceURL,
		CachePath:     s.cachePath,
		CachedAt:      cachedAt,
	}
}

func (s *Service) getLastRefreshError() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastRefreshError
}

func (s *Service) setLastRefreshError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.lastRefreshError = ""
		return
	}
	s.lastRefreshError = err.Error()
}

func emptyCatalog() Catalog {
	return Catalog{
		SchemaVersion: 1,
		Presets:       []Preset{},
	}
}
