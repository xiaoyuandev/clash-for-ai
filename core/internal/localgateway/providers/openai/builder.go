package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
)

func BuildRequest(request localgateway.Request, target localgateway.UpstreamTarget) (localgateway.UpstreamRequest, error) {
	if strings.TrimSpace(target.BaseURL) == "" {
		return localgateway.UpstreamRequest{}, fmt.Errorf("target base url is required")
	}

	baseURL, err := url.Parse(strings.TrimSpace(target.BaseURL))
	if err != nil {
		return localgateway.UpstreamRequest{}, fmt.Errorf("parse target base url: %w", err)
	}

	requestPath, err := resolvePath(request.Operation)
	if err != nil {
		return localgateway.UpstreamRequest{}, err
	}

	upstreamURL := *baseURL
	upstreamURL.Path = joinURLPath(baseURL.Path, requestPath)
	upstreamURL.RawPath = upstreamURL.Path

	body := request.Body
	if request.Method == http.MethodPost && strings.TrimSpace(target.DefaultModelID) != "" {
		body = rewriteModel(body, target.DefaultModelID)
	}

	headers := cloneHeaders(request.Headers)
	headers.Set("Content-Type", "application/json")
	headers.Set("Authorization", "Bearer "+strings.TrimSpace(target.APIKey))

	return localgateway.UpstreamRequest{
		Method:  request.Method,
		URL:     upstreamURL.String(),
		Headers: headers,
		Body:    body,
	}, nil
}

func resolvePath(operation localgateway.Operation) (string, error) {
	switch operation {
	case localgateway.OperationChatCompletions:
		return "/chat/completions", nil
	case localgateway.OperationResponses:
		return "/responses", nil
	case localgateway.OperationModels:
		return "/models", nil
	default:
		return "", fmt.Errorf("unsupported openai upstream operation: %s", operation)
	}
}

func rewriteModel(body []byte, modelID string) []byte {
	if len(body) == 0 {
		return body
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}

	payload["model"] = modelID
	encoded, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return encoded
}

func cloneHeaders(input map[string]string) http.Header {
	headers := make(http.Header, len(input))
	for key, value := range input {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "host", "content-length":
			continue
		default:
			headers.Set(key, value)
		}
	}
	return headers
}

func joinURLPath(basePath string, requestPath string) string {
	switch {
	case basePath == "":
		if requestPath == "" {
			return "/"
		}
		return requestPath
	case requestPath == "":
		return basePath
	case strings.HasSuffix(basePath, "/") && strings.HasPrefix(requestPath, "/"):
		return basePath + strings.TrimPrefix(requestPath, "/")
	case !strings.HasSuffix(basePath, "/") && !strings.HasPrefix(requestPath, "/"):
		return basePath + "/" + requestPath
	default:
		return basePath + requestPath
	}
}
