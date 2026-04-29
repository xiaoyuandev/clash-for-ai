package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestEnsureSystemLocalGatewayCreatesImmutableProvider(t *testing.T) {
	service := NewService(NewInMemoryRepository(), credential.NewInMemoryStore())

	item, err := service.EnsureSystemLocalGateway(context.Background(), "http://127.0.0.1:3456", DefaultLocalGatewayCapabilities())
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
	service := NewService(NewInMemoryRepository(), credential.NewInMemoryStore())

	if _, err := service.EnsureSystemLocalGateway(context.Background(), "http://127.0.0.1:3456", DefaultLocalGatewayCapabilities()); err != nil {
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
	service := NewService(NewInMemoryRepository(), credential.NewInMemoryStore())
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

	if _, err := service.EnsureSystemLocalGateway(context.Background(), "http://127.0.0.1:8788", DefaultLocalGatewayCapabilities()); err != nil {
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
