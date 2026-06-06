package extension

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	transcriptSourceClaudeCode = "claude-code"
	transcriptSourceCodexCLI   = "codex-cli"
)

type transcriptArchiveConfig struct {
	plugin             Plugin
	outputDirectory    string
	includeClaudeCode  bool
	includeCodexCLI    bool
	includeSystemEvent bool
	redactSecrets      bool
	overwriteExisting  bool
}

type transcriptSession struct {
	Source      string
	SourceTitle string
	SessionID   string
	Title       string
	ProjectPath string
	ProjectName string
	StartedAt   string
	UpdatedAt   string
	RawPath     string
	Messages    []transcriptMessage
	Metadata    map[string]string
}

type transcriptMessage struct {
	Role      string
	Text      string
	Timestamp string
	RawType   string
}

type transcriptPath struct {
	sourceID    string
	sourceTitle string
	path        string
}

func (s *Service) ListTranscriptSources(ctx context.Context) ([]TranscriptSource, error) {
	config, err := s.transcriptArchiveConfig(ctx, TranscriptSyncInput{})
	if err != nil && !errors.Is(err, ErrPluginNotFound) {
		return nil, err
	}
	if errors.Is(err, ErrPluginNotFound) {
		config = defaultTranscriptArchiveConfig()
	}

	sources := []TranscriptSource{}
	if config.includeClaudeCode {
		paths, err := discoverClaudeTranscriptPaths(s.homeDir)
		if err != nil {
			return nil, err
		}
		sources = append(sources, TranscriptSource{
			ID:           transcriptSourceClaudeCode,
			Title:        "Claude Code",
			Kind:         "local-jsonl",
			Enabled:      true,
			SessionCount: len(paths),
			Paths:        paths,
		})
	}
	if config.includeCodexCLI {
		paths, err := discoverCodexTranscriptPaths(s.homeDir)
		if err != nil {
			return nil, err
		}
		sources = append(sources, TranscriptSource{
			ID:           transcriptSourceCodexCLI,
			Title:        "Codex CLI",
			Kind:         "local-jsonl",
			Enabled:      true,
			SessionCount: len(paths),
			Paths:        paths,
		})
	}

	return sources, nil
}

func (s *Service) SyncTranscriptArchive(ctx context.Context, input TranscriptSyncInput) (TranscriptSyncResult, error) {
	started := time.Now().UTC()
	config, err := s.transcriptArchiveConfig(ctx, input)
	if err != nil {
		return TranscriptSyncResult{}, err
	}

	result := TranscriptSyncResult{
		PluginID:        config.plugin.ID,
		Status:          "success",
		OutputDirectory: config.outputDirectory,
		StartedAt:       started.Format(time.RFC3339),
		Sources:         []TranscriptSourceSyncResult{},
		Failures:        []TranscriptSyncFailure{},
	}

	finishWithAudit := func(status string, syncErr error) (TranscriptSyncResult, error) {
		finished := time.Now().UTC()
		result.Status = status
		result.FinishedAt = finished.Format(time.RFC3339)
		latency := finished.Sub(started).Milliseconds()
		auditStatus := status
		if syncErr != nil {
			auditStatus = "failed"
		}
		entry, auditErr := s.repository.RecordAudit(ctx, AuditLogEntry{
			PluginID:      config.plugin.ID,
			PluginVersion: config.plugin.Version,
			Capability:    "tool.transcripts.read",
			Action:        "markdownArchive.sync",
			ResourceType:  "directory",
			ResourceID:    config.outputDirectory,
			Status:        auditStatus,
			LatencyMs:     &latency,
			ErrorMessage:  errorMessage(syncErr),
			Metadata: map[string]any{
				"exported": result.ExportedCount,
				"skipped":  result.SkippedCount,
				"failed":   result.FailedCount,
			},
		})
		if auditErr != nil {
			return result, auditErr
		}
		result.AuditLogID = entry.ID
		return result, syncErr
	}

	if !config.plugin.Enabled || config.plugin.Status != PluginStatusEnabled {
		return finishWithAudit("failed", ErrPluginNotEnabled)
	}
	if strings.TrimSpace(config.outputDirectory) == "" {
		return finishWithAudit("failed", ErrTranscriptOutputDirectoryRequired)
	}

	outputDir, err := filepath.Abs(config.outputDirectory)
	if err != nil {
		return finishWithAudit("failed", fmt.Errorf("resolve output directory: %w", err))
	}
	config.outputDirectory = outputDir
	result.OutputDirectory = outputDir
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return finishWithAudit("failed", fmt.Errorf("create output directory: %w", err))
	}

	paths, err := s.discoverTranscriptPaths(config)
	if err != nil {
		return finishWithAudit("failed", err)
	}

	sourceResults := map[string]*TranscriptSourceSyncResult{}
	for _, item := range paths {
		if _, ok := sourceResults[item.sourceID]; !ok {
			sourceResults[item.sourceID] = &TranscriptSourceSyncResult{
				SourceID: item.sourceID,
			}
		}
		sourceResults[item.sourceID].Discovered++

		session, err := loadTranscriptSession(item, config.includeSystemEvent)
		if err != nil {
			result.FailedCount++
			sourceResults[item.sourceID].FailedCount++
			result.Failures = append(result.Failures, TranscriptSyncFailure{
				SourceID: item.sourceID,
				Path:     item.path,
				Error:    err.Error(),
			})
			continue
		}

		outcome, err := s.exportTranscriptSession(ctx, config, session)
		if err != nil {
			result.FailedCount++
			sourceResults[item.sourceID].FailedCount++
			result.Failures = append(result.Failures, TranscriptSyncFailure{
				SourceID:  item.sourceID,
				SessionID: session.SessionID,
				Path:      item.path,
				Error:     err.Error(),
			})
			continue
		}
		if outcome == "skipped" {
			result.SkippedCount++
			sourceResults[item.sourceID].SkippedCount++
			continue
		}
		result.ExportedCount++
		sourceResults[item.sourceID].ExportedCount++
	}

	sourceIDs := make([]string, 0, len(sourceResults))
	for sourceID := range sourceResults {
		sourceIDs = append(sourceIDs, sourceID)
	}
	sort.Strings(sourceIDs)
	for _, sourceID := range sourceIDs {
		result.Sources = append(result.Sources, *sourceResults[sourceID])
	}
	if result.FailedCount > 0 {
		return finishWithAudit("partial", nil)
	}
	return finishWithAudit("success", nil)
}

func (s *Service) transcriptArchiveConfig(ctx context.Context, input TranscriptSyncInput) (transcriptArchiveConfig, error) {
	pluginID := strings.TrimSpace(input.PluginID)
	if pluginID == "" {
		pluginID = MarkdownArchivePluginID
	}
	plugin, err := s.repository.GetByID(ctx, pluginID)
	if err != nil {
		return transcriptArchiveConfig{}, err
	}

	config := defaultTranscriptArchiveConfig()
	config.plugin = *plugin
	settings, err := s.GetSettings(ctx, plugin.ID)
	if err != nil {
		return transcriptArchiveConfig{}, err
	}

	config.outputDirectory = stringSetting(settings.EffectiveValues, "outputDirectory", config.outputDirectory)
	config.includeClaudeCode = boolSetting(settings.EffectiveValues, "includeClaudeCode", config.includeClaudeCode)
	config.includeCodexCLI = boolSetting(settings.EffectiveValues, "includeCodexCLI", config.includeCodexCLI)
	config.includeSystemEvent = boolSetting(settings.EffectiveValues, "includeSystemEvents", config.includeSystemEvent)
	config.redactSecrets = boolSetting(settings.EffectiveValues, "redactSecrets", config.redactSecrets)
	config.overwriteExisting = boolSetting(settings.EffectiveValues, "overwriteExisting", config.overwriteExisting)

	if strings.TrimSpace(input.OutputDirectory) != "" {
		config.outputDirectory = input.OutputDirectory
	}
	if input.IncludeClaudeCode != nil {
		config.includeClaudeCode = *input.IncludeClaudeCode
	}
	if input.IncludeCodexCLI != nil {
		config.includeCodexCLI = *input.IncludeCodexCLI
	}
	if input.IncludeSystemEvents != nil {
		config.includeSystemEvent = *input.IncludeSystemEvents
	}
	if input.RedactSecrets != nil {
		config.redactSecrets = *input.RedactSecrets
	}
	if input.OverwriteExisting != nil {
		config.overwriteExisting = *input.OverwriteExisting
	}

	return config, nil
}

func defaultTranscriptArchiveConfig() transcriptArchiveConfig {
	return transcriptArchiveConfig{
		includeClaudeCode:  true,
		includeCodexCLI:    true,
		includeSystemEvent: false,
		redactSecrets:      true,
		overwriteExisting:  true,
	}
}

func (s *Service) discoverTranscriptPaths(config transcriptArchiveConfig) ([]transcriptPath, error) {
	items := []transcriptPath{}
	if config.includeClaudeCode {
		paths, err := discoverClaudeTranscriptPaths(s.homeDir)
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			items = append(items, transcriptPath{
				sourceID:    transcriptSourceClaudeCode,
				sourceTitle: "Claude Code",
				path:        path,
			})
		}
	}
	if config.includeCodexCLI {
		paths, err := discoverCodexTranscriptPaths(s.homeDir)
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			items = append(items, transcriptPath{
				sourceID:    transcriptSourceCodexCLI,
				sourceTitle: "Codex CLI",
				path:        path,
			})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].path < items[j].path
	})
	return items, nil
}

func discoverClaudeTranscriptPaths(homeDir string) ([]string, error) {
	if homeDir == "" {
		return []string{}, nil
	}
	pattern := filepath.Join(homeDir, ".claude", "projects", "*", "*.jsonl")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("discover Claude Code transcripts: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func discoverCodexTranscriptPaths(homeDir string) ([]string, error) {
	if homeDir == "" {
		return []string{}, nil
	}
	roots := []string{
		filepath.Join(homeDir, ".codex", "sessions"),
		filepath.Join(homeDir, ".codex", "archived_sessions"),
	}

	paths := []string{}
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("inspect Codex transcript directory: %w", err)
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".jsonl") && name != "session_index.jsonl" {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("discover Codex transcripts: %w", err)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func loadTranscriptSession(item transcriptPath, includeSystemEvents bool) (transcriptSession, error) {
	file, err := os.Open(item.path)
	if err != nil {
		return transcriptSession{}, fmt.Errorf("open transcript: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return transcriptSession{}, fmt.Errorf("stat transcript: %w", err)
	}

	session := transcriptSession{
		Source:      item.sourceID,
		SourceTitle: item.sourceTitle,
		SessionID:   strings.TrimSuffix(filepath.Base(item.path), filepath.Ext(item.path)),
		RawPath:     item.path,
		UpdatedAt:   info.ModTime().UTC().Format(time.RFC3339),
		Metadata: map[string]string{
			"raw_path": item.path,
		},
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		if value := firstString(raw, "sessionId", "session_id", "conversation_id"); value != "" {
			session.SessionID = value
		}
		if value := firstNestedString(raw, []string{"payload", "session_id"}, []string{"payload", "sessionId"}); value != "" {
			session.SessionID = value
		}
		if value := firstString(raw, "cwd", "project_path", "projectPath"); value != "" {
			session.ProjectPath = value
		}
		if value := firstNestedString(raw, []string{"payload", "cwd"}, []string{"message", "cwd"}); value != "" && session.ProjectPath == "" {
			session.ProjectPath = value
		}
		if value := firstString(raw, "title", "summary"); value != "" && session.Title == "" {
			session.Title = value
		}

		rawType := firstString(raw, "type", "event", "kind")
		role := normalizeTranscriptRole(firstString(raw, "role"), rawType)
		if nestedRole := firstNestedString(raw, []string{"message", "role"}, []string{"payload", "role"}); nestedRole != "" {
			role = normalizeTranscriptRole(nestedRole, rawType)
		}
		if !includeSystemEvents && (role == "system" || role == "event") {
			continue
		}

		text := transcriptText(raw)
		if strings.TrimSpace(text) == "" && (role == "user" || role == "assistant") {
			continue
		}
		timestamp := firstString(raw, "timestamp", "created_at", "createdAt", "time")
		if value := firstNestedString(raw, []string{"payload", "timestamp"}, []string{"message", "timestamp"}); value != "" && timestamp == "" {
			timestamp = value
		}
		if session.StartedAt == "" && timestamp != "" {
			session.StartedAt = timestamp
		}
		if timestamp != "" {
			session.UpdatedAt = timestamp
		}

		session.Messages = append(session.Messages, transcriptMessage{
			Role:      role,
			Text:      text,
			Timestamp: timestamp,
			RawType:   rawType,
		})
	}
	if err := scanner.Err(); err != nil {
		return transcriptSession{}, fmt.Errorf("read transcript: %w", err)
	}
	if session.ProjectName == "" {
		session.ProjectName = projectName(session.ProjectPath)
	}
	if session.Title == "" {
		session.Title = fmt.Sprintf("%s - %s", session.SourceTitle, session.ProjectName)
	}
	if session.StartedAt == "" {
		session.StartedAt = session.UpdatedAt
	}
	return session, nil
}

func (s *Service) exportTranscriptSession(ctx context.Context, config transcriptArchiveConfig, session transcriptSession) (string, error) {
	info, err := os.Stat(session.RawPath)
	if err != nil {
		return "", fmt.Errorf("stat raw transcript: %w", err)
	}

	existing, err := s.repository.GetTranscriptExport(ctx, session.Source, session.SessionID)
	if err != nil {
		return "", err
	}
	rawMTime := info.ModTime().Unix()
	if existing != nil && existing.RawMTime == rawMTime && existing.RawSize == info.Size() {
		if _, err := os.Stat(existing.OutputPath); err == nil {
			return "skipped", nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("inspect indexed output path: %w", err)
		}
	}

	markdown := renderTranscriptMarkdown(session, config.redactSecrets)
	hashBytes := sha256.Sum256([]byte(markdown))
	contentHash := hex.EncodeToString(hashBytes[:])
	outputPath := transcriptOutputPath(config.outputDirectory, session)

	if existing == nil && !config.overwriteExisting {
		if _, err := os.Stat(outputPath); err == nil {
			return "skipped", nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("inspect output path: %w", err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", fmt.Errorf("create transcript output directory: %w", err)
	}
	if err := atomicWriteFile(outputPath, []byte(markdown), 0o644); err != nil {
		return "", err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.repository.UpsertTranscriptExport(ctx, TranscriptExportState{
		Source:      session.Source,
		SessionID:   session.SessionID,
		RawPath:     session.RawPath,
		RawMTime:    rawMTime,
		RawSize:     info.Size(),
		OutputPath:  outputPath,
		ExportedAt:  now,
		ContentHash: contentHash,
	}); err != nil {
		return "", err
	}

	return "exported", nil
}

func renderTranscriptMarkdown(session transcriptSession, redact bool) string {
	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString("source: " + yamlString(session.Source) + "\n")
	builder.WriteString("session_id: " + yamlString(session.SessionID) + "\n")
	builder.WriteString("project: " + yamlString(session.ProjectName) + "\n")
	builder.WriteString("project_path: " + yamlString(session.ProjectPath) + "\n")
	builder.WriteString("started_at: " + yamlString(session.StartedAt) + "\n")
	builder.WriteString("updated_at: " + yamlString(session.UpdatedAt) + "\n")
	builder.WriteString("raw_path: " + yamlString(session.RawPath) + "\n")
	builder.WriteString("tags:\n")
	builder.WriteString("  - ai/conversation\n")
	builder.WriteString("  - relay-switch\n")
	builder.WriteString("  - " + yamlString(session.Source) + "\n")
	builder.WriteString("---\n\n")
	builder.WriteString("# " + markdownText(session.Title, redact) + "\n\n")

	for _, message := range session.Messages {
		heading := transcriptRoleHeading(message.Role)
		builder.WriteString("## " + heading + "\n\n")
		if message.Timestamp != "" {
			builder.WriteString("`" + markdownText(message.Timestamp, redact) + "`\n\n")
		}
		text := markdownText(message.Text, redact)
		if strings.TrimSpace(text) == "" {
			text = "_No text content._"
		}
		builder.WriteString(text)
		builder.WriteString("\n\n")
	}

	return builder.String()
}

func transcriptRoleHeading(role string) string {
	switch role {
	case "user":
		return "User"
	case "assistant":
		return "Assistant"
	case "system":
		return "System"
	case "tool":
		return "Tool"
	default:
		return "Event"
	}
}

func transcriptOutputPath(outputDir string, session transcriptSession) string {
	stamp := timestampForFilename(session.StartedAt)
	fileName := sanitizePathSegment(stamp + " " + session.SessionID + ".md")
	return filepath.Join(
		outputDir,
		"AI Conversations",
		sanitizePathSegment(session.SourceTitle),
		sanitizePathSegment(session.ProjectName),
		fileName,
	)
}

func timestampForFilename(value string) string {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.Local().Format("2006-01-02 15-04")
	}
	return time.Now().Local().Format("2006-01-02 15-04")
}

func sanitizePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Unknown"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	value = replacer.Replace(value)
	value = strings.Trim(value, ". ")
	if value == "" {
		return "Unknown"
	}
	if len(value) > 120 {
		value = value[:120]
	}
	return value
}

func projectName(projectPath string) string {
	if strings.TrimSpace(projectPath) == "" {
		return "Unknown Project"
	}
	name := filepath.Base(projectPath)
	if name == "." || name == string(filepath.Separator) {
		return "Unknown Project"
	}
	return name
}

func atomicWriteFile(path string, content []byte, perm fs.FileMode) error {
	tempPath := fmt.Sprintf("%s.tmp-%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tempPath, content, perm); err != nil {
		return fmt.Errorf("write temp transcript output: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace transcript output: %w", err)
	}
	return nil
}

func normalizeTranscriptRole(role string, rawType string) string {
	value := strings.ToLower(strings.TrimSpace(role))
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(rawType))
	}
	switch value {
	case "user", "assistant", "system", "tool":
		return value
	case "human":
		return "user"
	case "":
		return "event"
	default:
		return "event"
	}
}

func transcriptText(raw map[string]any) string {
	for _, key := range []string{"text", "content", "message"} {
		if value, ok := raw[key]; ok {
			if text := textFromAny(value); text != "" {
				return text
			}
		}
	}
	for _, path := range [][]string{
		{"message", "content"},
		{"message", "text"},
		{"payload", "content"},
		{"payload", "text"},
		{"item", "content"},
	} {
		if value := nestedValue(raw, path...); value != nil {
			if text := textFromAny(value); text != "" {
				return text
			}
		}
	}
	return ""
}

func textFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		parts := []string{}
		for _, item := range typed {
			if text := textFromAny(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		for _, key := range []string{"text", "content", "input", "output"} {
			if value, ok := typed[key]; ok {
				if text := textFromAny(value); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func firstString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				return text
			}
		}
	}
	return ""
}

func firstNestedString(raw map[string]any, paths ...[]string) string {
	for _, path := range paths {
		if value := nestedValue(raw, path...); value != nil {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				return text
			}
		}
	}
	return ""
}

func nestedValue(raw map[string]any, path ...string) any {
	var current any = raw
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = object[key]
	}
	return current
}

func yamlString(value string) string {
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	return "\"" + escaped + "\""
}

func markdownText(value string, redact bool) string {
	if !redact {
		return value
	}
	return redactSecretText(value)
}

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`sk-[A-Za-z0-9_-]{12,}`),
	regexp.MustCompile(`(?i)(OPENAI_API_KEY|ANTHROPIC_API_KEY|GITHUB_TOKEN)=\S+`),
	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._~+/=-]{12,}`),
}

func redactSecretText(value string) string {
	for _, pattern := range secretPatterns {
		value = pattern.ReplaceAllString(value, "[redacted]")
	}
	return value
}

func boolSetting(values map[string]any, key string, fallback bool) bool {
	if value, ok := values[key].(bool); ok {
		return value
	}
	return fallback
}

func stringSetting(values map[string]any, key string, fallback string) string {
	if value, ok := values[key].(string); ok {
		return value
	}
	return fallback
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
