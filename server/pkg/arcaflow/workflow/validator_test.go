package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"go.flow.arcalot.io/engine"
	"go.flow.arcalot.io/engine/config"
	"go.flow.arcalot.io/engine/loadfile"
	"go.flow.arcalot.io/pluginsdk/schema"
)

// mockWorkflow implements engine.Workflow with a configurable input schema.
type mockWorkflow struct {
	inputSchema schema.Scope
}

func (m *mockWorkflow) Run(
	_ context.Context, _ []byte,
) (string, any, bool, error) {
	return "success", nil, false, nil
}

func (m *mockWorkflow) InputSchema() schema.Scope {
	return m.inputSchema
}

func (m *mockWorkflow) Outputs() map[string]schema.StepOutput {
	return nil
}

func (m *mockWorkflow) Namespaces() map[string]map[string]*schema.ObjectSchema {
	return nil
}

// mockEngine implements engine.WorkflowEngine for unit tests.
type mockEngine struct {
	workflow engine.Workflow
	parseErr error
}

func (m *mockEngine) RunWorkflow(
	_ context.Context, _ []byte, _ loadfile.FileCache, _ string,
) (string, any, bool, error) {
	return "", nil, false, nil
}

func (m *mockEngine) Parse(
	_ loadfile.FileCache, _ string,
) (engine.Workflow, error) {
	if m.parseErr != nil {
		return nil, m.parseErr
	}
	return m.workflow, nil
}

// newTestScope creates a simple scope with the given properties.
func newTestScope(props map[string]*schema.PropertySchema) schema.Scope {
	return schema.NewScopeSchema(
		schema.NewObjectSchema(
			"TestInput",
			props,
		),
	)
}

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
	if _, err := validator.Validate(
		context.Background(), Workflow{}, []byte(`{}`),
	); err == nil {
		t.Fatalf("expected error for empty workflow content")
	}
	workflow := Workflow{Content: []byte("version: v0.2.0")}
	if _, err := validator.Validate(
		context.Background(), workflow, nil,
	); err == nil {
		t.Fatalf("expected error for empty input payload")
	}
}

func TestInputValidatorContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	validator := NewInputValidator()
	_, err := validator.Validate(
		ctx,
		Workflow{Content: []byte("test")},
		[]byte(`{}`),
	)
	if err == nil {
		t.Fatalf("expected context error")
	}
}

func TestInputValidatorEngineParseError(t *testing.T) {
	eng := &mockEngine{parseErr: errors.New("parse failed")}
	validator := NewInputValidator(WithInputValidatorEngine(eng))
	result, err := validator.Validate(
		context.Background(),
		Workflow{Content: []byte("test")},
		[]byte(`{"name":"test"}`),
	)
	if err != nil {
		t.Fatalf("expected validation result, got error: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid result when engine parse fails")
	}
	if len(result.Issues) == 0 {
		t.Fatalf("expected issues from parse error")
	}
	if result.Issues[0].Path != "$" {
		t.Fatalf("expected root path, got %q", result.Issues[0].Path)
	}
}

func TestInputValidatorValidInput(t *testing.T) {
	inputScope := newTestScope(map[string]*schema.PropertySchema{
		"name": schema.NewPropertySchema(
			schema.NewStringSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
	})
	eng := &mockEngine{workflow: &mockWorkflow{inputSchema: inputScope}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	result, err := validator.Validate(
		context.Background(),
		Workflow{Content: []byte("test")},
		[]byte(`{"name":"arcaflow"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf(
			"expected valid, got issues: %v",
			result.Issues,
		)
	}
	if len(result.NormalizedInput) == 0 {
		t.Fatalf("expected normalized input")
	}
	// Verify normalized output contains the field
	var normalized map[string]any
	if err := json.Unmarshal(result.NormalizedInput, &normalized); err != nil {
		t.Fatalf("unmarshal normalized: %v", err)
	}
	if normalized["name"] != "arcaflow" {
		t.Fatalf("expected name=arcaflow, got %v", normalized["name"])
	}
}

func TestInputValidatorMissingRequiredField(t *testing.T) {
	inputScope := newTestScope(map[string]*schema.PropertySchema{
		"name": schema.NewPropertySchema(
			schema.NewStringSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
	})
	eng := &mockEngine{workflow: &mockWorkflow{inputSchema: inputScope}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	result, err := validator.Validate(
		context.Background(),
		Workflow{Content: []byte("test")},
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid for missing required field")
	}
	if len(result.Issues) == 0 {
		t.Fatalf("expected validation issues")
	}
}

func TestInputValidatorWrongType(t *testing.T) {
	inputScope := newTestScope(map[string]*schema.PropertySchema{
		"count": schema.NewPropertySchema(
			schema.NewIntSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
	})
	eng := &mockEngine{workflow: &mockWorkflow{inputSchema: inputScope}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	result, err := validator.Validate(
		context.Background(),
		Workflow{Content: []byte("test")},
		[]byte(`{"count":"not-a-number"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid for wrong type")
	}
}

func TestInputValidatorMultipleFields(t *testing.T) {
	inputScope := newTestScope(map[string]*schema.PropertySchema{
		"name": schema.NewPropertySchema(
			schema.NewStringSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
		"enabled": schema.NewPropertySchema(
			schema.NewBoolSchema(),
			nil, false, nil, nil, nil, nil, nil,
		),
		"count": schema.NewPropertySchema(
			schema.NewIntSchema(nil, nil, nil),
			nil, false, nil, nil, nil, nil, nil,
		),
	})
	eng := &mockEngine{workflow: &mockWorkflow{inputSchema: inputScope}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	result, err := validator.Validate(
		context.Background(),
		Workflow{Content: []byte("test")},
		[]byte(`{"name":"test","enabled":true,"count":42}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got issues: %v", result.Issues)
	}
}

func TestInputValidatorYAMLInput(t *testing.T) {
	inputScope := newTestScope(map[string]*schema.PropertySchema{
		"name": schema.NewPropertySchema(
			schema.NewStringSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
	})
	eng := &mockEngine{workflow: &mockWorkflow{inputSchema: inputScope}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	result, err := validator.Validate(
		context.Background(),
		Workflow{Content: []byte("test")},
		[]byte("name: arcaflow\n"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf(
			"expected valid for YAML input, got issues: %v",
			result.Issues,
		)
	}
}

func TestInputValidatorInvalidJSON(t *testing.T) {
	eng := &mockEngine{workflow: &mockWorkflow{
		inputSchema: newTestScope(nil),
	}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	_, err := validator.Validate(
		context.Background(),
		Workflow{Content: []byte("test")},
		[]byte(`{invalid`),
	)
	if err == nil {
		t.Fatalf("expected error for invalid JSON/YAML")
	}
}

func TestInputValidatorLegacyOptionNoop(t *testing.T) {
	// WithInputValidatorPluginSchemaProvider should be a no-op
	validator := NewInputValidator(
		WithInputValidatorPluginSchemaProvider(nil),
	)
	if validator == nil {
		t.Fatalf("expected non-nil validator")
	}
}

func TestSuggestionForConstraint(t *testing.T) {
	cases := []struct {
		message  string
		contains string
	}{
		{"field is required", "required"},
		{"value must be a string", "string"},
		{"value must be an int", "integer"},
		{"value must be an integer", "integer"},
		{"value must be a float", "numeric"},
		{"value must be a number", "numeric"},
		{"value must be a bool", "boolean"},
		{"value must be a boolean", "boolean"},
		{"unknown constraint", "Check the input"},
	}
	for _, tc := range cases {
		suggestion := suggestionForConstraint(tc.message)
		if suggestion == "" {
			t.Fatalf(
				"empty suggestion for %q", tc.message,
			)
		}
	}
}

func TestInputValidatorWithLocalPath(t *testing.T) {
	// The validator should use LocalPath for file cache base dir
	inputScope := newTestScope(map[string]*schema.PropertySchema{
		"name": schema.NewPropertySchema(
			schema.NewStringSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
	})
	eng := &mockEngine{workflow: &mockWorkflow{inputSchema: inputScope}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	result, err := validator.Validate(
		context.Background(),
		Workflow{
			Content:   []byte("test"),
			LocalPath: "/tmp/workflow.yaml",
			Path:      "workflow.yaml",
		},
		[]byte(`{"name":"test"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got issues: %v", result.Issues)
	}
}

func TestInputValidatorWithOnlyPath(t *testing.T) {
	inputScope := newTestScope(map[string]*schema.PropertySchema{
		"x": schema.NewPropertySchema(
			schema.NewIntSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
	})
	eng := &mockEngine{workflow: &mockWorkflow{inputSchema: inputScope}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	result, err := validator.Validate(
		context.Background(),
		Workflow{
			Content: []byte("test"),
			Path:    "my-workflow.yaml",
		},
		[]byte(`{"x":10}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got issues: %v", result.Issues)
	}
}

func TestNormalizeConstraintIssuesGenericError(t *testing.T) {
	issues := normalizeConstraintIssues(errors.New("some error"))
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Path != "$" {
		t.Fatalf("expected root path")
	}
	if issues[0].Message != "some error" {
		t.Fatalf("expected error message preserved")
	}
}

func TestValidationErrorMessage(t *testing.T) {
	err := &ValidationError{
		Issues: []ValidationIssue{
			{Path: "$.name", Message: "required"},
			{Path: "$.count", Message: "wrong type"},
		},
	}
	msg := err.Error()
	if msg == "" {
		t.Fatalf("expected non-empty error message")
	}
	if !errors.As(err, new(*ValidationError)) {
		t.Fatalf("expected ValidationError type")
	}
}

func TestValidationErrorEmpty(t *testing.T) {
	err := &ValidationError{}
	msg := err.Error()
	if msg != "workflow input validation failed" {
		t.Fatalf("expected default message, got %q", msg)
	}
}

func TestGenerateInputFileValidJSON(t *testing.T) {
	inputScope := newTestScope(map[string]*schema.PropertySchema{
		"name": schema.NewPropertySchema(
			schema.NewStringSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
	})
	eng := &mockEngine{workflow: &mockWorkflow{inputSchema: inputScope}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	parsed := ParsedWorkflow{
		Workflow: Workflow{
			ID:      "test",
			Content: []byte("test"),
		},
		InputSchemaPath: "input",
	}

	exported, err := GenerateInputFile(
		context.Background(),
		parsed,
		[]byte(`{"name":"arcaflow"}`),
		ExportFormatJSON,
		WithInputValidator(validator),
	)
	if err != nil {
		t.Fatalf("generate input: %v", err)
	}
	if exported.Format != ExportFormatJSON {
		t.Fatalf("expected json format, got %v", exported.Format)
	}
	if len(exported.Payload) == 0 {
		t.Fatalf("expected payload")
	}
	if !exported.Metadata.Validated {
		t.Fatalf("expected validated=true")
	}
}

func TestGenerateInputFileValidYAML(t *testing.T) {
	inputScope := newTestScope(map[string]*schema.PropertySchema{
		"name": schema.NewPropertySchema(
			schema.NewStringSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
	})
	eng := &mockEngine{workflow: &mockWorkflow{inputSchema: inputScope}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	parsed := ParsedWorkflow{
		Workflow: Workflow{
			ID:      "test",
			Content: []byte("test"),
		},
	}

	exported, err := GenerateInputFile(
		context.Background(),
		parsed,
		[]byte(`{"name":"arcaflow"}`),
		ExportFormatYAML,
		WithInputValidator(validator),
	)
	if err != nil {
		t.Fatalf("generate input: %v", err)
	}
	if exported.Format != ExportFormatYAML {
		t.Fatalf("expected yaml format, got %v", exported.Format)
	}
}

func TestGenerateInputFileInvalidInput(t *testing.T) {
	inputScope := newTestScope(map[string]*schema.PropertySchema{
		"name": schema.NewPropertySchema(
			schema.NewStringSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
	})
	eng := &mockEngine{workflow: &mockWorkflow{inputSchema: inputScope}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	parsed := ParsedWorkflow{
		Workflow: Workflow{
			ID:      "test",
			Content: []byte("test"),
		},
	}

	_, err := GenerateInputFile(
		context.Background(),
		parsed,
		[]byte(`{}`), // missing required "name"
		ExportFormatJSON,
		WithInputValidator(validator),
	)
	if err == nil {
		t.Fatalf("expected validation error for missing field")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestGenerateInputFileUnsupportedFormat(t *testing.T) {
	inputScope := newTestScope(map[string]*schema.PropertySchema{
		"name": schema.NewPropertySchema(
			schema.NewStringSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
	})
	eng := &mockEngine{workflow: &mockWorkflow{inputSchema: inputScope}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	parsed := ParsedWorkflow{
		Workflow: Workflow{
			ID:      "test",
			Content: []byte("test"),
		},
	}

	_, err := GenerateInputFile(
		context.Background(),
		parsed,
		[]byte(`{"name":"test"}`),
		ExportFormat("xml"),
		WithInputValidator(validator),
	)
	if err == nil {
		t.Fatalf("expected error for unsupported format")
	}
}

func TestGenerateInputFileDefaultFormat(t *testing.T) {
	inputScope := newTestScope(map[string]*schema.PropertySchema{
		"name": schema.NewPropertySchema(
			schema.NewStringSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
	})
	eng := &mockEngine{workflow: &mockWorkflow{inputSchema: inputScope}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	parsed := ParsedWorkflow{
		Workflow: Workflow{
			ID:      "test",
			Content: []byte("test"),
		},
	}

	exported, err := GenerateInputFile(
		context.Background(),
		parsed,
		[]byte(`{"name":"test"}`),
		"", // empty format defaults to JSON
		WithInputValidator(validator),
	)
	if err != nil {
		t.Fatalf("generate input: %v", err)
	}
	if exported.Format != ExportFormatJSON {
		t.Fatalf("expected default json format")
	}
}

func TestInputValidatorLocalPathWithSiblingFiles(t *testing.T) {
	dir := t.TempDir()
	// Create a main workflow file and a sibling YAML file
	mainPath := dir + "/workflow.yaml"
	siblingPath := dir + "/config.yaml"
	if err := writeTestFile(mainPath, "version: v0.2.0"); err != nil {
		t.Fatalf("write main: %v", err)
	}
	if err := writeTestFile(siblingPath, "key: value"); err != nil {
		t.Fatalf("write sibling: %v", err)
	}
	// Also create a non-yaml file that should be ignored
	if err := writeTestFile(dir+"/notes.txt", "ignore me"); err != nil {
		t.Fatalf("write txt: %v", err)
	}

	inputScope := newTestScope(map[string]*schema.PropertySchema{
		"name": schema.NewPropertySchema(
			schema.NewStringSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
	})
	eng := &mockEngine{workflow: &mockWorkflow{inputSchema: inputScope}}
	validator := NewInputValidator(WithInputValidatorEngine(eng))

	result, err := validator.Validate(
		context.Background(),
		Workflow{
			Content:   []byte("version: v0.2.0"),
			LocalPath: mainPath,
		},
		[]byte(`{"name":"test"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got issues: %v", result.Issues)
	}
}

func TestInputValidatorUnresolvedPluginRefs(t *testing.T) {
	// Workflow with input referencing a type that
	// would only exist in a plugin schema. The
	// fallback path should detect this and return
	// a clear error, not panic.
	content := `
version: v0.2.0
input:
  root: CompositeInput
  objects:
    CompositeInput:
      id: CompositeInput
      properties:
        fio_params:
          required: true
          type:
            type_id: ref
            id: FioInputParams
            namespace: "$.steps.fio.starting.inputs.input"
steps: {}
outputs:
  success:
    status: ok
`
	validator := NewInputValidator()
	result, err := validator.validateFromInputScope(
		Workflow{Content: []byte(content)},
		[]byte(`{"fio_params":{"filename":"/dev/sda"}}`),
	)
	// Should return an error about unresolved refs,
	// not panic.
	if err == nil {
		t.Fatalf(
			"expected error for unresolved refs, "+
				"got valid=%v",
			result.Valid,
		)
	}
	if !strings.Contains(
		err.Error(), "container runtime",
	) {
		t.Errorf(
			"error should mention container runtime: %v",
			err,
		)
	}
}

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func TestGenerateInputFileContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	parsed := ParsedWorkflow{
		Workflow: Workflow{Content: []byte("test")},
	}

	_, err := GenerateInputFile(
		ctx, parsed, []byte(`{}`), ExportFormatJSON,
	)
	if err == nil {
		t.Fatalf("expected context error")
	}
}
