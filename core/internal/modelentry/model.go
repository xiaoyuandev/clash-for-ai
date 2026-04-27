package modelentry

type Entry struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ModelID      string `json:"model_id"`
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	ProviderType string `json:"provider_type"`
	Protocol     string `json:"protocol"`
	Enabled      bool   `json:"enabled"`
	Position     int    `json:"position"`
}

type CreateInput struct {
	Name         string `json:"name"`
	ModelID      string `json:"model_id"`
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	ProviderType string `json:"provider_type"`
	Protocol     string `json:"protocol"`
	Enabled      bool   `json:"enabled"`
}

type UpdateInput struct {
	Name         string `json:"name"`
	ModelID      string `json:"model_id"`
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	ProviderType string `json:"provider_type"`
	Protocol     string `json:"protocol"`
	Enabled      bool   `json:"enabled"`
}
