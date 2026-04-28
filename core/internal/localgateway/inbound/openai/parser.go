package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
)

type chatCompletionRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

type responsesRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

func ParseRequest(r *http.Request, body []byte) (localgateway.Request, error) {
	operation, err := resolveOperation(r.Method, r.URL.Path)
	if err != nil {
		return localgateway.Request{}, err
	}

	request := localgateway.Request{
		Protocol:  localgateway.InboundOpenAI,
		Operation: operation,
		Method:    r.Method,
		Path:      r.URL.Path,
		Headers:   flattenHeaders(r.Header),
		Body:      body,
	}

	switch operation {
	case localgateway.OperationChatCompletions:
		var payload chatCompletionRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			return localgateway.Request{}, fmt.Errorf("decode chat completions request: %w", err)
		}
		request.Model = strings.TrimSpace(payload.Model)
		request.Stream = payload.Stream
	case localgateway.OperationResponses:
		var payload responsesRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			return localgateway.Request{}, fmt.Errorf("decode responses request: %w", err)
		}
		request.Model = strings.TrimSpace(payload.Model)
		request.Stream = payload.Stream
	case localgateway.OperationModels:
		// no-op
	}

	return request, nil
}

func resolveOperation(method string, path string) (localgateway.Operation, error) {
	switch {
	case method == http.MethodPost && path == "/v1/chat/completions":
		return localgateway.OperationChatCompletions, nil
	case method == http.MethodPost && path == "/v1/responses":
		return localgateway.OperationResponses, nil
	case method == http.MethodGet && path == "/v1/models":
		return localgateway.OperationModels, nil
	default:
		return "", fmt.Errorf("unsupported openai route: %s %s", method, path)
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
