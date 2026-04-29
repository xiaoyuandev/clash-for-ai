package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
)

type stubModelSourceReader struct {
	items []modelsource.Source
}

func (r stubModelSourceReader) List(context.Context) ([]modelsource.Source, error) {
	return r.items, nil
}

type stubLocalRuntimeState struct {
	items []SelectedModel
}

func (s *stubLocalRuntimeState) ListSelectedModels(context.Context) ([]SelectedModel, error) {
	return s.items, nil
}

func (s *stubLocalRuntimeState) ReplaceSelectedModels(_ context.Context, items []SelectedModel) ([]SelectedModel, error) {
	s.items = items
	return items, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestEnsureSystemLocalGatewayCreatesImmutableProvider(t *testing.T) {
	service := NewService(NewInMemoryRepository(), credential.NewInMemoryStore(), stubModelSourceReader{})

	item, err := service.EnsureSystemLocalGateway(context.Background(), "http://127.0.0.1:3456")
	if err != nil {
		t.Fatalf("EnsureSystemLocalGateway returned error: %v", err)
	}

	if item.ID != LocalGatewayProviderID {
		t.Fatalf("unexpected provider id: %s", item.ID)
	}
	if item.Name != LocalGatewayProviderName {
		t.Fatalf("unexpected provider name: %s", item.Name)
	}
	if !item.IsSystem {
		t.Fatalf("expected system provider")
	}
	if !item.IsImmutable {
		t.Fatalf("expected immutable provider")
	}
	if !item.Status.IsActive {
		t.Fatalf("expected local gateway provider to be active by default")
	}
}

func TestUpdateAndDeleteRejectImmutableProvider(t *testing.T) {
	service := NewService(NewInMemoryRepository(), credential.NewInMemoryStore(), stubModelSourceReader{})

	if _, err := service.EnsureSystemLocalGateway(context.Background(), "http://127.0.0.1:3456"); err != nil {
		t.Fatalf("EnsureSystemLocalGateway returned error: %v", err)
	}

	_, updateErr := service.Update(context.Background(), LocalGatewayProviderID, UpdateInput{
		Name:    "Changed",
		BaseURL: "http://127.0.0.1:9999",
	})
	if !errors.Is(updateErr, ErrProviderImmutable) {
		t.Fatalf("expected ErrProviderImmutable from Update, got %v", updateErr)
	}

	deleteErr := service.Delete(context.Background(), LocalGatewayProviderID)
	if !errors.Is(deleteErr, ErrProviderImmutable) {
		t.Fatalf("expected ErrProviderImmutable from Delete, got %v", deleteErr)
	}
}

func TestFetchModelsForSystemLocalGatewayUsesRuntimeEndpoint(t *testing.T) {
	service := NewService(
		NewInMemoryRepository(),
		credential.NewInMemoryStore(),
		stubModelSourceReader{},
	)
	service.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "http://127.0.0.1:8788/v1/models" {
				t.Fatalf("unexpected request url: %s", req.URL.String())
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"gpt-4.1","owned_by":"openai-compatible"},{"id":"claude-sonnet-4","owned_by":"anthropic-compatible"}]}`)),
			}, nil
		}),
	}

	if _, err := service.EnsureSystemLocalGateway(context.Background(), "http://127.0.0.1:8788"); err != nil {
		t.Fatalf("EnsureSystemLocalGateway returned error: %v", err)
	}

	items, err := service.FetchModels(context.Background(), LocalGatewayProviderID)
	if err != nil {
		t.Fatalf("FetchModels returned error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 exposed models, got %d", len(items))
	}
	if items[0].ID != "gpt-4.1" || items[0].OwnedBy != "openai-compatible" {
		t.Fatalf("unexpected first model: %+v", items[0])
	}
	if items[1].ID != "claude-sonnet-4" || items[1].OwnedBy != "anthropic-compatible" {
		t.Fatalf("unexpected second model: %+v", items[1])
	}
}

func TestLocalGatewaySelectedModelsUseBoundRuntimeState(t *testing.T) {
	service := NewService(NewInMemoryRepository(), credential.NewInMemoryStore(), stubModelSourceReader{})
	if _, err := service.EnsureSystemLocalGateway(context.Background(), "http://127.0.0.1:8788"); err != nil {
		t.Fatalf("EnsureSystemLocalGateway returned error: %v", err)
	}

	runtimeState := &stubLocalRuntimeState{
		items: []SelectedModel{{ModelID: "model-a", Position: 0}},
	}
	service.BindLocalRuntimeState(runtimeState)

	items, err := service.ListSelectedModels(context.Background(), LocalGatewayProviderID)
	if err != nil {
		t.Fatalf("ListSelectedModels returned error: %v", err)
	}
	if len(items) != 1 || items[0].ModelID != "model-a" {
		t.Fatalf("unexpected selected models: %+v", items)
	}

	saved, err := service.ReplaceSelectedModels(context.Background(), LocalGatewayProviderID, []SelectedModel{
		{ModelID: "model-b", Position: 9},
	})
	if err != nil {
		t.Fatalf("ReplaceSelectedModels returned error: %v", err)
	}
	if len(saved) != 1 || saved[0].ModelID != "model-b" {
		t.Fatalf("unexpected saved models: %+v", saved)
	}
	if len(runtimeState.items) != 1 || runtimeState.items[0].ModelID != "model-b" {
		t.Fatalf("unexpected runtime state: %+v", runtimeState.items)
	}
}
