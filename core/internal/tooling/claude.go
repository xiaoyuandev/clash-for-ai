package tooling

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

func claudeConfigPath() string {
	return filepath.Join(currentRuntime().HomeDir, ".claude", "settings.json")
}

func inspectClaudeIntegration(apiPort int) (ToolIntegrationState, error) {
	configPath := claudeConfigPath()
	executable := resolveExecutable(executableName("claude"))
	configured, err := isClaudeConfigured(apiPort)
	if err != nil {
		return ToolIntegrationState{}, err
	}

	return ToolIntegrationState{
		ID:              ToolClaudeCode,
		Detected:        executable != "" || fileExists(configPath),
		Configured:      configured,
		SupportsAdapter: true,
		ConfigPath:      configPath,
		ExecutablePath:  executable,
		BackupPath:      latestBackupPath(ToolClaudeCode),
		Message:         claudeMessage(configured),
	}, nil
}

func applyClaudeIntegration(apiPort int, modelMap provider.ClaudeCodeModelMap) (ToolIntegrationState, error) {
	configPath := claudeConfigPath()
	currentRaw, err := readOptionalText(configPath)
	if err != nil {
		return ToolIntegrationState{}, err
	}

	backupPath, err := backupIfExists(ToolClaudeCode, configPath)
	if err != nil {
		return ToolIntegrationState{}, err
	}

	parsed := map[string]any{}
	if currentRaw != "" {
		_ = json.Unmarshal([]byte(currentRaw), &parsed)
	}

	env := map[string]any{}
	if existingEnv, ok := parsed["env"].(map[string]any); ok {
		env = existingEnv
	}

	env["ANTHROPIC_BASE_URL"] = "http://127.0.0.1:" + intToString(apiPort)
	env["ANTHROPIC_AUTH_TOKEN"] = "dummy"
	syncClaudeCodeModelEnv(env, modelMap)
	parsed["env"] = env

	content, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return ToolIntegrationState{}, err
	}

	if err := ensureDir(configPath); err != nil {
		return ToolIntegrationState{}, err
	}
	if err := os.WriteFile(configPath, append(content, '\n'), 0o644); err != nil {
		return ToolIntegrationState{}, err
	}

	state, err := inspectClaudeIntegration(apiPort)
	if err != nil {
		return ToolIntegrationState{}, err
	}
	state.BackupPath = backupPath
	state.Message = "Configured Claude Code to use the local Clash for AI gateway and synced the active provider model slots."
	return state, nil
}

func restoreClaudeIntegration(apiPort int) (ToolIntegrationState, error) {
	backupPath := latestBackupPath(ToolClaudeCode)
	if backupPath == "" {
		return ToolIntegrationState{}, ErrNoBackupAvailable
	}

	content, err := os.ReadFile(backupPath)
	if err != nil {
		return ToolIntegrationState{}, err
	}
	configPath := claudeConfigPath()
	if err := ensureDir(configPath); err != nil {
		return ToolIntegrationState{}, err
	}
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		return ToolIntegrationState{}, err
	}

	state, err := inspectClaudeIntegration(apiPort)
	if err != nil {
		return ToolIntegrationState{}, err
	}
	state.BackupPath = backupPath
	state.Message = "Restored the most recent backup for claude-code."
	return state, nil
}

func isClaudeConfigured(apiPort int) (bool, error) {
	content, err := readOptionalText(claudeConfigPath())
	if err != nil {
		return false, err
	}
	if content == "" {
		return false, nil
	}

	parsed := map[string]any{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return false, nil
	}
	env, ok := parsed["env"].(map[string]any)
	if !ok {
		return false, nil
	}

	return env["ANTHROPIC_BASE_URL"] == "http://127.0.0.1:"+intToString(apiPort) &&
		env["ANTHROPIC_AUTH_TOKEN"] == "dummy", nil
}

func syncClaudeCodeModelEnv(env map[string]any, modelMap provider.ClaudeCodeModelMap) {
	assignOrDelete(env, "ANTHROPIC_MODEL", modelMap.Sonnet)
	assignOrDelete(env, "ANTHROPIC_DEFAULT_OPUS_MODEL", modelMap.Opus)
	assignOrDelete(env, "ANTHROPIC_DEFAULT_SONNET_MODEL", modelMap.Sonnet)
	assignOrDelete(env, "ANTHROPIC_DEFAULT_HAIKU_MODEL", modelMap.Haiku)
}

func assignOrDelete(env map[string]any, key string, value string) {
	if value != "" {
		env[key] = value
		return
	}
	delete(env, key)
}

func claudeMessage(configured bool) string {
	if configured {
		return "Claude Code is already configured with the local Clash for AI gateway variables."
	}
	return "Claude Code can be configured by writing env overrides into ~/.claude/settings.json."
}
