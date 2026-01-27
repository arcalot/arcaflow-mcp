package workflowtools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/pluginschema"
	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

const pluginSchemaGetInputSchema = `{
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
    "step_id": {
      "type": "string",
      "description": "Optional workflow step ID to filter results."
    }
  },
  "required": ["source"],
  "additionalProperties": false
}`

// PluginSchemaGetParams defines the plugin_schema_get tool input.
type PluginSchemaGetParams struct {
	Source   ListSourceParams   `json:"source"`
	Selector LoadSelectorParams `json:"selector,omitempty"`
	StepID   string             `json:"step_id,omitempty"`
}

// PluginSchemaEntry captures a plugin schema reference and payload.
type PluginSchemaEntry struct {
	StepID   string          `json:"step_id"`
	Location string          `json:"location"`
	Schema   json.RawMessage `json:"schema"`
}

// PluginSchemaGetResult is the plugin_schema_get tool output payload.
type PluginSchemaGetResult struct {
	Workflow SchemaWorkflow      `json:"workflow"`
	Schemas  []PluginSchemaEntry `json:"schemas"`
}

// NewPluginSchemaGetTool registers the plugin_schema_get tool.
func NewPluginSchemaGetTool(
	loader *workflow.Loader,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name:        "plugin_schema_get",
			Description: "Get plugin schemas referenced by a workflow.",
			InputSchema: json.RawMessage(pluginSchemaGetInputSchema),
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
			var params PluginSchemaGetParams
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

			handler := pluginschema.NewHandler(logger)
			entries, err := handler.LoadFromWorkflow(
				ctx,
				selected.Content,
				selected.LocalPath,
			)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"plugin schema load failed",
					map[string]string{"error": err.Error()},
				)
			}

			var schemas []PluginSchemaEntry
			for _, entry := range entries {
				if params.StepID != "" && entry.StepID != params.StepID {
					continue
				}
				schemas = append(schemas, PluginSchemaEntry{
					StepID:   entry.StepID,
					Location: entry.Location,
					Schema:   entry.Schema,
				})
			}

			if params.StepID != "" && len(schemas) == 0 {
				available := []string{}
				for _, entry := range entries {
					available = append(available, entry.StepID)
				}
				details := map[string]string{
					"step_id": params.StepID,
				}
				if len(available) > 0 {
					details["available_steps"] = strings.Join(available, ", ")
				} else {
					details["hint"] = "No plugin schema references found in workflow."
				}
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"no plugin schemas found for step",
					details,
				)
			}

			result := PluginSchemaGetResult{
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
				Schemas: schemas,
			}

			return renderJSONResult(result, logger)
		},
	}
}
