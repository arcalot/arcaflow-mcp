package pluginschema

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromWorkflow(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "plugin-schema.yaml")
	if err := os.WriteFile(schemaPath, []byte("type: object\n"), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	workflow := []byte(`
steps:
  step-a:
    plugin_schema_ref: plugin-schema.yaml
`)

	handler := NewHandler(nil)
	entries, err := handler.LoadFromWorkflow(
		context.Background(),
		workflow,
		filepath.Join(dir, "workflow.yaml"),
	)
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 schema entry, got %d", len(entries))
	}
	if entries[0].StepID != "step-a" {
		t.Fatalf("expected step-a, got %s", entries[0].StepID)
	}
	if len(entries[0].Schema) == 0 {
		t.Fatalf("expected schema payload")
	}
}

func TestLoadFromWorkflowMissingSchema(t *testing.T) {
	t.Parallel()
	workflow := []byte(`
steps:
  step-a:
    plugin_schema_ref: missing.yaml
`)

	handler := NewHandler(nil)
	_, err := handler.LoadFromWorkflow(context.Background(), workflow, "")
	if err == nil {
		t.Fatalf("expected error for missing schema file")
	}
}

func TestCacheReturnsCopy(t *testing.T) {
	t.Parallel()
	handler := NewHandler(nil)
	handler.cache["step|schema"] = SchemaEntry{
		StepID:   "step",
		Location: "schema",
		Schema:   json.RawMessage(`{"type":"object"}`),
	}
	cached := handler.Cache()
	if len(cached) != 1 {
		t.Fatalf("expected 1 cached entry, got %d", len(cached))
	}
	delete(cached, "step|schema")
	if len(handler.cache) != 1 {
		t.Fatalf("expected cache copy isolation")
	}
}

func TestReadSchemaErrors(t *testing.T) {
	t.Parallel()
	handler := NewHandler(nil)
	if _, err := handler.readSchema("", ""); err == nil {
		t.Fatalf("expected error for empty schema location")
	}
	if _, err := handler.readSchema("schema.yaml", ""); err == nil {
		t.Fatalf("expected error for missing workflow path")
	}
}

func TestParseWorkflowDocumentNormalizesYAML(t *testing.T) {
	t.Parallel()
	workflow := []byte(`
steps:
  step-a:
    plugin:
      schema_ref: plugin.yaml
`)
	root, err := parseWorkflowDocument(workflow)
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	steps, ok := root["steps"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected steps map")
	}
	step, ok := steps["step-a"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected step map")
	}
	plugin, ok := step["plugin"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected plugin map")
	}
	if plugin["schema_ref"] != "plugin.yaml" {
		t.Fatalf("expected schema_ref to normalize")
	}
}

func TestReadSchemaFromFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.json")
	if err := os.WriteFile(schemaPath, []byte(`{"type":"object"}`), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	handler := NewHandler(nil)
	payload, err := handler.readSchema(schemaPath, "")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if len(payload) == 0 {
		t.Fatalf("expected schema payload")
	}
}

func TestReadSchemaFromHTTP(t *testing.T) {
	t.Parallel()
	handler := NewHandler(nil)
	handler.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("type: object\n")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}
	payload, err := handler.readSchema("https://example.com/schema.yaml", "")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if len(payload) == 0 {
		t.Fatalf("expected schema payload")
	}
}

func TestParseWorkflowDocumentJSON(t *testing.T) {
	t.Parallel()
	workflow := []byte(`{"steps":{"step-a":{"plugin_schema_ref":"schema.json"}}}`)
	root, err := parseWorkflowDocument(workflow)
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	steps, ok := root["steps"].(map[string]interface{})
	if !ok || steps["step-a"] == nil {
		t.Fatalf("expected step map")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTripper roundTripFunc) RoundTrip(
	request *http.Request,
) (*http.Response, error) {
	return roundTripper(request)
}

func TestNormalizeYAMLTypes(t *testing.T) {
	t.Parallel()
	raw := map[interface{}]interface{}{
		"list": []interface{}{
			map[interface{}]interface{}{"key": "value"},
		},
	}
	normalized := normalizeYAML(raw)
	root, ok := normalized.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map normalization")
	}
	list, ok := root["list"].([]interface{})
	if !ok || len(list) != 1 {
		t.Fatalf("expected list normalization")
	}
	item, ok := list[0].(map[string]interface{})
	if !ok || item["key"] != "value" {
		t.Fatalf("expected nested map normalization")
	}
}

func TestExtractSchemaLocations(t *testing.T) {
	t.Parallel()
	step := map[string]interface{}{
		"plugin_schema_ref": "ref-a",
		"plugin_schema":     "schema-a",
		"plugin": map[string]interface{}{
			"schema_ref": "ref-b",
			"schema":     "schema-b",
		},
	}
	locations := extractSchemaLocations(step)
	if len(locations) != 4 {
		t.Fatalf("expected 4 schema locations, got %d", len(locations))
	}
}
