package workflow

import (
	"encoding/json"
	"fmt"
)

// GenerateExampleInput builds a minimal example payload from a JSON schema.
func GenerateExampleInput(schema json.RawMessage) (json.RawMessage, error) {
	var root interface{}
	if err := json.Unmarshal(schema, &root); err != nil {
		return nil, fmt.Errorf("parse schema for example: %w", err)
	}
	example, ok := buildExample(root)
	if !ok {
		return nil, fmt.Errorf("unable to generate example input")
	}
	raw, err := json.Marshal(example)
	if err != nil {
		return nil, fmt.Errorf("marshal example input: %w", err)
	}
	return raw, nil
}

// GenerateExampleInputFromInputScope builds a minimal example from Arcaflow input.
func GenerateExampleInputFromInputScope(
	inputSchema json.RawMessage,
) (json.RawMessage, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(inputSchema, &payload); err != nil {
		return nil, fmt.Errorf("parse input scope: %w", err)
	}
	rootID, _ := payload["root"].(string)
	objects, _ := payload["objects"].(map[string]interface{})
	if rootID == "" || objects == nil {
		return nil, fmt.Errorf("input scope missing root objects")
	}
	rootObject, ok := objects[rootID].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("input scope missing root object %q", rootID)
	}
	example := buildInputObjectExample(rootObject, objects)
	raw, err := json.Marshal(example)
	if err != nil {
		return nil, fmt.Errorf("marshal input example: %w", err)
	}
	return raw, nil
}

func buildInputObjectExample(
	object map[string]interface{},
	objects map[string]interface{},
) map[string]interface{} {
	props, _ := object["properties"].(map[string]interface{})
	example := make(map[string]interface{}, len(props))
	for name, raw := range props {
		property, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if required, _ := property["required"].(bool); !required {
			continue
		}
		example[name] = exampleValueForType(property["type"], objects)
	}
	return example
}

func exampleValueForType(
	typeSpec interface{},
	objects map[string]interface{},
) interface{} {
	spec, ok := typeSpec.(map[string]interface{})
	if !ok {
		return nil
	}
	typeID, _ := spec["type_id"].(string)
	switch typeID {
	case "string":
		return ""
	case "integer", "int":
		return 0
	case "number", "float":
		return 0
	case "bool", "boolean":
		return false
	case "object":
		objectID, _ := spec["id"].(string)
		if objectID == "" {
			objectID, _ = spec["object_id"].(string)
		}
		if objectID != "" {
			if raw, ok := objects[objectID].(map[string]interface{}); ok {
				return buildInputObjectExample(raw, objects)
			}
		}
		if rawProps, ok := spec["properties"].(map[string]interface{}); ok {
			return buildInputObjectExample(
				map[string]interface{}{"properties": rawProps},
				objects,
			)
		}
		return map[string]interface{}{}
	case "list":
		item := exampleValueForType(spec["items"], objects)
		return []interface{}{item}
	default:
		return nil
	}
}

func buildExample(value interface{}) (interface{}, bool) {
	schema, ok := value.(map[string]interface{})
	if !ok {
		return nil, false
	}

	if example, ok := schema["example"]; ok {
		return example, true
	}
	if examples, ok := schema["examples"].([]interface{}); ok && len(examples) > 0 {
		return examples[0], true
	}
	if def, ok := schema["default"]; ok {
		return def, true
	}
	if enum, ok := schema["enum"].([]interface{}); ok && len(enum) > 0 {
		return enum[0], true
	}

	schemaType := extractSchemaType(schema)
	switch schemaType {
	case "object":
		return buildObjectExample(schema), true
	case "array":
		return buildArrayExample(schema), true
	case "string":
		return "", true
	case "integer":
		return 0, true
	case "number":
		return 0, true
	case "boolean":
		return false, true
	default:
		return nil, false
	}
}

func extractSchemaType(schema map[string]interface{}) string {
	if value, ok := schema["type"]; ok {
		switch typed := value.(type) {
		case string:
			return typed
		case []interface{}:
			for _, entry := range typed {
				if name, ok := entry.(string); ok {
					return name
				}
			}
		}
	}
	if _, ok := schema["properties"]; ok {
		return "object"
	}
	return ""
}

func buildObjectExample(schema map[string]interface{}) map[string]interface{} {
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}

	required := requiredSet(schema["required"])
	limitToRequired := len(required) > 0
	example := make(map[string]interface{}, len(props))
	for name, raw := range props {
		if limitToRequired && !required[name] {
			continue
		}
		value, ok := buildExample(raw)
		if !ok {
			value = zeroValueForSchema(raw)
		}
		example[name] = value
	}
	return example
}

func buildArrayExample(schema map[string]interface{}) []interface{} {
	rawItems, ok := schema["items"]
	if !ok {
		return []interface{}{}
	}
	item, ok := buildExample(rawItems)
	if !ok {
		item = zeroValueForSchema(rawItems)
	}
	return []interface{}{item}
}

func zeroValueForSchema(schema interface{}) interface{} {
	typed, ok := schema.(map[string]interface{})
	if !ok {
		return nil
	}
	switch extractSchemaType(typed) {
	case "object":
		return map[string]interface{}{}
	case "array":
		return []interface{}{}
	case "string":
		return ""
	case "integer", "number":
		return 0
	case "boolean":
		return false
	default:
		return nil
	}
}

func requiredSet(value interface{}) map[string]bool {
	items, ok := value.([]interface{})
	if !ok {
		return map[string]bool{}
	}
	set := make(map[string]bool, len(items))
	for _, item := range items {
		if name, ok := item.(string); ok {
			set[name] = true
		}
	}
	return set
}
