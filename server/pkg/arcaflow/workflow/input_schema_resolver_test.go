package workflow

import (
	"testing"

	engineconfig "go.flow.arcalot.io/engine/config"
	"go.flow.arcalot.io/pluginsdk/schema"
)

func TestNewInputSchemaResolver(t *testing.T) {
	resolver := NewInputSchemaResolver()
	if resolver == nil {
		t.Fatalf("expected resolver")
	}
}

func TestInputSchemaResolverConfigOption(t *testing.T) {
	cfg := &engineconfig.Config{}
	resolver := NewInputSchemaResolver(
		WithInputSchemaResolverConfig(cfg),
	)
	if resolver == nil {
		t.Fatalf("expected resolver")
	}
	if resolver.config != cfg {
		t.Fatalf("expected config override")
	}
}

func TestInputSchemaResolverNilConfigIgnored(t *testing.T) {
	resolver := NewInputSchemaResolver(
		WithInputSchemaResolverConfig(nil),
	)
	if resolver.config == nil {
		t.Fatalf("expected default config preserved")
	}
}

func TestInputSchemaResolverWithLegacyOption(t *testing.T) {
	resolver := NewInputSchemaResolver(WithPluginSchemaProvider(nil))
	if resolver == nil {
		t.Fatalf("expected resolver")
	}
}

func TestArcaflowScopeToJSONSchemaEmpty(t *testing.T) {
	scope := schema.NewScopeSchema(
		schema.NewObjectSchema("Empty", nil),
	)
	result := arcaflowScopeToJSONSchema(scope)
	if result["type"] != "object" {
		t.Fatalf("expected type=object, got %v", result["type"])
	}
}

func TestArcaflowScopeToJSONSchemaStringProperty(t *testing.T) {
	scope := schema.NewScopeSchema(
		schema.NewObjectSchema("Input", map[string]*schema.PropertySchema{
			"name": schema.NewPropertySchema(
				schema.NewStringSchema(nil, nil, nil),
				nil, true, nil, nil, nil, nil, nil,
			),
		}),
	)
	result := arcaflowScopeToJSONSchema(scope)
	props, ok := result["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map")
	}
	nameProp, ok := props["name"].(map[string]any)
	if !ok {
		t.Fatalf("expected name property")
	}
	if nameProp["type"] != "string" {
		t.Fatalf("expected string type, got %v", nameProp["type"])
	}
	req, ok := result["required"].([]string)
	if !ok {
		t.Fatalf("expected required array")
	}
	if len(req) != 1 || req[0] != "name" {
		t.Fatalf("expected [name] required, got %v", req)
	}
}

func TestArcaflowTypeToJSONSchemaInt(t *testing.T) {
	scope := schema.NewScopeSchema(
		schema.NewObjectSchema("X", nil),
	)
	result := arcaflowTypeToJSONSchema(
		schema.NewIntSchema(nil, nil, nil), scope,
	)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["type"] != "integer" {
		t.Fatalf("expected integer, got %v", m["type"])
	}
}

func TestArcaflowTypeToJSONSchemaFloat(t *testing.T) {
	scope := schema.NewScopeSchema(
		schema.NewObjectSchema("X", nil),
	)
	result := arcaflowTypeToJSONSchema(
		schema.NewFloatSchema(nil, nil, nil), scope,
	)
	m := result.(map[string]any)
	if m["type"] != "number" {
		t.Fatalf("expected number, got %v", m["type"])
	}
}

func TestArcaflowTypeToJSONSchemaBool(t *testing.T) {
	scope := schema.NewScopeSchema(
		schema.NewObjectSchema("X", nil),
	)
	result := arcaflowTypeToJSONSchema(
		schema.NewBoolSchema(), scope,
	)
	m := result.(map[string]any)
	if m["type"] != "boolean" {
		t.Fatalf("expected boolean, got %v", m["type"])
	}
}

func TestArcaflowTypeToJSONSchemaList(t *testing.T) {
	scope := schema.NewScopeSchema(
		schema.NewObjectSchema("X", nil),
	)
	result := arcaflowTypeToJSONSchema(
		schema.NewListSchema(
			schema.NewStringSchema(nil, nil, nil), nil, nil,
		),
		scope,
	)
	m := result.(map[string]any)
	if m["type"] != "array" {
		t.Fatalf("expected array, got %v", m["type"])
	}
	items, ok := m["items"].(map[string]any)
	if !ok {
		t.Fatalf("expected items map")
	}
	if items["type"] != "string" {
		t.Fatalf("expected string items, got %v", items["type"])
	}
}

func TestArcaflowTypeToJSONSchemaMap(t *testing.T) {
	scope := schema.NewScopeSchema(
		schema.NewObjectSchema("X", nil),
	)
	result := arcaflowTypeToJSONSchema(
		schema.NewMapSchema(
			schema.NewStringSchema(nil, nil, nil),
			schema.NewIntSchema(nil, nil, nil),
			nil, nil,
		),
		scope,
	)
	m := result.(map[string]any)
	if m["type"] != "object" {
		t.Fatalf("expected object, got %v", m["type"])
	}
	addlProps, ok := m["additionalProperties"].(map[string]any)
	if !ok {
		t.Fatalf("expected additionalProperties map")
	}
	if addlProps["type"] != "integer" {
		t.Fatalf(
			"expected integer values, got %v",
			addlProps["type"],
		)
	}
}

func TestArcaflowTypeToJSONSchemaObject(t *testing.T) {
	scope := schema.NewScopeSchema(
		schema.NewObjectSchema("X", nil),
	)
	objSchema := schema.NewObjectSchema("Nested", map[string]*schema.PropertySchema{
		"value": schema.NewPropertySchema(
			schema.NewStringSchema(nil, nil, nil),
			nil, true, nil, nil, nil, nil, nil,
		),
	})
	result := arcaflowTypeToJSONSchema(objSchema, scope)
	m := result.(map[string]any)
	if m["type"] != "object" {
		t.Fatalf("expected object, got %v", m["type"])
	}
	props := m["properties"].(map[string]any)
	if _, ok := props["value"]; !ok {
		t.Fatalf("expected value property")
	}
}

func TestArcaflowTypeToJSONSchemaStringEnum(t *testing.T) {
	scope := schema.NewScopeSchema(
		schema.NewObjectSchema("X", nil),
	)
	enumSchema := schema.NewStringEnumSchema(map[string]*schema.DisplayValue{
		"a": nil,
		"b": nil,
	})
	result := arcaflowTypeToJSONSchema(enumSchema, scope)
	m := result.(map[string]any)
	if m["type"] != "string" {
		t.Fatalf("expected string, got %v", m["type"])
	}
	enumVals, ok := m["enum"].([]string)
	if !ok {
		t.Fatalf("expected enum array")
	}
	if len(enumVals) != 2 {
		t.Fatalf("expected 2 enum values, got %d", len(enumVals))
	}
}

func TestArcaflowTypeToJSONSchemaRef(t *testing.T) {
	refObj := schema.NewObjectSchema("RefTarget", nil)
	scope := schema.NewScopeSchema(refObj)
	refSchema := schema.NewRefSchema("RefTarget", nil)
	result := arcaflowTypeToJSONSchema(refSchema, scope)
	m := result.(map[string]any)
	if m["type"] != "object" {
		t.Fatalf("expected object, got %v", m["type"])
	}
}

func TestArcaflowScopeToJSONSchemaOptionalField(t *testing.T) {
	scope := schema.NewScopeSchema(
		schema.NewObjectSchema("Input", map[string]*schema.PropertySchema{
			"optional": schema.NewPropertySchema(
				schema.NewStringSchema(nil, nil, nil),
				nil, false, nil, nil, nil, nil, nil,
			),
		}),
	)
	result := arcaflowScopeToJSONSchema(scope)
	if _, ok := result["required"]; ok {
		t.Fatalf("expected no required field for optional-only schema")
	}
}
