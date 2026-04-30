package settings

type LocalGatewaySettings struct {
	Enabled    bool   `json:"enabled"`
	ListenHost string `json:"listen_host"`
	ListenPort int    `json:"listen_port"`
}

type SelectedModel struct {
	ModelID  string `json:"model_id"`
	Position int    `json:"position"`
}

type ClaudeCodeModelMap struct {
	Opus   string `json:"opus"`
	Sonnet string `json:"sonnet"`
	Haiku  string `json:"haiku"`
}

type AppSettings struct {
	LocalGateway LocalGatewaySettings `json:"local_gateway"`
	// Legacy migration-only field. Runtime selected models now live in localgatewayruntime/state.
	LocalGatewaySelected []SelectedModel `json:"local_gateway_selected_models"`
	// Legacy migration-only field. Provider-scoped Claude model slots now live on provider metadata.
	LocalGatewayClaude ClaudeCodeModelMap `json:"local_gateway_claude_code_model_map"`
}
