package modelsource

type Source struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	ProviderType   string `json:"provider_type"`
	DefaultModelID string `json:"default_model_id"`
	Enabled        bool   `json:"enabled"`
	Position       int    `json:"position"`
	APIKeyRef      string `json:"-"`
	APIKey         string `json:"api_key"`
	APIKeyMasked   string `json:"api_key_masked"`
}

type CreateInput struct {
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	ProviderType   string `json:"provider_type"`
	DefaultModelID string `json:"default_model_id"`
	Enabled        bool   `json:"enabled"`
	APIKey         string `json:"api_key"`
}

type UpdateInput struct {
	Name           string `json:"name"`
	BaseURL        string `json:"base_url"`
	ProviderType   string `json:"provider_type"`
	DefaultModelID string `json:"default_model_id"`
	Enabled        bool   `json:"enabled"`
	APIKey         string `json:"api_key"`
}
