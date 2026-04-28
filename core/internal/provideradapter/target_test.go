package provideradapter

import "testing"

func TestResolveTargetAnthropicCompatible(t *testing.T) {
	target := ResolveTarget("anthropic-compatible", "https://api.example.com/anthropic/v1", "key", "claude-sonnet")

	if target.ProviderType != TypeAnthropicCompatible {
		t.Fatalf("unexpected provider type: %s", target.ProviderType)
	}
	if target.Route.UpstreamKind != "anthropic" {
		t.Fatalf("unexpected upstream kind: %s", target.Route.UpstreamKind)
	}
	if target.Route.RequestPath != "/v1/messages" {
		t.Fatalf("unexpected request path: %s", target.Route.RequestPath)
	}
	if target.DefaultModelID != "claude-sonnet" {
		t.Fatalf("unexpected default model id: %s", target.DefaultModelID)
	}
}

func TestResolveTargetOpenAICompatibleByDefault(t *testing.T) {
	target := ResolveTarget("", "https://api.example.com/v1", "key", "gpt-4.1")

	if target.ProviderType != TypeOpenAICompatible {
		t.Fatalf("unexpected provider type: %s", target.ProviderType)
	}
	if target.Route.UpstreamKind != "openai" {
		t.Fatalf("unexpected upstream kind: %s", target.Route.UpstreamKind)
	}
	if target.Route.RequestPath != "/v1/chat/completions" {
		t.Fatalf("unexpected request path: %s", target.Route.RequestPath)
	}
}
