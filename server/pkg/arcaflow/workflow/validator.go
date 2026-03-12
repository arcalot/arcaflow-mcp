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

	"go.flow.arcalot.io/engine"
	"go.flow.arcalot.io/engine/config"
	"go.flow.arcalot.io/engine/loadfile"
	"go.flow.arcalot.io/pluginsdk/schema"
)

// PluginSchemaProvider fetches plugin input JSON schemas.
type PluginSchemaProvider interface {
	InputJSONSchema(ctx context.Context, image string, stepID string) (json.RawMessage, error)
}

// ValidationIssue describes a single validation failure.
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

// InputValidator validates workflow inputs against Arcaflow input schemas using the engine SDK.
type InputValidator struct {
	logger *slog.Logger
	config *config.Config
	engine engine.WorkflowEngine // Optional: injected engine
}

// InputValidatorOption configures the input validator.
type InputValidatorOption func(*InputValidator)

// NewInputValidator constructs a workflow input validator.
func NewInputValidator(options ...InputValidatorOption) *InputValidator {
	v := &InputValidator{
		logger: slog.Default(),
		config: config.Default(),
	}
	for _, option := range options {
		if option != nil {
			option(v)
		}
	}
	return v
}

// WithInputValidatorLogger sets the validator logger.
func WithInputValidatorLogger(logger *slog.Logger) InputValidatorOption {
	return func(v *InputValidator) {
		if logger != nil {
			v.logger = logger
		}
	}
}

// WithInputValidatorConfig sets the engine configuration.
func WithInputValidatorConfig(cfg *config.Config) InputValidatorOption {
	return func(v *InputValidator) {
		if cfg != nil {
			v.config = cfg
		}
	}
}

// WithInputValidatorEngine injects a pre-configured engine instance.
func WithInputValidatorEngine(eng engine.WorkflowEngine) InputValidatorOption {
	return func(v *InputValidator) {
		v.engine = eng
	}
}

// WithInputValidatorPluginSchemaProvider is a legacy option kept for compatibility.
func WithInputValidatorPluginSchemaProvider(_ PluginSchemaProvider) InputValidatorOption {
	return func(_ *InputValidator) {}
}

// Validate validates input payloads against the Arcaflow workflow using the full engine logic.
func (v *InputValidator) Validate(
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

	// 1. Initialize the engine if not injected
	eng := v.engine
	if eng == nil {
		var err error
		eng, err = engine.New(v.config)
		if err != nil {
			return ValidationResult{}, fmt.Errorf("failed to create engine: %w", err)
		}
	}

	// 2. Prepare the file cache
	workflowFileName := "workflow.yaml"
	if workflow.Path != "" {
		workflowFileName = filepath.Base(workflow.Path)
	} else if workflow.LocalPath != "" {
		workflowFileName = filepath.Base(workflow.LocalPath)
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

	// 3. Full parse & validation of the workflow
	wf, err := eng.Parse(fc, workflowFileName)
	if err != nil {
		return ValidationResult{
			Valid:  false,
			Issues: []ValidationIssue{{Path: "$", Message: err.Error()}},
		}, nil
	}

	// 4. Validate the input payload against the resolved schema
	inputData, err := parseAny(inputPayload)
	if err != nil {
		return ValidationResult{}, err
	}

	inputSchema := wf.InputSchema()
	unserialized, err := inputSchema.Unserialize(inputData)
	if err != nil {
		return ValidationResult{
			Valid:  false,
			Issues: normalizeConstraintIssues(err),
		}, nil
	}

	// 5. Success! Normalize the input
	serialized, err := inputSchema.Serialize(unserialized)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("serialize validated input: %w", err)
	}

	normalized, err := json.Marshal(serialized)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("normalize validated input: %w", err)
	}

	return ValidationResult{
		Valid:           true,
		NormalizedInput: normalized,
	}, nil
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
