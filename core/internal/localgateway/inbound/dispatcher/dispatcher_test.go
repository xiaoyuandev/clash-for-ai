package dispatcher

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
)

func TestParseRequestDispatchesOpenAI(t *testing.T) {
	body := `{"model":"gpt-4.1","stream":false,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))

	parsed, err := ParseRequest(req, []byte(body))
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if parsed.Protocol != localgateway.InboundOpenAI {
		t.Fatalf("unexpected protocol: %s", parsed.Protocol)
	}
}

func TestParseRequestDispatchesAnthropic(t *testing.T) {
	body := `{"model":"claude-sonnet-4","stream":false,"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(body))

	parsed, err := ParseRequest(req, []byte(body))
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if parsed.Protocol != localgateway.InboundAnthropic {
		t.Fatalf("unexpected protocol: %s", parsed.Protocol)
	}
}

func TestParseRequestRejectsUnknownRoute(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/unknown", nil)

	if _, err := ParseRequest(req, nil); err == nil {
		t.Fatalf("expected error for unknown route")
	}
}
