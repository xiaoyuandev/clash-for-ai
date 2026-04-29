package gatewayadapter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
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
	adapter.client = &http.Client{
		Transport: externalRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotImplemented,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		}),
	}
	adapter.admin.client = adapter.client

	if _, err := adapter.ListSelectedModels(context.Background()); !errors.Is(err, ErrRuntimeAdminUnsupported) {
		t.Fatalf("expected ErrRuntimeAdminUnsupported, got %v", err)
	}
}

func TestExternalRuntimeAdapterAdminOperations(t *testing.T) {
	selectedModels := []provider.SelectedModel{{ModelID: "gpt-4.1", Position: 0}}
	modelSources := []modelsource.Source{{
		ID:             "source-1",
		Name:           "OpenAI",
		BaseURL:        "https://example.com",
		ProviderType:   "openai-compatible",
		DefaultModelID: "gpt-4.1",
		Enabled:        true,
		Position:       0,
	}}

	adapter := NewExternalRuntimeAdapter("http://127.0.0.1:8788")
	adapter.client = &http.Client{
		Transport: externalRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			header := http.Header{"Content-Type": []string{"application/json"}}
			switch req.Method + " " + req.URL.Path {
			case http.MethodGet + " /admin/model-sources":
				payload, err := json.Marshal(modelSources)
				if err != nil {
					t.Fatalf("marshal model sources: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(string(payload))),
					Header:     header,
				}, nil
			case http.MethodPost + " /admin/model-sources":
				var input modelsource.CreateInput
				if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
					t.Fatalf("decode create input: %v", err)
				}
				payload, err := json.Marshal(modelsource.Source{
					ID:             "created",
					Name:           input.Name,
					BaseURL:        input.BaseURL,
					ProviderType:   input.ProviderType,
					DefaultModelID: input.DefaultModelID,
					Enabled:        input.Enabled,
				})
				if err != nil {
					t.Fatalf("marshal created source: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(string(payload))),
					Header:     header,
				}, nil
			case http.MethodPut + " /admin/model-sources/source-1":
				var input modelsource.UpdateInput
				if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
					t.Fatalf("decode update input: %v", err)
				}
				payload, err := json.Marshal(modelsource.Source{
					ID:             "source-1",
					Name:           input.Name,
					BaseURL:        input.BaseURL,
					ProviderType:   input.ProviderType,
					DefaultModelID: input.DefaultModelID,
					Enabled:        input.Enabled,
				})
				if err != nil {
					t.Fatalf("marshal updated source: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(string(payload))),
					Header:     header,
				}, nil
			case http.MethodDelete + " /admin/model-sources/source-1":
				return &http.Response{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     header,
				}, nil
			case http.MethodPut + " /admin/model-sources/order":
				var input []modelsource.Source
				if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
					t.Fatalf("decode reorder input: %v", err)
				}
				payload, err := json.Marshal(input)
				if err != nil {
					t.Fatalf("marshal reordered sources: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(string(payload))),
					Header:     header,
				}, nil
			case http.MethodGet + " /admin/selected-models":
				payload, err := json.Marshal(selectedModels)
				if err != nil {
					t.Fatalf("marshal selected models: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(string(payload))),
					Header:     header,
				}, nil
			case http.MethodPut + " /admin/selected-models":
				var input []provider.SelectedModel
				if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
					t.Fatalf("decode selected models input: %v", err)
				}
				payload, err := json.Marshal(input)
				if err != nil {
					t.Fatalf("marshal selected models response: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(string(payload))),
					Header:     header,
				}, nil
			default:
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				return nil, nil
			}
		}),
	}
	adapter.admin.client = adapter.client

	gotSources, err := adapter.ListModelSources(context.Background())
	if err != nil {
		t.Fatalf("ListModelSources returned error: %v", err)
	}
	if len(gotSources) != 1 || gotSources[0].ID != "source-1" {
		t.Fatalf("unexpected model sources: %+v", gotSources)
	}

	created, err := adapter.CreateModelSource(context.Background(), modelsource.CreateInput{
		Name:           "Created",
		BaseURL:        "https://created.example.com",
		ProviderType:   "openai-compatible",
		DefaultModelID: "gpt-4.1",
		Enabled:        true,
		APIKey:         "secret",
	})
	if err != nil {
		t.Fatalf("CreateModelSource returned error: %v", err)
	}
	if created.ID != "created" || created.Name != "Created" {
		t.Fatalf("unexpected created source: %+v", created)
	}

	updated, err := adapter.UpdateModelSource(context.Background(), "source-1", modelsource.UpdateInput{
		Name:           "Updated",
		BaseURL:        "https://updated.example.com",
		ProviderType:   "openai-compatible",
		DefaultModelID: "gpt-4.1-mini",
		Enabled:        false,
	})
	if err != nil {
		t.Fatalf("UpdateModelSource returned error: %v", err)
	}
	if updated.Name != "Updated" || updated.DefaultModelID != "gpt-4.1-mini" {
		t.Fatalf("unexpected updated source: %+v", updated)
	}

	if err := adapter.DeleteModelSource(context.Background(), "source-1"); err != nil {
		t.Fatalf("DeleteModelSource returned error: %v", err)
	}

	reordered, err := adapter.ReplaceModelSourceOrder(context.Background(), modelSources)
	if err != nil {
		t.Fatalf("ReplaceModelSourceOrder returned error: %v", err)
	}
	if len(reordered) != 1 || reordered[0].ID != "source-1" {
		t.Fatalf("unexpected reordered sources: %+v", reordered)
	}

	gotSelected, err := adapter.ListSelectedModels(context.Background())
	if err != nil {
		t.Fatalf("ListSelectedModels returned error: %v", err)
	}
	if len(gotSelected) != 1 || gotSelected[0].ModelID != "gpt-4.1" {
		t.Fatalf("unexpected selected models: %+v", gotSelected)
	}

	replacedSelected, err := adapter.ReplaceSelectedModels(context.Background(), selectedModels)
	if err != nil {
		t.Fatalf("ReplaceSelectedModels returned error: %v", err)
	}
	if len(replacedSelected) != 1 || replacedSelected[0].ModelID != "gpt-4.1" {
		t.Fatalf("unexpected replaced selected models: %+v", replacedSelected)
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
