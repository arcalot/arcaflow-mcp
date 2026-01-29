package workflowtools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

const workflowDiscoverInputSchema = `{
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

const gitLoadTimeout = 60 * time.Second

// DiscoverParams defines the workflow_discover tool input.
type DiscoverParams struct {
	Source ListSourceParams `json:"source"`
}

// DiscoveryResult captures workflow discovery output and selection guidance.
type DiscoveryResult struct {
	Source    ListSource        `json:"source"`
	Workflows []Summary         `json:"workflows"`
	Selection SelectionGuidance `json:"selection"`
	Cache     CacheInfo         `json:"cache"`
	Timing    TimingInfo        `json:"timing"`
	Progress  []ProgressInfo    `json:"progress,omitempty"`
}

// SelectionGuidance describes how to select a workflow from discovery output.
type SelectionGuidance struct {
	Required          bool                `json:"required"`
	Reason            string              `json:"reason,omitempty"`
	SuggestedSelector *LoadSelectorParams `json:"suggested_selector,omitempty"`
	AvailablePaths    []string            `json:"available_paths,omitempty"`
	AvailableIDs      []string            `json:"available_ids,omitempty"`
}

// CacheInfo reports cache details for a discovery run.
type CacheInfo struct {
	Status    string       `json:"status"`
	Snapshot  SnapshotInfo `json:"snapshot,omitempty"`
	FetchedAt string       `json:"fetched_at,omitempty"`
}

// SnapshotInfo reports source snapshot metadata.
type SnapshotInfo struct {
	FilesystemFingerprint string `json:"filesystem_fingerprint,omitempty"`
	ETag                  string `json:"etag,omitempty"`
	LastModified          string `json:"last_modified,omitempty"`
	Commit                string `json:"commit,omitempty"`
}

// TimingInfo reports discovery timings in milliseconds.
type TimingInfo struct {
	TotalMs    int64 `json:"total_ms"`
	FetchMs    int64 `json:"fetch_ms,omitempty"`
	CheckoutMs int64 `json:"checkout_ms,omitempty"`
	ScanMs     int64 `json:"scan_ms,omitempty"`
}

// ProgressInfo reports a progress milestone from discovery.
type ProgressInfo struct {
	Stage       string `json:"stage"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
	DurationMs  int64  `json:"duration_ms,omitempty"`
	Error       string `json:"error,omitempty"`
}

// NewWorkflowDiscoverTool registers the workflow_discover tool.
func NewWorkflowDiscoverTool(
	loader *workflow.Loader,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name:        "workflow_discover",
			Description: "Discover workflows and selection guidance from a source.",
			InputSchema: json.RawMessage(workflowDiscoverInputSchema),
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
			var params DiscoverParams
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

			result := buildDiscoveryResult(details)
			result.Selection = selectionGuidanceForWorkflows(details.Index.Workflows)
			return renderJSONResult(result, logger)
		},
	}
}

type loadTimeoutError struct {
	timeout time.Duration
	err     error
}

func (loadTimeoutError) Error() string {
	return "workflow discovery timed out"
}

func (err loadTimeoutError) Unwrap() error {
	return err.err
}

func loadDetails(
	ctx context.Context,
	loader *workflow.Loader,
	source ListSourceParams,
) (workflow.LoadDetails, error) {
	kind := strings.ToLower(strings.TrimSpace(source.Kind))
	switch kind {
	case string(workflow.SourceFilesystem):
		return loader.LoadWithDetails(ctx, workflow.SourceMetadata{
			Kind:     workflow.SourceFilesystem,
			Location: source.Location,
		})
	case string(workflow.SourceURL):
		return loader.LoadWithDetails(ctx, workflow.SourceMetadata{
			Kind:     workflow.SourceURL,
			Location: source.Location,
		})
	case string(workflow.SourceGit):
		metadata := workflow.SourceMetadata{
			Kind:     workflow.SourceGit,
			Location: source.Location,
			Ref:      source.Ref,
			Subdir:   source.Subdir,
		}
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			timeoutCtx, cancel := context.WithTimeout(ctx, gitLoadTimeout)
			defer cancel()
			details, err := loader.LoadWithDetails(timeoutCtx, metadata)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					return workflow.LoadDetails{}, loadTimeoutError{
						timeout: gitLoadTimeout,
						err:     err,
					}
				}
				return workflow.LoadDetails{}, err
			}
			return details, nil
		}
		return loader.LoadWithDetails(ctx, metadata)
	default:
		return workflow.LoadDetails{}, fmt.Errorf(
			"unsupported source kind %q",
			source.Kind,
		)
	}
}

func loadErrorData(err error) map[string]interface{} {
	data := map[string]interface{}{
		"error": err.Error(),
	}
	var timeoutErr loadTimeoutError
	if errors.As(err, &timeoutErr) {
		data["timeout_seconds"] = int(timeoutErr.timeout.Seconds())
		data["retry_guidance"] = fmt.Sprintf(
			"Retry with a smaller repository or a shorter subdir scope, "+
				"or rerun with a longer timeout (>%ds).",
			int(timeoutErr.timeout.Seconds()),
		)
	}
	return data
}

func buildDiscoveryResult(details workflow.LoadDetails) DiscoveryResult {
	source := details.Index.Source
	result := DiscoveryResult{
		Source: ListSource{
			Kind:     string(source.Kind),
			Location: source.Location,
			Ref:      source.Ref,
			Subdir:   source.Subdir,
		},
		Workflows: summarizeWorkflows(details.Index.Workflows),
		Cache:     buildCacheInfo(details),
		Timing:    buildTimingInfo(details.Timing),
	}
	progress := buildProgressInfo(details.Progress)
	if len(progress) > 0 {
		result.Progress = progress
	}
	return result
}

func buildCacheInfo(details workflow.LoadDetails) CacheInfo {
	info := CacheInfo{
		Status: string(details.CacheStatus),
	}
	snapshot := SnapshotInfo{
		FilesystemFingerprint: details.Index.Snapshot.FilesystemFingerprint,
		ETag:                  details.Index.Snapshot.ETag,
		Commit:                details.Index.Snapshot.Commit,
	}
	if !details.Index.Snapshot.LastModified.IsZero() {
		snapshot.LastModified = formatTimestamp(details.Index.Snapshot.LastModified)
	}
	if snapshot != (SnapshotInfo{}) {
		info.Snapshot = snapshot
	}
	if !details.Index.FetchedAt.IsZero() {
		info.FetchedAt = formatTimestamp(details.Index.FetchedAt)
	}
	return info
}

func buildTimingInfo(timing workflow.LoadTiming) TimingInfo {
	info := TimingInfo{
		TotalMs: timing.Total.Milliseconds(),
	}
	if timing.Fetch != 0 {
		info.FetchMs = timing.Fetch.Milliseconds()
	}
	if timing.Checkout != 0 {
		info.CheckoutMs = timing.Checkout.Milliseconds()
	}
	if timing.Scan != 0 {
		info.ScanMs = timing.Scan.Milliseconds()
	}
	return info
}

func buildProgressInfo(progress []workflow.ProgressEvent) []ProgressInfo {
	if len(progress) == 0 {
		return nil
	}
	output := make([]ProgressInfo, 0, len(progress))
	for _, item := range progress {
		info := ProgressInfo{
			Stage:      string(item.Stage),
			DurationMs: item.Duration.Milliseconds(),
		}
		if !item.StartedAt.IsZero() {
			info.StartedAt = formatTimestamp(item.StartedAt)
		}
		if !item.CompletedAt.IsZero() {
			info.CompletedAt = formatTimestamp(item.CompletedAt)
		}
		if item.Error != "" {
			info.Error = item.Error
		}
		output = append(output, info)
	}
	return output
}

func selectionGuidanceForWorkflows(
	workflows []workflow.Workflow,
) SelectionGuidance {
	guidance := SelectionGuidance{
		AvailablePaths: availableWorkflowPathsList(workflows),
		AvailableIDs:   availableWorkflowIDsList(workflows),
	}
	if len(workflows) == 0 {
		guidance.Required = true
		guidance.Reason = "no workflows found in source"
		return guidance
	}
	if len(workflows) == 1 {
		guidance.Required = false
		guidance.SuggestedSelector = suggestedSelectorForWorkflow(workflows[0])
		return guidance
	}
	guidance.Required = true
	guidance.Reason = "multiple workflows found; provide selector.id or selector.path"
	guidance.SuggestedSelector = suggestedSelectorForWorkflows(workflows)
	return guidance
}

func suggestedSelectorForWorkflows(
	workflows []workflow.Workflow,
) *LoadSelectorParams {
	if len(workflows) == 1 {
		return suggestedSelectorForWorkflow(workflows[0])
	}
	if selected, ok := autoSelectPrimaryWorkflow(workflows); ok {
		return suggestedSelectorForWorkflow(selected)
	}
	if len(workflows) == 0 {
		return nil
	}
	return suggestedSelectorForWorkflow(workflows[0])
}

func suggestedSelectorForWorkflow(item workflow.Workflow) *LoadSelectorParams {
	if item.Path != "" {
		return &LoadSelectorParams{Path: item.Path}
	}
	if item.ID != "" {
		return &LoadSelectorParams{ID: item.ID}
	}
	return nil
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2006-01-02T15:04:05Z")
}
