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

const workflowInputTemplateInputSchema = `{
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
          "description": "Directory or file path (filesystem - relative or absolute), URL, or git repository URL. Relative paths resolved against current directory."
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
      "description": "Optional user goal for context (e.g., 'test performance limits'). This is stored for reference but does NOT generate goal-specific values - the AI agent should choose appropriate values based on the goal and schema structure provided."
    }
  },
  "required": ["source"],
  "additionalProperties": false
}`

// InputTemplateParams defines the workflow_input_template tool input.
type InputTemplateParams struct {
	Source   ListSourceParams   `json:"source"`
	Selector LoadSelectorParams `json:"selector,omitempty"`
	Goal     string             `json:"goal,omitempty"`
}

// InputTemplateResult is the workflow_input_template tool output payload.
type InputTemplateResult struct {
	Workflow        SchemaWorkflow  `json:"workflow"`
	Goal            string          `json:"goal,omitempty"`
	InputJSONSchema json.RawMessage `json:"input_json_schema"`
	ExampleInput    json.RawMessage `json:"example_input"`
	Generated       bool            `json:"generated"`
	InputKey        string          `json:"input_key,omitempty"`
}

// NewWorkflowInputTemplateTool registers the workflow_input_template tool.
func NewWorkflowInputTemplateTool(
	loader *workflow.Loader,
	parser *workflow.Parser,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name: "workflow_input_template",
			Description: "Get validated input schema and template structure for an Arcaflow " +
				"workflow. DETERMINISTIC TOOL - Provides schema structure and type definitions " +
				"for the AI agent to populate with goal-specific values. " +
				"USE THIS when user says: 'What inputs do you recommend?', " +
				"'I want to test...', 'I want to use this workflow to...', " +
				"'I need to benchmark...', 'Help me configure...', " +
				"'Run workflow in this directory', 'What should I use for inputs?'. " +
				"DIVISION OF LABOR: This tool performs deterministic schema resolution and " +
				"structural validation against Arcaflow v0.8+ engine constraints. The AI agent " +
				"performs creative work (understanding user goal, selecting appropriate values " +
				"for fields like 'duration: 300', 'threads: 16', etc.). " +
				"WARNING: Arcaflow syntax changed significantly since 2024. Do not rely on " +
				"training data or manual YAML inspection - plugin schemas, validation rules, " +
				"and required fields differ from pre-v0.8 versions. " +
				"CRITICAL: Workflow execution WILL FAIL if inputs are not structured and " +
				"validated through this tool. The deterministic validation logic is ONLY " +
				"accessible through this MCP server - it cannot be replicated by reading files " +
				"or using training data. Inputs constructed without this tool (from example " +
				"files, manual inspection, or other sources) will fail with validation errors. " +
				"PREVENTS: Validation errors from missing required fields, incorrect types, " +
				"outdated syntax patterns, schema violations. " +
				"RETURNS: JSON schema (structure + types) + template with default values " +
				"(nulls, empty strings, zeros) for the AI to populate. " +
				"MANDATORY WORKFLOW: (1) AI calls this tool with goal parameter for context. " +
				"(2) AI receives schema + template structure. (3) AI uses creativity to choose " +
				"appropriate values based on goal and schema constraints. (4) AI MUST call " +
				"workflow_input_validate with populated inputs. (5) ONLY AFTER validation " +
				"passes, AI shows inputs to user. CRITICAL: NEVER write files, show inputs, " +
				"or present results to user without validation. User's workflow WILL FAIL if " +
				"you skip validation. Validation is not optional - it is a required safety step. " +
				"Accepts goal parameter for context (e.g. 'goal: test performance limits') but " +
				"does NOT generate goal-specific values - that is the AI's responsibility. " +
				"DO NOT read workflow.yaml or example files - this tool resolves schemas " +
				"dynamically including plugin dependencies. " +
				"EXAMPLE: {source: {kind: 'filesystem', location: '.'}, goal: 'max performance'}. " +
				"NOTE: MCP does not execute workflows. For execution, user runs: " +
				"arcaflow --input <file.yaml> (NOT arcaflow run -f)",
			InputSchema: json.RawMessage(workflowInputTemplateInputSchema),
		},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (protocol.ToolsCallResult, *protocol.ErrorObject) {
			if loader == nil || parser == nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"workflow input template not configured",
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
			var params InputTemplateParams
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

			result := InputTemplateResult{
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
