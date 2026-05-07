package tooling

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func codexConfigPath() string {
	return filepath.Join(currentRuntime().HomeDir, ".codex", "config.toml")
}

func codexAuthPath() string {
	return filepath.Join(currentRuntime().HomeDir, ".codex", "auth.json")
}

func inspectCodexIntegration(apiPort int) (ToolIntegrationState, error) {
	configPath := codexConfigPath()
	authPath := codexAuthPath()
	executable := resolveExecutable(executableName("codex"))
	configured, err := isCodexConfigured(apiPort)
	if err != nil {
		return ToolIntegrationState{}, err
	}

	return ToolIntegrationState{
		ID:                  ToolCodexCLI,
		Detected:            executable != "" || fileExists(configPath) || fileExists(authPath),
		Configured:          configured,
		SupportsAdapter:     true,
		ConfigPath:          configPath,
		SecondaryConfigPath: authPath,
		ExecutablePath:      executable,
		BackupPath:          latestBackupPath(ToolCodexCLI),
		Message:             codexMessage(configured),
	}, nil
}

func applyCodexIntegration(apiPort int) (ToolIntegrationState, error) {
	configPath := codexConfigPath()
	authPath := codexAuthPath()
	currentConfig, err := readOptionalText(configPath)
	if err != nil {
		return ToolIntegrationState{}, err
	}
	currentAuth, err := readOptionalText(authPath)
	if err != nil {
		return ToolIntegrationState{}, err
	}

	backupPath, err := backupCodexFiles(configPath, authPath)
	if err != nil {
		return ToolIntegrationState{}, err
	}

	if err := ensureDir(configPath); err != nil {
		return ToolIntegrationState{}, err
	}
	if err := os.WriteFile(configPath, []byte(buildCodexConfig(currentConfig, apiPort)), 0o644); err != nil {
		return ToolIntegrationState{}, err
	}
	if err := os.WriteFile(authPath, []byte(buildCodexAuth(currentAuth)), 0o644); err != nil {
		return ToolIntegrationState{}, err
	}

	state, err := inspectCodexIntegration(apiPort)
	if err != nil {
		return ToolIntegrationState{}, err
	}
	state.BackupPath = backupPath
	state.Message = "Configured Codex CLI to use the local Clash for AI gateway."
	return state, nil
}

func restoreCodexIntegration(apiPort int) (ToolIntegrationState, error) {
	backupPath := latestBackupPath(ToolCodexCLI)
	if backupPath == "" {
		return ToolIntegrationState{}, ErrNoBackupAvailable
	}

	configPath := codexConfigPath()
	authPath := codexAuthPath()
	if strings.HasSuffix(backupPath, ".toml") || strings.HasSuffix(backupPath, ".json") {
		content, err := os.ReadFile(backupPath)
		if err != nil {
			return ToolIntegrationState{}, err
		}
		if err := ensureDir(configPath); err != nil {
			return ToolIntegrationState{}, err
		}
		if err := os.WriteFile(configPath, content, 0o644); err != nil {
			return ToolIntegrationState{}, err
		}
	} else {
		if err := ensureDir(configPath); err != nil {
			return ToolIntegrationState{}, err
		}
		if content, err := os.ReadFile(filepath.Join(backupPath, "config.toml")); err == nil {
			if err := os.WriteFile(configPath, content, 0o644); err != nil {
				return ToolIntegrationState{}, err
			}
		}
		if content, err := os.ReadFile(filepath.Join(backupPath, "auth.json")); err == nil {
			if err := os.WriteFile(authPath, content, 0o644); err != nil {
				return ToolIntegrationState{}, err
			}
		}
	}

	state, err := inspectCodexIntegration(apiPort)
	if err != nil {
		return ToolIntegrationState{}, err
	}
	state.BackupPath = backupPath
	state.Message = "Restored the most recent backup for codex-cli."
	return state, nil
}

func isCodexConfigured(apiPort int) (bool, error) {
	content, err := readOptionalText(codexConfigPath())
	if err != nil {
		return false, err
	}
	authContent, err := readOptionalText(codexAuthPath())
	if err != nil {
		return false, err
	}
	if content == "" || authContent == "" {
		return false, nil
	}

	topLevelProvider := readTopLevelTomlValue(content, "model_provider")
	var auth map[string]any
	if err := json.Unmarshal([]byte(authContent), &auth); err != nil {
		auth = map[string]any{}
	}

	return topLevelProvider == "OpenAI" &&
		strings.Contains(content, "[model_providers.OpenAI]") &&
		strings.Contains(content, `base_url = "http://127.0.0.1:`+intToString(apiPort)+`/v1"`) &&
		auth["OPENAI_API_KEY"] == "dummy", nil
}

func buildCodexConfig(existingContent string, apiPort int) string {
	lines := []string{}
	if strings.TrimSpace(existingContent) != "" {
		lines = strings.Split(strings.TrimRight(existingContent, "\n\r\t "), "\n")
	}

	nextModelProviderLine := `model_provider = "OpenAI"`
	replacedTopLevelProvider := false
	currentTable := ""
	inOpenAISection := false
	replacedBaseURL := false
	rewritten := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentTable = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			inOpenAISection = currentTable == "model_providers.OpenAI"
			rewritten = append(rewritten, line)
			continue
		}

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			rewritten = append(rewritten, line)
			continue
		}

		if currentTable == "" && strings.HasPrefix(trimmed, "model_provider") {
			replacedTopLevelProvider = true
			rewritten = append(rewritten, nextModelProviderLine)
			continue
		}

		if inOpenAISection && strings.HasPrefix(trimmed, "base_url") {
			replacedBaseURL = true
			rewritten = append(rewritten, `base_url = "http://127.0.0.1:`+intToString(apiPort)+`/v1"`)
			continue
		}

		rewritten = append(rewritten, line)
	}

	if !replacedTopLevelProvider {
		firstTableIndex := -1
		for idx, line := range rewritten {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
				firstTableIndex = idx
				break
			}
		}

		if firstTableIndex == -1 {
			rewritten = append(rewritten, nextModelProviderLine)
		} else {
			rewritten = append(rewritten[:firstTableIndex], append([]string{nextModelProviderLine, ""}, rewritten[firstTableIndex:]...)...)
		}
	}

	hasOpenAISection := false
	for _, line := range rewritten {
		if strings.TrimSpace(line) == "[model_providers.OpenAI]" {
			hasOpenAISection = true
			break
		}
	}

	nextLines := append([]string{}, rewritten...)
	if !hasOpenAISection {
		nextLines = append(nextLines,
			"",
			"[model_providers.OpenAI]",
			`name = "OpenAI"`,
			`base_url = "http://127.0.0.1:`+intToString(apiPort)+`/v1"`,
			`wire_api = "responses"`,
			"requires_openai_auth = true",
		)
	} else if !replacedBaseURL {
		sectionIndex := -1
		for idx, line := range nextLines {
			if strings.TrimSpace(line) == "[model_providers.OpenAI]" {
				sectionIndex = idx
				break
			}
		}
		if sectionIndex >= 0 {
			insertIndex := sectionIndex + 1
			for insertIndex < len(nextLines) {
				trimmed := strings.TrimSpace(nextLines[insertIndex])
				if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
					break
				}
				insertIndex++
			}
			nextLines = append(nextLines[:insertIndex], append([]string{`base_url = "http://127.0.0.1:` + intToString(apiPort) + `/v1"`}, nextLines[insertIndex:]...)...)
		}
	}

	return strings.TrimRight(strings.Join(nextLines, "\n"), "\n\r\t ") + "\n"
}

func buildCodexAuth(existingContent string) string {
	auth := map[string]any{}
	if strings.TrimSpace(existingContent) != "" {
		_ = json.Unmarshal([]byte(existingContent), &auth)
	}
	auth["OPENAI_API_KEY"] = "dummy"
	content, _ := json.MarshalIndent(auth, "", "  ")
	return string(content) + "\n"
}

func readTopLevelTomlValue(content string, key string) string {
	currentTable := ""
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(key) + `\s*=\s*"([^"]+)"\s*$`)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentTable = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			continue
		}
		if currentTable != "" || trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		match := re.FindStringSubmatch(trimmed)
		if len(match) == 2 {
			return match[1]
		}
	}
	return ""
}

func codexMessage(configured bool) string {
	if configured {
		return "Codex CLI is already pointed at the local Clash for AI gateway."
	}
	return "Codex CLI can be configured by updating ~/.codex/config.toml and ~/.codex/auth.json."
}
