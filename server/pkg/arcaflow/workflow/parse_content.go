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

	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("parse yaml content: %w", err)
	}
	if doc.Kind == 0 {
		return nil, nil
	}
	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	return nodeToInterface(root), nil
}

// nodeToInterface converts a yaml.Node tree to Go types, preserving
// custom-tagged values (like !expr) as strings.
func nodeToInterface(n *yaml.Node) interface{} {
	switch n.Kind {
	case yaml.MappingNode:
		result := make(map[string]interface{}, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			key := n.Content[i].Value
			result[key] = nodeToInterface(n.Content[i+1])
		}
		return result
	case yaml.SequenceNode:
		result := make([]interface{}, 0, len(n.Content))
		for _, child := range n.Content {
			result = append(result, nodeToInterface(child))
		}
		return result
	case yaml.ScalarNode:
		return scalarToInterface(n)
	case yaml.AliasNode:
		if n.Alias != nil {
			return nodeToInterface(n.Alias)
		}
		return nil
	default:
		return n.Value
	}
}

func scalarToInterface(n *yaml.Node) interface{} {
	switch n.Tag {
	case "!!null":
		return nil
	case "!!bool":
		return n.Value == "true"
	case "!!int":
		var v int64
		if err := n.Decode(&v); err == nil {
			return v
		}
		return n.Value
	case "!!float":
		var v float64
		if err := n.Decode(&v); err == nil {
			return v
		}
		return n.Value
	default:
		return n.Value
	}
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
