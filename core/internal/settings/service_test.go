package settings

import (
	"context"
	"testing"
)

type stubRepository struct {
	settings AppSettings
}

func (r *stubRepository) Get(context.Context) (AppSettings, error) {
	return r.settings, nil
}

func (r *stubRepository) Save(_ context.Context, settings AppSettings) (AppSettings, error) {
	r.settings = settings
	return settings, nil
}

func TestNormalizeSettingsAppliesDefaults(t *testing.T) {
	settings := NormalizeSettings(AppSettings{})

	if !settings.LocalGateway.Enabled {
		t.Fatalf("expected LocalGateway.Enabled to default to true")
	}
	if settings.LocalGateway.ListenHost != "127.0.0.1" {
		t.Fatalf("unexpected LocalGateway.ListenHost: %s", settings.LocalGateway.ListenHost)
	}
	if settings.LocalGateway.ListenPort != 8788 {
		t.Fatalf("unexpected LocalGateway.ListenPort: %d", settings.LocalGateway.ListenPort)
	}
	if settings.LocalGatewaySelected == nil {
		t.Fatalf("expected LocalGatewaySelected to be initialized")
	}
}

func TestUpdateLocalGatewaySelectedModelsNormalizesOrder(t *testing.T) {
	repo := &stubRepository{settings: DefaultSettings()}
	service := NewService(repo)

	items, err := service.UpdateLocalGatewaySelectedModels(context.Background(), []SelectedModel{
		{ModelID: "model-a", Position: 9},
		{ModelID: "model-a", Position: 1},
		{ModelID: "model-b", Position: 3},
	})
	if err != nil {
		t.Fatalf("UpdateLocalGatewaySelectedModels returned error: %v", err)
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

func TestUpdateLocalGatewayClaudeMapTrimsValues(t *testing.T) {
	repo := &stubRepository{settings: DefaultSettings()}
	service := NewService(repo)

	item, err := service.UpdateLocalGatewayClaudeMap(context.Background(), ClaudeCodeModelMap{
		Opus:   " opus-model ",
		Sonnet: " sonnet-model ",
		Haiku:  " haiku-model ",
	})
	if err != nil {
		t.Fatalf("UpdateLocalGatewayClaudeMap returned error: %v", err)
	}

	if item.Opus != "opus-model" || item.Sonnet != "sonnet-model" || item.Haiku != "haiku-model" {
		t.Fatalf("unexpected normalized claude map: %+v", item)
	}
}
