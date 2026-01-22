package workflow

import (
	"encoding/json"
	"testing"
)

func TestGenerateExampleInputUsesDefaults(t *testing.T) {
	t.Parallel()
	schema := json.RawMessage(`{
  "type": "object",
  "required": ["name", "count"],
  "properties": {
    "name": {"type": "string"},
    "count": {"type": "integer"}
  }
}`)

	example, err := GenerateExampleInput(schema)
	if err != nil {
		t.Fatalf("generate example: %v", err)
	}

	if !json.Valid(example) {
		t.Fatalf("expected valid JSON example")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(example, &decoded); err != nil {
		t.Fatalf("decode example: %v", err)
	}
	if _, ok := decoded["name"]; !ok {
		t.Fatalf("expected name in example")
	}
	if _, ok := decoded["count"]; !ok {
		t.Fatalf("expected count in example")
	}
}

func TestGenerateExampleInputUsesExampleValue(t *testing.T) {
	t.Parallel()
	schema := json.RawMessage(`{
  "type": "object",
  "example": {"value": 123}
}`)

	example, err := GenerateExampleInput(schema)
	if err != nil {
		t.Fatalf("generate example: %v", err)
	}

	if string(example) != `{"value":123}` {
		t.Fatalf("unexpected example output: %s", string(example))
	}
}

func TestGenerateExampleInputUsesArrayDefaults(t *testing.T) {
	t.Parallel()
	schema := json.RawMessage(`{
  "type": "object",
  "required": ["items"],
  "properties": {
    "items": {
      "type": "array",
      "items": {"type": "integer"}
    }
  }
}`)

	example, err := GenerateExampleInput(schema)
	if err != nil {
		t.Fatalf("generate example: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(example, &decoded); err != nil {
		t.Fatalf("decode example: %v", err)
	}
	items, ok := decoded["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("expected array example with one item")
	}
	if items[0].(float64) != 0 {
		t.Fatalf("expected zero-value item")
	}
}

func TestZeroValueForSchema(t *testing.T) {
	t.Parallel()
	if value := zeroValueForSchema(map[string]interface{}{"type": "string"}); value != "" {
		t.Fatalf("expected empty string zero value")
	}
	if value := zeroValueForSchema(map[string]interface{}{"type": "array"}); value == nil {
		t.Fatalf("expected array zero value")
	}
	if value := zeroValueForSchema(map[string]interface{}{"type": "object"}); value == nil {
		t.Fatalf("expected object zero value")
	}
	if value := zeroValueForSchema(map[string]interface{}{"type": "integer"}); value != 0 {
		t.Fatalf("expected integer zero value")
	}
	if value := zeroValueForSchema(map[string]interface{}{"type": "number"}); value != 0 {
		t.Fatalf("expected number zero value")
	}
	if value := zeroValueForSchema(map[string]interface{}{"type": "boolean"}); value != false {
		t.Fatalf("expected boolean zero value")
	}
	if value := zeroValueForSchema("not-a-map"); value != nil {
		t.Fatalf("expected nil for non-schema")
	}
}

func TestExtractSchemaTypeArray(t *testing.T) {
	t.Parallel()
	value := extractSchemaType(map[string]interface{}{
		"type": []interface{}{"integer", "string"},
	})
	if value != "integer" {
		t.Fatalf("expected integer type, got %s", value)
	}
}

func TestBuildExamplePriorityOrder(t *testing.T) {
	t.Parallel()
	example, ok := buildExample(map[string]interface{}{
		"examples": []interface{}{map[string]interface{}{"value": 1}},
	})
	if !ok {
		t.Fatalf("expected example from examples")
	}
	if example.(map[string]interface{})["value"].(int) != 1 {
		t.Fatalf("unexpected examples output")
	}

	example, ok = buildExample(map[string]interface{}{
		"default": "value",
	})
	if !ok || example.(string) != "value" {
		t.Fatalf("expected default example")
	}

	example, ok = buildExample(map[string]interface{}{
		"enum": []interface{}{"first", "second"},
	})
	if !ok || example.(string) != "first" {
		t.Fatalf("expected enum example")
	}
}

func TestExtractSchemaTypeFromProperties(t *testing.T) {
	t.Parallel()
	value := extractSchemaType(map[string]interface{}{
		"properties": map[string]interface{}{"name": map[string]interface{}{}},
	})
	if value != "object" {
		t.Fatalf("expected object type, got %s", value)
	}
}

func TestGenerateExampleInputRejectsInvalidSchema(t *testing.T) {
	t.Parallel()
	if _, err := GenerateExampleInput(json.RawMessage(`"not-an-object"`)); err == nil {
		t.Fatalf("expected error for invalid schema")
	}
}

func TestBuildArrayExampleWithoutItems(t *testing.T) {
	t.Parallel()
	result := buildArrayExample(map[string]interface{}{})
	if len(result) != 0 {
		t.Fatalf("expected empty array example")
	}
}

func TestBuildArrayExampleFallback(t *testing.T) {
	t.Parallel()
	result := buildArrayExample(map[string]interface{}{
		"items": "bad",
	})
	if len(result) != 1 {
		t.Fatalf("expected array fallback")
	}
	if result[0] != nil {
		t.Fatalf("expected nil fallback value")
	}
}
