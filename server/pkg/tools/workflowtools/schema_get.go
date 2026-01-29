package workflowtools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

const workflowSchemaGetInputSchema = `{
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
    }
  },
  "required": ["source"],
  "additionalProperties": false
}`

// SchemaGetParams defines the workflow_schema_get tool input.
type SchemaGetParams struct {
	Source   ListSourceParams   `json:"source"`
	Selector LoadSelectorParams `json:"selector,omitempty"`
}

// SchemaGetResult is the workflow_schema_get tool output payload.
type SchemaGetResult struct {
	Workflow         SchemaWorkflow    `json:"workflow"`
	InputJSONSchema  json.RawMessage   `json:"input_json_schema"`
	OutputJSONSchema json.RawMessage   `json:"output_json_schema,omitempty"`
	ExampleInput     json.RawMessage   `json:"example_input,omitempty"`
	SchemaKeys       SchemaKeyMetadata `json:"schema_keys"`
}

// SchemaWorkflow identifies the workflow metadata used for schema extraction.
type SchemaWorkflow struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Path   string     `json:"path"`
	Source ListSource `json:"source"`
}

// SchemaPathMetadata records where schemas were found in the workflow.
type SchemaKeyMetadata struct {
	InputKey  string `json:"input_key,omitempty"`
	OutputKey string `json:"output_key,omitempty"`
}

// NewWorkflowSchemaGetTool registers the workflow_schema_get tool.
func NewWorkflowSchemaGetTool(
	loader *workflow.Loader,
	parser *workflow.Parser,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name:        "workflow_schema_get",
			Description: "Get workflow input and output schemas.",
			InputSchema: json.RawMessage(workflowSchemaGetInputSchema),
		},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (protocol.ToolsCallResult, *protocol.ErrorObject) {
			if loader == nil || parser == nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"workflow schema parser not configured",
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
			payload, err := json.Marshal(arguments)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"invalid tool arguments",
					map[string]string{"error": err.Error()},
				)
			}
			var params SchemaGetParams
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

			parsed, err := parser.Parse(ctx, selected)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"workflow schema parse failed",
					map[string]string{"error": err.Error()},
				)
			}

			resolver := workflow.NewInputSchemaResolver()
			resolvedInput, err := resolver.ResolveInputJSONSchema(ctx, selected)
			if err != nil {
				details := map[string]string{"error": err.Error()}
				var resolutionErr workflow.NamespaceResolutionError
				if errors.As(err, &resolutionErr) && resolutionErr.Hint != "" {
					details["hint"] = resolutionErr.Hint
				}
				if strings.Contains(err.Error(), "no container runtime available") {
					details["hint"] = "Install podman or docker to resolve plugin schemas."
				}
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"workflow input schema resolution failed",
					details,
				)
			}
			result := SchemaGetResult{
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
				InputJSONSchema:  resolvedInput,
				OutputJSONSchema: parsed.OutputSchema,
				ExampleInput:     parsed.InputExample,
				SchemaKeys: SchemaKeyMetadata{
					InputKey:  parsed.InputSchemaPath,
					OutputKey: parsed.OutputSchemaPath,
				},
			}

			return renderJSONResult(result, logger)
		},
	}
}
