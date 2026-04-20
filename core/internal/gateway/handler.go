package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/logging"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type ActiveProviderResolver interface {
	GetActive(ctx context.Context) (*provider.Provider, error)
}

type Handler struct {
	providers   ActiveProviderResolver
	credentials credential.Store
	logs        *logging.Service
}

func NewHandler(providers ActiveProviderResolver, credentials credential.Store, logs *logging.Service) *Handler {
	return &Handler{
		providers:   providers,
		credentials: credentials,
		logs:        logs,
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

	model := extractModel(r)
	recorder := newStatusRecorder(w)

	proxy := httputil.NewSingleHostReverseProxy(baseURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		req.URL.Path = joinURLPath(baseURL.Path, strings.TrimPrefix(r.URL.Path, "/v1"))
		req.URL.RawPath = req.URL.Path
		req.Host = baseURL.Host

		provider.ApplyCredentialHeaders(req, *activeProvider, apiKey, r.Header)
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, proxyErr error) {
		statusCode := http.StatusBadGateway
		message := proxyErr.Error()
		snippet := fmt.Sprintf(`{"error":"upstream_request_failed","message":%q}`, proxyErr.Error())
		recorder.errorMessage = &message
		recorder.errorSnippet = &snippet
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(statusCode)
		_ = json.NewEncoder(rw).Encode(map[string]string{
			"error":   "upstream_request_failed",
			"message": proxyErr.Error(),
		})
	}

	proxy.ServeHTTP(recorder, r)

	h.recordLog(r.Context(), logging.Entry{
		ProviderID:   activeProvider.ID,
		ProviderName: activeProvider.Name,
		Method:       r.Method,
		Path:         r.URL.Path,
		Model:        model,
		StatusCode:   recorder.statusCodePtr(),
		IsStream:     isStreamRequest(r),
		UpstreamHost: baseURL.Host,
		LatencyMs:    time.Since(startedAt).Milliseconds(),
		FirstByteMs:  recorder.firstByteMsPtr(startedAt),
		ErrorMessage: recorder.errorMessage,
		ErrorSnippet: recorder.errorSnippetOrBody(),
	})
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

func extractModel(r *http.Request) *string {
	if r.Body == nil {
		return nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}

	modelValue, ok := payload["model"].(string)
	if !ok || strings.TrimSpace(modelValue) == "" {
		return nil
	}

	return &modelValue
}

func isStreamRequest(r *http.Request) bool {
	if r.Body == nil {
		return false
	}
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/event-stream")
}

func (h *Handler) recordLog(ctx context.Context, entry logging.Entry) {
	if h.logs == nil {
		return
	}
	_ = h.logs.Record(ctx, entry)
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode    int
	wroteHeader   bool
	firstByteAt   time.Time
	snippetBuffer bytes.Buffer
	errorMessage  *string
	errorSnippet  *string
}

func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w}
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	if !r.wroteHeader {
		r.statusCode = statusCode
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	if r.firstByteAt.IsZero() {
		r.firstByteAt = time.Now()
	}
	if r.snippetBuffer.Len() < 200 {
		remaining := 200 - r.snippetBuffer.Len()
		if len(data) < remaining {
			remaining = len(data)
		}
		r.snippetBuffer.Write(data[:remaining])
	}
	return r.ResponseWriter.Write(data)
}

func (r *statusRecorder) statusCodePtr() *int {
	if r.statusCode == 0 {
		return nil
	}
	value := r.statusCode
	return &value
}

func (r *statusRecorder) firstByteMsPtr(startedAt time.Time) *int64 {
	if r.firstByteAt.IsZero() {
		return nil
	}
	value := r.firstByteAt.Sub(startedAt).Milliseconds()
	return &value
}

func (r *statusRecorder) snippetPtr() *string {
	if r.snippetBuffer.Len() == 0 {
		return nil
	}
	value := r.snippetBuffer.String()
	return &value
}

func (r *statusRecorder) errorSnippetOrBody() *string {
	if r.errorSnippet != nil {
		return r.errorSnippet
	}
	return r.snippetPtr()
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
