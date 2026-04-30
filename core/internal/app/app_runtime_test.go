package app

import "testing"

func TestResolveEmbeddedRuntimeExecutablePathPrefersExplicitRuntimeBinary(t *testing.T) {
	t.Setenv("LOCAL_GATEWAY_RUNTIME_EXECUTABLE", "/tmp/standalone-gateway")

	path := resolveEmbeddedRuntimeExecutablePath("/tmp/current-core")
	if path != "/tmp/standalone-gateway" {
		t.Fatalf("expected explicit runtime executable, got %s", path)
	}
}

func TestResolveEmbeddedRuntimeExecutablePathFallsBackToCurrentExecutable(t *testing.T) {
	t.Setenv("LOCAL_GATEWAY_RUNTIME_EXECUTABLE", "")

	path := resolveEmbeddedRuntimeExecutablePath("/tmp/current-core")
	if path != "/tmp/current-core" {
		t.Fatalf("expected current executable fallback, got %s", path)
	}
}
