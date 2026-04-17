package provider

type AuthMode string

const (
	AuthModeBearer AuthMode = "bearer"
	AuthModeAPIKey AuthMode = "x-api-key"
	AuthModeBoth   AuthMode = "both"
)

type Capabilities struct {
	SupportsOpenAICompatible    bool `json:"supports_openai_compatible"`
	SupportsAnthropicCompatible bool `json:"supports_anthropic_compatible"`
	SupportsModelsAPI           bool `json:"supports_models_api"`
	SupportsBalanceAPI          bool `json:"supports_balance_api"`
	SupportsStream              bool `json:"supports_stream"`
}

type Status struct {
	IsActive          bool   `json:"is_active"`
	LastHealthStatus  string `json:"last_health_status"`
	LastHealthcheckAt string `json:"last_healthcheck_at,omitempty"`
}

type Provider struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	BaseURL      string            `json:"base_url"`
	APIKeyRef    string            `json:"-"`
	AuthMode     AuthMode          `json:"auth_mode"`
	ExtraHeaders map[string]string `json:"extra_headers"`
	Capabilities Capabilities      `json:"capabilities"`
	Status       Status            `json:"status"`
	APIKeyMasked string            `json:"api_key_masked"`
}
