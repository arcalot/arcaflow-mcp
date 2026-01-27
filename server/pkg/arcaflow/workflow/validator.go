package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"go.flow.arcalot.io/pluginsdk/schema"
)

// ValidationIssue describes a single input validation failure.
type ValidationIssue struct {
	Path       string
	Message    string
	Suggestion string
}

// ValidationResult captures the validation outcome and normalized input payload.
type ValidationResult struct {
	Valid           bool
	NormalizedInput json.RawMessage
	Issues          []ValidationIssue
}

// InputValidator validates workflow inputs against Arcaflow input schemas.
type InputValidator struct {
	logger         *slog.Logger
	pluginProvider PluginSchemaProvider
}

// InputValidatorOption configures the input validator.
type InputValidatorOption func(*InputValidator)

// NewInputValidator constructs a workflow input validator.
func NewInputValidator(options ...InputValidatorOption) *InputValidator {
	validator := &InputValidator{
		logger:         slog.Default(),
		pluginProvider: defaultPluginSchemaProvider(),
	}
	for _, option := range options {
		if option != nil {
			option(validator)
		}
	}
	return validator
}

// WithInputValidatorLogger sets the validator logger.
func WithInputValidatorLogger(logger *slog.Logger) InputValidatorOption {
	return func(validator *InputValidator) {
		if logger != nil {
			validator.logger = logger
		}
	}
}

// WithInputValidatorPluginSchemaProvider sets the plugin schema provider.
func WithInputValidatorPluginSchemaProvider(
	provider PluginSchemaProvider,
) InputValidatorOption {
	return func(validator *InputValidator) {
		if provider != nil {
			validator.pluginProvider = provider
		}
	}
}

// Validate validates input payloads against the Arcaflow workflow input schema.
func (validator *InputValidator) Validate(
	ctx context.Context,
	workflow Workflow,
	inputPayload []byte,
) (ValidationResult, error) {
	if ctx.Err() != nil {
		return ValidationResult{}, ctx.Err()
	}
	if len(workflow.Content) == 0 {
		return ValidationResult{}, fmt.Errorf("workflow content is empty")
	}
	if len(inputPayload) == 0 {
		return ValidationResult{}, fmt.Errorf("input payload is empty")
	}

	scope, err := extractInputScope(ctx, workflow, validator.pluginProvider)
	if err != nil {
		return ValidationResult{}, err
	}

	inputData, err := parseAny(inputPayload)
	if err != nil {
		return ValidationResult{}, err
	}

	unserialized, err := scope.Unserialize(inputData)
	if err != nil {
		return ValidationResult{
			Valid:  false,
			Issues: normalizeConstraintIssues(err),
		}, nil
	}

	serialized, err := scope.Serialize(unserialized)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("serialize validated input: %w", err)
	}

	normalized, err := json.Marshal(serialized)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("normalize validated input: %w", err)
	}

	validator.logger.Debug(
		"validated workflow input",
		"workflow_id",
		workflow.ID,
	)

	return ValidationResult{
		Valid:           true,
		NormalizedInput: normalized,
	}, nil
}

func extractInputScope(
	ctx context.Context,
	workflow Workflow,
	pluginProvider PluginSchemaProvider,
) (*schema.ScopeSchema, error) {
	root, err := parseContent(workflow.Content)
	if err != nil {
		return nil, err
	}

	inputDefinition, ok := root["input"]
	if !ok {
		return nil, fmt.Errorf("workflow input section is missing")
	}

	normalized := normalizeScopeDefaults(inputDefinition)
	scope, err := schema.UnserializeScope(normalized)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow input section: %w", err)
	}

	scope.ApplySelf()
	baseDir := ""
	if strings.TrimSpace(workflow.LocalPath) != "" {
		baseDir = filepath.Dir(workflow.LocalPath)
	}
	if err := applyExternalNamespaces(
		ctx,
		scope,
		inputDefinition,
		root,
		baseDir,
		pluginProvider,
	); err != nil {
		return nil, fmt.Errorf("apply input namespaces: %w", err)
	}
	if err := scope.ValidateReferences(); err != nil {
		return nil, fmt.Errorf("invalid workflow input references: %w", err)
	}

	return scope, nil
}

func applyExternalNamespaces(
	ctx context.Context,
	scope *schema.ScopeSchema,
	inputDefinition interface{},
	root map[string]interface{},
	baseDir string,
	pluginProvider PluginSchemaProvider,
) error {
	namespaceRefs := map[string]map[string]struct{}{}
	collectNamespaceRefs(inputDefinition, namespaceRefs)
	if len(namespaceRefs) == 0 {
		return nil
	}
	steps, _ := root["steps"].(map[string]interface{})
	for namespace, refIDs := range namespaceRefs {
		stepName, path, ok := parseNamespace(namespace)
		if !ok {
			return fmt.Errorf("unsupported namespace %q", namespace)
		}
		step, _ := steps[stepName].(map[string]interface{})
		if step == nil {
			return fmt.Errorf("step %s not found", stepName)
		}
		if workflowPath, ok := step["workflow"].(string); ok && workflowPath != "" {
			if err := validateWorkflowNamespace(path); err != nil {
				return err
			}
			if baseDir == "" {
				return fmt.Errorf("workflow base directory missing for %q", namespace)
			}
			subPath := workflowPath
			if !filepath.IsAbs(subPath) {
				subPath = filepath.Join(baseDir, workflowPath)
			}
			content, err := os.ReadFile(subPath)
			if err != nil {
				return fmt.Errorf("read subworkflow %s: %w", subPath, err)
			}
			subScope, err := extractInputScope(ctx, Workflow{
				Content:   content,
				LocalPath: subPath,
			}, pluginProvider)
			if err != nil {
				return err
			}
			objects := map[string]*schema.ObjectSchema{}
			for id, obj := range subScope.Objects() {
				objects[id] = obj
			}
			for refID := range refIDs {
				if _, ok := objects[refID]; ok {
					continue
				}
				objects[refID] = schema.NewObjectSchema(
					refID,
					map[string]*schema.PropertySchema{},
				)
			}
			if err := applyNamespaceSafely(scope, objects, namespace); err != nil {
				return err
			}
			continue
		}
		if pluginSpec, ok := step["plugin"].(map[string]interface{}); ok {
			if pluginProvider == nil {
				return fmt.Errorf("plugin namespace %q missing schema provider", namespace)
			}
			if err := validatePluginNamespace(path); err != nil {
				return err
			}
			image, _ := pluginSpec["src"].(string)
			stepID, _ := step["step"].(string)
			if image == "" {
				return fmt.Errorf("plugin image missing for step %s", stepName)
			}
			rawSchema, err := pluginProvider.InputJSONSchema(ctx, image, stepID)
			if err != nil {
				return fmt.Errorf("load plugin schema for %s: %w", stepName, err)
			}
			objects := map[string]*schema.ObjectSchema{}
			for refID := range refIDs {
				objectSchema, err := objectSchemaFromJSON(rawSchema, refID)
				if err != nil {
					return fmt.Errorf("convert plugin schema for %s: %w", stepName, err)
				}
				objects[refID] = objectSchema
			}
			if err := applyNamespaceSafely(
				scope,
				objects,
				namespace,
			); err != nil {
				return err
			}
			continue
		}
		return fmt.Errorf("unsupported namespace %q", namespace)
	}
	return nil
}

func applyNamespaceSafely(
	scope *schema.ScopeSchema,
	objects map[string]*schema.ObjectSchema,
	namespace string,
) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("namespace %q: %v", namespace, recovered)
		}
	}()
	scope.ApplyNamespace(objects, namespace)
	return nil
}

func normalizeScopeDefaults(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		normalized := make(map[string]interface{}, len(typed))
		for key, child := range typed {
			switch key {
			case "default":
				normalized[key] = normalizeDefaultValue(child)
			case "examples":
				normalized[key] = normalizeExamples(child)
			default:
				normalized[key] = normalizeScopeDefaults(child)
			}
		}
		return normalized
	case []interface{}:
		normalized := make([]interface{}, 0, len(typed))
		for _, child := range typed {
			normalized = append(normalized, normalizeScopeDefaults(child))
		}
		return normalized
	default:
		return typed
	}
}

func normalizeDefaultValue(value interface{}) interface{} {
	if _, ok := value.(string); ok {
		return value
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return value
	}
	return string(raw)
}

func normalizeExamples(value interface{}) interface{} {
	list, ok := value.([]interface{})
	if !ok {
		return value
	}
	normalized := make([]interface{}, 0, len(list))
	for _, entry := range list {
		normalized = append(normalized, normalizeDefaultValue(entry))
	}
	return normalized
}

func collectNamespaceRefs(
	value interface{},
	namespaces map[string]map[string]struct{},
) {
	switch typed := value.(type) {
	case map[string]interface{}:
		rawNamespace, _ := typed["namespace"].(string)
		rawID, _ := typed["id"].(string)
		if rawNamespace != "" {
			entry := namespaces[rawNamespace]
			if entry == nil {
				entry = map[string]struct{}{}
				namespaces[rawNamespace] = entry
			}
			if rawID != "" {
				entry[rawID] = struct{}{}
			}
		}
		for _, child := range typed {
			collectNamespaceRefs(child, namespaces)
		}
	case []interface{}:
		for _, child := range typed {
			collectNamespaceRefs(child, namespaces)
		}
	}
}

func objectSchemaFromJSON(
	raw json.RawMessage,
	objectID string,
) (*schema.ObjectSchema, error) {
	var root map[string]interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parse json schema: %w", err)
	}
	propertiesRaw, _ := root["properties"].(map[string]interface{})
	required := requiredSet(root["required"])
	properties := make(map[string]*schema.PropertySchema, len(propertiesRaw))
	for name := range propertiesRaw {
		isRequired := required[name]
		properties[name] = schema.NewPropertySchema(
			schema.NewAnySchema(),
			nil,
			isRequired,
			nil,
			nil,
			nil,
			nil,
			nil,
		)
	}
	return schema.NewObjectSchema(objectID, properties), nil
}

func defaultPluginSchemaProvider() PluginSchemaProvider {
	if strings.EqualFold(os.Getenv("ARCAFLOW_MCP_PLUGIN_SCHEMA_MODE"), "stub") {
		return validatorStubPluginSchemaProvider{}
	}
	return NewContainerPluginSchemaProvider()
}

type validatorStubPluginSchemaProvider struct{}

func (validatorStubPluginSchemaProvider) InputJSONSchema(
	_ context.Context,
	_ string,
	_ string,
) (json.RawMessage, error) {
	return json.RawMessage(`{"type":"object","properties":{}}`), nil
}

func normalizeConstraintIssues(err error) []ValidationIssue {
	var constraint *schema.ConstraintError
	if errors.As(err, &constraint) {
		path := "$"
		if len(constraint.Path) > 0 {
			path = "$." + strings.Join(constraint.Path, ".")
		}
		return []ValidationIssue{{
			Path:       path,
			Message:    constraint.Message,
			Suggestion: suggestionForConstraint(constraint.Message),
		}}
	}

	return []ValidationIssue{{
		Path:    "$",
		Message: err.Error(),
	}}
}

func suggestionForConstraint(message string) string {
	lowered := strings.ToLower(message)
	switch {
	case strings.Contains(lowered, "required"):
		return "Add the required field to the input."
	case strings.Contains(lowered, "must be a string"):
		return "Provide a string value for this field."
	case strings.Contains(lowered, "must be an int"),
		strings.Contains(lowered, "must be an integer"):
		return "Provide an integer value for this field."
	case strings.Contains(lowered, "must be a float"),
		strings.Contains(lowered, "must be a number"):
		return "Provide a numeric value for this field."
	case strings.Contains(lowered, "must be a bool"),
		strings.Contains(lowered, "must be a boolean"):
		return "Provide a boolean value for this field."
	default:
		return "Check the input value against the workflow schema."
	}
}
