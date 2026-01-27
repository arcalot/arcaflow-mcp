package workflowtools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
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

			details, err := loadDetails(ctx, loader, params.Source)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"workflow source load failed",
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
		if selected, ok := autoSelectPrimaryWorkflow(workflows); ok {
			return selected, nil
		}
		return workflow.Workflow{}, fmt.Errorf(
			"multiple workflows found; provide selector.id or selector.path. "+
				"Available: %s",
			availableWorkflowPaths(workflows),
		)
	}
	if selector.ID != "" {
		for _, item := range workflows {
			if item.ID == selector.ID {
				return item, nil
			}
		}
		return workflow.Workflow{}, fmt.Errorf(
			"workflow id %q not found. Available: %s",
			selector.ID,
			availableWorkflowIDs(workflows),
		)
	}
	for _, item := range workflows {
		if item.Path == selector.Path || item.Name == selector.Path {
			return item, nil
		}
	}
	return workflow.Workflow{}, fmt.Errorf(
		"workflow path %q not found. Available: %s",
		selector.Path,
		availableWorkflowPaths(workflows),
	)
}

func normalizeSelector(selector LoadSelectorParams) LoadSelectorParams {
	selector.ID = strings.TrimSpace(selector.ID)
	selector.Path = strings.TrimSpace(selector.Path)
	return selector
}

func availableWorkflowPaths(workflows []workflow.Workflow) string {
	paths := availableWorkflowPathsList(workflows)
	if len(paths) == 0 {
		return "none"
	}
	return strings.Join(paths, ", ")
}

func availableWorkflowIDs(workflows []workflow.Workflow) string {
	ids := availableWorkflowIDsList(workflows)
	if len(ids) == 0 {
		return "none"
	}
	return strings.Join(ids, ", ")
}

func availableWorkflowPathsList(workflows []workflow.Workflow) []string {
	paths := make([]string, 0, len(workflows))
	for _, item := range workflows {
		if item.Path != "" {
			paths = append(paths, item.Path)
		} else if item.Name != "" {
			paths = append(paths, item.Name)
		}
	}
	sort.Strings(paths)
	return paths
}

func availableWorkflowIDsList(workflows []workflow.Workflow) []string {
	ids := make([]string, 0, len(workflows))
	for _, item := range workflows {
		if item.ID != "" {
			ids = append(ids, item.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func autoSelectPrimaryWorkflow(
	workflows []workflow.Workflow,
) (workflow.Workflow, bool) {
	if selected, ok := selectWorkflowByFilename(
		workflows,
		map[string]struct{}{"workflow.yaml": {}, "workflow.yml": {}},
	); ok {
		return selected, true
	}
	if selected, ok := selectParentWorkflow(workflows); ok {
		return selected, true
	}
	return workflow.Workflow{}, false
}

func selectWorkflowByFilename(
	workflows []workflow.Workflow,
	names map[string]struct{},
) (workflow.Workflow, bool) {
	var matches []workflow.Workflow
	for _, item := range workflows {
		if item.Path == "" {
			continue
		}
		base := strings.ToLower(filepath.Base(item.Path))
		if _, ok := names[base]; ok {
			matches = append(matches, item)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	if len(matches) > 1 {
		selected, ok := selectShallowestWorkflow(matches)
		if ok {
			return selected, true
		}
	}
	return workflow.Workflow{}, false
}

func selectParentWorkflow(
	workflows []workflow.Workflow,
) (workflow.Workflow, bool) {
	referenced := map[string]struct{}{}
	for _, item := range workflows {
		root, err := workflow.ParseDocument(item.Content)
		if err != nil {
			continue
		}
		steps, _ := root["steps"].(map[string]interface{})
		for _, step := range steps {
			stepMap, ok := step.(map[string]interface{})
			if !ok {
				continue
			}
			ref, ok := stepMap["workflow"].(string)
			if !ok || strings.TrimSpace(ref) == "" {
				continue
			}
			referenced[filepath.Clean(ref)] = struct{}{}
		}
	}
	var candidates []workflow.Workflow
	for _, item := range workflows {
		if item.Path == "" {
			continue
		}
		if _, ok := referenced[filepath.Clean(item.Path)]; ok {
			continue
		}
		candidates = append(candidates, item)
	}
	if len(candidates) == 1 {
		return candidates[0], true
	}
	return workflow.Workflow{}, false
}

func selectShallowestWorkflow(
	workflows []workflow.Workflow,
) (workflow.Workflow, bool) {
	var (
		selected workflow.Workflow
		depth    = -1
		tie      = false
	)
	for _, item := range workflows {
		currentDepth := pathDepth(item.Path)
		if depth == -1 || currentDepth < depth {
			selected = item
			depth = currentDepth
			tie = false
			continue
		}
		if currentDepth == depth {
			tie = true
		}
	}
	if tie {
		return workflow.Workflow{}, false
	}
	return selected, depth >= 0
}

func pathDepth(path string) int {
	if path == "" {
		return 0
	}
	cleaned := filepath.ToSlash(filepath.Clean(path))
	return strings.Count(cleaned, "/")
}
