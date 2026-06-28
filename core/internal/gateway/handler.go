package gateway

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/xiaoyuandev/relay-switch/core/internal/credential"
	"github.com/xiaoyuandev/relay-switch/core/internal/logging"
	"github.com/xiaoyuandev/relay-switch/core/internal/provider"
)

type ActiveProviderResolver interface {
	GetActive(ctx context.Context) (*provider.Provider, error)
	ListSelectedModels(ctx context.Context, id string) ([]provider.SelectedModel, error)
}

type Handler struct {
	providers    ActiveProviderResolver
	credentials  credential.Store
	logs         *logging.Service
	client       *http.Client
	streamClient *http.Client
}

func NewHandler(providers ActiveProviderResolver, credentials credential.Store, logs *logging.Service) *Handler {
	transport := http.DefaultTransport

	return &Handler{
		providers:   providers,
		credentials: credentials,
		logs:        logs,
		client: &http.Client{
			Transport: transport,
			Timeout:   120 * time.Second,
		},
		streamClient: &http.Client{
			Transport: transport,
		},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()

	if r.URL.Path != "/v1" && !strings.HasPrefix(r.URL.Path, "/v1/") {
		http.NotFound(w, r)
		return
	}

	activeProvider, err := h.providers.GetActive(r.Context())
	if err != nil {
		http.Error(w, "failed to resolve active provider", http.StatusInternalServerError)
		return
	}

	if activeProvider == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		payload := map[string]string{
			"error": "no_active_provider",
		}
		_ = json.NewEncoder(w).Encode(payload)
		message := "no_active_provider"
		statusCode := http.StatusBadGateway
		snippet := `{"error":"no_active_provider"}`
		h.recordLog(r.Context(), logging.Entry{
			Method:       r.Method,
			Path:         r.URL.Path,
			StatusCode:   &statusCode,
			LatencyMs:    time.Since(startedAt).Milliseconds(),
			ErrorMessage: &message,
			ErrorSnippet: &snippet,
		})
		return
	}

	baseURL, err := url.Parse(activeProvider.BaseURL)
	if err != nil {
		http.Error(w, "invalid provider base_url", http.StatusBadGateway)
		return
	}

	apiKey, err := h.credentials.Get(r.Context(), activeProvider.APIKeyRef)
	if err != nil {
		http.Error(w, "failed to load provider credential", http.StatusInternalServerError)
		message := err.Error()
		statusCode := http.StatusInternalServerError
		h.recordLog(r.Context(), logging.Entry{
			ProviderID:   activeProvider.ID,
			ProviderName: activeProvider.Name,
			Method:       r.Method,
			Path:         r.URL.Path,
			StatusCode:   &statusCode,
			UpstreamHost: baseURL.Host,
			LatencyMs:    time.Since(startedAt).Milliseconds(),
			ErrorMessage: &message,
		})
		return
	}

	body, bodyErr := readRequestBody(r)
	if bodyErr != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	selectedModels, err := h.providers.ListSelectedModels(r.Context(), activeProvider.ID)
	if err != nil {
		http.Error(w, "failed to load selected models", http.StatusInternalServerError)
		return
	}

	attempts, model := buildModelAttempts(r, body, selectedModels, activeProvider.ClaudeCodeModelMap)
	result := h.forwardWithFallback(r.Context(), forwardInput{
		baseURL:        baseURL,
		activeProvider: *activeProvider,
		apiKey:         apiKey,
		originalHeader: r.Header,
		method:         r.Method,
		path:           r.URL.Path,
		rawQuery:       r.URL.RawQuery,
		isStream:       isStreamingRequest(r, body),
		attempts:       attempts,
	})

	if result.networkError != nil {
		statusCode := http.StatusBadGateway
		message := result.networkError.Error()
		snippet := fmt.Sprintf(`{"error":"upstream_request_failed","message":%q}`, message)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(snippet))
		h.recordLog(r.Context(), logging.Entry{
			ProviderID:   activeProvider.ID,
			ProviderName: activeProvider.Name,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        result.finalModel,
			StatusCode:   &statusCode,
			IsStream:     isStreamingRequest(r, body),
			UpstreamHost: baseURL.Host,
			LatencyMs:    time.Since(startedAt).Milliseconds(),
			ErrorMessage: &message,
			ErrorSnippet: &snippet,
		})
		return
	}

	shouldExtractResponseModel := result.finalModel == nil && model == nil && r.URL.Path == "/v1/responses"
	responseModel := writeResponse(w, result, shouldExtractResponseModel)

	h.recordLog(r.Context(), logging.Entry{
		ProviderID:   activeProvider.ID,
		ProviderName: activeProvider.Name,
		Method:       r.Method,
		Path:         r.URL.Path,
		Model:        chooseLogModel(result.finalModel, model, responseModel),
		StatusCode:   intPtr(result.statusCode),
		IsStream:     isStreamingRequest(r, body),
		UpstreamHost: baseURL.Host,
		LatencyMs:    time.Since(startedAt).Milliseconds(),
		FirstByteMs:  result.firstByteMsPtr(startedAt),
		ErrorMessage: result.errorMessage,
		ErrorSnippet: result.errorSnippet,
	})
}

func resolveUpstreamPath(basePath string, requestPath string) string {
	trimmedBase := strings.TrimRight(basePath, "/")
	suffix := strings.TrimPrefix(requestPath, "/v1")

	if trimmedBase == "" {
		return "/v1" + suffix
	}
	if strings.HasSuffix(trimmedBase, "/v1") {
		return trimmedBase + suffix
	}
	return trimmedBase + "/v1" + suffix
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

func (h *Handler) recordLog(ctx context.Context, entry logging.Entry) {
	if h.logs == nil {
		return
	}
	_ = h.logs.Record(ctx, entry)
}

type attemptSpec struct {
	model *string
	body  []byte
}

type forwardInput struct {
	baseURL        *url.URL
	activeProvider provider.Provider
	apiKey         string
	originalHeader http.Header
	method         string
	path           string
	rawQuery       string
	isStream       bool
	attempts       []attemptSpec
}

type forwardResult struct {
	statusCode   int
	header       http.Header
	body         []byte
	streamBody   io.ReadCloser
	firstByteAt  time.Time
	errorMessage *string
	errorSnippet *string
	finalModel   *string
	networkError error
}

func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	body, err := decodeRequestBody(rawBody, r.Header.Get("Content-Encoding"))
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	r.ContentLength = int64(len(body))
	r.Header.Del("Content-Encoding")
	r.Header.Set("Content-Length", strconv.Itoa(len(body)))
	return body, nil
}

func decodeRequestBody(body []byte, contentEncoding string) ([]byte, error) {
	encodings := contentEncodings(contentEncoding)
	if len(encodings) == 0 {
		return body, nil
	}

	decoded := body
	for index := len(encodings) - 1; index >= 0; index-- {
		switch encoding := encodings[index]; encoding {
		case "", "identity":
			continue
		case "gzip", "x-gzip":
			reader, err := gzip.NewReader(bytes.NewReader(decoded))
			if err != nil {
				return nil, fmt.Errorf("decode gzip request body: %w", err)
			}
			next, err := io.ReadAll(reader)
			closeErr := reader.Close()
			if err != nil {
				return nil, fmt.Errorf("read gzip request body: %w", err)
			}
			if closeErr != nil {
				return nil, fmt.Errorf("close gzip request body: %w", closeErr)
			}
			decoded = next
		case "zstd", "zstandard":
			reader, err := zstd.NewReader(bytes.NewReader(decoded))
			if err != nil {
				return nil, fmt.Errorf("decode zstd request body: %w", err)
			}
			next, err := io.ReadAll(reader)
			reader.Close()
			if err != nil {
				return nil, fmt.Errorf("read zstd request body: %w", err)
			}
			decoded = next
		default:
			return nil, fmt.Errorf("unsupported request content encoding: %s", encoding)
		}
	}

	return decoded, nil
}

func contentEncodings(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	encodings := make([]string, 0, len(parts))
	for _, part := range parts {
		encoding := strings.ToLower(strings.TrimSpace(part))
		if encoding == "" || encoding == "identity" {
			continue
		}
		encodings = append(encodings, encoding)
	}
	return encodings
}

func buildModelAttempts(
	r *http.Request,
	body []byte,
	selected []provider.SelectedModel,
	claudeCodeModels provider.ClaudeCodeModelMap,
) ([]attemptSpec, *string) {
	currentModel, payload := extractModelFromBody(body)
	if rewritten := buildClaudeCodeModelAttempt(r, currentModel, payload, body, claudeCodeModels); rewritten != nil {
		return rewritten, currentModel
	}

	if len(selected) == 0 || payload == nil || r.Method != http.MethodPost {
		return []attemptSpec{{model: currentModel, body: body}}, currentModel
	}

	orderedModels := make([]string, 0, len(selected))
	for _, item := range selected {
		orderedModels = append(orderedModels, item.ModelID)
	}

	startIndex := 0
	if currentModel != nil {
		found := -1
		for index, modelID := range orderedModels {
			if modelID == *currentModel {
				found = index
				break
			}
		}
		if found < 0 {
			return []attemptSpec{{model: currentModel, body: body}}, currentModel
		}
		startIndex = found
	}

	attempts := make([]attemptSpec, 0, len(orderedModels)-startIndex)
	for _, modelID := range orderedModels[startIndex:] {
		attemptModel := modelID
		updatedBody := bodyWithModel(payload, modelID, body)
		attempts = append(attempts, attemptSpec{
			model: &attemptModel,
			body:  updatedBody,
		})
	}

	return attempts, currentModel
}

func buildClaudeCodeModelAttempt(
	r *http.Request,
	currentModel *string,
	payload map[string]any,
	fallback []byte,
	claudeCodeModels provider.ClaudeCodeModelMap,
) []attemptSpec {
	if currentModel == nil || payload == nil || r.Method != http.MethodPost {
		return nil
	}

	targetModel := resolveClaudeCodeTargetModel(r, *currentModel, claudeCodeModels)
	if targetModel == "" || targetModel == *currentModel {
		return nil
	}

	updatedBody := bodyWithModel(payload, targetModel, fallback)
	return []attemptSpec{
		{
			model: stringPtr(targetModel),
			body:  updatedBody,
		},
	}
}

func resolveClaudeCodeTargetModel(
	r *http.Request,
	requestModel string,
	claudeCodeModels provider.ClaudeCodeModelMap,
) string {
	if !isAnthropicModelRequest(r) {
		return ""
	}

	normalized := strings.ToLower(strings.TrimSpace(requestModel))
	switch {
	case strings.Contains(normalized, "haiku"):
		return strings.TrimSpace(claudeCodeModels.Haiku)
	case strings.Contains(normalized, "sonnet"):
		return strings.TrimSpace(claudeCodeModels.Sonnet)
	case strings.Contains(normalized, "opus"):
		return strings.TrimSpace(claudeCodeModels.Opus)
	default:
		return ""
	}
}

func isAnthropicModelRequest(r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("anthropic-version")) != "" {
		return true
	}

	switch r.URL.Path {
	case "/v1/messages", "/v1/complete", "/v1/complete/stream":
		return true
	default:
		return false
	}
}

func extractModelFromBody(body []byte) (*string, map[string]any) {
	if len(body) == 0 {
		return nil, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, nil
	}

	modelValue, ok := payload["model"].(string)
	if !ok || strings.TrimSpace(modelValue) == "" {
		return nil, payload
	}

	return &modelValue, payload
}

func extractStreamFlag(body []byte) *bool {
	if len(body) == 0 {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}

	value, ok := payload["stream"].(bool)
	if !ok {
		return nil
	}

	return &value
}

func isStreamingRequest(r *http.Request, body []byte) bool {
	if strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/event-stream") {
		return true
	}

	streamFlag := extractStreamFlag(body)
	return streamFlag != nil && *streamFlag
}

func bodyWithModel(payload map[string]any, modelID string, fallback []byte) []byte {
	clone := make(map[string]any, len(payload))
	for key, value := range payload {
		clone[key] = value
	}
	clone["model"] = modelID

	encoded, err := json.Marshal(clone)
	if err != nil {
		return fallback
	}
	return encoded
}

func (h *Handler) forwardWithFallback(ctx context.Context, input forwardInput) forwardResult {
	var lastResult forwardResult

	for index, attempt := range input.attempts {
		reqURL := *input.baseURL
		if input.path == "/v1/models" {
			reqURL.Path = provider.ResolveModelsPath(input.baseURL.Path, input.activeProvider.ModelsPath)
		} else {
			reqURL.Path = resolveUpstreamPath(input.baseURL.Path, input.path)
		}
		reqURL.RawPath = reqURL.Path
		reqURL.RawQuery = input.rawQuery

		req, err := http.NewRequestWithContext(ctx, input.method, reqURL.String(), bytes.NewReader(attempt.body))
		if err != nil {
			lastResult.networkError = err
			lastResult.finalModel = attempt.model
			return lastResult
		}

		req.Header = cloneHeader(input.originalHeader)
		req.Header.Del("Content-Encoding")
		req.Host = input.baseURL.Host
		req.ContentLength = int64(len(attempt.body))
		if len(attempt.body) > 0 {
			req.Header.Set("Content-Length", strconv.Itoa(len(attempt.body)))
		}
		provider.ApplyCredentialHeaders(req, input.activeProvider, input.apiKey, input.originalHeader)

		httpClient := h.client
		if input.isStream {
			httpClient = h.streamClient
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			lastResult.finalModel = attempt.model
			if index < len(input.attempts)-1 {
				continue
			}
			lastResult.networkError = err
			return lastResult
		}

		firstByteAt := time.Now()

		if input.isStream && resp.StatusCode < 400 {
			lastResult = forwardResult{
				statusCode:  resp.StatusCode,
				header:      resp.Header.Clone(),
				streamBody:  resp.Body,
				firstByteAt: firstByteAt,
				finalModel:  attempt.model,
			}
			return lastResult
		}

		responseBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastResult.finalModel = attempt.model
			lastResult.networkError = readErr
			return lastResult
		}

		lastResult = forwardResult{
			statusCode:  resp.StatusCode,
			header:      resp.Header.Clone(),
			body:        responseBody,
			firstByteAt: firstByteAt,
			finalModel:  attempt.model,
		}

		if resp.StatusCode < 400 {
			return lastResult
		}

		message := strings.TrimSpace(string(responseBody))
		if message != "" {
			lastResult.errorMessage = &message
			lastResult.errorSnippet = &message
		}

		if !isRetryableStatus(resp.StatusCode) || index == len(input.attempts)-1 {
			return lastResult
		}
	}

	return lastResult
}

func writeResponse(w http.ResponseWriter, result forwardResult, extractResponseModel bool) *string {
	copyHeaders(w.Header(), result.header)
	w.WriteHeader(result.statusCode)

	if result.streamBody != nil {
		defer result.streamBody.Close()

		sniffer := &streamModelSniffer{}
		if flusher, ok := w.(http.Flusher); ok {
			writer := io.Writer(w)
			if extractResponseModel {
				writer = sniffWriter{writer: writer, sniffer: sniffer}
			}
			_, _ = io.Copy(flushWriter{writer: writer, flusher: flusher}, result.streamBody)
			flusher.Flush()
			return sniffer.Model()
		}

		writer := io.Writer(w)
		if extractResponseModel {
			writer = sniffWriter{writer: writer, sniffer: sniffer}
		}
		_, _ = io.Copy(writer, result.streamBody)
		return sniffer.Model()
	}

	if len(result.body) > 0 {
		_, _ = w.Write(result.body)
	}
	if !extractResponseModel {
		return nil
	}
	return extractModelFromResponseBody(result.body)
}

type sniffWriter struct {
	writer  io.Writer
	sniffer *streamModelSniffer
}

func (w sniffWriter) Write(p []byte) (int, error) {
	w.sniffer.Write(p)
	return w.writer.Write(p)
}

type streamModelSniffer struct {
	pending string
	model   *string
}

func (s *streamModelSniffer) Write(p []byte) {
	if s == nil || s.model != nil {
		return
	}

	s.pending += string(p)
	for {
		index := strings.IndexByte(s.pending, '\n')
		if index < 0 {
			const maxPendingLineBytes = 1024 * 1024
			if len(s.pending) > maxPendingLineBytes {
				s.pending = s.pending[len(s.pending)-maxPendingLineBytes:]
			}
			return
		}

		line := s.pending[:index]
		s.pending = s.pending[index+1:]
		if model := extractModelFromStreamLine(line); model != nil {
			s.model = model
			return
		}
	}
}

func (s *streamModelSniffer) Model() *string {
	if s == nil || s.model != nil {
		return s.model
	}
	if s.pending == "" {
		return nil
	}
	return extractModelFromStreamLine(s.pending)
}

type flushWriter struct {
	writer  io.Writer
	flusher http.Flusher
}

func (w flushWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if err == nil {
		w.flusher.Flush()
	}
	return n, err
}

func isRetryableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func cloneHeader(source http.Header) http.Header {
	cloned := make(http.Header, len(source))
	for key, values := range source {
		nextValues := make([]string, len(values))
		copy(nextValues, values)
		cloned[key] = nextValues
	}
	return cloned
}

func copyHeaders(target http.Header, source http.Header) {
	for key := range target {
		target.Del(key)
	}
	for key, values := range source {
		for _, value := range values {
			target.Add(key, value)
		}
	}
}

func extractModelFromResponseBody(body []byte) *string {
	if len(body) == 0 {
		return nil
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	return responseModelID(payload)
}

func extractModelFromStreamBuffer(buffer []byte) *string {
	if len(buffer) == 0 {
		return nil
	}

	for _, line := range strings.Split(string(buffer), "\n") {
		if model := extractModelFromStreamLine(line); model != nil {
			return model
		}
	}

	return nil
}

func extractModelFromStreamLine(line string) *string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "data:") {
		return nil
	}

	data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if data == "" || data == "[DONE]" {
		return nil
	}

	return extractModelFromResponseBody([]byte(data))
}

func responseModelID(value any) *string {
	payload, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	if model := modelField(payload); model != nil {
		return model
	}

	for _, key := range []string{"response", "message", "metadata"} {
		nested, ok := payload[key].(map[string]any)
		if !ok {
			continue
		}
		if model := responseModelID(nested); model != nil {
			return model
		}
	}

	return nil
}

func modelField(payload map[string]any) *string {
	for _, key := range []string{"model", "model_id", "model_slug"} {
		if model := stringField(payload, key); model != nil {
			return model
		}
	}
	return nil
}

func stringField(payload map[string]any, key string) *string {
	value, ok := payload[key].(string)
	if !ok {
		return nil
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func chooseLogModel(models ...*string) *string {
	for _, model := range models {
		if model != nil && strings.TrimSpace(*model) != "" {
			return model
		}
	}
	return nil
}

func intPtr(value int) *int {
	if value == 0 {
		return nil
	}
	next := value
	return &next
}

func stringPtr(value string) *string {
	next := value
	return &next
}

func (r *forwardResult) firstByteMsPtr(startedAt time.Time) *int64 {
	if r.firstByteAt.IsZero() {
		return nil
	}
	value := r.firstByteAt.Sub(startedAt).Milliseconds()
	return &value
}
