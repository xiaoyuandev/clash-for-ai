package conversation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Service struct {
	dataDir    string
	configPath string
	adapters   map[SourceID]Adapter

	mu      sync.Mutex
	running bool
	started bool
}

func NewService(dataDir string) *Service {
	service := &Service{
		dataDir:    dataDir,
		configPath: filepath.Join(dataDir, "conversation-backup-config.json"),
		adapters:   map[SourceID]Adapter{},
	}
	service.adapters[SourceCodexCLI] = newCodexAdapter()
	service.adapters[SourceClaudeCode] = newClaudeAdapter()
	return service
}

func (s *Service) Start(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runScheduledBackup(ctx)
			}
		}
	}()
}

func (s *Service) runScheduledBackup(ctx context.Context) {
	config, err := s.GetConfig(ctx)
	if err != nil || !config.Enabled || strings.TrimSpace(config.OutputDir) == "" {
		return
	}
	interval := time.Duration(config.IntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = time.Hour
	}
	if !config.LastRunAt.IsZero() && time.Since(config.LastRunAt) < interval {
		return
	}
	_, _ = s.BackupNow(ctx)
}

func (s *Service) Catalog(_ context.Context) (Catalog, error) {
	config, _ := s.GetConfig(context.Background())
	backedUp := map[string]bool{}
	if strings.TrimSpace(config.OutputDir) != "" {
		for key := range readManifest(filepath.Join(config.OutputDir, "manifest.json")) {
			backedUp[key] = true
		}
	}

	sources := make([]SourceCatalog, 0, len(s.adapters))
	for _, sourceID := range []SourceID{SourceCodexCLI, SourceClaudeCode} {
		adapter := s.adapters[sourceID]
		refs, warnings := adapter.Discover()
		projects := groupProjects(refs, backedUp)
		sources = append(sources, SourceCatalog{
			ID:       sourceID,
			Title:    adapter.Title(),
			Detected: len(refs) > 0,
			Projects: projects,
			Warnings: warnings,
		})
	}

	return Catalog{Sources: sources}, nil
}

func (s *Service) GetSession(_ context.Context, source SourceID, sessionID string) (Session, error) {
	adapter, ok := s.adapters[source]
	if !ok {
		return Session{}, ErrSourceNotFound
	}
	refs, _ := adapter.Discover()
	for _, ref := range refs {
		if ref.ID == sessionID {
			return adapter.Load(ref)
		}
	}
	return Session{}, ErrSessionNotFound
}

func (s *Service) GetConfig(_ context.Context) (BackupConfig, error) {
	config := defaultBackupConfig()
	content, err := os.ReadFile(s.configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config, nil
		}
		return config, err
	}
	if err := json.Unmarshal(content, &config); err != nil {
		return defaultBackupConfig(), err
	}
	return normalizeConfig(config), nil
}

func (s *Service) UpdateConfig(_ context.Context, input BackupConfig) (BackupConfig, error) {
	config := normalizeConfig(input)
	if config.Enabled && strings.TrimSpace(config.OutputDir) == "" {
		return BackupConfig{}, ErrOutputDirRequired
	}
	config.UpdatedAt = time.Now().UTC()

	if err := os.MkdirAll(filepath.Dir(s.configPath), 0o755); err != nil {
		return BackupConfig{}, err
	}
	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return BackupConfig{}, err
	}
	if err := atomicWriteFile(s.configPath, append(content, '\n'), 0o644); err != nil {
		return BackupConfig{}, err
	}
	return config, nil
}

func (s *Service) BackupNow(ctx context.Context) (BackupResult, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return BackupResult{}, ErrBackupAlreadyRunning
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	result := BackupResult{
		StartedAt: time.Now().UTC(),
		Warnings:  []string{},
		Errors:    []string{},
	}

	config, err := s.GetConfig(ctx)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.FinishedAt = time.Now().UTC()
		return result, err
	}
	if strings.TrimSpace(config.OutputDir) == "" {
		result.Errors = append(result.Errors, ErrOutputDirRequired.Error())
		result.FinishedAt = time.Now().UTC()
		return result, ErrOutputDirRequired
	}

	if err := os.MkdirAll(config.OutputDir, 0o755); err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.FinishedAt = time.Now().UTC()
		return result, err
	}

	manifestPath := filepath.Join(config.OutputDir, "manifest.json")
	manifest := readManifest(manifestPath)
	selected := selectedSources(config.Sources)

	for _, sourceID := range []SourceID{SourceCodexCLI, SourceClaudeCode} {
		if !selected[sourceID] {
			continue
		}
		adapter := s.adapters[sourceID]
		refs, warnings := adapter.Discover()
		result.Warnings = append(result.Warnings, warnings...)
		for _, ref := range refs {
			if ctx.Err() != nil {
				result.Errors = append(result.Errors, ctx.Err().Error())
				result.FinishedAt = time.Now().UTC()
				return result, ctx.Err()
			}
			written, warnings, err := s.backupSession(adapter, ref, config, manifest)
			result.Warnings = append(result.Warnings, warnings...)
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
				continue
			}
			if written {
				result.Exported++
			} else {
				result.Skipped++
			}
		}
	}

	if err := writeManifest(manifestPath, manifest); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}

	config.LastRunAt = time.Now().UTC()
	config.LastError = strings.Join(result.Errors, "; ")
	_, _ = s.UpdateConfig(ctx, config)

	result.FinishedAt = time.Now().UTC()
	return result, nil
}

func (s *Service) backupSession(adapter Adapter, ref sessionRef, config BackupConfig, manifest map[string]manifestEntry) (bool, []string, error) {
	content, err := os.ReadFile(ref.SourcePath)
	if err != nil {
		return false, nil, err
	}
	hash := sha256.Sum256(content)
	hashText := hex.EncodeToString(hash[:])
	key := manifestKey(ref.Source, ref.ID)
	if existing, ok := manifest[key]; ok && existing.SHA256 == hashText && existing.IncludeMarkdown == config.IncludeMarkdown {
		return false, nil, nil
	}

	relativeDir := filepath.Join(string(ref.Source), "projects", safeFilename(ref.ProjectName), "sessions")
	outputDir := filepath.Join(config.OutputDir, relativeDir)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return false, nil, err
	}

	baseName := backupBaseName(ref)
	jsonPath := filepath.Join(outputDir, baseName+".jsonl")
	if !isInsideDir(config.OutputDir, jsonPath) {
		return false, nil, fmt.Errorf("backup path escapes output dir")
	}
	if err := atomicWriteFile(jsonPath, content, 0o644); err != nil {
		return false, nil, err
	}

	markdownRel := ""
	warnings := []string{}
	if config.IncludeMarkdown {
		session, loadErr := adapter.Load(ref)
		if loadErr != nil {
			warnings = append(warnings, fmt.Sprintf("%s: markdown skipped: %v", ref.ID, loadErr))
		} else {
			markdown := RenderMarkdown(session, config.RedactSecrets)
			mdPath := filepath.Join(outputDir, baseName+".md")
			if !isInsideDir(config.OutputDir, mdPath) {
				return false, warnings, fmt.Errorf("markdown path escapes output dir")
			}
			if err := atomicWriteFile(mdPath, []byte(markdown), 0o644); err != nil {
				warnings = append(warnings, fmt.Sprintf("%s: markdown failed: %v", ref.ID, err))
			} else {
				markdownRel, _ = filepath.Rel(config.OutputDir, mdPath)
			}
		}
	}

	jsonRel, _ := filepath.Rel(config.OutputDir, jsonPath)
	manifest[key] = manifestEntry{
		Source:          ref.Source,
		ProjectID:       ref.ProjectID,
		ProjectName:     ref.ProjectName,
		ProjectPath:     ref.ProjectPath,
		SessionID:       ref.ID,
		SourcePath:      ref.SourcePath,
		BackupJSONLPath: jsonRel,
		BackupMDPath:    markdownRel,
		Size:            int64(len(content)),
		SHA256:          hashText,
		UpdatedAt:       ref.UpdatedAt,
		BackedUpAt:      time.Now().UTC(),
		IncludeMarkdown: config.IncludeMarkdown,
	}
	return true, warnings, nil
}

func groupProjects(refs []sessionRef, backedUp map[string]bool) []Project {
	byID := map[string]*Project{}
	for _, ref := range refs {
		project := byID[ref.ProjectID]
		if project == nil {
			project = &Project{
				ID:       ref.ProjectID,
				Name:     ref.ProjectName,
				Path:     ref.ProjectPath,
				Sessions: []SessionSummary{},
			}
			byID[ref.ProjectID] = project
		}
		if ref.UpdatedAt.After(project.UpdatedAt) {
			project.UpdatedAt = ref.UpdatedAt
		}
		project.Sessions = append(project.Sessions, SessionSummary{
			ID:           ref.ID,
			Source:       ref.Source,
			ProjectID:    ref.ProjectID,
			ProjectName:  ref.ProjectName,
			ProjectPath:  ref.ProjectPath,
			Title:        titleFromRef(ref),
			SourcePath:   ref.SourcePath,
			StartedAt:    ref.StartedAt,
			UpdatedAt:    ref.UpdatedAt,
			MessageCount: ref.MessageCount,
			BackedUp:     backedUp[manifestKey(ref.Source, ref.ID)],
		})
	}

	projects := make([]Project, 0, len(byID))
	for _, project := range byID {
		sort.Slice(project.Sessions, func(i, j int) bool {
			return project.Sessions[i].UpdatedAt.After(project.Sessions[j].UpdatedAt)
		})
		projects = append(projects, *project)
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].UpdatedAt.After(projects[j].UpdatedAt)
	})
	return projects
}

func defaultBackupConfig() BackupConfig {
	return BackupConfig{
		Enabled:         false,
		IntervalMinutes: 60,
		Sources:         []SourceID{SourceCodexCLI, SourceClaudeCode},
		IncludeMarkdown: false,
		RedactSecrets:   true,
	}
}

func normalizeConfig(config BackupConfig) BackupConfig {
	if config.IntervalMinutes <= 0 {
		config.IntervalMinutes = 60
	}
	if len(config.Sources) == 0 {
		config.Sources = []SourceID{SourceCodexCLI, SourceClaudeCode}
	}
	config.OutputDir = strings.TrimSpace(config.OutputDir)
	return config
}

func selectedSources(sources []SourceID) map[SourceID]bool {
	selected := map[SourceID]bool{}
	for _, source := range sources {
		selected[source] = true
	}
	if len(selected) == 0 {
		selected[SourceCodexCLI] = true
		selected[SourceClaudeCode] = true
	}
	return selected
}

func atomicWriteFile(path string, content []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func isInsideDir(root string, target string) bool {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return relative == "." || (!strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != "..")
}

var (
	ErrSourceNotFound       = errors.New("conversation source not found")
	ErrSessionNotFound      = errors.New("conversation session not found")
	ErrOutputDirRequired    = errors.New("backup output directory is required")
	ErrBackupAlreadyRunning = errors.New("conversation backup is already running")
)
