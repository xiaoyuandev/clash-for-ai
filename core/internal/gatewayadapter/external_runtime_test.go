package gatewayadapter

import (
	"context"
	"errors"
	"testing"
)

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
}

func TestExternalRuntimeAdapterAdminUnsupported(t *testing.T) {
	adapter := NewExternalRuntimeAdapter("http://127.0.0.1:8788")

	if _, err := adapter.ListSelectedModels(context.Background()); !errors.Is(err, ErrRuntimeAdminUnsupported) {
		t.Fatalf("expected ErrRuntimeAdminUnsupported, got %v", err)
	}
}
