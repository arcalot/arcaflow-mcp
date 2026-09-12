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

func TestGenerateExampleInputFromInputScope(t *testing.T) {
	t.Parallel()
	inputSchema := json.RawMessage(`{
  "root": "RootObject",
  "objects": {
    "RootObject": {
      "id": "RootObject",
      "properties": {
        "nickname": {
          "required": true,
          "type": {"type_id": "string"}
        }
      }
    }
  }
}`)
	example, err := GenerateExampleInputFromInputScope(inputSchema)
	if err != nil {
		t.Fatalf("generate input scope example: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(example, &decoded); err != nil {
		t.Fatalf("decode input scope example: %v", err)
	}
	if _, ok := decoded["nickname"]; !ok {
		t.Fatalf("expected nickname in example")
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

func TestGenerateExampleInputInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := GenerateExampleInput(json.RawMessage("{invalid"))
	if err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestGenerateExampleInputUnsupportedSchema(t *testing.T) {
	t.Parallel()

	_, err := GenerateExampleInput(json.RawMessage(`{"type":"unknown"}`))
	if err == nil {
		t.Fatalf("expected unsupported schema error")
	}
}

func TestGenerateExampleInputFromInputScopeErrors(t *testing.T) {
	t.Parallel()

	_, err := GenerateExampleInputFromInputScope(json.RawMessage(`{"root":""}`))
	if err == nil {
		t.Fatalf("expected missing root error")
	}

	_, err = GenerateExampleInputFromInputScope(json.RawMessage(`{"root":"Root","objects":{}}`))
	if err == nil {
		t.Fatalf("expected missing root object error")
	}
}

func TestExampleValueForTypeBranches(t *testing.T) {
	t.Parallel()

	objects := map[string]interface{}{
		"Obj": map[string]interface{}{
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"required": true,
					"type": map[string]interface{}{
						"type_id": "string",
					},
				},
			},
		},
	}
	value := exampleValueForType(map[string]interface{}{
		"type_id": "object",
		"id":      "Obj",
	}, objects)
	if _, ok := value.(map[string]interface{}); !ok {
		t.Fatalf("expected object example")
	}

	value = exampleValueForType(map[string]interface{}{
		"type_id": "object",
		"properties": map[string]interface{}{
			"count": map[string]interface{}{
				"required": true,
				"type": map[string]interface{}{
					"type_id": "integer",
				},
			},
		},
	}, objects)
	if _, ok := value.(map[string]interface{}); !ok {
		t.Fatalf("expected inline object example")
	}

	value = exampleValueForType(map[string]interface{}{
		"type_id": "list",
		"items": map[string]interface{}{
			"type_id": "boolean",
		},
	}, objects)
	if list, ok := value.([]interface{}); !ok || len(list) != 1 {
		t.Fatalf("expected list example")
	}

	if value := exampleValueForType("invalid", objects); value != nil {
		t.Fatalf("expected nil for invalid type")
	}
}
