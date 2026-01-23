package workflowtools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

const workflowLoadInputSchema = `{
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

// LoadParams defines the workflow_load tool input.
type LoadParams struct {
	Source   ListSourceParams   `json:"source"`
	Selector LoadSelectorParams `json:"selector,omitempty"`
}

// LoadSelectorParams narrows the workflow selection.
type LoadSelectorParams struct {
	ID   string `json:"id,omitempty"`
	Path string `json:"path,omitempty"`
}

// LoadResult is the workflow_load tool output payload.
type LoadResult struct {
	Workflow LoadedWorkflow `json:"workflow"`
}

// LoadedWorkflow captures a workflow and its content.
type LoadedWorkflow struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Path          string      `json:"path"`
	Source        ListSource  `json:"source"`
	Content       string      `json:"content"`
	ContentSHA256 string      `json:"content_sha256"`
	SizeBytes     int64       `json:"size_bytes,omitempty"`
	ModifiedAt    string      `json:"modified_at,omitempty"`
	Metadata      interface{} `json:"metadata,omitempty"`
}

// NewWorkflowLoadTool registers the workflow_load tool.
func NewWorkflowLoadTool(
	loader *workflow.Loader,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name:        "workflow_load",
			Description: "Load a workflow document from a source.",
			InputSchema: json.RawMessage(workflowLoadInputSchema),
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
			var params LoadParams
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
					"workflow selection failed",
					map[string]string{"error": err.Error()},
				)
			}

			result := LoadResult{
				Workflow: LoadedWorkflow{
					ID:   selected.ID,
					Name: selected.Name,
					Path: selected.Path,
					Source: ListSource{
						Kind:     string(selected.Source.Kind),
						Location: selected.Source.Location,
						Ref:      selected.Source.Ref,
						Subdir:   selected.Source.Subdir,
					},
					Content:       string(selected.Content),
					ContentSHA256: selected.ContentSHA256,
					SizeBytes:     selected.SizeBytes,
				},
			}
			if !selected.ModifiedAt.IsZero() {
				result.Workflow.ModifiedAt = selected.ModifiedAt.UTC().Format(
					"2006-01-02T15:04:05Z",
				)
			}

			return renderJSONResult(result, logger)
		},
	}
}

func selectWorkflow(
	workflows []workflow.Workflow,
	selector LoadSelectorParams,
) (workflow.Workflow, error) {
	if len(workflows) == 0 {
		return workflow.Workflow{}, fmt.Errorf("no workflows found")
	}
	selector = normalizeSelector(selector)
	if selector.ID == "" && selector.Path == "" {
		if len(workflows) == 1 {
			return workflows[0], nil
		}
		return workflow.Workflow{}, fmt.Errorf("workflow selector required")
	}
	if selector.ID != "" {
		for _, item := range workflows {
			if item.ID == selector.ID {
				return item, nil
			}
		}
		return workflow.Workflow{}, fmt.Errorf("workflow id not found")
	}
	for _, item := range workflows {
		if item.Path == selector.Path || item.Name == selector.Path {
			return item, nil
		}
	}
	return workflow.Workflow{}, fmt.Errorf("workflow path not found")
}

func normalizeSelector(selector LoadSelectorParams) LoadSelectorParams {
	selector.ID = strings.TrimSpace(selector.ID)
	selector.Path = strings.TrimSpace(selector.Path)
	return selector
}
