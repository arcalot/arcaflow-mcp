package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveFilesystemPath(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	tests := []struct {
		name      string
		input     string
		wantAbs   bool
		wantErr   bool
		errString string
	}{
		{
			name:    "absolute path unchanged",
			input:   "/absolute/path/to/workflow.yaml",
			wantAbs: true,
		},
		{
			name:    "relative file resolved",
			input:   "workflow.yaml",
			wantAbs: true,
		},
		{
			name:    "relative path resolved",
			input:   "subdir/workflow.yaml",
			wantAbs: true,
		},
		{
			name:    "dot path resolved",
			input:   ".",
			wantAbs: true,
		},
		{
			name:    "parent path resolved",
			input:   "../workflow.yaml",
			wantAbs: true,
		},
		{
			name:      "empty path error",
			input:     "",
			wantErr:   true,
			errString: "location cannot be empty",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ResolveFilesystemPath(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errString)
				}
				if !strings.Contains(err.Error(), tt.errString) {
					t.Fatalf("expected error containing %q, got %q", tt.errString, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantAbs && !filepath.IsAbs(result) {
				t.Fatalf("expected absolute path, got %q", result)
			}

			// For relative inputs, verify resolution worked correctly
			if !filepath.IsAbs(tt.input) && tt.input != "" {
				expected := filepath.Join(cwd, filepath.Clean(tt.input))
				if result != expected {
					t.Fatalf("expected %q, got %q", expected, result)
				}
			}

			// For absolute inputs, verify unchanged
			if filepath.IsAbs(tt.input) {
				if result != filepath.Clean(tt.input) {
					t.Fatalf("absolute path changed: expected %q, got %q", filepath.Clean(tt.input), result)
				}
			}
		})
	}
}

func TestResolveFilesystemPathIdempotent(t *testing.T) {
	t.Parallel()

	// Resolving an already-resolved path should return the same result
	input := "relative/path/to/file.yaml"
	first, err := ResolveFilesystemPath(input)
	if err != nil {
		t.Fatalf("first resolution failed: %v", err)
	}

	second, err := ResolveFilesystemPath(first)
	if err != nil {
		t.Fatalf("second resolution failed: %v", err)
	}

	if first != second {
		t.Fatalf("resolution not idempotent: first=%q, second=%q", first, second)
	}
}
