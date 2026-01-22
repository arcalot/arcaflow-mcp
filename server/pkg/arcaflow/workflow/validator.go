package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	logger *slog.Logger
}

// InputValidatorOption configures the input validator.
type InputValidatorOption func(*InputValidator)

// NewInputValidator constructs a workflow input validator.
func NewInputValidator(options ...InputValidatorOption) *InputValidator {
	validator := &InputValidator{
		logger: slog.Default(),
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

	scope, err := extractInputScope(workflow.Content)
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

func extractInputScope(workflowContent []byte) (*schema.ScopeSchema, error) {
	root, err := parseContent(workflowContent)
	if err != nil {
		return nil, err
	}

	inputDefinition, ok := root["input"]
	if !ok {
		return nil, fmt.Errorf("workflow input section is missing")
	}

	scope, err := schema.UnserializeScope(inputDefinition)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow input section: %w", err)
	}

	scope.ApplySelf()
	if err := scope.ValidateReferences(); err != nil {
		return nil, fmt.Errorf("invalid workflow input references: %w", err)
	}

	return scope, nil
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
