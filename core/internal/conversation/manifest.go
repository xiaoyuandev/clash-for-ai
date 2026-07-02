package conversation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type manifestFile struct {
	Version   int             `json:"version"`
	UpdatedAt time.Time       `json:"updated_at"`
	Sessions  []manifestEntry `json:"sessions"`
}

type manifestEntry struct {
	Source          SourceID  `json:"source"`
	ProjectID       string    `json:"project_id"`
	ProjectName     string    `json:"project_name"`
	ProjectPath     string    `json:"project_path"`
	SessionID       string    `json:"session_id"`
	SourcePath      string    `json:"source_path"`
	BackupJSONLPath string    `json:"backup_jsonl_path"`
	BackupMDPath    string    `json:"backup_markdown_path,omitempty"`
	Size            int64     `json:"size"`
	SHA256          string    `json:"sha256"`
	UpdatedAt       time.Time `json:"updated_at"`
	BackedUpAt      time.Time `json:"backed_up_at"`
	IncludeMarkdown bool      `json:"include_markdown"`
}

func readManifest(path string) map[string]manifestEntry {
	content, err := os.ReadFile(path)
	if err != nil {
		return map[string]manifestEntry{}
	}
	var file manifestFile
	if err := json.Unmarshal(content, &file); err != nil {
		return map[string]manifestEntry{}
	}
	result := map[string]manifestEntry{}
	for _, entry := range file.Sessions {
		result[manifestKey(entry.Source, entry.SessionID)] = entry
	}
	return result
}

func writeManifest(path string, entries map[string]manifestEntry) error {
	items := make([]manifestEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, entry)
	}
	content, err := json.MarshalIndent(manifestFile{
		Version:   1,
		UpdatedAt: time.Now().UTC(),
		Sessions:  items,
	}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return atomicWriteFile(path, append(content, '\n'), 0o644)
}

func manifestKey(source SourceID, sessionID string) string {
	return string(source) + ":" + sessionID
}

func backupBaseName(ref sessionRef) string {
	date := "unknown-date"
	if !ref.UpdatedAt.IsZero() {
		date = ref.UpdatedAt.Format("2006-01-02")
	}
	return safeFilename(date + "_" + ref.ID)
}

func safeFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "untitled"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	value = replacer.Replace(value)
	value = strings.Join(strings.Fields(value), "-")
	value = strings.Trim(value, ".- ")
	if value == "" {
		return "untitled"
	}
	if len([]rune(value)) > 80 {
		value = string([]rune(value)[:80])
	}
	return value
}
