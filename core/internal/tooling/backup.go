package tooling

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func backupRootDir() string {
	return filepath.Join(currentRuntime().HomeDir, ".clash-for-ai", "tool-backups")
}

func backupIfExists(toolID ToolIntegrationID, filePath string) (string, error) {
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	stamp := time.Now().UTC().Format(time.RFC3339)
	stamp = strings.NewReplacer(":", "-", ".", "-").Replace(stamp)
	backupDir := filepath.Join(backupRootDir(), string(toolID))
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}

	backupPath := filepath.Join(backupDir, stamp+"-"+filepath.Base(filePath))
	if err := os.WriteFile(backupPath, content, 0o644); err != nil {
		return "", err
	}

	return backupPath, nil
}

func backupCodexFiles(configPath string, authPath string) (string, error) {
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			if _, authErr := os.Stat(authPath); authErr != nil && os.IsNotExist(authErr) {
				return "", nil
			}
		} else {
			return "", err
		}
	}

	stamp := time.Now().UTC().Format(time.RFC3339)
	stamp = strings.NewReplacer(":", "-", ".", "-").Replace(stamp)
	backupDir := filepath.Join(backupRootDir(), string(ToolCodexCLI), stamp)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}

	if content, err := os.ReadFile(configPath); err == nil {
		if err := os.WriteFile(filepath.Join(backupDir, "config.toml"), content, 0o644); err != nil {
			return "", err
		}
	}

	if content, err := os.ReadFile(authPath); err == nil {
		if err := os.WriteFile(filepath.Join(backupDir, "auth.json"), content, 0o644); err != nil {
			return "", err
		}
	}

	return backupDir, nil
}

func latestBackupPath(toolID ToolIntegrationID) string {
	entries, err := os.ReadDir(filepath.Join(backupRootDir(), string(toolID)))
	if err != nil {
		return ""
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	if len(names) == 0 {
		return ""
	}

	return filepath.Join(backupRootDir(), string(toolID), names[0])
}
