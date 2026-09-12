package workflow

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestParserParseYAML(t *testing.T) {
	t.Parallel()
	parser := NewParser()

	workflow := Workflow{
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
outputSchema:
  type: object
  properties:
    status:
      type: string
`),
	}

	parsed, err := parser.Parse(context.Background(), workflow)
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}

	if parsed.InputSchemaPath != "input" {
		t.Fatalf("expected input path, got %s", parsed.InputSchemaPath)
	}
	if parsed.OutputSchemaPath != "outputSchema" {
		t.Fatalf("expected outputSchema path, got %s", parsed.OutputSchemaPath)
	}
	if !json.Valid(parsed.Document) {
		t.Fatalf("expected normalized document to be valid JSON")
	}
	if len(parsed.InputExample) != 0 {
		t.Fatalf("expected no example from arcaflow input schema")
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

func TestParserHandlesDerivedOutputSchema(t *testing.T) {
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
	if len(parsed.OutputSchema) == 0 {
		t.Fatalf("expected derived output schema")
	}
	if len(parsed.InputExample) != 0 {
		t.Fatalf("expected no example from inferred schema")
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

func TestParserHTTPClientOption(t *testing.T) {
	t.Parallel()
	client := &http.Client{}
	parser := NewParser(WithParserHTTPClient(client))
	if parser.httpClient != client {
		t.Fatalf("expected http client override")
	}
}

func TestParserRejectsNullInputSchema(t *testing.T) {
	t.Parallel()

	parser := NewParser()
	workflow := Workflow{
		ID: "workflow-null",
		Content: []byte(`
version: v0.2.0
input: null
`),
	}

	if _, err := parser.Parse(context.Background(), workflow); err == nil {
		t.Fatalf("expected null schema error")
	}
}
