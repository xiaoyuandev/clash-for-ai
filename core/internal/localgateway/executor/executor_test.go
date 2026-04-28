package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
)

func TestHandleOpenAI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if payload["model"] != "gpt-4.1" {
			t.Fatalf("unexpected rewritten model: %v", payload["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	exec := New(server.Client())
	response, err := exec.Handle(context.Background(), localgateway.Request{
		Protocol:  localgateway.InboundOpenAI,
		Operation: localgateway.OperationChatCompletions,
		Method:    "POST",
		Path:      "/v1/chat/completions",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"model":"placeholder","messages":[{"role":"user","content":"hi"}]}`),
	}, localgateway.ModelSource{
		Name:           "OpenAI source",
		BaseURL:        server.URL + "/v1",
		APIKey:         "test-key",
		ProviderType:   "openai-compatible",
		DefaultModelID: "gpt-4.1",
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", response.StatusCode)
	}
}

func TestHandleAnthropic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/anthropic/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Fatalf("unexpected x-api-key header: %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Fatalf("missing anthropic-version header")
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if payload["model"] != "claude-sonnet-4" {
			t.Fatalf("unexpected rewritten model: %v", payload["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	exec := New(server.Client())
	response, err := exec.Handle(context.Background(), localgateway.Request{
		Protocol:  localgateway.InboundAnthropic,
		Operation: localgateway.OperationMessages,
		Method:    "POST",
		Path:      "/v1/messages",
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"anthropic-version": "2023-06-01",
		},
		Body: []byte(`{"model":"placeholder","messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`),
	}, localgateway.ModelSource{
		Name:           "Anthropic source",
		BaseURL:        server.URL + "/anthropic/v1",
		APIKey:         "test-key",
		ProviderType:   "anthropic-compatible",
		DefaultModelID: "claude-sonnet-4",
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", response.StatusCode)
	}
}
