package workflow

import (
	"context"
	"errors"
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

func TestContainerPluginSchemaProviderHonorsContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	script := []byte("#!/bin/sh\necho '{\"type\":\"object\"}'\n")
	path := filepath.Join(t.TempDir(), "runtime.sh")
	if err := os.WriteFile(path, script, 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	provider := NewContainerPluginSchemaProvider().WithRuntime(path)
	_, err := provider.InputJSONSchema(ctx, "image", "")
	if err == nil {
		t.Fatalf("expected context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestFullSchemaNoRuntime(t *testing.T) {
	provider := NewContainerPluginSchemaProvider()
	provider.runtime = ""

	_, err := provider.FullSchema(
		context.Background(), "image",
	)
	if err == nil {
		t.Fatal("expected error when runtime is empty")
	}
	expected := "no container runtime available"
	if err.Error() != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected, err.Error(),
		)
	}
}

func TestHasRuntime(t *testing.T) {
	provider := NewContainerPluginSchemaProvider()

	provider.runtime = "podman"
	if !provider.HasRuntime() {
		t.Fatal("expected HasRuntime true with runtime set")
	}

	provider.runtime = ""
	if provider.HasRuntime() {
		t.Fatal("expected HasRuntime false with empty runtime")
	}
}

func TestRuntime(t *testing.T) {
	provider := NewContainerPluginSchemaProvider()

	provider.runtime = "docker"
	if provider.Runtime() != "docker" {
		t.Fatalf(
			"expected 'docker', got %q",
			provider.Runtime(),
		)
	}

	provider.runtime = ""
	if provider.Runtime() != "" {
		t.Fatal("expected empty string for no runtime")
	}
}

func TestFullSchemaEmptyOutput(t *testing.T) {
	// Script outputs nothing, testing empty output detection
	script := []byte("#!/bin/sh\necho ''\n")
	path := filepath.Join(t.TempDir(), "runtime.sh")
	if err := os.WriteFile(path, script, 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	provider := NewContainerPluginSchemaProvider().
		WithRuntime(path)
	_, err := provider.FullSchema(
		context.Background(), "image",
	)
	if err == nil {
		t.Fatal("expected error for empty output")
	}
	if err.Error() != "empty schema output" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFullSchemaCommandExecution(t *testing.T) {
	// Use "false" binary which always exits 1,
	// verifying error handling for failed commands.
	provider := NewContainerPluginSchemaProvider().
		WithRuntime("false")
	_, err := provider.FullSchema(
		context.Background(), "test:latest",
	)
	if err == nil {
		t.Fatal("expected error from false command")
	}
}

func TestFullSchemaSuccess(t *testing.T) {
	// Script outputs valid YAML schema content
	script := []byte(
		"#!/bin/sh\necho 'steps:\n  hello:\n" +
			"    id: hello'\n",
	)
	path := filepath.Join(t.TempDir(), "runtime.sh")
	if err := os.WriteFile(path, script, 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	provider := NewContainerPluginSchemaProvider().
		WithRuntime(path)
	output, err := provider.FullSchema(
		context.Background(), "image",
	)
	if err != nil {
		t.Fatalf("expected schema, got %v", err)
	}
	if len(output) == 0 {
		t.Fatal("expected non-empty schema output")
	}
}

func TestNamespaceResolutionErrorMessage(t *testing.T) {
	err := NamespaceResolutionError{Message: "failed"}
	if err.Error() != "failed" {
		t.Fatalf("expected error message to match")
	}
}
