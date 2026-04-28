package provideradapter

type Type string

const (
	TypeOpenAICompatible    Type = "openai-compatible"
	TypeAnthropicCompatible Type = "anthropic-compatible"
	TypeGeminiCompatible    Type = "gemini-compatible"
)

type ResolvedRoute struct {
	ProviderType Type   `json:"provider_type"`
	UpstreamKind string `json:"upstream_kind"`
	RequestPath  string `json:"request_path"`
	AuthHeader   string `json:"auth_header"`
}
