package tooling

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xiaoyuandev/relay-switch/core/internal/provider"
)

const (
	codexDefaultModelsURL           = "https://raw.githubusercontent.com/openai/codex/main/codex-rs/models-manager/models.json"
	codexDefaultModelsCacheFilename = "codex-models.json"
	codexRelaySwitchModelsFilename  = "relay-switch-models.json"
	codexRelaySwitchCatalogFilename = "relay-switch-model-catalog.json"
	codexHideOfficialModelsKey      = "relay_switch_hide_official_models"
	defaultCodexContextWindow       = 128000
)

//go:embed assets/codex-models.json
var embeddedCodexModelsJSON []byte

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func httpClientWithTimeout(timeout time.Duration) httpClient {
	return &http.Client{Timeout: timeout}
}

type codexModelsCatalog struct {
	Models []map[string]any `json:"models"`
}

type CodexModelCatalogState struct {
	Enabled            bool   `json:"enabled"`
	CatalogPath        string `json:"catalog_path"`
	HideOfficialModels bool   `json:"hide_official_models"`
}

func (s *Service) BootstrapCodexModelCatalog(ctx context.Context) {
	if err := s.SyncCodexModelCatalog(ctx); err != nil {
		log.Printf("[codex] initial model catalog sync failed: %v", err)
	}

	go func() {
		refreshCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.RefreshCodexDefaultModelsCatalog(refreshCtx); err != nil {
			log.Printf("[codex] refresh default model catalog failed: %v", err)
			return
		}
		if err := s.SyncCodexModelCatalog(context.Background()); err != nil {
			log.Printf("[codex] model catalog sync after refresh failed: %v", err)
		}
	}()
}

func (s *Service) RefreshCodexDefaultModelsCatalog(ctx context.Context) error {
	if s.catalogClient == nil {
		s.catalogClient = httpClientWithTimeout(8 * time.Second)
	}
	if strings.TrimSpace(s.catalogURL) == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.catalogURL, nil)
	if err != nil {
		return fmt.Errorf("build default codex models request: %w", err)
	}

	resp, err := s.catalogClient.Do(req)
	if err != nil {
		return fmt.Errorf("request default codex models: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return fmt.Errorf("read default codex models: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("default codex models request failed: HTTP %d", resp.StatusCode)
	}
	if _, err := decodeCodexModelsCatalog(body); err != nil {
		return fmt.Errorf("validate default codex models: %w", err)
	}

	cachePath := s.codexDefaultModelsCachePath()
	if err := ensureDir(cachePath); err != nil {
		return err
	}
	if err := os.WriteFile(cachePath, append(bytes.TrimSpace(body), '\n'), 0o644); err != nil {
		return fmt.Errorf("write default codex models cache: %w", err)
	}
	return nil
}

func (s *Service) SyncCodexModelCatalog(ctx context.Context) error {
	hideOfficialModels, err := s.codexHideOfficialModels()
	if err != nil {
		return err
	}
	return s.syncCodexModelCatalog(ctx, hideOfficialModels)
}

func (s *Service) syncCodexModelCatalog(ctx context.Context, hideOfficialModels bool) error {
	defaultCatalog, err := s.loadCodexDefaultModelsCatalog()
	if err != nil {
		return err
	}

	activeModels, err := s.activeCodexModels(ctx)
	if err != nil {
		return err
	}

	relayModels := buildRelaySwitchCodexModels(activeModels)
	catalogModels := make([]map[string]any, 0, len(defaultCatalog.Models)+len(relayModels))
	catalogModels = append(catalogModels, cloneCodexModelMaps(defaultCatalog.Models)...)
	if hideOfficialModels {
		hideCodexModelMaps(catalogModels)
	}
	catalogModels = append(catalogModels, relayModels...)

	codexDir := filepath.Dir(codexConfigPath())
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		return fmt.Errorf("create codex dir: %w", err)
	}

	if err := writeCodexCatalogFile(codexRelaySwitchModelsPath(), codexModelsCatalog{Models: relayModels}); err != nil {
		return err
	}
	if err := writeCodexCatalogFile(codexRelaySwitchCatalogPath(), codexModelsCatalog{Models: catalogModels}); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetCodexModelCatalogState() (CodexModelCatalogState, error) {
	content, err := readOptionalText(codexConfigPath())
	if err != nil {
		return CodexModelCatalogState{}, err
	}

	catalogPath := codexRelaySwitchCatalogPath()
	return CodexModelCatalogState{
		Enabled:            readTopLevelTomlValue(content, "model_catalog_json") == catalogPath,
		CatalogPath:        catalogPath,
		HideOfficialModels: readTopLevelTomlBool(content, codexHideOfficialModelsKey),
	}, nil
}

func (s *Service) SetCodexModelCatalogEnabled(ctx context.Context, enabled bool) (CodexModelCatalogState, error) {
	return s.UpdateCodexModelCatalogState(ctx, &enabled, nil)
}

func (s *Service) UpdateCodexModelCatalogState(ctx context.Context, enabled *bool, hideOfficialModels *bool) (CodexModelCatalogState, error) {
	content, err := readOptionalText(codexConfigPath())
	if err != nil {
		return CodexModelCatalogState{}, err
	}

	desiredHideOfficialModels := readTopLevelTomlBool(content, codexHideOfficialModelsKey)
	if hideOfficialModels != nil {
		desiredHideOfficialModels = *hideOfficialModels
	}
	if hideOfficialModels != nil || (enabled != nil && *enabled) {
		if err := s.syncCodexModelCatalog(ctx, desiredHideOfficialModels); err != nil {
			return CodexModelCatalogState{}, err
		}
	}

	nextContent := content
	if hideOfficialModels != nil {
		nextContent = removeTopLevelTomlKey(nextContent, codexHideOfficialModelsKey)
		if *hideOfficialModels {
			nextContent = setTopLevelTomlRaw(nextContent, codexHideOfficialModelsKey, "true")
		}
	}

	if enabled != nil {
		nextContent = removeTopLevelTomlKey(nextContent, "model_catalog_json")
		if *enabled {
			nextContent = setTopLevelTomlString(nextContent, "model_catalog_json", codexRelaySwitchCatalogPath())
		}
	}

	if nextContent != content {
		if err := ensureDir(codexConfigPath()); err != nil {
			return CodexModelCatalogState{}, err
		}
		if err := os.WriteFile(codexConfigPath(), []byte(nextContent), 0o644); err != nil {
			return CodexModelCatalogState{}, fmt.Errorf("write codex config model_catalog_json: %w", err)
		}
	}

	return s.GetCodexModelCatalogState()
}

func (s *Service) codexHideOfficialModels() (bool, error) {
	content, err := readOptionalText(codexConfigPath())
	if err != nil {
		return false, err
	}
	return readTopLevelTomlBool(content, codexHideOfficialModelsKey), nil
}

func (s *Service) activeCodexModels(ctx context.Context) ([]provider.CodexModel, error) {
	if s.providers == nil {
		return []provider.CodexModel{}, nil
	}
	active, err := s.providers.GetActive(ctx)
	if err != nil || active == nil {
		return []provider.CodexModel{}, err
	}
	return s.providers.ListCodexModels(ctx, active.ID)
}

func (s *Service) loadCodexDefaultModelsCatalog() (codexModelsCatalog, error) {
	if content, err := os.ReadFile(s.codexDefaultModelsCachePath()); err == nil {
		if catalog, decodeErr := decodeCodexModelsCatalog(content); decodeErr == nil {
			return catalog, nil
		}
	}
	return decodeCodexModelsCatalog(embeddedCodexModelsJSON)
}

func (s *Service) codexDefaultModelsCachePath() string {
	if strings.TrimSpace(s.dataDir) == "" {
		return filepath.Join(currentRuntime().HomeDir, ".relay-switch", codexDefaultModelsCacheFilename)
	}
	return filepath.Join(s.dataDir, codexDefaultModelsCacheFilename)
}

func decodeCodexModelsCatalog(content []byte) (codexModelsCatalog, error) {
	var catalog codexModelsCatalog
	if err := json.Unmarshal(content, &catalog); err != nil {
		return codexModelsCatalog{}, err
	}
	if catalog.Models == nil {
		return codexModelsCatalog{}, fmt.Errorf("models array is required")
	}
	return catalog, nil
}

func writeCodexCatalogFile(filePath string, catalog codexModelsCatalog) error {
	content, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal codex catalog: %w", err)
	}
	if err := ensureDir(filePath); err != nil {
		return err
	}
	if err := os.WriteFile(filePath, append(content, '\n'), 0o644); err != nil {
		return fmt.Errorf("write codex catalog %s: %w", filePath, err)
	}
	return nil
}

func codexRelaySwitchModelsPath() string {
	return filepath.Join(currentRuntime().HomeDir, ".codex", codexRelaySwitchModelsFilename)
}

func codexRelaySwitchCatalogPath() string {
	return filepath.Join(currentRuntime().HomeDir, ".codex", codexRelaySwitchCatalogFilename)
}

func buildRelaySwitchCodexModels(items []provider.CodexModel) []map[string]any {
	models := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		modelID := strings.TrimSpace(item.ModelID)
		if modelID == "" {
			continue
		}

		displayName := strings.TrimSpace(item.DisplayName)
		if displayName == "" {
			displayName = modelID
		}
		contextWindow := defaultCodexContextWindow
		if item.ContextWindow != nil && *item.ContextWindow > 0 {
			contextWindow = *item.ContextWindow
		}

		models = append(models, map[string]any{
			"slug":                               modelID,
			"display_name":                       displayName,
			"description":                        displayName,
			"default_reasoning_level":            "medium",
			"supported_reasoning_levels":         standardCodexReasoningLevels(),
			"shell_type":                         "shell_command",
			"visibility":                         "list",
			"supported_in_api":                   true,
			"priority":                           item.Position,
			"additional_speed_tiers":             []string{},
			"service_tiers":                      []any{},
			"availability_nux":                   nil,
			"upgrade":                            nil,
			"base_instructions":                  "You are Codex, a coding agent. Follow the system and developer instructions.",
			"model_messages":                     map[string]any{"instructions_template": "You are Codex, a coding agent. Follow the system and developer instructions.", "instructions_variables": map[string]string{}},
			"supports_reasoning_summaries":       true,
			"default_reasoning_summary":          "none",
			"support_verbosity":                  true,
			"default_verbosity":                  "low",
			"apply_patch_tool_type":              "freeform",
			"web_search_tool_type":               "text_and_image",
			"truncation_policy":                  map[string]any{"mode": "tokens", "limit": 10000},
			"supports_parallel_tool_calls":       true,
			"supports_image_detail_original":     true,
			"context_window":                     contextWindow,
			"max_context_window":                 contextWindow,
			"effective_context_window_percent":   95,
			"experimental_supported_tools":       []any{},
			"input_modalities":                   []string{"text"},
			"supports_search_tool":               false,
			"auto_compact_token_limit":           nil,
			"reasoning_summary_format":           "experimental",
			"prefer_websockets":                  true,
			"supports_reasoning_summary_options": false,
		})
	}
	return models
}

func standardCodexReasoningLevels() []map[string]string {
	return []map[string]string{
		{"effort": "low", "description": "Fast responses with lighter reasoning"},
		{"effort": "medium", "description": "Balances speed and reasoning depth for everyday tasks"},
		{"effort": "high", "description": "Greater reasoning depth for complex problems"},
	}
}

func cloneCodexModelMaps(items []map[string]any) []map[string]any {
	cloned := make([]map[string]any, 0, len(items))
	for _, item := range items {
		next := make(map[string]any, len(item))
		for key, value := range item {
			next[key] = value
		}
		cloned = append(cloned, next)
	}
	return cloned
}

func hideCodexModelMaps(items []map[string]any) {
	for _, item := range items {
		item["visibility"] = "hide"
	}
}

func setTopLevelTomlString(content string, key string, value string) string {
	return setTopLevelTomlRaw(content, key, tomlQuote(value))
}

func setTopLevelTomlRaw(content string, key string, value string) string {
	lines := []string{}
	if strings.TrimSpace(content) != "" {
		lines = strings.Split(strings.TrimRight(content, "\n\r\t "), "\n")
	}

	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			break
		}
		if isTomlAssignment(trimmed, key) {
			lines[index] = key + " = " + value
			return strings.TrimRight(strings.Join(lines, "\n"), "\n\r\t ") + "\n"
		}
	}

	insertAt := len(lines)
	for index, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			insertAt = index
			break
		}
	}
	nextLine := key + " = " + value
	if insertAt == len(lines) {
		lines = append(lines, nextLine)
	} else {
		lines = append(lines[:insertAt], append([]string{nextLine, ""}, lines[insertAt:]...)...)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n\r\t ") + "\n"
}

func removeTopLevelTomlKey(content string, key string) string {
	lines := []string{}
	if strings.TrimSpace(content) != "" {
		lines = strings.Split(strings.TrimRight(content, "\n\r\t "), "\n")
	}

	removed := false
	inTopLevel := true
	next := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inTopLevel = false
			next = append(next, line)
			continue
		}
		if inTopLevel && trimmed != "" && !strings.HasPrefix(trimmed, "#") && isTomlAssignment(trimmed, key) {
			removed = true
			continue
		}
		next = append(next, line)
	}
	if !removed {
		return content
	}
	if len(next) == 0 {
		return ""
	}
	return strings.TrimRight(strings.Join(next, "\n"), "\n\r\t ") + "\n"
}

func tomlQuote(value string) string {
	content, _ := json.Marshal(value)
	return string(content)
}

func isTomlAssignment(trimmedLine string, key string) bool {
	if !strings.HasPrefix(trimmedLine, key) {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(trimmedLine[len(key):]), "=")
}

func readTopLevelTomlBool(content string, key string) bool {
	currentTable := ""
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentTable = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			continue
		}
		if currentTable != "" || trimmed == "" || strings.HasPrefix(trimmed, "#") || !isTomlAssignment(trimmed, key) {
			continue
		}
		rawValue := strings.TrimSpace(trimmed[len(key):])
		rawValue = strings.TrimSpace(strings.TrimPrefix(rawValue, "="))
		fields := strings.Fields(rawValue)
		return len(fields) > 0 && fields[0] == "true"
	}
	return false
}
