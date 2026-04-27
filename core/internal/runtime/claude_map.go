package runtime

type ClaudeCodeModelMap struct {
	Opus   string `json:"opus"`
	Sonnet string `json:"sonnet"`
	Haiku  string `json:"haiku"`
}

type SelectedModel struct {
	ModelID  string `json:"model_id"`
	Position int    `json:"position"`
}
