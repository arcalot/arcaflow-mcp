package workflowtools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"github.com/arcalot/arcaflow-mcp/server/pkg/state"
)

const workflowInputValidateInputSchema = `{
  "type": "object",
  "properties": {
    "source": {
      "type": "object",
      "properties": {
        "kind": {
          "type": "string",
          "description": "Workflow source kind: filesystem, url, or git."
        },
        "location": {
          "type": "string",
          "description": "Filesystem root, URL, or git repository URL."
        },
        "ref": {
          "type": "string",
          "description": "Optional git ref (branch, tag, or commit)."
        },
        "subdir": {
          "type": "string",
          "description": "Optional git subdirectory to scan for workflows."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
    },
    "selector": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "description": "Workflow ID to load."
        },
        "path": {
          "type": "string",
          "description": "Workflow path to load."
        }
      },
      "additionalProperties": false
    },
    "session_id": {
      "type": "string",
      "description": "Advanced: session ID for a stored draft. Only use if you just ran workflow_input_build and received this session_id."
    },
    "input": {
      "type": "object",
      "description": "Recommended: pass the workflow input payload here for validation."
    }
  },
  "required": ["source"],
  "additionalProperties": false
}`

// InputValidateParams defines the workflow_input_validate tool input.
type InputValidateParams struct {
	Source    ListSourceParams       `json:"source"`
	Selector  LoadSelectorParams     `json:"selector,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
}

// InputValidateResult is the workflow_input_validate tool output payload.
type InputValidateResult struct {
	SessionID  string          `json:"session_id,omitempty"`
	Workflow   SchemaWorkflow  `json:"workflow"`
	Validation InputValidation `json:"validation"`
}

// NewWorkflowInputValidateTool registers the workflow_input_validate tool.
func NewWorkflowInputValidateTool(
	loader *workflow.Loader,
	stateManager *state.Manager,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name: "workflow_input_validate",
			Description: "Validate workflow inputs against Arcaflow v0.8+ schema engine. " +
				"USE THIS when user asks to validate, check, or verify inputs: " +
				"'Validate the inputs', 'Validate this input', 'Check if inputs are valid', " +
				"'Are these inputs correct?', 'Verify inputs', 'Is this input valid?'. " +
				"PROVIDES: Structured validation feedback with specific error messages, " +
				"field-level issues, and schema constraint violations that you can parse " +
				"and explain to the user. " +
				"RETURNS: Boolean valid status + detailed error array for debugging. " +
				"BETTER THAN shell/engine validation because: (1) Structured error output " +
				"the AI can parse and present clearly, (2) Same validation logic as " +
				"workflow_input_template for consistency, (3) No need to manage file paths " +
				"or engine configuration - pass input directly as JSON. " +
				"MANDATORY SAFETY NET: If you have already constructed inputs manually " +
				"(which you should not have done), you MUST validate them through this " +
				"tool before giving to the user. The user's workflow WILL FAIL if you " +
				"provide unvalidated inputs. Pass the input payload in the `input` " +
				"parameter for validation. Only use session_id if you just created a " +
				"draft with workflow_input_build in the same conversation. " +
				"NOTE: Prefer workflow_input_template to get validated structure that AI " +
				"populates, rather than construct and validate separately.",
			InputSchema: json.RawMessage(workflowInputValidateInputSchema),
		},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (protocol.ToolsCallResult, *protocol.ErrorObject) {
			if loader == nil || stateManager == nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"workflow input validator not configured",
					nil,
				)
			}
			if ctx.Err() != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"context cancelled",
					map[string]string{"error": ctx.Err().Error()},
				)
			}
			if _, ok := auth.TenantIDFromContext(ctx); !ok {
				ctx = auth.WithTenantID(ctx, "local")
			}

			payload, err := json.Marshal(arguments)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"invalid tool arguments",
					map[string]string{"error": err.Error()},
				)
			}
			var params InputValidateParams
			if err := json.Unmarshal(payload, &params); err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"invalid tool arguments",
					map[string]string{"error": err.Error()},
				)
			}
			if params.Source.Kind == "" || params.Source.Location == "" {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"source.kind and source.location are required",
					nil,
				)
			}

			details, err := loadDetails(ctx, loader, params.Source)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					fmt.Sprintf(
						"workflow source load failed: %s",
						err.Error(),
					),
					loadErrorData(err),
				)
			}

			selected, discovery, err := selectWorkflowWithDiscovery(
				details,
				params.Selector,
			)
			if discovery != nil && err == nil {
				return renderJSONResult(*discovery, logger)
			}
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					fmt.Sprintf("workflow selection failed: %s", err.Error()),
					selectionErrorData(err, discovery, details.Index.Workflows),
				)
			}

			rawInput, sessionID, err := resolveValidationInput(
				ctx,
				stateManager,
				params.SessionID,
				params.Input,
			)
			if err != nil {
				if resolutionErr, ok := err.(*inputResolutionError); ok {
					return protocol.ToolsCallResult{}, toolError(
						protocol.ErrInvalidParams,
						resolutionErr.Message,
						map[string]string{"hint": resolutionErr.Hint},
					)
				}
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"input resolution failed",
					map[string]string{"error": err.Error()},
				)
			}

			validator := workflow.NewInputValidator()
			result, err := validator.Validate(ctx, selected, rawInput)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"workflow input validation failed",
					map[string]string{"error": err.Error()},
				)
			}

			validation := InputValidation{
				Performed:       true,
				Valid:           result.Valid,
				NormalizedInput: result.NormalizedInput,
				Issues:          result.Issues,
			}

			response := InputValidateResult{
				SessionID: sessionID,
				Workflow: SchemaWorkflow{
					ID:   selected.ID,
					Name: selected.Name,
					Path: selected.Path,
					Source: ListSource{
						Kind:     string(selected.Source.Kind),
						Location: selected.Source.Location,
						Ref:      selected.Source.Ref,
						Subdir:   selected.Source.Subdir,
					},
				},
				Validation: validation,
			}

			return renderJSONResult(response, logger)
		},
	}
}

func resolveValidationInput(
	ctx context.Context,
	manager *state.Manager,
	sessionID string,
	payload map[string]interface{},
) ([]byte, string, error) {
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, sessionID, err
		}
		return raw, sessionID, nil
	}
	if sessionID == "" {
		return nil, sessionID, &inputResolutionError{
			Message: "input is required when session_id is empty. " +
				"Hint: pass input directly or provide session_id.",
			Hint: "Call workflow_input_validate with input, or build a session first.",
		}
	}
	data, ok, err := manager.Get(ctx, sessionID)
	if err != nil {
		return nil, sessionID, err
	}
	if !ok || len(data.DraftInput) == 0 {
		return nil, sessionID, &inputResolutionError{
			Message: "draft input not found for session_id. " +
				"Hint: call workflow_input_validate with input or run workflow_input_build.",
			Hint: "Use workflow_input_build to create a draft or pass input directly.",
		}
	}
	return data.DraftInput, sessionID, nil
}

type inputResolutionError struct {
	Message string
	Hint    string
}

func (err *inputResolutionError) Error() string {
	return err.Message
}
