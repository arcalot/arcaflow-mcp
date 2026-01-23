package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PluginSchemaProvider fetches plugin input JSON schemas.
type PluginSchemaProvider interface {
	InputJSONSchema(ctx context.Context, image string, stepID string) (json.RawMessage, error)
}

// NamespaceResolutionError captures resolver failures with actionable hints.
type NamespaceResolutionError struct {
	Message string
	Hint    string
}

func (err NamespaceResolutionError) Error() string {
	return err.Message
}

// InputSchemaResolver resolves workflow input JSON schemas.
type InputSchemaResolver struct {
	pluginProvider PluginSchemaProvider
	workflowCache  map[string]map[string]interface{}
	pluginCache    map[string]json.RawMessage
}

// InputSchemaResolverOption configures the resolver.
type InputSchemaResolverOption func(*InputSchemaResolver)

// NewInputSchemaResolver constructs an input schema resolver.
func NewInputSchemaResolver(options ...InputSchemaResolverOption) *InputSchemaResolver {
	resolver := &InputSchemaResolver{
		pluginProvider: NewContainerPluginSchemaProvider(),
		workflowCache:  make(map[string]map[string]interface{}),
		pluginCache:    make(map[string]json.RawMessage),
	}
	for _, option := range options {
		if option != nil {
			option(resolver)
		}
	}
	return resolver
}

// WithPluginSchemaProvider overrides the plugin schema provider.
func WithPluginSchemaProvider(provider PluginSchemaProvider) InputSchemaResolverOption {
	return func(resolver *InputSchemaResolver) {
		if provider != nil {
			resolver.pluginProvider = provider
		}
	}
}

// ResolveInputJSONSchema returns a resolved JSON schema for a workflow input.
func (resolver *InputSchemaResolver) ResolveInputJSONSchema(
	ctx context.Context,
	workflow Workflow,
) (json.RawMessage, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	root, err := parseContent(workflow.Content)
	if err != nil {
		return nil, err
	}
	inputScope, ok := root["input"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("input section is missing or invalid")
	}
	steps, _ := root["steps"].(map[string]interface{})
	baseDir := filepath.Dir(workflow.LocalPath)
	schema := resolver.buildScopeSchema(ctx, inputScope, steps, baseDir)
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("marshal input json schema: %w", err)
	}
	return raw, nil
}

func (resolver *InputSchemaResolver) buildScopeSchema(
	ctx context.Context,
	inputScope map[string]interface{},
	steps map[string]interface{},
	baseDir string,
) map[string]interface{} {
	rootID, _ := inputScope["root"].(string)
	objects, _ := inputScope["objects"].(map[string]interface{})
	if rootID == "" || objects == nil {
		return map[string]interface{}{"type": "object"}
	}
	rootObj, _ := objects[rootID].(map[string]interface{})
	return resolver.objectSchema(ctx, rootObj, objects, steps, baseDir)
}

func (resolver *InputSchemaResolver) objectSchema(
	ctx context.Context,
	object map[string]interface{},
	objects map[string]interface{},
	steps map[string]interface{},
	baseDir string,
) map[string]interface{} {
	properties := map[string]interface{}{}
	required := []string{}
	rawProps, _ := object["properties"].(map[string]interface{})
	for name, raw := range rawProps {
		property, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if requiredFlag, _ := property["required"].(bool); requiredFlag {
			required = append(required, name)
		}
		properties[name] = resolver.typeSchema(
			ctx,
			property["type"],
			objects,
			steps,
			baseDir,
		)
	}
	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func (resolver *InputSchemaResolver) typeSchema(
	ctx context.Context,
	typeSpec interface{},
	objects map[string]interface{},
	steps map[string]interface{},
	baseDir string,
) interface{} {
	spec, ok := typeSpec.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	typeID, _ := spec["type_id"].(string)
	switch typeID {
	case "string":
		return map[string]interface{}{"type": "string"}
	case "integer", "int":
		return map[string]interface{}{"type": "integer"}
	case "number", "float":
		return map[string]interface{}{"type": "number"}
	case "bool", "boolean":
		return map[string]interface{}{"type": "boolean"}
	case "enum_string":
		values := []string{}
		rawValues, _ := spec["values"].(map[string]interface{})
		for key := range rawValues {
			values = append(values, key)
		}
		return map[string]interface{}{
			"type": "string",
			"enum": values,
		}
	case "list":
		items := resolver.typeSchema(
			ctx,
			spec["items"],
			objects,
			steps,
			baseDir,
		)
		return map[string]interface{}{
			"type":  "array",
			"items": items,
		}
	case "ref":
		refID, _ := spec["id"].(string)
		if refID == "" {
			return map[string]interface{}{}
		}
		namespace, _ := spec["namespace"].(string)
		if namespace != "" {
			if resolved, err := resolver.resolveNamespaceSchema(
				ctx,
				namespace,
				refID,
				steps,
				baseDir,
			); err == nil && resolved != nil {
				return resolved
			}
		}
		if rawObj, ok := objects[refID].(map[string]interface{}); ok {
			return resolver.objectSchema(ctx, rawObj, objects, steps, baseDir)
		}
		return map[string]interface{}{}
	default:
		return map[string]interface{}{}
	}
}

func (resolver *InputSchemaResolver) resolveNamespaceSchema(
	ctx context.Context,
	namespace string,
	refID string,
	steps map[string]interface{},
	baseDir string,
) (interface{}, error) {
	stepName, path, ok := parseNamespace(namespace)
	if !ok {
		return nil, NamespaceResolutionError{
			Message: "unsupported namespace reference",
			Hint: "use step input namespaces like " +
				"$.steps.<step>.starting.inputs.input or " +
				"$.steps.<step>.execute.inputs.items.item",
		}
	}
	step, _ := steps[stepName].(map[string]interface{})
	if step == nil {
		return nil, fmt.Errorf("step %s not found", stepName)
	}
	if workflowPath, ok := step["workflow"].(string); ok && workflowPath != "" {
		if err := validateWorkflowNamespace(path); err != nil {
			return nil, err
		}
		subRoot, err := resolver.loadWorkflowRoot(baseDir, workflowPath)
		if err != nil {
			return nil, err
		}
		inputScope, _ := subRoot["input"].(map[string]interface{})
		objects, _ := inputScope["objects"].(map[string]interface{})
		subSteps, _ := subRoot["steps"].(map[string]interface{})
		if rawObj, ok := objects[refID].(map[string]interface{}); ok {
			return resolver.objectSchema(ctx, rawObj, objects, subSteps, baseDir), nil
		}
		return nil, fmt.Errorf("object %s not found in subworkflow", refID)
	}
	if pluginSpec, ok := step["plugin"].(map[string]interface{}); ok {
		if err := validatePluginNamespace(path); err != nil {
			return nil, err
		}
		image, _ := pluginSpec["src"].(string)
		stepID, _ := step["step"].(string)
		if image == "" {
			return nil, fmt.Errorf("plugin image missing for step %s", stepName)
		}
		cacheKey := fmt.Sprintf("%s::%s", image, stepID)
		rawSchema, ok := resolver.pluginCache[cacheKey]
		if !ok {
			fetched, err := resolver.pluginProvider.InputJSONSchema(ctx, image, stepID)
			if err != nil {
				return nil, err
			}
			resolver.pluginCache[cacheKey] = fetched
			rawSchema = fetched
		}
		var schema interface{}
		if err := json.Unmarshal(rawSchema, &schema); err != nil {
			return nil, err
		}
		return schema, nil
	}
	return nil, fmt.Errorf("unsupported step %s", stepName)
}

func (resolver *InputSchemaResolver) loadWorkflowRoot(
	baseDir string,
	workflowPath string,
) (map[string]interface{}, error) {
	path := workflowPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, workflowPath)
	}
	if cached, ok := resolver.workflowCache[path]; ok {
		return cached, nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workflow %s: %w", path, err)
	}
	root, err := parseContent(content)
	if err != nil {
		return nil, err
	}
	resolver.workflowCache[path] = root
	return root, nil
}

func parseNamespace(namespace string) (string, []string, bool) {
	trimmed := strings.TrimSpace(namespace)
	if !strings.HasPrefix(trimmed, "$.steps.") {
		return "", nil, false
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) < 3 {
		return "", nil, false
	}
	return parts[2], parts[3:], true
}

func validateWorkflowNamespace(path []string) error {
	if !containsPathToken(path, "execute") || !containsPathToken(path, "inputs") {
		return NamespaceResolutionError{
			Message: "unsupported sub-workflow namespace path",
			Hint: "use execute inputs like " +
				"$.steps.<step>.execute.inputs.items.item",
		}
	}
	return nil
}

func validatePluginNamespace(path []string) error {
	if !containsPathToken(path, "inputs") {
		return NamespaceResolutionError{
			Message: "unsupported plugin namespace path",
			Hint: "use starting inputs like " +
				"$.steps.<step>.starting.inputs.input",
		}
	}
	return nil
}

func containsPathToken(path []string, token string) bool {
	for _, part := range path {
		if part == token {
			return true
		}
	}
	return false
}
