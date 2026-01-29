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

const workflowInputRecommendInputSchema = `{
  "type": "object",
  "properties": {
    "source": {
      "type": "object",
      "description": "Workflow source. Use when user provides directory or file path instead of reading workflow files directly.",
      "properties": {
        "kind": {
          "type": "string",
          "description": "Workflow source kind: filesystem (for local paths), url, or git."
        },
        "location": {
          "type": "string",
          "description": "Directory or file path (filesystem), URL, or git repository URL."
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
    "goal": {
      "type": "string",
      "description": "Optional goal for the recommendation (e.g. max performance)."
    }
  },
  "required": ["source"],
  "additionalProperties": false
}`

// InputRecommendParams defines the workflow_input_recommend tool input.
type InputRecommendParams struct {
	Source   ListSourceParams   `json:"source"`
	Selector LoadSelectorParams `json:"selector,omitempty"`
	Goal     string             `json:"goal,omitempty"`
}

// InputRecommendResult is the workflow_input_recommend tool output payload.
type InputRecommendResult struct {
	Workflow        SchemaWorkflow  `json:"workflow"`
	Goal            string          `json:"goal,omitempty"`
	InputJSONSchema json.RawMessage `json:"input_json_schema"`
	ExampleInput    json.RawMessage `json:"example_input"`
	Generated       bool            `json:"generated"`
	InputKey        string          `json:"input_key,omitempty"`
}

// NewWorkflowInputRecommendTool registers the workflow_input_recommend tool.
func NewWorkflowInputRecommendTool(
	loader *workflow.Loader,
	parser *workflow.Parser,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name: "workflow_input_recommend",
			Description: "Recommend workflow inputs validated against the workflow schema. " +
				"USE THIS when user says: 'What inputs do you recommend?', " +
				"'Run workflow in this directory', 'What should I use for inputs?'. " +
				"PREVENTS: Validation errors from missing required fields or incorrect types. " +
				"Returns schema + example inputs guaranteed to pass validation. " +
				"Accepts source.kind=filesystem for local workflows. " +
				"DO NOT read workflow.yaml or example files - this tool uses the schema internally. " +
				"EXAMPLE: {source: {kind: 'filesystem', location: '.'}, goal: 'max performance'}. " +
				"NOTE: MCP does not execute workflows. For execution, user runs: " +
				"arcaflow --input <file.yaml> (NOT arcaflow run -f)",
			InputSchema: json.RawMessage(workflowInputRecommendInputSchema),
		},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (protocol.ToolsCallResult, *protocol.ErrorObject) {
			if loader == nil || parser == nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"workflow input recommendations not configured",
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
			var params InputRecommendParams
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

			example := parsed.InputExample
			generated := false
			if len(example) == 0 {
				example, err = workflow.GenerateExampleInput(resolvedInput)
				if err != nil {
					return protocol.ToolsCallResult{}, toolError(
						protocol.ErrInvalidParams,
						"example input generation failed",
						map[string]string{"error": err.Error()},
					)
				}
				generated = true
			}

			result := InputRecommendResult{
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
				Goal:            strings.TrimSpace(params.Goal),
				InputJSONSchema: resolvedInput,
				ExampleInput:    example,
				Generated:       generated,
				InputKey:        parsed.InputSchemaPath,
			}

			return renderJSONResult(result, logger)
		},
	}
}
