package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParserParseYAML(t *testing.T) {
	t.Parallel()
	parser := NewParser()

	workflow := Workflow{
		ID: "workflow-yaml",
		Content: []byte(`
input_schema:
  type: object
output_schema:
  type: object
`),
	}

	parsed, err := parser.Parse(context.Background(), workflow)
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}

	if parsed.InputSchemaPath != "input_schema" {
		t.Fatalf("expected input_schema path, got %s", parsed.InputSchemaPath)
	}
	if parsed.OutputSchemaPath != "output_schema" {
		t.Fatalf("expected output_schema path, got %s", parsed.OutputSchemaPath)
	}
	if !json.Valid(parsed.Document) {
		t.Fatalf("expected normalized document to be valid JSON")
	}
	if !json.Valid(parsed.InputExample) {
		t.Fatalf("expected input example to be valid JSON")
	}
}

func TestParserParseJSONNestedSchema(t *testing.T) {
	t.Parallel()
	parser := NewParser()

	workflow := Workflow{
		ID: "workflow-json",
		Content: []byte(`{
  "input": {"schema": {"type": "object"}},
  "output": {"schema": {"type": "object"}}
}`),
	}

	parsed, err := parser.Parse(context.Background(), workflow)
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}

	if parsed.InputSchemaPath != "input.schema" {
		t.Fatalf("expected input.schema path, got %s", parsed.InputSchemaPath)
	}
	if parsed.OutputSchemaPath != "output.schema" {
		t.Fatalf("expected output.schema path, got %s", parsed.OutputSchemaPath)
	}
	if !json.Valid(parsed.InputSchema) || !json.Valid(parsed.OutputSchema) {
		t.Fatalf("expected schemas to be valid JSON")
	}
}

func TestParserErrorsOnMissingSchemas(t *testing.T) {
	t.Parallel()
	parser := NewParser()

	workflow := Workflow{
		ID:      "workflow-missing",
		Content: []byte("name: missing"),
	}

	_, err := parser.Parse(context.Background(), workflow)
	if err == nil {
		t.Fatalf("expected error on missing schemas")
	}
	if !strings.Contains(err.Error(), "input schema not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParserInfersSchemaFromInputOutput(t *testing.T) {
	t.Parallel()
	parser := NewParser()

	workflow := Workflow{
		ID: "workflow-infer",
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
outputs:
  success:
    status: ok
`),
	}

	parsed, err := parser.Parse(context.Background(), workflow)
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	if parsed.InputSchemaPath != "input" {
		t.Fatalf("expected input path, got %s", parsed.InputSchemaPath)
	}
	if parsed.OutputSchemaPath != "outputs" {
		t.Fatalf("expected outputs path, got %s", parsed.OutputSchemaPath)
	}
	if !json.Valid(parsed.InputSchema) || !json.Valid(parsed.OutputSchema) {
		t.Fatalf("expected schemas to be valid JSON")
	}
	if len(parsed.InputExample) != 0 {
		t.Fatalf("expected no example from inferred schema")
	}
}

func TestParserLoadsSchemaFromURL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(
					bytes.NewBufferString(`{"type":"object"}`),
				),
				Header:  make(http.Header),
				Request: req,
			}, nil
		}),
	}
	parser := NewParser(WithParserHTTPClient(client))

	workflow := Workflow{
		Content: []byte(`
input_schema_ref: "https://example.com/schema.json"
output_schema:
  type: object
`),
	}

	parsed, err := parser.Parse(ctx, workflow)
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	if len(parsed.InputSchema) == 0 {
		t.Fatalf("expected input schema from url")
	}
}

func TestParserLoggerOption(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := NewParser(WithParserLogger(logger))
	if parser.logger != logger {
		t.Fatalf("expected logger override")
	}
}

func TestParserSchemaRefHTTPError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBufferString("error")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}
	parser := NewParser(WithParserHTTPClient(client))

	workflow := Workflow{
		Content: []byte(`
input_schema_ref: "https://example.com/schema.json"
output_schema:
  type: object
`),
	}

	if _, err := parser.Parse(ctx, workflow); err == nil {
		t.Fatalf("expected schema ref error")
	}
}

func TestParserResolvesSchemaRefs(t *testing.T) {
	t.Parallel()
	parser := NewParser()
	root := t.TempDir()

	inputPath := filepath.Join(root, "input-schema.json")
	outputPath := filepath.Join(root, "output-schema.yaml")

	if err := os.WriteFile(
		inputPath,
		[]byte(`{"type":"object"}`),
		0o644,
	); err != nil {
		t.Fatalf("write input schema: %v", err)
	}
	if err := os.WriteFile(
		outputPath,
		[]byte("type: object"),
		0o644,
	); err != nil {
		t.Fatalf("write output schema: %v", err)
	}

	workflow := Workflow{
		ID:        "workflow-refs",
		LocalPath: filepath.Join(root, "workflow.yaml"),
		Content: []byte(`
input_schema_ref: input-schema.json
output_schema_ref: output-schema.yaml
`),
	}

	parsed, err := parser.Parse(context.Background(), workflow)
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}

	if parsed.InputSchemaPath != "input_schema_ref" {
		t.Fatalf("expected input_schema_ref path, got %s", parsed.InputSchemaPath)
	}
	if parsed.OutputSchemaPath != "output_schema_ref" {
		t.Fatalf("expected output_schema_ref path, got %s", parsed.OutputSchemaPath)
	}
	if !json.Valid(parsed.InputSchema) || !json.Valid(parsed.OutputSchema) {
		t.Fatalf("expected schemas to be valid JSON")
	}
}
