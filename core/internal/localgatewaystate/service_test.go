package localgatewaystate

import (
	"context"
	"testing"
)

type stubRepository struct {
	items []SelectedModel
}

func (r *stubRepository) ListSelectedModels(context.Context) ([]SelectedModel, error) {
	return r.items, nil
}

func (r *stubRepository) ReplaceSelectedModels(_ context.Context, items []SelectedModel) error {
	r.items = items
	return nil
}

func TestReplaceSelectedModelsNormalizesOrder(t *testing.T) {
	repo := &stubRepository{}
	service := NewService(repo)

	items, err := service.ReplaceSelectedModels(context.Background(), []SelectedModel{
		{ModelID: "model-a", Position: 9},
		{ModelID: "model-a", Position: 1},
		{ModelID: "model-b", Position: 3},
	})
	if err != nil {
		t.Fatalf("ReplaceSelectedModels returned error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 unique items, got %d", len(items))
	}
	if items[0].ModelID != "model-a" || items[0].Position != 0 {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[1].ModelID != "model-b" || items[1].Position != 1 {
		t.Fatalf("unexpected second item: %+v", items[1])
	}
}
