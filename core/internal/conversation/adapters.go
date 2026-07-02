package conversation

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type codexAdapter struct {
	homeDir string
}

type claudeAdapter struct {
	homeDir string
}

func newCodexAdapter() Adapter {
	home, _ := os.UserHomeDir()
	return codexAdapter{homeDir: home}
}

func newClaudeAdapter() Adapter {
	home, _ := os.UserHomeDir()
	return claudeAdapter{homeDir: home}
}

func (a codexAdapter) Source() SourceID { return SourceCodexCLI }
func (a codexAdapter) Title() string    { return "Codex CLI" }

func (a codexAdapter) Discover() ([]sessionRef, []string) {
	return discoverJSONLSessions(SourceCodexCLI, filepath.Join(a.homeDir, ".codex", "sessions"), "Codex", "")
}

func (a codexAdapter) Load(ref sessionRef) (Session, error) {
	return loadJSONLSession(ref)
}

func (a claudeAdapter) Source() SourceID { return SourceClaudeCode }
func (a claudeAdapter) Title() string    { return "Claude Code" }

func (a claudeAdapter) Discover() ([]sessionRef, []string) {
	root := filepath.Join(a.homeDir, ".claude", "projects")
	refs, warnings := discoverJSONLSessions(SourceClaudeCode, root, "Claude Code", root)
	for idx := range refs {
		if refs[idx].ProjectPath == "" || strings.HasPrefix(refs[idx].ProjectName, "project-") {
			if decoded := decodeClaudeProjectPath(filepath.Dir(refs[idx].SourcePath), root); decoded != "" {
				refs[idx].ProjectPath = decoded
				refs[idx].ProjectName = filepath.Base(decoded)
				refs[idx].ProjectID = projectID(refs[idx].ProjectPath)
			}
		}
	}
	return refs, warnings
}

func (a claudeAdapter) Load(ref sessionRef) (Session, error) {
	return loadJSONLSession(ref)
}

func discoverJSONLSessions(source SourceID, root string, fallbackProject string, claudeRoot string) ([]sessionRef, []string) {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return []sessionRef{}, nil
	}

	refs := []sessionRef{}
	warnings := []string{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = append(warnings, walkErr.Error())
			return nil
		}
		if source == SourceClaudeCode && entry.IsDir() && entry.Name() == "subagents" {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".jsonl") {
			return nil
		}
		ref := sessionRefFromPath(source, path, fallbackProject)
		if claudeRoot != "" {
			if decoded := decodeClaudeProjectPath(filepath.Dir(path), claudeRoot); decoded != "" {
				ref.ProjectPath = decoded
				ref.ProjectName = filepath.Base(decoded)
				ref.ProjectID = projectID(decoded)
			}
		}
		if summary, summaryWarnings := summarizeJSONL(path); summary.ProjectPath != "" || summary.ProjectName != "" || summary.Title != "" {
			warnings = append(warnings, summaryWarnings...)
			if summary.ProjectPath != "" {
				ref.ProjectPath = summary.ProjectPath
				ref.ProjectName = filepath.Base(summary.ProjectPath)
				ref.ProjectID = projectID(summary.ProjectPath)
			}
			if summary.ProjectName != "" && ref.ProjectName == fallbackProject {
				ref.ProjectName = summary.ProjectName
			}
			if summary.Title != "" {
				ref.Title = summary.Title
			}
			if !summary.StartedAt.IsZero() {
				ref.StartedAt = summary.StartedAt
			}
			if !summary.UpdatedAt.IsZero() {
				ref.UpdatedAt = summary.UpdatedAt
			}
			ref.MessageCount = summary.MessageCount
		}
		refs = append(refs, ref)
		return nil
	})
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	sort.Slice(refs, func(i, j int) bool {
		return refs[i].UpdatedAt.After(refs[j].UpdatedAt)
	})
	return refs, warnings
}

type jsonlSummary struct {
	ProjectPath  string
	ProjectName  string
	Title        string
	StartedAt    time.Time
	UpdatedAt    time.Time
	MessageCount int
}

func summarizeJSONL(path string) (jsonlSummary, []string) {
	file, err := os.Open(path)
	if err != nil {
		return jsonlSummary{}, []string{err.Error()}
	}
	defer file.Close()

	summary := jsonlSummary{}
	warnings := []string{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024*8)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s:%d: invalid jsonl line", path, lineNumber))
			continue
		}
		if summary.ProjectPath == "" {
			summary.ProjectPath = firstStringDeep(raw, "cwd", "project_path", "projectPath", "workdir")
		}
		if summary.ProjectName == "" {
			summary.ProjectName = firstStringDeep(raw, "project", "project_name", "projectName")
		}
		event := eventFromRaw(raw, lineNumber)
		if summary.Title == "" && event.Role == "user" && event.Kind == "message" {
			summary.Title = event.Text
		}
		if timestamp := parseTimestamp(firstStringDeep(raw, "timestamp", "created_at", "createdAt", "time")); !timestamp.IsZero() {
			if summary.StartedAt.IsZero() || timestamp.Before(summary.StartedAt) {
				summary.StartedAt = timestamp
			}
			summary.UpdatedAt = timestamp
		}
		if event.Kind == "message" && (event.Role == "user" || event.Role == "assistant") {
			summary.MessageCount++
		}
	}
	if err := scanner.Err(); err != nil {
		warnings = append(warnings, err.Error())
	}
	return summary, warnings
}

func loadJSONLSession(ref sessionRef) (Session, error) {
	file, err := os.Open(ref.SourcePath)
	if err != nil {
		return Session{}, err
	}
	defer file.Close()

	events := []Event{}
	warnings := []string{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024*16)
	lineNumber := 0
	var startedAt time.Time
	var updatedAt time.Time
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			warnings = append(warnings, fmt.Sprintf("line %d: invalid json", lineNumber))
			continue
		}
		event := eventFromRaw(raw, lineNumber)
		if event.Text == "" && event.Kind == "raw" {
			event.Text = line
		}
		if !event.CreatedAt.IsZero() {
			if startedAt.IsZero() || event.CreatedAt.Before(startedAt) {
				startedAt = event.CreatedAt
			}
			if event.CreatedAt.After(updatedAt) {
				updatedAt = event.CreatedAt
			}
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		warnings = append(warnings, err.Error())
	}
	if updatedAt.IsZero() {
		updatedAt = ref.UpdatedAt
	}

	summary := SessionSummary{
		ID:           ref.ID,
		Source:       ref.Source,
		ProjectID:    ref.ProjectID,
		ProjectName:  ref.ProjectName,
		ProjectPath:  ref.ProjectPath,
		Title:        titleFromEvents(events, ref),
		SourcePath:   ref.SourcePath,
		StartedAt:    startedAt,
		UpdatedAt:    updatedAt,
		MessageCount: countMessages(events),
	}

	session := Session{Summary: summary, Events: events, Warnings: warnings}
	session.Markdown = RenderMarkdown(session, false)
	return session, nil
}

func eventFromRaw(raw map[string]any, lineNumber int) Event {
	rawType := strings.ToLower(firstString(raw, "type"))
	container := messageContainer(raw)
	containerType := strings.ToLower(firstString(container, "type"))
	role := strings.ToLower(firstString(container, "role", "speaker"))
	if role == "" {
		role = strings.ToLower(firstString(raw, "role", "speaker"))
	}
	if role == "" && (rawType == "user" || rawType == "assistant" || rawType == "system") {
		role = rawType
	}
	text := normalizeMessageText(firstMessageText(raw), role)
	kind := "message"
	language := ""
	if role == "" {
		role = "system"
	}

	if isMetaEvent(rawType, containerType, role, text) {
		kind = "meta"
		text = ""
	}
	if role == "developer" || role == "system" {
		kind = "meta"
		text = ""
	}
	if hasToolContent(container) || hasToolContent(raw) || strings.Contains(role, "tool") {
		kind = "tool"
	}
	if command := firstStringDeep(raw, "command", "cmd"); command != "" {
		kind = "tool_call"
		text = command
		language = "sh"
	}
	if isToolCallType(containerType) || isToolCallType(rawType) {
		kind = "tool_call"
		if text == "" {
			text = toolText(container)
		}
	}
	if isToolOutputType(containerType) || isToolOutputType(rawType) {
		kind = "tool"
	}
	if text == "" {
		if kind == "message" {
			kind = "raw"
		}
	}
	return Event{
		ID:        firstStringDeep(raw, "id", "uuid", "event_id", "eventId", "request_id", "call_id"),
		Role:      role,
		Kind:      kind,
		Text:      text,
		Language:  language,
		Metadata:  map[string]any{"line": lineNumber},
		CreatedAt: parseTimestamp(firstStringDeep(raw, "timestamp", "created_at", "createdAt", "time")),
	}
}

func firstMessageText(raw map[string]any) string {
	if value := firstString(raw, "text", "content", "summary", "title"); value != "" {
		return value
	}
	if message, ok := raw["message"]; ok {
		return textFromAny(message)
	}
	if payload, ok := raw["payload"]; ok {
		return textFromAny(payload)
	}
	if item, ok := raw["item"]; ok {
		return textFromAny(item)
	}
	return ""
}

func textFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case map[string]any:
		itemType := strings.ToLower(firstString(typed, "type"))
		if itemType == "thinking" || itemType == "reasoning" || itemType == "tool_use" || itemType == "function_call" || itemType == "local_shell_call" || itemType == "custom_tool_call" {
			return ""
		}
		if itemType == "tool_result" || itemType == "function_call_output" {
			return textFromAny(typed["content"])
		}
		if value := firstString(typed, "text", "content", "summary"); value != "" {
			return value
		}
		if content, ok := typed["content"]; ok {
			return textFromAny(content)
		}
	case []any:
		parts := []string{}
		for _, item := range typed {
			text := textFromAny(item)
			if strings.TrimSpace(text) != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func normalizeMessageText(value string, role string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	if role == "user" {
		if arguments := textAfterSkillArguments(text); arguments != "" {
			return arguments
		}
		text = removeTaggedBlock(text, "skill")
		text = removeTaggedBlock(text, "skills_instructions")
		text = strings.TrimSpace(text)
		if text == "" {
			return ""
		}
		for _, prefix := range []string{
			"<environment_context>",
			"<permissions instructions>",
			"<collaboration_mode>",
			"<skills_instructions>",
			"<plugins_instructions>",
		} {
			if strings.HasPrefix(text, prefix) {
				return ""
			}
		}
		for _, marker := range []string{"## My request for Codex:", "## My request for Claude:"} {
			if idx := strings.Index(text, marker); idx >= 0 {
				return strings.TrimSpace(text[idx+len(marker):])
			}
		}
	}
	return text
}

func textAfterSkillArguments(text string) string {
	for _, marker := range []string{"\nARGUMENTS:", "\nArguments:", "\narguments:"} {
		if idx := strings.LastIndex(text, marker); idx >= 0 {
			return strings.TrimSpace(text[idx+len(marker):])
		}
	}
	return ""
}

func removeTaggedBlock(text string, tag string) string {
	openTag := "<" + tag + ">"
	closeTag := "</" + tag + ">"
	for {
		start := strings.Index(text, openTag)
		if start < 0 {
			return text
		}
		end := strings.Index(text[start+len(openTag):], closeTag)
		if end < 0 {
			return strings.TrimSpace(text[:start])
		}
		end = start + len(openTag) + end + len(closeTag)
		text = text[:start] + text[end:]
	}
}

func messageContainer(raw map[string]any) map[string]any {
	for _, key := range []string{"message", "payload", "item"} {
		if nested, ok := raw[key].(map[string]any); ok {
			return nested
		}
	}
	return raw
}

func firstStringDeep(raw map[string]any, keys ...string) string {
	if value := firstString(raw, keys...); value != "" {
		return value
	}
	for _, key := range []string{"message", "payload", "item"} {
		nested, ok := raw[key].(map[string]any)
		if !ok {
			continue
		}
		if value := firstString(nested, keys...); value != "" {
			return value
		}
	}
	return ""
}

func isMetaEvent(rawType string, containerType string, role string, text string) bool {
	if text != "" && (role == "user" || role == "assistant") && containerType == "message" {
		return false
	}
	switch rawType {
	case "session_meta", "turn_context", "event_msg", "mode", "permission-mode", "file-history-snapshot", "attachment", "ai-title", "queue-operation":
		return true
	}
	switch containerType {
	case "task_started", "token_count", "reasoning":
		return true
	}
	return false
}

func hasToolContent(raw map[string]any) bool {
	if raw == nil {
		return false
	}
	if isToolCallType(strings.ToLower(firstString(raw, "type"))) || isToolOutputType(strings.ToLower(firstString(raw, "type"))) {
		return true
	}
	content, ok := raw["content"].([]any)
	if !ok {
		return false
	}
	for _, item := range content {
		nested, ok := item.(map[string]any)
		if !ok {
			continue
		}
		itemType := strings.ToLower(firstString(nested, "type"))
		if isToolCallType(itemType) || isToolOutputType(itemType) {
			return true
		}
	}
	return false
}

func isToolCallType(value string) bool {
	switch value {
	case "tool_use", "function_call", "local_shell_call", "custom_tool_call":
		return true
	default:
		return false
	}
}

func isToolOutputType(value string) bool {
	switch value {
	case "tool_result", "function_call_output":
		return true
	default:
		return false
	}
}

func toolText(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	name := firstString(raw, "name", "command")
	if name == "" {
		name = firstStringDeep(raw, "name", "command")
	}
	input := raw["input"]
	if input == nil {
		input = raw["arguments"]
	}
	if input == nil {
		return name
	}
	encoded, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return name
	}
	if name == "" {
		return string(encoded)
	}
	return name + "\n" + string(encoded)
}

func firstString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case fmt.Stringer:
			if strings.TrimSpace(typed.String()) != "" {
				return strings.TrimSpace(typed.String())
			}
		}
	}
	return ""
}

func sessionRefFromPath(source SourceID, path string, fallbackProject string) sessionRef {
	info, _ := os.Stat(path)
	updatedAt := time.Time{}
	if info != nil {
		updatedAt = info.ModTime().UTC()
	}
	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	projectPath := filepath.Dir(path)
	return sessionRef{
		Source:      source,
		ID:          id,
		ProjectID:   projectID(projectPath),
		ProjectName: fallbackProject,
		ProjectPath: projectPath,
		SourcePath:  path,
		UpdatedAt:   updatedAt,
	}
}

func decodeClaudeProjectPath(dir string, root string) string {
	relative, err := filepath.Rel(root, dir)
	if err != nil || relative == "." || strings.HasPrefix(relative, "..") {
		return ""
	}
	firstPart := strings.Split(relative, string(filepath.Separator))[0]
	if strings.HasPrefix(firstPart, "-") {
		firstPart = string(filepath.Separator) + strings.TrimLeft(firstPart, "-")
	}
	decoded := strings.ReplaceAll(firstPart, "-", string(filepath.Separator))
	if filepath.IsAbs(decoded) {
		return filepath.Clean(decoded)
	}
	return ""
}

func projectID(path string) string {
	if strings.TrimSpace(path) == "" {
		return "unknown"
	}
	sum := sha1.Sum([]byte(path))
	return safeFilename(filepath.Base(path)) + "-" + hex.EncodeToString(sum[:])[:8]
}

func titleFromRef(ref sessionRef) string {
	if strings.TrimSpace(ref.Title) != "" {
		return truncateTitle(ref.Title)
	}
	if ref.ID != "" {
		return ref.ID
	}
	return "Untitled session"
}

func titleFromEvents(events []Event, ref sessionRef) string {
	for _, event := range events {
		if event.Role == "user" && strings.TrimSpace(event.Text) != "" {
			return truncateTitle(event.Text)
		}
	}
	return titleFromRef(ref)
}

func truncateTitle(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len([]rune(value)) <= 48 {
		return value
	}
	return string([]rune(value)[:48]) + "..."
}

func countMessages(events []Event) int {
	count := 0
	for _, event := range events {
		if event.Kind == "message" {
			count++
		}
	}
	return count
}

func parseTimestamp(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}
