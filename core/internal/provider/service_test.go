package provider

import (
	"context"
	"errors"
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

func TestFetchModelsForSystemLocalGatewayUsesEnabledModelSources(t *testing.T) {
	service := NewService(
		NewInMemoryRepository(),
		credential.NewInMemoryStore(),
		stubModelSourceReader{
			items: []modelsource.Source{
				{
					ID:             "source-1",
					Name:           "OpenAI",
					ProviderType:   "openai-compatible",
					DefaultModelID: "gpt-4.1",
					Enabled:        true,
				},
				{
					ID:             "source-2",
					Name:           "Anthropic",
					ProviderType:   "anthropic-compatible",
					DefaultModelID: "claude-sonnet-4",
					Enabled:        true,
				},
				{
					ID:             "source-3",
					Name:           "Duplicate",
					ProviderType:   "openai-compatible",
					DefaultModelID: "gpt-4.1",
					Enabled:        true,
				},
				{
					ID:             "source-4",
					Name:           "Disabled",
					ProviderType:   "openai-compatible",
					DefaultModelID: "gpt-disabled",
					Enabled:        false,
				},
			},
		},
	)

	if _, err := service.EnsureSystemLocalGateway(context.Background(), "http://127.0.0.1:3456"); err != nil {
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
