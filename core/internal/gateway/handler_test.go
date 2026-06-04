package gateway

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/xiaoyuandev/relay-switch/core/internal/credential"
	"github.com/xiaoyuandev/relay-switch/core/internal/logging"
	"github.com/xiaoyuandev/relay-switch/core/internal/provider"
	"github.com/xiaoyuandev/relay-switch/core/internal/storage"
)

func TestHandlerRecordsModelFromStreamingResponse(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.created\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.created","response":{"id":"resp_1","model":"chatgpt-5.5"}}` + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	logs := newGatewayTestLogs(t)
	credentialStore := credential.NewInMemoryStore()
	apiKeyRef, err := credentialStore.Save(context.Background(), "test", "secret")
	if err != nil {
		t.Fatalf("save credential: %v", err)
	}

	handler := NewHandler(
		gatewayTestResolver{
			active: provider.Provider{
				ID:        "provider-1",
				Name:      "Test Provider",
				BaseURL:   upstream.URL,
				APIKeyRef: apiKeyRef,
			},
		},
		credentialStore,
		logs,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"stream":true,"input":"hello"}`))
	req.Header.Set("Accept", "text/event-stream")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	items, err := logs.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one log, got %d", len(items))
	}
	if items[0].Model == nil || *items[0].Model != "chatgpt-5.5" {
		t.Fatalf("unexpected logged model: %+v", items[0].Model)
	}
}

func TestHandlerRecordsModelFromJSONResponse(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1","object":"response","model":"chatgpt-5.5"}`))
	}))
	defer upstream.Close()

	logs := newGatewayTestLogs(t)
	credentialStore := credential.NewInMemoryStore()
	apiKeyRef, err := credentialStore.Save(context.Background(), "test", "secret")
	if err != nil {
		t.Fatalf("save credential: %v", err)
	}

	handler := NewHandler(
		gatewayTestResolver{
			active: provider.Provider{
				ID:        "provider-1",
				Name:      "Test Provider",
				BaseURL:   upstream.URL,
				APIKeyRef: apiKeyRef,
			},
		},
		credentialStore,
		logs,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"input":"hello"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	items, err := logs.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one log, got %d", len(items))
	}
	if items[0].Model == nil || *items[0].Model != "chatgpt-5.5" {
		t.Fatalf("unexpected logged model: %+v", items[0].Model)
	}
}

func TestHandlerRecordsModelFromLateStreamingResponse(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: response.reasoning_summary_text.delta\n"))
		_, _ = w.Write([]byte("data: " + strings.Repeat("x", 80*1024) + "\n\n"))
		_, _ = w.Write([]byte("event: response.completed\n"))
		_, _ = w.Write([]byte(`data: {"type":"response.completed","response":{"id":"resp_1","model":"chatgpt-5.5-thinking-high"}}` + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	logs := newGatewayTestLogs(t)
	credentialStore := credential.NewInMemoryStore()
	apiKeyRef, err := credentialStore.Save(context.Background(), "test", "secret")
	if err != nil {
		t.Fatalf("save credential: %v", err)
	}

	handler := NewHandler(
		gatewayTestResolver{
			active: provider.Provider{
				ID:        "provider-1",
				Name:      "Test Provider",
				BaseURL:   upstream.URL,
				APIKeyRef: apiKeyRef,
			},
		},
		credentialStore,
		logs,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"stream":true,"input":"hello"}`))
	req.Header.Set("Accept", "text/event-stream")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	items, err := logs.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one log, got %d", len(items))
	}
	if items[0].Model == nil || *items[0].Model != "chatgpt-5.5-thinking-high" {
		t.Fatalf("unexpected logged model: %+v", items[0].Model)
	}
}

func TestHandlerRecordsModelSlugFromResponseMetadata(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1","object":"response","metadata":{"model_slug":"chatgpt-5.5-thinking-high"}}`))
	}))
	defer upstream.Close()

	logs := newGatewayTestLogs(t)
	credentialStore := credential.NewInMemoryStore()
	apiKeyRef, err := credentialStore.Save(context.Background(), "test", "secret")
	if err != nil {
		t.Fatalf("save credential: %v", err)
	}

	handler := NewHandler(
		gatewayTestResolver{
			active: provider.Provider{
				ID:        "provider-1",
				Name:      "Test Provider",
				BaseURL:   upstream.URL,
				APIKeyRef: apiKeyRef,
			},
		},
		credentialStore,
		logs,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"input":"hello"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	items, err := logs.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one log, got %d", len(items))
	}
	if items[0].Model == nil || *items[0].Model != "chatgpt-5.5-thinking-high" {
		t.Fatalf("unexpected logged model: %+v", items[0].Model)
	}
}

func TestHandlerDoesNotUseResponseModelForClaudeMessages(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"type":"message_start","message":{"model":"chatgpt-5.5"}}` + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	logs := newGatewayTestLogs(t)
	credentialStore := credential.NewInMemoryStore()
	apiKeyRef, err := credentialStore.Save(context.Background(), "test", "secret")
	if err != nil {
		t.Fatalf("save credential: %v", err)
	}

	handler := NewHandler(
		gatewayTestResolver{
			active: provider.Provider{
				ID:        "provider-1",
				Name:      "Claude Provider",
				BaseURL:   upstream.URL,
				APIKeyRef: apiKeyRef,
			},
		},
		credentialStore,
		logs,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`{"model":"claude-sonnet-4-5","stream":true,"messages":[]}`))
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("anthropic-version", "2023-06-01")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	items, err := logs.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one log, got %d", len(items))
	}
	if items[0].Model == nil || *items[0].Model != "claude-sonnet-4-5" {
		t.Fatalf("unexpected logged model: %+v", items[0].Model)
	}
}

func TestHandlerDecodesZstdEncodedRequestBeforeForwarding(t *testing.T) {
	t.Parallel()

	var upstreamBody string
	var upstreamContentEncoding string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read upstream body: %v", err)
		}
		upstreamBody = string(content)
		upstreamContentEncoding = r.Header.Get("Content-Encoding")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1","object":"response"}`))
	}))
	defer upstream.Close()

	logs := newGatewayTestLogs(t)
	credentialStore := credential.NewInMemoryStore()
	apiKeyRef, err := credentialStore.Save(context.Background(), "test", "secret")
	if err != nil {
		t.Fatalf("save credential: %v", err)
	}

	handler := NewHandler(
		gatewayTestResolver{
			active: provider.Provider{
				ID:        "provider-local-gateway",
				Name:      "Local Gateway",
				BaseURL:   upstream.URL,
				APIKeyRef: apiKeyRef,
			},
		},
		credentialStore,
		logs,
	)

	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatalf("create zstd encoder: %v", err)
	}
	defer encoder.Close()
	compressedBody := encoder.EncodeAll([]byte(`{"model":"deepseek-v4-flash","input":"hello","stream":true}`), nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(compressedBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "zstd")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if upstreamContentEncoding != "" {
		t.Fatalf("expected upstream content encoding to be cleared, got %q", upstreamContentEncoding)
	}
	if upstreamBody != `{"input":"hello","model":"deepseek-v4-flash","stream":true}` &&
		upstreamBody != `{"model":"deepseek-v4-flash","input":"hello","stream":true}` {
		t.Fatalf("expected upstream body to be decoded JSON, got %s", upstreamBody)
	}

	items, err := logs.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one log, got %d", len(items))
	}
	if items[0].Model == nil || *items[0].Model != "deepseek-v4-flash" {
		t.Fatalf("unexpected logged model: %+v", items[0].Model)
	}
}

func TestDecodeRequestBodyGzip(t *testing.T) {
	t.Parallel()

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, _ = writer.Write([]byte(`{"model":"gpt-4.1-mini","messages":[]}`))
	if err := writer.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}

	body, err := decodeRequestBody(compressed.Bytes(), "gzip")
	if err != nil {
		t.Fatalf("decode gzip body: %v", err)
	}
	if string(body) != `{"model":"gpt-4.1-mini","messages":[]}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestDecodeRequestBodyRejectsUnsupportedEncoding(t *testing.T) {
	t.Parallel()

	if _, err := decodeRequestBody([]byte("{}"), "br"); err == nil {
		t.Fatal("expected unsupported encoding error")
	}
}

func newGatewayTestLogs(t *testing.T) *logging.Service {
	t.Helper()

	sqliteStore, err := storage.NewSQLite(filepath.Join(t.TempDir(), "gateway.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = sqliteStore.Close() })

	return logging.NewService(logging.NewSQLiteRepository(sqliteStore.DB), 7, 1000)
}

type gatewayTestResolver struct {
	active   provider.Provider
	selected []provider.SelectedModel
}

func (r gatewayTestResolver) GetActive(context.Context) (*provider.Provider, error) {
	return &r.active, nil
}

func (r gatewayTestResolver) ListSelectedModels(context.Context, string) ([]provider.SelectedModel, error) {
	return r.selected, nil
}
