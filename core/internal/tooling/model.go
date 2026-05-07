package tooling

type ToolIntegrationID string

const (
	ToolCodexCLI   ToolIntegrationID = "codex-cli"
	ToolClaudeCode ToolIntegrationID = "claude-code"
	ToolCursor     ToolIntegrationID = "cursor"
	ToolCherry     ToolIntegrationID = "cherry-studio"
	ToolOpenCode   ToolIntegrationID = "open-code"
	ToolOpenAISDK  ToolIntegrationID = "openai-sdk"
)

type ToolIntegrationState struct {
	ID                  ToolIntegrationID `json:"id"`
	Detected            bool              `json:"detected"`
	Configured          bool              `json:"configured"`
	SupportsAdapter     bool              `json:"supports_adapter"`
	ConfigPath          string            `json:"config_path,omitempty"`
	SecondaryConfigPath string            `json:"secondary_config_path,omitempty"`
	ExecutablePath      string            `json:"executable_path,omitempty"`
	BackupPath          string            `json:"backup_path,omitempty"`
	Message             string            `json:"message,omitempty"`
}

type RuntimeInfo struct {
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	IsWSL   bool   `json:"is_wsl"`
	HomeDir string `json:"home_dir"`
}
