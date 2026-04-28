package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
)

func TestBuildRequestForMessages(t *testing.T) {
	body := []byte(`{"model":"placeholder","messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}],"stream":true}`)
	request := localgateway.Request{
		Operation: localgateway.OperationMessages,
		Method:    "POST",
		Path:      "/v1/messages",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}
	target := localgateway.UpstreamTarget{
		BaseURL:        "https://api.example.com/anthropic/v1",
		APIKey:         "test-key",
		DefaultModelID: "claude-sonnet-4",
	}

	upstream, err := BuildRequest(request, target)
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if upstream.URL != "https://api.example.com/anthropic/v1/messages" {
		t.Fatalf("unexpected upstream url: %s", upstream.URL)
	}

	if upstream.Headers.Get("x-api-key") != "test-key" {
		t.Fatalf("unexpected x-api-key header: %s", upstream.Headers.Get("x-api-key"))
	}

	if upstream.Headers.Get("anthropic-version") != defaultAnthropicVersion {
		t.Fatalf("unexpected anthropic-version header: %s", upstream.Headers.Get("anthropic-version"))
	}

	var payload map[string]any
	if err := json.Unmarshal(upstream.Body, &payload); err != nil {
		t.Fatalf("failed to decode upstream body: %v", err)
	}
	if payload["model"] != "claude-sonnet-4" {
		t.Fatalf("unexpected rewritten model: %v", payload["model"])
	}
}

func TestBuildRequestForCountTokens(t *testing.T) {
	request := localgateway.Request{
		Operation: localgateway.OperationCountTokens,
		Method:    "POST",
		Path:      "/v1/messages/count_tokens",
		Headers:   map[string]string{},
		Body:      []byte(`{"model":"placeholder"}`),
	}
	target := localgateway.UpstreamTarget{
		BaseURL:        "https://api.example.com/anthropic/v1",
		APIKey:         "test-key",
		DefaultModelID: "claude-sonnet-4",
	}

	upstream, err := BuildRequest(request, target)
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if upstream.URL != "https://api.example.com/anthropic/v1/messages/count_tokens" {
		t.Fatalf("unexpected upstream url: %s", upstream.URL)
	}
}
