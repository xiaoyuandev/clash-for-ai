package openai

import (
	"encoding/json"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
)

func TestBuildRequestForChatCompletions(t *testing.T) {
	body := []byte(`{"model":"placeholder","messages":[{"role":"user","content":"hi"}],"stream":true}`)
	request := localgateway.Request{
		Operation: localgateway.OperationChatCompletions,
		Method:    "POST",
		Path:      "/v1/chat/completions",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}
	target := localgateway.UpstreamTarget{
		BaseURL:        "https://api.example.com/v1",
		APIKey:         "test-key",
		DefaultModelID: "gpt-4.1",
	}

	upstream, err := BuildRequest(request, target)
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if upstream.URL != "https://api.example.com/v1/chat/completions" {
		t.Fatalf("unexpected upstream url: %s", upstream.URL)
	}

	if upstream.Headers.Get("Authorization") != "Bearer test-key" {
		t.Fatalf("unexpected authorization header: %s", upstream.Headers.Get("Authorization"))
	}

	var payload map[string]any
	if err := json.Unmarshal(upstream.Body, &payload); err != nil {
		t.Fatalf("failed to decode upstream body: %v", err)
	}
	if payload["model"] != "gpt-4.1" {
		t.Fatalf("unexpected rewritten model: %v", payload["model"])
	}
}

func TestBuildRequestForModels(t *testing.T) {
	request := localgateway.Request{
		Operation: localgateway.OperationModels,
		Method:    "GET",
		Path:      "/v1/models",
		Headers:   map[string]string{},
	}
	target := localgateway.UpstreamTarget{
		BaseURL: "https://api.example.com/v1",
		APIKey:  "test-key",
	}

	upstream, err := BuildRequest(request, target)
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if upstream.URL != "https://api.example.com/v1/models" {
		t.Fatalf("unexpected upstream url: %s", upstream.URL)
	}
}
