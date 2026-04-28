package openai

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
)

func TestParseChatCompletionsRequest(t *testing.T) {
	body := `{"model":"gpt-4.1","stream":true,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	parsed, err := ParseRequest(req, []byte(body))
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if parsed.Protocol != localgateway.InboundOpenAI {
		t.Fatalf("unexpected protocol: %s", parsed.Protocol)
	}
	if parsed.Operation != localgateway.OperationChatCompletions {
		t.Fatalf("unexpected operation: %s", parsed.Operation)
	}
	if parsed.Model != "gpt-4.1" {
		t.Fatalf("unexpected model: %s", parsed.Model)
	}
	if !parsed.Stream {
		t.Fatalf("expected stream=true")
	}
}

func TestParseModelsRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/models", nil)

	parsed, err := ParseRequest(req, nil)
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if parsed.Operation != localgateway.OperationModels {
		t.Fatalf("unexpected operation: %s", parsed.Operation)
	}
}
