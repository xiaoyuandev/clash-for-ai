package anthropic

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
)

func TestParseMessagesRequest(t *testing.T) {
	body := `{"model":"claude-sonnet-4","stream":true,"max_tokens":64,"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	parsed, err := ParseRequest(req, []byte(body))
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if parsed.Protocol != localgateway.InboundAnthropic {
		t.Fatalf("unexpected protocol: %s", parsed.Protocol)
	}
	if parsed.Operation != localgateway.OperationMessages {
		t.Fatalf("unexpected operation: %s", parsed.Operation)
	}
	if parsed.Model != "claude-sonnet-4" {
		t.Fatalf("unexpected model: %s", parsed.Model)
	}
	if !parsed.Stream {
		t.Fatalf("expected stream=true")
	}
}

func TestParseCountTokensRequest(t *testing.T) {
	body := `{"model":"claude-sonnet-4","messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`
	req := httptest.NewRequest("POST", "/v1/messages/count_tokens", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	parsed, err := ParseRequest(req, []byte(body))
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if parsed.Operation != localgateway.OperationCountTokens {
		t.Fatalf("unexpected operation: %s", parsed.Operation)
	}
	if parsed.Model != "claude-sonnet-4" {
		t.Fatalf("unexpected model: %s", parsed.Model)
	}
}
