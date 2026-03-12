package workflow

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"go.flow.arcalot.io/engine/config"
)

func TestInputValidatorLoggerOption(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	validator := NewInputValidator(WithInputValidatorLogger(logger))
	if validator.logger != logger {
		t.Fatalf("expected logger override")
	}
}

func TestInputValidatorConfigOption(t *testing.T) {
	cfg := config.Default()
	validator := NewInputValidator(WithInputValidatorConfig(cfg))
	if validator.config != cfg {
		t.Fatalf("expected config override")
	}
}

func TestInputValidatorRejectsEmptyInputs(t *testing.T) {
	validator := NewInputValidator()
	if _, err := validator.Validate(context.Background(), Workflow{}, []byte(`{}`)); err == nil {
		t.Fatalf("expected error for empty workflow content")
	}
	workflow := Workflow{Content: []byte("version: v0.2.0")}
	if _, err := validator.Validate(context.Background(), workflow, nil); err == nil {
		t.Fatalf("expected error for empty input payload")
	}
}

// NOTE: Full engine validation tests are omitted here as they require a 
// working container runtime or complex mocking of the Arcaflow engine SDK.
// Integration tests should be used for end-to-end validation.
