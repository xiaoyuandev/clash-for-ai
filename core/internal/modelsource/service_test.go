package modelsource

import (
	"context"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
)

type stubRepository struct {
	items []Source
}

func (r *stubRepository) List(context.Context) ([]Source, error) {
	items := make([]Source, len(r.items))
	copy(items, r.items)
	return items, nil
}

func (r *stubRepository) GetByID(_ context.Context, id string) (*Source, error) {
	for _, item := range r.items {
		if item.ID == id {
			next := item
			return &next, nil
		}
	}
	return nil, ErrSourceNotFound
}

func (r *stubRepository) Create(_ context.Context, source Source) (Source, error) {
	r.items = append(r.items, source)
	return source, nil
}

func (r *stubRepository) Update(_ context.Context, source Source) (Source, error) {
	for index := range r.items {
		if r.items[index].ID == source.ID {
			r.items[index] = source
			return source, nil
		}
	}
	return Source{}, ErrSourceNotFound
}

func (r *stubRepository) Delete(_ context.Context, id string) error {
	for index := range r.items {
		if r.items[index].ID == id {
			r.items = append(r.items[:index], r.items[index+1:]...)
			return nil
		}
	}
	return ErrSourceNotFound
}

func (r *stubRepository) ReplaceOrder(_ context.Context, sources []Source) error {
	for index := range sources {
		for itemIndex := range r.items {
			if r.items[itemIndex].ID == sources[index].ID {
				r.items[itemIndex].Position = index
			}
		}
	}
	return nil
}

func TestCreateMasksApiKeyAndAssignsPosition(t *testing.T) {
	repo := &stubRepository{
		items: []Source{
			{ID: "existing", Position: 0},
		},
	}
	service := NewService(repo, credential.NewInMemoryStore())

	item, err := service.Create(context.Background(), CreateInput{
		Name:           " Test Source ",
		BaseURL:        " https://api.example.com/v1 ",
		ProviderType:   " openai-compatible ",
		DefaultModelID: " gpt-4.1 ",
		Enabled:        true,
		APIKey:         "sk-1234567890abcdef",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if item.Position != 1 {
		t.Fatalf("expected position 1, got %d", item.Position)
	}
	if item.Name != "Test Source" {
		t.Fatalf("unexpected normalized name: %s", item.Name)
	}
	if item.DefaultModelID != "gpt-4.1" {
		t.Fatalf("unexpected normalized model id: %s", item.DefaultModelID)
	}
	if item.APIKeyMasked == "" || item.APIKeyMasked == item.APIKey {
		t.Fatalf("expected masked api key, got %+v", item)
	}
}

func TestReplaceOrderNormalizesPositions(t *testing.T) {
	repo := &stubRepository{
		items: []Source{
			{ID: "a", Position: 0},
			{ID: "b", Position: 1},
			{ID: "c", Position: 2},
		},
	}
	service := NewService(repo, credential.NewInMemoryStore())

	items, err := service.ReplaceOrder(context.Background(), []Source{
		{ID: "c"},
		{ID: "a"},
		{ID: "c"},
		{ID: "b"},
	})
	if err != nil {
		t.Fatalf("ReplaceOrder returned error: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].ID != "c" || items[0].Position != 0 {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[1].ID != "a" || items[1].Position != 1 {
		t.Fatalf("unexpected second item: %+v", items[1])
	}
	if items[2].ID != "b" || items[2].Position != 2 {
		t.Fatalf("unexpected third item: %+v", items[2])
	}
}
