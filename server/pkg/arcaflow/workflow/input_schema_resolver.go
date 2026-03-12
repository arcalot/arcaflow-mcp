package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.flow.arcalot.io/engine"
	"go.flow.arcalot.io/engine/config"
	"go.flow.arcalot.io/engine/loadfile"
	"go.flow.arcalot.io/pluginsdk/schema"
)

// NamespaceResolutionError captures resolver failures with actionable hints.
type NamespaceResolutionError struct {
	Message string
	Hint    string
}

func (err NamespaceResolutionError) Error() string {
	return err.Message
}

// InputSchemaResolver resolves workflow input JSON schemas using the engine SDK.
type InputSchemaResolver struct {
	config *config.Config
}

// InputSchemaResolverOption configures the resolver.
type InputSchemaResolverOption func(*InputSchemaResolver)

// NewInputSchemaResolver constructs an input schema resolver.
func NewInputSchemaResolver(options ...InputSchemaResolverOption) *InputSchemaResolver {
	resolver := &InputSchemaResolver{
		config: config.Default(),
	}
	for _, option := range options {
		if option != nil {
			option(resolver)
		}
	}
	return resolver
}

// WithInputSchemaResolverConfig sets the engine configuration.
func WithInputSchemaResolverConfig(cfg *config.Config) InputSchemaResolverOption {
	return func(resolver *InputSchemaResolver) {
		if cfg != nil {
			resolver.config = cfg
		}
	}
}

// WithPluginSchemaProvider is a legacy option kept for compatibility.
func WithPluginSchemaProvider(_ PluginSchemaProvider) InputSchemaResolverOption {
	return func(_ *InputSchemaResolver) {}
}

// ResolveInputJSONSchema returns a resolved JSON schema for a workflow input.
func (resolver *InputSchemaResolver) ResolveInputJSONSchema(
	ctx context.Context,
	workflow Workflow,
) (json.RawMessage, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// 1. Initialize the engine
	eng, err := engine.New(resolver.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	// 2. Prepare the file cache
	workflowFileName := "workflow.yaml"
	if workflow.Path != "" {
		workflowFileName = filepath.Base(workflow.Path)
	}
	fileContents := map[string][]byte{
		workflowFileName: workflow.Content,
	}

	if workflow.LocalPath != "" {
		baseDir := filepath.Dir(workflow.LocalPath)
		entries, err := os.ReadDir(baseDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml")) {
					if entry.Name() == workflowFileName {
						continue
					}
					content, err := os.ReadFile(filepath.Join(baseDir, entry.Name()))
					if err == nil {
						fileContents[entry.Name()] = content
					}
				}
			}
		}
	}

	fc := loadfile.NewFileCache(filepath.Dir(workflow.LocalPath), fileContents)

	// 3. Parse the workflow to resolve all schemas and expressions
	wf, err := eng.Parse(fc, workflowFileName)
	if err != nil {
		return nil, fmt.Errorf("parse workflow: %w", err)
	}

	// 4. Convert the Arcaflow scope to JSON Schema
	inputScope := wf.InputSchema()
	jsonSchema := arcaflowScopeToJSONSchema(inputScope)

	raw, err := json.Marshal(jsonSchema)
	if err != nil {
		return nil, fmt.Errorf("marshal input json schema: %w", err)
	}
	return raw, nil
}

func arcaflowScopeToJSONSchema(scope schema.Scope) map[string]any {
	rootObj := scope.RootObject()
	if rootObj == nil {
		return map[string]any{"type": "object"}
	}
	return arcaflowObjectToJSONSchema(rootObj, scope)
}

func arcaflowObjectToJSONSchema(obj *schema.ObjectSchema, scope schema.Scope) map[string]any {
	properties := map[string]any{}
	required := []string{}

	for id, prop := range obj.Properties() {
		properties[id] = arcaflowTypeToJSONSchema(prop.Type(), scope)
		if prop.Required() {
			required = append(required, id)
		}
	}

	result := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

func arcaflowTypeToJSONSchema(t schema.Type, scope schema.Scope) any {
	switch t.TypeID() {
	case schema.TypeIDString:
		return map[string]any{"type": "string"}
	case schema.TypeIDInt:
		return map[string]any{"type": "integer"}
	case schema.TypeIDFloat:
		return map[string]any{"type": "number"}
	case schema.TypeIDBool:
		return map[string]any{"type": "boolean"}
	case schema.TypeIDList:
		listT := t.(schema.UntypedList)
		return map[string]any{
			"type":  "array",
			"items": arcaflowTypeToJSONSchema(listT.Items(), scope),
		}
	case schema.TypeIDMap:
		mapT := t.(schema.UntypedMap)
		return map[string]any{
			"type":                 "object",
			"additionalProperties": arcaflowTypeToJSONSchema(mapT.Values(), scope),
		}
	case schema.TypeIDObject:
		objT := t.(*schema.ObjectSchema)
		return arcaflowObjectToJSONSchema(objT, scope)
	case schema.TypeIDRef:
		refT := t.(schema.Ref)
		refID := refT.ID()
		// Try to resolve the ref in the scope
		if objects := scope.Objects(); objects != nil {
			if _, ok := objects[refID]; ok {
				// To avoid infinite recursion in circular refs for JSON Schema (which we don't handle deeply here),
				// we just return an object type. Real JSON Schema would use $ref.
				// For MCP's simplified needs, this is usually enough.
				return map[string]any{"type": "object", "title": refID}
			}
		}
		return map[string]any{"type": "object"}
	case schema.TypeIDStringEnum:
		enumT := t.(schema.StringEnum)
		values := []string{}
		for v := range enumT.ValidValues() {
			values = append(values, v)
		}
		return map[string]any{
			"type": "string",
			"enum": values,
		}
	default:
		return map[string]any{}
	}
}
