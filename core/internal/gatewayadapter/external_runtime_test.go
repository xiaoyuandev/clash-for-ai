package gatewayadapter

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type externalRoundTripFunc func(*http.Request) (*http.Response, error)

func (f externalRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestExternalRuntimeAdapterDiscover(t *testing.T) {
	adapter := NewExternalRuntimeAdapter("http://127.0.0.1:8788/")

	info, err := adapter.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	if info.BaseURL != "http://127.0.0.1:8788" {
		t.Fatalf("unexpected base url: %s", info.BaseURL)
	}
	if info.Mode != "external" || info.Embedded {
		t.Fatalf("unexpected runtime info: %+v", info)
	}
}

func TestExternalRuntimeAdapterCapabilities(t *testing.T) {
	adapter := NewExternalRuntimeAdapter("http://127.0.0.1:8788")
	adapter.client = &http.Client{
		Transport: externalRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/v1/chat/completions", "/v1/messages", "/v1/models":
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			case "/admin/model-sources", "/admin/selected-models":
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			default:
				t.Fatalf("unexpected request path: %s", req.URL.Path)
				return nil, nil
			}
		}),
	}

	capabilities, err := adapter.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities returned error: %v", err)
	}

	if !capabilities.SupportsOpenAICompatible || !capabilities.SupportsAnthropicCompatible {
		t.Fatalf("unexpected protocol capabilities: %+v", capabilities)
	}
	if capabilities.SupportsAdminAPI || capabilities.SupportsModelSourceAdmin || capabilities.SupportsSelectedModelAdmin {
		t.Fatalf("external runtime should not expose embedded admin capabilities: %+v", capabilities)
	}
	if missing := capabilities.MissingRequiredCapabilities(); len(missing) != 0 {
		t.Fatalf("unexpected missing required capabilities: %v", missing)
	}
	if missing := capabilities.MissingOptionalCapabilities(); len(missing) != 3 {
		t.Fatalf("unexpected missing optional capabilities: %v", missing)
	}
}

func TestExternalRuntimeAdapterAdminUnsupported(t *testing.T) {
	adapter := NewExternalRuntimeAdapter("http://127.0.0.1:8788")

	if _, err := adapter.ListSelectedModels(context.Background()); !errors.Is(err, ErrRuntimeAdminUnsupported) {
		t.Fatalf("expected ErrRuntimeAdminUnsupported, got %v", err)
	}
}

func TestExternalRuntimeAdapterHealthFallsBackToModels(t *testing.T) {
	adapter := NewExternalRuntimeAdapter("http://127.0.0.1:8788")
	adapter.client = &http.Client{
		Transport: externalRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/health":
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			case "/v1/chat/completions", "/v1/messages", "/v1/models":
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			case "/admin/model-sources", "/admin/selected-models":
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			default:
				t.Fatalf("unexpected request path: %s", req.URL.Path)
				return nil, nil
			}
		}),
	}

	health, err := adapter.CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if health.Status != "ok" {
		t.Fatalf("unexpected health: %+v", health)
	}
}
