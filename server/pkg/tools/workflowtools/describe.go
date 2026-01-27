package workflowtools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

const workflowDescribeInputSchema = `{
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

// DescribeParams defines the workflow_describe tool input.
type DescribeParams struct {
	Source   ListSourceParams   `json:"source"`
	Selector LoadSelectorParams `json:"selector,omitempty"`
}

// DescribeResult is the workflow_describe tool output payload.
type DescribeResult struct {
	Workflow    SchemaWorkflow `json:"workflow"`
	Version     string         `json:"version,omitempty"`
	Description string         `json:"description,omitempty"`
	Steps       []StepSummary  `json:"steps,omitempty"`
	StepCount   int            `json:"step_count"`
}

// StepSummary captures a high-level view of a workflow step.
type StepSummary struct {
	ID              string `json:"id"`
	PluginName      string `json:"plugin_name,omitempty"`
	PluginImage     string `json:"plugin_image,omitempty"`
	PluginSchema    string `json:"plugin_schema,omitempty"`
	PluginSchemaRef string `json:"plugin_schema_ref,omitempty"`
}

// NewWorkflowDescribeTool registers the workflow_describe tool.
func NewWorkflowDescribeTool(
	loader *workflow.Loader,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name:        "workflow_describe",
			Description: "Get a human-readable description of a workflow.",
			InputSchema: json.RawMessage(workflowDescribeInputSchema),
		},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (protocol.ToolsCallResult, *protocol.ErrorObject) {
			if loader == nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"workflow loader not configured",
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
			var params DescribeParams
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

			index, err := loadIndex(ctx, loader, params.Source)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"workflow source load failed",
					map[string]string{"error": err.Error()},
				)
			}

			selected, err := selectWorkflow(index.Workflows, params.Selector)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					fmt.Sprintf("workflow selection failed: %s", err.Error()),
					map[string]string{"error": err.Error()},
				)
			}

			root, err := workflow.ParseDocument(selected.Content)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"workflow parse failed",
					map[string]string{"error": err.Error()},
				)
			}

			result := DescribeResult{
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
				Version:     stringValue(root, "version"),
				Description: descriptionValue(root),
			}

			steps := extractStepSummaries(root)
			result.Steps = steps
			result.StepCount = len(steps)

			return renderJSONResult(result, logger)
		},
	}
}

func descriptionValue(root map[string]interface{}) string {
	for _, key := range []string{"description", "summary", "title"} {
		if value := stringValue(root, key); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(root map[string]interface{}, key string) string {
	if root == nil {
		return ""
	}
	value, ok := root[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func extractStepSummaries(root map[string]interface{}) []StepSummary {
	stepsRaw, ok := root["steps"]
	if !ok {
		return nil
	}
	stepMap, ok := stepsRaw.(map[string]interface{})
	if !ok {
		return nil
	}
	ids := make([]string, 0, len(stepMap))
	for id := range stepMap {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	summaries := make([]StepSummary, 0, len(ids))
	for _, id := range ids {
		stepRaw := stepMap[id]
		step, ok := stepRaw.(map[string]interface{})
		if !ok {
			summaries = append(summaries, StepSummary{ID: id})
			continue
		}
		summary := StepSummary{
			ID:              id,
			PluginSchema:    stringField(step, "plugin_schema"),
			PluginSchemaRef: stringField(step, "plugin_schema_ref"),
		}
		if plugin, ok := step["plugin"].(map[string]interface{}); ok {
			summary.PluginName = stringField(plugin, "name")
			summary.PluginImage = stringField(plugin, "image")
			if summary.PluginSchema == "" {
				summary.PluginSchema = stringField(plugin, "schema")
			}
			if summary.PluginSchemaRef == "" {
				summary.PluginSchemaRef = stringField(plugin, "schema_ref")
			}
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func stringField(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	raw, ok := values[key]
	if !ok {
		return ""
	}
	text, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}
