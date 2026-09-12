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

func TestParseDocumentJSON(t *testing.T) {
	t.Parallel()
	root, err := ParseDocument([]byte(`{"name":"workflow"}`))
	if err != nil {
		t.Fatalf("parse document: %v", err)
	}
	if root["name"] != "workflow" {
		t.Fatalf("expected name to be parsed")
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

func TestParseAnyHandlesExprTags(t *testing.T) {
	t.Parallel()
	payload, err := parseAny([]byte(`
version: v0.2.0
steps:
  hello:
    input:
      message: !expr $.input.name
`))
	if err != nil {
		t.Fatalf("parse yaml with !expr: %v", err)
	}
	root, ok := payload.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map payload")
	}
	if root["version"] != "v0.2.0" {
		t.Fatalf("expected version, got %v", root["version"])
	}
	steps, ok := root["steps"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected steps map")
	}
	hello, ok := steps["hello"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected hello step")
	}
	input, ok := hello["input"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected input map")
	}
	if input["message"] != "$.input.name" {
		t.Fatalf(
			"expected expr value preserved as string, got %v",
			input["message"],
		)
	}
}

func TestParseAnyHandlesQuotedExpr(t *testing.T) {
	t.Parallel()
	payload, err := parseAny([]byte(
		"value: !expr '`Hello` + $.input.name'\n",
	))
	if err != nil {
		t.Fatalf("parse yaml with quoted !expr: %v", err)
	}
	root := payload.(map[string]interface{})
	if root["value"] != "`Hello` + $.input.name" {
		t.Fatalf("expected expr string, got %v", root["value"])
	}
}

func TestParseAnyPreservesYAMLTypes(t *testing.T) {
	t.Parallel()
	payload, err := parseAny([]byte(`
str: hello
num: 42
flt: 3.14
flag: true
empty: null
`))
	if err != nil {
		t.Fatalf("parse yaml types: %v", err)
	}
	root := payload.(map[string]interface{})
	if root["str"] != "hello" {
		t.Fatalf("expected string, got %T %v", root["str"], root["str"])
	}
	if root["num"] != int64(42) {
		t.Fatalf("expected int64, got %T %v", root["num"], root["num"])
	}
	if root["flt"] != 3.14 {
		t.Fatalf("expected float64, got %T %v", root["flt"], root["flt"])
	}
	if root["flag"] != true {
		t.Fatalf("expected bool, got %T %v", root["flag"], root["flag"])
	}
	if root["empty"] != nil {
		t.Fatalf("expected nil, got %T %v", root["empty"], root["empty"])
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
