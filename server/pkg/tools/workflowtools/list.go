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

const workflowListInputSchema = `{
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
    }
  },
  "required": ["source"],
  "additionalProperties": false
}`

// ListParams defines the workflow_list tool input.
type ListParams struct {
	Source ListSourceParams `json:"source"`
}

// ListSourceParams defines the workflow source selector.
type ListSourceParams struct {
	Kind     string `json:"kind"`
	Location string `json:"location"`
	Ref      string `json:"ref,omitempty"`
	Subdir   string `json:"subdir,omitempty"`
}

// ListResult is the workflow_list tool output payload.
type ListResult struct {
	Source    ListSource `json:"source"`
	Workflows []Summary  `json:"workflows"`
}

// ListSource describes the resolved source metadata.
type ListSource struct {
	Kind     string `json:"kind"`
	Location string `json:"location"`
	Ref      string `json:"ref,omitempty"`
	Subdir   string `json:"subdir,omitempty"`
}

// Summary captures metadata about a discovered workflow.
type Summary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	ContentSHA256 string `json:"content_sha256"`
	SizeBytes     int64  `json:"size_bytes,omitempty"`
	ModifiedAt    string `json:"modified_at,omitempty"`
}

// NewWorkflowListTool registers the workflow_list tool.
func NewWorkflowListTool(
	loader *workflow.Loader,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name:        "workflow_list",
			Description: "List workflows available from a source.",
			InputSchema: json.RawMessage(workflowListInputSchema),
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
			var params ListParams
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

			result := ListResult{
				Source: ListSource{
					Kind:     string(index.Source.Kind),
					Location: index.Source.Location,
					Ref:      index.Source.Ref,
					Subdir:   index.Source.Subdir,
				},
				Workflows: summarizeWorkflows(index.Workflows),
			}

			return renderJSONResult(result, logger)
		},
	}
}

func loadIndex(
	ctx context.Context,
	loader *workflow.Loader,
	source ListSourceParams,
) (workflow.WorkflowIndex, error) {
	kind := strings.ToLower(strings.TrimSpace(source.Kind))
	switch kind {
	case string(workflow.SourceFilesystem):
		return loader.LoadFromFilesystem(ctx, source.Location)
	case string(workflow.SourceURL):
		return loader.LoadFromURL(ctx, source.Location)
	case string(workflow.SourceGit):
		return loader.LoadFromGit(ctx, source.Location, source.Ref, source.Subdir)
	default:
		return workflow.WorkflowIndex{}, fmt.Errorf("unsupported source kind %q", source.Kind)
	}
}

func summarizeWorkflows(workflows []workflow.Workflow) []Summary {
	summaries := make([]Summary, 0, len(workflows))
	for _, item := range workflows {
		summary := Summary{
			ID:            item.ID,
			Name:          item.Name,
			Path:          item.Path,
			ContentSHA256: item.ContentSHA256,
			SizeBytes:     item.SizeBytes,
		}
		if !item.ModifiedAt.IsZero() {
			summary.ModifiedAt = item.ModifiedAt.UTC().Format(
				"2006-01-02T15:04:05Z",
			)
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func renderJSONResult(
	result interface{},
	logger *slog.Logger,
) (protocol.ToolsCallResult, *protocol.ErrorObject) {
	payload, err := json.Marshal(result)
	if err != nil {
		if logger != nil {
			logger.Error("marshal tool result", "error", err)
		}
		return protocol.ToolsCallResult{}, toolError(
			protocol.ErrInternal,
			"failed to encode tool response",
			map[string]string{"error": err.Error()},
		)
	}
	return protocol.ToolsCallResult{
		Content: []protocol.ToolContent{
			{
				Type: "text",
				Text: string(payload),
			},
		},
	}, nil
}

func toolError(
	code int,
	message string,
	data interface{},
) *protocol.ErrorObject {
	return &protocol.ErrorObject{
		Code:    code,
		Message: message,
		Data:    data,
	}
}
