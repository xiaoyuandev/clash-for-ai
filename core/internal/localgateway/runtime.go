package localgateway

const (
	RuntimeStateStopped  = "stopped"
	RuntimeStateStarting = "starting"
	RuntimeStateRunning  = "running"
	RuntimeStateDegraded = "degraded"
	RuntimeStateError    = "error"
)

type StartRuntimeInput struct {
	Executable  string            `json:"executable"`
	Host        string            `json:"host"`
	Port        int               `json:"port"`
	DataDir     string            `json:"data_dir"`
	Environment map[string]string `json:"environment"`
	Arguments   []string          `json:"arguments"`
}

type RuntimeStatus struct {
	RuntimeKind string `json:"runtime_kind"`
	State       string `json:"state"`
	Managed     bool   `json:"managed"`
	Running     bool   `json:"running"`
	Healthy     bool   `json:"healthy"`
	APIBase     string `json:"api_base"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	PID         int    `json:"pid,omitempty"`
	Version     string `json:"version,omitempty"`
	Commit      string `json:"commit,omitempty"`
	LastError   string `json:"last_error,omitempty"`
}

type RuntimeCapabilities struct {
	SupportsOpenAICompatible     bool `json:"supports_openai_compatible"`
	SupportsAnthropicCompatible  bool `json:"supports_anthropic_compatible"`
	SupportsModelsAPI            bool `json:"supports_models_api"`
	SupportsStream               bool `json:"supports_stream"`
	SupportsAdminAPI             bool `json:"supports_admin_api"`
	SupportsModelSourceAdmin     bool `json:"supports_model_source_admin"`
	SupportsSelectedModelAdmin   bool `json:"supports_selected_model_admin"`
	SupportsSourceCapabilities   bool `json:"supports_source_capabilities"`
	SupportsAtomicSourceSync     bool `json:"supports_atomic_source_sync"`
	SupportsRuntimeVersion       bool `json:"supports_runtime_version"`
	SupportsExplicitSourceHealth bool `json:"supports_explicit_source_health"`
}

type RuntimeModelSource struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	BaseURL         string   `json:"base_url"`
	ProviderType    string   `json:"provider_type"`
	DefaultModelID  string   `json:"default_model_id"`
	ExposedModelIDs []string `json:"exposed_model_ids"`
	Enabled         bool     `json:"enabled"`
	Position        int      `json:"position"`
	APIKeyMasked    string   `json:"api_key_masked"`
}

type RuntimeModelSourceInput struct {
	Name            string   `json:"name"`
	BaseURL         string   `json:"base_url"`
	APIKey          string   `json:"api_key"`
	ProviderType    string   `json:"provider_type"`
	DefaultModelID  string   `json:"default_model_id"`
	ExposedModelIDs []string `json:"exposed_model_ids"`
	Enabled         bool     `json:"enabled"`
	Position        int      `json:"position"`
}

type SyncModelSource struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	BaseURL         string   `json:"base_url"`
	APIKey          string   `json:"api_key"`
	ProviderType    string   `json:"provider_type"`
	DefaultModelID  string   `json:"default_model_id"`
	ExposedModelIDs []string `json:"exposed_model_ids"`
	Enabled         bool     `json:"enabled"`
	Position        int      `json:"position"`
}

type ModelSourceCapability struct {
	SourceID                      string `json:"source_id"`
	Name                          string `json:"name"`
	ProviderType                  string `json:"provider_type"`
	SupportsModelsAPI             bool   `json:"supports_models_api"`
	ModelsAPIStatus               string `json:"models_api_status"`
	SupportsOpenAIChatCompletions bool   `json:"supports_openai_chat_completions"`
	OpenAIChatCompletionsStatus   string `json:"openai_chat_completions_status"`
	SupportsOpenAIResponses       bool   `json:"supports_openai_responses"`
	OpenAIResponsesStatus         string `json:"openai_responses_status"`
	SupportsAnthropicMessages     bool   `json:"supports_anthropic_messages"`
	AnthropicMessagesStatus       string `json:"anthropic_messages_status"`
	SupportsAnthropicCountTokens  bool   `json:"supports_anthropic_count_tokens"`
	AnthropicCountTokensStatus    string `json:"anthropic_count_tokens_status"`
	SupportsStream                bool   `json:"supports_stream"`
	StreamStatus                  string `json:"stream_status"`
}

type SyncInput struct {
	Sources        []SyncModelSource `json:"sources"`
	SelectedModels []SelectedModel   `json:"selected_models"`
}

type SyncResult struct {
	AppliedSources        int    `json:"applied_sources"`
	AppliedSelectedModels int    `json:"applied_selected_models"`
	LastSyncedAt          string `json:"last_synced_at"`
}
