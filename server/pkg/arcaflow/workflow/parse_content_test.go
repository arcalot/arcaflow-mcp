package workflow

import (
	"encoding/json"
	"testing"
)

func TestParseContentJSON(t *testing.T) {
	t.Parallel()
	root, err := parseContent([]byte(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("parse content: %v", err)
	}
	if root["key"] != "value" {
		t.Fatalf("expected key to be parsed")
	}
}

func TestParseContentRejectsNonObject(t *testing.T) {
	t.Parallel()
	if _, err := parseContent([]byte(`["value"]`)); err == nil {
		t.Fatalf("expected error for non-object")
	}
}

func TestParseAnyNormalizesYAML(t *testing.T) {
	t.Parallel()
	payload, err := parseAny([]byte(`
items:
  - name: one
`))
	if err != nil {
		t.Fatalf("parse yaml: %v", err)
	}
	root, ok := payload.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map payload")
	}
	items, ok := root["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("expected items list")
	}
	item, ok := items[0].(map[string]interface{})
	if !ok || item["name"] != "one" {
		t.Fatalf("expected normalized map entry")
	}
}

func TestParseSchemaContentYAML(t *testing.T) {
	t.Parallel()
	raw, err := parseSchemaContent([]byte("type: object\n"))
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("expected schema JSON")
	}
}

func TestNormalizeYAMLMapInterface(t *testing.T) {
	t.Parallel()
	value := map[interface{}]interface{}{
		"key": map[interface{}]interface{}{
			"nested": "value",
		},
	}
	normalized := normalizeYAML(value)
	root, ok := normalized.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map normalization")
	}
	nested, ok := root["key"].(map[string]interface{})
	if !ok || nested["nested"] != "value" {
		t.Fatalf("expected nested normalization")
	}
}
