package executiontools

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

const workflowCancelInputSchema = `{
  "type": "object",
  "properties": {
    "execution_id": {
      "type": "string",
      "description": "Execution ID to cancel."
    }
  },
  "required": ["execution_id"],
  "additionalProperties": false
}`

// CancelResult is the workflow_execution_cancel output.
type CancelResult struct {
	ExecutionID string `json:"execution_id"`
	Status      Status `json:"status"`
	Message     string `json:"message"`
}

// NewWorkflowExecutionCancelTool registers the
// workflow_execution_cancel MCP tool.
func NewWorkflowExecutionCancelTool(
	manager *ExecutionManager,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name: "workflow_execution_cancel",
			Description: "Cancel a running workflow " +
				"execution. The engine context is " +
				"cancelled, which triggers graceful " +
				"shutdown of running plugins.",
			InputSchema: json.RawMessage(
				workflowCancelInputSchema,
			),
		},
		Handler: handleCancel(manager, logger),
	}
}

func handleCancel(
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

		if err := manager.Cancel(execID); err != nil {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInvalidParams,
					err.Error(),
					map[string]string{
						"execution_id": execID,
					},
				)
		}

		result := CancelResult{
			ExecutionID: execID,
			Status:      StatusCancelled,
			Message:     "Execution cancelled by client",
		}

		return renderJSONResult(result, logger)
	}
}
