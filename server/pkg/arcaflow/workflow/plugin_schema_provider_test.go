package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestContainerPluginSchemaProviderOptions(t *testing.T) {
	provider := NewContainerPluginSchemaProvider()
	provider.WithRuntime("podman")
	provider.WithTimeout(5 * time.Second)

	if provider.runtime != "podman" {
		t.Fatalf("expected runtime to be set")
	}
	if provider.timeout != 5*time.Second {
		t.Fatalf("expected timeout to be set")
	}
}

func TestContainerPluginSchemaProviderMissingRuntime(t *testing.T) {
	provider := NewContainerPluginSchemaProvider()
	provider.runtime = ""

	_, err := provider.InputJSONSchema(context.Background(), "image", "step")
	if err == nil {
		t.Fatalf("expected runtime error")
	}
}

func TestContainerPluginSchemaProviderSuccess(t *testing.T) {
	script := []byte("#!/bin/sh\necho '{\"type\":\"object\"}'\n")
	path := filepath.Join(t.TempDir(), "runtime.sh")
	if err := os.WriteFile(path, script, 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	provider := NewContainerPluginSchemaProvider().WithRuntime(path)
	payload, err := provider.InputJSONSchema(context.Background(), "image", "")
	if err != nil {
		t.Fatalf("expected schema, got %v", err)
	}
	if len(payload) == 0 {
		t.Fatalf("expected schema payload")
	}
}

func TestContainerPluginSchemaProviderInvalidJSON(t *testing.T) {
	script := []byte("#!/bin/sh\necho 'not-json'\n")
	path := filepath.Join(t.TempDir(), "runtime.sh")
	if err := os.WriteFile(path, script, 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	provider := NewContainerPluginSchemaProvider().WithRuntime(path)
	_, err := provider.InputJSONSchema(context.Background(), "image", "")
	if err == nil {
		t.Fatalf("expected json error")
	}
}

func TestNamespaceResolutionErrorMessage(t *testing.T) {
	err := NamespaceResolutionError{Message: "failed"}
	if err.Error() != "failed" {
		t.Fatalf("expected error message to match")
	}
}
