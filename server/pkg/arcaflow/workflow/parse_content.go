package workflow

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

func parseContent(content []byte) (map[string]interface{}, error) {
	root, err := parseAny(content)
	if err != nil {
		return nil, err
	}
	return ensureObject(root)
}

// ParseDocument parses workflow content into a normalized object.
func ParseDocument(content []byte) (map[string]interface{}, error) {
	return parseContent(content)
}

func parseSchemaContent(content []byte) (json.RawMessage, error) {
	root, err := parseAny(content)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}
	return raw, nil
}

func parseAny(content []byte) (interface{}, error) {
	if json.Valid(content) {
		var root interface{}
		if err := json.Unmarshal(content, &root); err != nil {
			return nil, fmt.Errorf("parse json content: %w", err)
		}
		return root, nil
	}

	var root interface{}
	if err := yaml.Unmarshal(content, &root); err != nil {
		return nil, fmt.Errorf("parse yaml content: %w", err)
	}

	return normalizeYAML(root), nil
}

func ensureObject(value interface{}) (map[string]interface{}, error) {
	root, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("workflow document must be a JSON object")
	}
	return root, nil
}

func normalizeYAML(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		normalized := make(map[string]interface{}, len(typed))
		for key, child := range typed {
			normalized[key] = normalizeYAML(child)
		}
		return normalized
	case map[interface{}]interface{}:
		normalized := make(map[string]interface{}, len(typed))
		for key, child := range typed {
			normalized[fmt.Sprint(key)] = normalizeYAML(child)
		}
		return normalized
	case []interface{}:
		normalized := make([]interface{}, 0, len(typed))
		for _, child := range typed {
			normalized = append(normalized, normalizeYAML(child))
		}
		return normalized
	default:
		return typed
	}
}
