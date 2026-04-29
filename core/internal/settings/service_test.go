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
}
