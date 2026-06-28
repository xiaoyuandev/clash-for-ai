package provider

type AuthMode string

const (
	AuthModeBearer              AuthMode = "bearer"
	AuthModeAPIKey              AuthMode = "x-api-key"
	AuthModeBoth                AuthMode = "both"
	RuntimeKindExternal                  = "external"
	RuntimeKindManagedLocalGate          = "local-gateway"
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

type ClaudeCodeModelMap struct {
	Opus   string `json:"opus"`
	Sonnet string `json:"sonnet"`
	Haiku  string `json:"haiku"`
}

type Provider struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	BaseURL            string             `json:"base_url"`
	ModelsPath         string             `json:"models_path"`
	APIKeyRef          string             `json:"-"`
	APIKey             string             `json:"api_key"`
	AuthMode           AuthMode           `json:"auth_mode"`
	ExtraHeaders       map[string]string  `json:"extra_headers"`
	Capabilities       Capabilities       `json:"capabilities"`
	Status             Status             `json:"status"`
	APIKeyMasked       string             `json:"api_key_masked"`
	ClaudeCodeModelMap ClaudeCodeModelMap `json:"claude_code_model_map"`
	IsSystemManaged    bool               `json:"is_system_managed"`
	IsEditable         bool               `json:"is_editable"`
	IsDeletable        bool               `json:"is_deletable"`
	RuntimeKind        string             `json:"runtime_kind"`
}

type SelectedModel struct {
	ModelID  string `json:"model_id"`
	Position int    `json:"position"`
}

type CodexModel struct {
	ProviderID    string `json:"provider_id,omitempty"`
	ModelID       string `json:"model_id"`
	DisplayName   string `json:"display_name"`
	Enabled       bool   `json:"enabled"`
	Position      int    `json:"position"`
	ContextWindow *int   `json:"context_window,omitempty"`
}

type ModelTestResult struct {
	ModelID     string `json:"model_id"`
	Status      string `json:"status"`
	StatusCode  int    `json:"status_code"`
	LatencyMs   int64  `json:"latency_ms"`
	Summary     string `json:"summary"`
	CheckedAt   string `json:"checked_at"`
	ProviderID  string `json:"provider_id"`
	ProviderURL string `json:"provider_url"`
	Protocol    string `json:"protocol"`
	RequestPath string `json:"request_path"`
}
