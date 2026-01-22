package workflow

import (
	"context"
	"math"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateInputFileJSON(t *testing.T) {
	t.Parallel()

	parsed := ParsedWorkflow{
		Workflow: Workflow{
			ID: "workflow-json",
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
		},
		InputSchemaPath: "input",
	}

	output, err := GenerateInputFile(
		context.Background(),
		parsed,
		[]byte(`{"name":"arcaflow"}`),
		ExportFormatJSON,
	)
	if err != nil {
		t.Fatalf("generate input: %v", err)
	}
	if output.Format != ExportFormatJSON {
		t.Fatalf("expected json format, got %s", output.Format)
	}
	if !strings.Contains(string(output.Payload), `"name":"arcaflow"`) {
		t.Fatalf("expected payload to include name")
	}
	if !output.Metadata.Validated {
		t.Fatalf("expected validation metadata")
	}
}

func TestGenerateInputFileYAML(t *testing.T) {
	t.Parallel()

	parsed := ParsedWorkflow{
		Workflow: Workflow{
			ID: "workflow-yaml",
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
        count:
          required: true
          type:
            type_id: integer
`),
		},
		InputSchemaPath: "input",
	}

	output, err := GenerateInputFile(
		context.Background(),
		parsed,
		[]byte(`{"count":2,"name":"arcaflow"}`),
		ExportFormatYAML,
	)
	if err != nil {
		t.Fatalf("generate input: %v", err)
	}
	if output.Format != ExportFormatYAML {
		t.Fatalf("expected yaml format, got %s", output.Format)
	}
	payload := string(output.Payload)
	if !strings.Contains(payload, "name: arcaflow") {
		t.Fatalf("expected payload to include name")
	}
	var decoded map[string]interface{}
	if err := yaml.Unmarshal(output.Payload, &decoded); err != nil {
		t.Fatalf("parse yaml payload: %v", err)
	}
	if _, ok := decoded["count"]; !ok {
		t.Fatalf("expected count in payload")
	}
}

func TestGenerateInputFileRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	parsed := ParsedWorkflow{
		Workflow: Workflow{
			ID: "workflow-invalid",
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
		},
		InputSchemaPath: "input",
	}

	_, err := GenerateInputFile(
		context.Background(),
		parsed,
		[]byte(`{}`),
		ExportFormatJSON,
	)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "workflow input validation failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMarshalDeterministicYAMLScalars(t *testing.T) {
	t.Parallel()

	payload, err := marshalDeterministicYAML(map[string]interface{}{
		"flag":    true,
		"ratio":   1.5,
		"note":    "ok",
		"nothing": nil,
		"items":   []interface{}{1, "two"},
	})
	if err != nil {
		t.Fatalf("marshal yaml: %v", err)
	}
	text := string(payload)
	if !strings.Contains(text, "flag: true") {
		t.Fatalf("expected bool encoding")
	}
	if !strings.Contains(text, "ratio: 1.5") {
		t.Fatalf("expected float encoding")
	}
	if !strings.Contains(text, "note: ok") {
		t.Fatalf("expected string encoding")
	}
	if !strings.Contains(text, "nothing: null") {
		t.Fatalf("expected null encoding")
	}
}

func TestMarshalDeterministicJSONOrdering(t *testing.T) {
	t.Parallel()

	payload, err := marshalDeterministicJSON(map[string]interface{}{
		"b": "two",
		"a": []interface{}{1, 2},
	})
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	if string(payload) != `{"a":[1,2],"b":"two"}` {
		t.Fatalf("unexpected json output: %s", string(payload))
	}
}

func TestMarshalDeterministicJSONRejectsNaN(t *testing.T) {
	t.Parallel()
	if _, err := marshalDeterministicJSON(math.NaN()); err == nil {
		t.Fatalf("expected NaN marshal error")
	}
}

func TestGenerateInputFileUnsupportedFormat(t *testing.T) {
	t.Parallel()
	parsed := ParsedWorkflow{
		Workflow: Workflow{
			ID: "workflow-bad-format",
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
		},
		InputSchemaPath: "input",
	}
	_, err := GenerateInputFile(
		context.Background(),
		parsed,
		[]byte(`{"name":"arcaflow"}`),
		ExportFormat("xml"),
	)
	if err == nil {
		t.Fatalf("expected unsupported format error")
	}
}

func TestGenerateInputFileDefaultFormat(t *testing.T) {
	t.Parallel()
	parsed := ParsedWorkflow{
		Workflow: Workflow{
			ID: "workflow-default-format",
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
		},
		InputSchemaPath: "input",
	}
	output, err := GenerateInputFile(
		context.Background(),
		parsed,
		[]byte(`{"name":"arcaflow"}`),
		"",
	)
	if err != nil {
		t.Fatalf("generate input: %v", err)
	}
	if output.Format != ExportFormatJSON {
		t.Fatalf("expected default JSON format")
	}
}

func TestValidationErrorMessage(t *testing.T) {
	t.Parallel()
	err := (&ValidationError{}).Error()
	if !strings.Contains(err, "workflow input validation failed") {
		t.Fatalf("expected validation error message")
	}
}

func TestGenerateInputFileCanceledContext(t *testing.T) {
	t.Parallel()
	parsed := ParsedWorkflow{
		Workflow: Workflow{
			ID: "workflow-canceled",
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
		},
		InputSchemaPath: "input",
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := GenerateInputFile(ctx, parsed, []byte(`{"name":"arcaflow"}`), ExportFormatJSON); err == nil {
		t.Fatalf("expected context error")
	}
}

func TestValidationErrorMessageWithIssues(t *testing.T) {
	t.Parallel()
	err := (&ValidationError{
		Issues: []ValidationIssue{{
			Path:    "$.name",
			Message: "required",
		}},
	}).Error()
	if !strings.Contains(err, "$.name: required") {
		t.Fatalf("expected issue details in message")
	}
}
