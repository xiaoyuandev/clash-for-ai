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
	LocalGateway         LocalGatewaySettings `json:"local_gateway"`
	LocalGatewaySelected []SelectedModel      `json:"local_gateway_selected_models"`
	LocalGatewayClaude   ClaudeCodeModelMap   `json:"local_gateway_claude_code_model_map"`
}
