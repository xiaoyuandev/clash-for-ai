package anthropic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
)

type messagesRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

func ParseRequest(r *http.Request, body []byte) (localgateway.Request, error) {
	operation, err := resolveOperation(r.Method, r.URL.Path)
	if err != nil {
		return localgateway.Request{}, err
	}

	request := localgateway.Request{
		Protocol:  localgateway.InboundAnthropic,
		Operation: operation,
		Method:    r.Method,
		Path:      r.URL.Path,
		Headers:   flattenHeaders(r.Header),
		Body:      body,
	}

	switch operation {
	case localgateway.OperationMessages, localgateway.OperationCountTokens:
		var payload messagesRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			return localgateway.Request{}, fmt.Errorf("decode anthropic messages request: %w", err)
		}
		request.Model = strings.TrimSpace(payload.Model)
		request.Stream = payload.Stream
	}

	return request, nil
}

func resolveOperation(method string, path string) (localgateway.Operation, error) {
	switch {
	case method == http.MethodPost && path == "/v1/messages":
		return localgateway.OperationMessages, nil
	case method == http.MethodPost && path == "/v1/messages/count_tokens":
		return localgateway.OperationCountTokens, nil
	default:
		return "", fmt.Errorf("unsupported anthropic route: %s %s", method, path)
	}
}

func flattenHeaders(header http.Header) map[string]string {
	items := make(map[string]string, len(header))
	for key, values := range header {
		if len(values) == 0 {
			continue
		}
		items[key] = values[0]
	}
	return items
}
