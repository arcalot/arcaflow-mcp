package workflowtools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/auth"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"github.com/arcalot/arcaflow-mcp/server/pkg/state"
)

const workflowInputExportInputSchema = `{
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
      "description": "Optional workflow input payload to export."
    },
    "format": {
      "type": "string",
      "description": "Export format: json or yaml.",
      "default": "json"
    }
  },
  "required": ["source"],
  "additionalProperties": false
}`

// InputExportParams defines the workflow_input_export tool input.
type InputExportParams struct {
	Source    ListSourceParams       `json:"source"`
	Selector  LoadSelectorParams     `json:"selector,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Format    string                 `json:"format,omitempty"`
}

// InputExportResult is the workflow_input_export tool output payload.
type InputExportResult struct {
	SessionID string                  `json:"session_id,omitempty"`
	Workflow  SchemaWorkflow          `json:"workflow"`
	Format    string                  `json:"format"`
	Payload   string                  `json:"payload"`
	Metadata  workflow.ExportMetadata `json:"metadata"`
}

// NewWorkflowInputExportTool registers the workflow_input_export tool.
func NewWorkflowInputExportTool(
	loader *workflow.Loader,
	parser *workflow.Parser,
	stateManager *state.Manager,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name: "workflow_input_export",
			Description: "Validate and export inputs. STRONGLY prefer passing `input` " +
				"directly; only use session_id if you just created a draft with " +
				"workflow_input_build in the same conversation.",
			InputSchema: json.RawMessage(workflowInputExportInputSchema),
		},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (protocol.ToolsCallResult, *protocol.ErrorObject) {
			if loader == nil || parser == nil || stateManager == nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"workflow input export not configured",
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
			var params InputExportParams
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
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"input resolution failed",
					map[string]string{"error": err.Error()},
				)
			}

			parsed, err := parser.Parse(ctx, selected)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"workflow schema parse failed",
					map[string]string{"error": err.Error()},
				)
			}

			format, err := normalizeExportFormat(params.Format)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"unsupported export format",
					map[string]string{"error": err.Error()},
				)
			}

			exported, err := workflow.GenerateInputFile(
				ctx,
				parsed,
				rawInput,
				format,
			)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"input export failed",
					map[string]string{"error": err.Error()},
				)
			}

			result := InputExportResult{
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
				Format:   string(exported.Format),
				Payload:  string(exported.Payload),
				Metadata: exported.Metadata,
			}

			return renderJSONResult(result, logger)
		},
	}
}

func normalizeExportFormat(value string) (workflow.ExportFormat, error) {
	if strings.TrimSpace(value) == "" {
		return workflow.ExportFormatJSON, nil
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(workflow.ExportFormatJSON):
		return workflow.ExportFormatJSON, nil
	case string(workflow.ExportFormatYAML):
		return workflow.ExportFormatYAML, nil
	default:
		return "", fmt.Errorf("unknown format %q", value)
	}
}
