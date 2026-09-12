package workflowtools

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/arcalot/arcaflow-mcp/server/pkg/analysis"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

const workflowHistoryLoadInputSchema = `{
  "type": "object",
  "properties": {
    "workflow_id": {
      "type": "string",
      "description": "Optional workflow ID to filter history."
    },
    "run_id": {
      "type": "string",
      "description": "Optional run ID to fetch a specific record."
    }
  },
  "additionalProperties": false
}`

// HistoryLoadParams defines workflow_history_load input.
type HistoryLoadParams struct {
	WorkflowID string `json:"workflow_id,omitempty"`
	RunID      string `json:"run_id,omitempty"`
}

// HistoryLoadResult returns history summaries or a single record.
type HistoryLoadResult struct {
	Runs []analysis.HistoryRunSummary `json:"runs,omitempty"`
	Run  *analysis.HistoryRunRecord   `json:"run,omitempty"`
}

// NewWorkflowHistoryLoadTool registers workflow_history_load.
func NewWorkflowHistoryLoadTool(
	analysisClient *analysis.Client,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name: "workflow_history_load",
			Description: "Load historical analysis runs. USE THIS when user says: " +
				"'Show me previous results', 'Load analysis history', 'Past runs'. " +
				"Returns stored analysis metadata and summaries.",
			InputSchema: json.RawMessage(workflowHistoryLoadInputSchema),
		},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (protocol.ToolsCallResult, *protocol.ErrorObject) {
			if analysisClient == nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"analysis service not configured",
					map[string]string{
						"hint": "set analysis.analysis_http_url in configuration",
					},
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
			var params HistoryLoadParams
			if err := json.Unmarshal(payload, &params); err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"invalid tool arguments",
					map[string]string{"error": err.Error()},
				)
			}
			result := HistoryLoadResult{}
			if params.RunID != "" {
				run, err := analysisClient.HistoryGet(ctx, params.RunID)
				if err != nil {
					return protocol.ToolsCallResult{}, toolError(
						protocol.ErrInternal,
						"history fetch failed",
						map[string]string{"error": err.Error()},
					)
				}
				result.Run = &run
				return renderJSONResult(result, logger)
			}
			runs, err := analysisClient.HistoryList(ctx, params.WorkflowID)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"history list failed",
					map[string]string{"error": err.Error()},
				)
			}
			result.Runs = runs
			return renderJSONResult(result, logger)
		},
	}
}
