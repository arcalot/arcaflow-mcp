package workflow

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestInputValidatorAcceptsValidInput(t *testing.T) {
	t.Parallel()
	validator := NewInputValidator()

	workflow := Workflow{
		ID: "workflow-valid",
		Content: []byte(`
version: v0.2.0
input:
  root: InputParams
  objects:
    InputParams:
      id: InputParams
      properties:
        name:
          required: true
          type:
            type_id: string
`),
	}

	result, err := validator.Validate(
		context.Background(),
		workflow,
		[]byte(`{"name":"arcaflow"}`),
	)
	if err != nil {
		t.Fatalf("validate input: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected input to be valid")
	}
	if len(result.NormalizedInput) == 0 {
		t.Fatalf("expected normalized input")
	}
}

func TestInputValidatorRejectsMissingRequiredField(t *testing.T) {
	t.Parallel()
	validator := NewInputValidator()

	workflow := Workflow{
		ID: "workflow-missing",
		Content: []byte(`
version: v0.2.0
input:
  root: InputParams
  objects:
    InputParams:
      id: InputParams
      properties:
        name:
          required: true
          type:
            type_id: string
`),
	}

	result, err := validator.Validate(
		context.Background(),
		workflow,
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("validate input: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected input to be invalid")
	}
	if len(result.Issues) == 0 {
		t.Fatalf("expected validation issues")
	}
	if !strings.Contains(strings.ToLower(result.Issues[0].Message), "required") {
		t.Fatalf("expected required message, got %s", result.Issues[0].Message)
	}
}

func TestInputValidatorRejectsInvalidType(t *testing.T) {
	t.Parallel()
	validator := NewInputValidator()

	workflow := Workflow{
		ID: "workflow-invalid-type",
		Content: []byte(`
version: v0.2.0
input:
  root: InputParams
  objects:
    InputParams:
      id: InputParams
      properties:
        count:
          required: true
          type:
            type_id: integer
`),
	}

	result, err := validator.Validate(
		context.Background(),
		workflow,
		[]byte(`{"count":"nope"}`),
	)
	if err != nil {
		t.Fatalf("validate input: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected input to be invalid")
	}
	if len(result.Issues) == 0 {
		t.Fatalf("expected validation issues")
	}
	if result.Issues[0].Message == "" {
		t.Fatalf("expected issue message")
	}
}

func TestInputValidatorLoggerOption(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	validator := NewInputValidator(WithInputValidatorLogger(logger))
	if validator.logger != logger {
		t.Fatalf("expected logger override")
	}
}

func TestSuggestionForConstraint(t *testing.T) {
	t.Parallel()
	cases := []struct {
		message  string
		expected string
	}{
		{"field is required", "Add the required field to the input."},
		{"value must be a string", "Provide a string value for this field."},
		{"value must be an int", "Provide an integer value for this field."},
		{"value must be a float", "Provide a numeric value for this field."},
		{"value must be a bool", "Provide a boolean value for this field."},
		{"unexpected", "Check the input value against the workflow schema."},
	}
	for _, entry := range cases {
		if suggestion := suggestionForConstraint(entry.message); suggestion != entry.expected {
			t.Fatalf("expected %q for %q, got %q", entry.expected, entry.message, suggestion)
		}
	}
}

func TestInputValidatorRejectsEmptyInputs(t *testing.T) {
	t.Parallel()
	validator := NewInputValidator()
	if _, err := validator.Validate(context.Background(), Workflow{}, []byte(`{}`)); err == nil {
		t.Fatalf("expected error for empty workflow content")
	}
	workflow := Workflow{Content: []byte("version: v0.2.0")}
	if _, err := validator.Validate(context.Background(), workflow, nil); err == nil {
		t.Fatalf("expected error for empty input payload")
	}
}

func TestExtractInputScopeMissingInput(t *testing.T) {
	t.Parallel()
	if _, err := extractInputScope([]byte("version: v0.2.0")); err == nil {
		t.Fatalf("expected error for missing input section")
	}
}
