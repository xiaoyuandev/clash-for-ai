package config

import "testing"

func TestLoadHTTPBindDefaults(t *testing.T) {
	t.Setenv("HTTP_HOST", "")
	t.Setenv("HTTP_PORT", "")

	cfg := Load()

	if cfg.GatewayBind != "127.0.0.1" {
		t.Fatalf("expected default HTTP host 127.0.0.1, got %q", cfg.GatewayBind)
	}
	if cfg.HTTPPort != 3456 {
		t.Fatalf("expected default HTTP port 3456, got %d", cfg.HTTPPort)
	}
}

func TestLoadHTTPBindFromEnvironment(t *testing.T) {
	t.Setenv("HTTP_HOST", "0.0.0.0")
	t.Setenv("HTTP_PORT", "8080")

	cfg := Load()

	if cfg.GatewayBind != "0.0.0.0" {
		t.Fatalf("expected env HTTP host 0.0.0.0, got %q", cfg.GatewayBind)
	}
	if cfg.HTTPPort != 8080 {
		t.Fatalf("expected env HTTP port 8080, got %d", cfg.HTTPPort)
	}
}
