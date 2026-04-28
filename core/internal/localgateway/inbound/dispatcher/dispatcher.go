package dispatcher

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
	anthropicinbound "github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway/inbound/anthropic"
	openaiinbound "github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway/inbound/openai"
)

func ParseRequest(r *http.Request, body []byte) (localgateway.Request, error) {
	switch {
	case isAnthropicRoute(r.Method, r.URL.Path):
		return anthropicinbound.ParseRequest(r, body)
	case isOpenAIRoute(r.Method, r.URL.Path):
		return openaiinbound.ParseRequest(r, body)
	default:
		return localgateway.Request{}, fmt.Errorf("unsupported inbound route: %s %s", r.Method, r.URL.Path)
	}
}

func isOpenAIRoute(method string, path string) bool {
	switch {
	case method == http.MethodPost && path == "/v1/chat/completions":
		return true
	case method == http.MethodPost && path == "/v1/responses":
		return true
	case method == http.MethodGet && path == "/v1/models":
		return true
	default:
		return false
	}
}

func isAnthropicRoute(method string, path string) bool {
	switch {
	case method == http.MethodPost && path == "/v1/messages":
		return true
	case method == http.MethodPost && path == "/v1/messages/count_tokens":
		return true
	case strings.TrimSpace(path) == "/v1/complete":
		return true
	default:
		return false
	}
}
