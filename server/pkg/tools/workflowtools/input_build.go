package workflowtools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"github.com/arcalot/arcaflow-mcp/server/pkg/state"
)

const workflowInputBuildInputSchema = `{
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
      "description": "Optional session ID for iterative input building. Use the returned session_id immediately in the same conversation."
    },
    "input": {
      "type": "object",
      "description": "Partial or full workflow input payload to merge or replace."
    },
    "merge": {
      "type": "boolean",
      "description": "Merge input into existing draft when true. Defaults to true.",
      "default": true
    },
    "validate": {
      "type": "boolean",
      "description": "Validate the draft input against the workflow schema. Disable for partial inputs.",
      "default": true
    }
  },
  "required": ["source"],
  "additionalProperties": false
}`

// InputBuildParams defines the workflow_input_build tool input.
type InputBuildParams struct {
	Source    ListSourceParams       `json:"source"`
	Selector  LoadSelectorParams     `json:"selector,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Merge     *bool                  `json:"merge,omitempty"`
	Validate  *bool                  `json:"validate,omitempty"`
}

// InputBuildResult is the workflow_input_build tool output payload.
type InputBuildResult struct {
	SessionID  string          `json:"session_id"`
	Workflow   SchemaWorkflow  `json:"workflow"`
	Draft      json.RawMessage `json:"draft_input"`
	Validation InputValidation `json:"validation"`
}

// InputValidation captures validation results for draft inputs.
type InputValidation struct {
	Performed       bool                       `json:"performed"`
	Valid           bool                       `json:"valid"`
	NormalizedInput json.RawMessage            `json:"normalized_input,omitempty"`
	Issues          []workflow.ValidationIssue `json:"issues,omitempty"`
}

// NewWorkflowInputBuildTool registers the workflow_input_build tool.
func NewWorkflowInputBuildTool(
	loader *workflow.Loader,
	stateManager *state.Manager,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name: "workflow_input_build",
			Description: "Build or update draft inputs. For multi-step builds, pass " +
				"`input` each time; disable `validate` until required fields are set.",
			InputSchema: json.RawMessage(workflowInputBuildInputSchema),
		},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (protocol.ToolsCallResult, *protocol.ErrorObject) {
			if loader == nil || stateManager == nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"workflow input builder not configured",
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
			var params InputBuildParams
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

			sessionID := params.SessionID
			if sessionID == "" {
				sessionID, err = newSessionID()
				if err != nil {
					return protocol.ToolsCallResult{}, toolError(
						protocol.ErrInternal,
						"failed to create session",
						map[string]string{"error": err.Error()},
					)
				}
			}

			draft, err := loadDraft(ctx, stateManager, sessionID)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"failed to load draft input",
					map[string]string{"error": err.Error()},
				)
			}

			if params.Input != nil {
				if shouldMerge(params.Merge) && len(draft) > 0 {
					draft = mergeObjects(draft, params.Input)
				} else {
					draft = params.Input
				}
			}

			if len(draft) == 0 {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"input is required to build draft. "+
						"Hint: pass input each call; disable validate until complete.",
					map[string]string{
						"hint": "Pass input directly (even partial) and set validate=false " +
							"until required fields are present.",
					},
				)
			}

			draftRaw, err := json.Marshal(draft)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"failed to encode draft input",
					map[string]string{"error": err.Error()},
				)
			}

			validation := InputValidation{}
			if shouldValidate(params.Validate) {
				validation.Performed = true
				validator := workflow.NewInputValidator()
				result, err := validator.Validate(ctx, selected, draftRaw)
				if err != nil {
					return protocol.ToolsCallResult{}, toolError(
						protocol.ErrInvalidParams,
						"workflow input validation failed",
						map[string]string{"error": err.Error()},
					)
				}
				validation.Valid = result.Valid
				validation.NormalizedInput = result.NormalizedInput
				validation.Issues = result.Issues
				if result.Valid && len(result.NormalizedInput) > 0 {
					draftRaw = result.NormalizedInput
				}
			}

			if err := stateManager.Set(ctx, state.SessionData{
				SessionID:  sessionID,
				WorkflowID: selected.ID,
				DraftInput: draftRaw,
			}); err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"failed to store draft input",
					map[string]string{"error": err.Error()},
				)
			}

			result := InputBuildResult{
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
				Draft:      draftRaw,
				Validation: validation,
			}

			return renderJSONResult(result, logger)
		},
	}
}

func loadDraft(
	ctx context.Context,
	manager *state.Manager,
	sessionID string,
) (map[string]interface{}, error) {
	if sessionID == "" {
		return map[string]interface{}{}, nil
	}
	data, ok, err := manager.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !ok || len(data.DraftInput) == 0 {
		return map[string]interface{}{}, nil
	}
	var draft map[string]interface{}
	if err := json.Unmarshal(data.DraftInput, &draft); err != nil {
		return nil, fmt.Errorf("parse draft input: %w", err)
	}
	return draft, nil
}

func mergeObjects(
	base map[string]interface{},
	overlay map[string]interface{},
) map[string]interface{} {
	result := make(map[string]interface{}, len(base)+len(overlay))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range overlay {
		baseValue, ok := result[key]
		if !ok {
			result[key] = value
			continue
		}
		baseMap, okBase := baseValue.(map[string]interface{})
		overlayMap, okOverlay := value.(map[string]interface{})
		if okBase && okOverlay {
			result[key] = mergeObjects(baseMap, overlayMap)
			continue
		}
		result[key] = value
	}
	return result
}

func shouldMerge(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

func shouldValidate(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

func newSessionID() (string, error) {
	seed := make([]byte, 16)
	if _, err := rand.Read(seed); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(seed), nil
}
