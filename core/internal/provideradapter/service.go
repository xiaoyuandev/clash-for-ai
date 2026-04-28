package provideradapter

import "strings"

func NormalizeType(value string) Type {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case string(TypeAnthropicCompatible):
		return TypeAnthropicCompatible
	case string(TypeGeminiCompatible):
		return TypeGeminiCompatible
	default:
		return TypeOpenAICompatible
	}
}

func ResolveRoute(value string) ResolvedRoute {
	providerType := NormalizeType(value)

	switch providerType {
	case TypeAnthropicCompatible:
		return ResolvedRoute{
			ProviderType: providerType,
			UpstreamKind: "anthropic",
			RequestPath:  "/v1/messages",
			AuthHeader:   "x-api-key",
		}
	case TypeGeminiCompatible:
		return ResolvedRoute{
			ProviderType: providerType,
			UpstreamKind: "gemini",
			RequestPath:  "/v1beta/models",
			AuthHeader:   "x-goog-api-key",
		}
	default:
		return ResolvedRoute{
			ProviderType: providerType,
			UpstreamKind: "openai",
			RequestPath:  "/v1/chat/completions",
			AuthHeader:   "authorization",
		}
	}
}
