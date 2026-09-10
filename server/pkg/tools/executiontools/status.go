package executiontools

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

const workflowStatusInputSchema = `{
  "type": "object",
  "properties": {
    "execution_id": {
      "type": "string",
      "description": "Execution ID from workflow_execute."
    }
  },
  "required": ["execution_id"],
  "additionalProperties": false
}`

// StatusResult is the workflow_execution_status output.
type StatusResult struct {
	ExecutionID    string      `json:"execution_id"`
	Status         Status      `json:"status"`
	ElapsedSeconds float64     `json:"elapsed_seconds"`
	OutputLines    int         `json:"output_lines,omitempty"`
	OutputID       string      `json:"output_id,omitempty"`
	Output         interface{} `json:"output,omitempty"`
	Error          string      `json:"error,omitempty"`
}

// NewWorkflowExecutionStatusTool registers the
// workflow_execution_status MCP tool.
func NewWorkflowExecutionStatusTool(
	manager *ExecutionManager,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name: "workflow_execution_status",
			Description: "Poll the status of a " +
				"running workflow execution. " +
				"Returns current status, elapsed " +
				"time, and results when complete.",
			InputSchema: json.RawMessage(
				workflowStatusInputSchema,
			),
		},
		Handler: handleStatus(manager, logger),
	}
}

func handleStatus(
	manager *ExecutionManager,
	logger *slog.Logger,
) protocol.ToolHandler {
	return func(
		ctx context.Context,
		arguments map[string]interface{},
	) (protocol.ToolsCallResult, *protocol.ErrorObject) {
		execID, _ := arguments["execution_id"].(string)
		if execID == "" {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInvalidParams,
					"execution_id is required", nil,
				)
		}

		exec := manager.Get(execID)
		if exec == nil {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInvalidParams,
					"execution not found",
					map[string]string{
						"execution_id": execID,
					},
				)
		}

		elapsed := elapsedSeconds(exec)
		result := StatusResult{
			ExecutionID:    exec.ID,
			Status:         exec.Status,
			ElapsedSeconds: elapsed,
		}

		switch exec.Status {
		case StatusRunning:
			if exec.LogBuffer != nil {
				result.OutputLines = exec.LogBuffer.Len()
			}
		case StatusCompleted:
			result.OutputID = exec.OutputID
			result.Output = exec.OutputData
		case StatusFailed:
			if exec.Error != nil {
				result.Error = exec.Error.Error()
			}
		case StatusCancelled:
			result.Error = "execution cancelled"
		}

		return renderJSONResult(result, logger)
	}
}

// elapsedSeconds returns the execution duration in
// seconds, using CompletedAt if finished, or now if
// still running.
func elapsedSeconds(exec *Execution) float64 {
	end := exec.CompletedAt
	if end.IsZero() {
		end = time.Now()
	}
	secs := end.Sub(exec.StartedAt).Seconds()
	return math.Round(secs*10) / 10
}
