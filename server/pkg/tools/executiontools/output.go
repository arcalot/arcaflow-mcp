package executiontools

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

// maxOutputBytes caps the total output returned to
// avoid overwhelming the MCP response.
const maxOutputBytes = 50 * 1024

const workflowOutputInputSchema = `{
  "type": "object",
  "properties": {
    "execution_id": {
      "type": "string",
      "description": "Execution ID."
    },
    "stream": {
      "type": "string",
      "enum": ["stdout", "stderr", "both"],
      "default": "both",
      "description": "Which output stream to return."
    },
    "tail_lines": {
      "type": "integer",
      "description": "Return only the last N lines. 0 = all.",
      "default": 0
    }
  },
  "required": ["execution_id"],
  "additionalProperties": false
}`

// OutputResult is the workflow_execution_output output.
type OutputResult struct {
	ExecutionID string `json:"execution_id"`
	Status      Status `json:"status"`
	Lines       int    `json:"lines"`
	Output      string `json:"output"`
	Truncated   bool   `json:"truncated"`
}

// NewWorkflowExecutionOutputTool registers the
// workflow_execution_output MCP tool.
func NewWorkflowExecutionOutputTool(
	manager *ExecutionManager,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name: "workflow_execution_output",
			Description: "Retrieve engine log output " +
				"for a running or completed " +
				"execution. Useful for diagnostics " +
				"and monitoring progress.",
			InputSchema: json.RawMessage(
				workflowOutputInputSchema,
			),
		},
		Handler: handleOutput(manager, logger),
	}
}

func handleOutput(
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

		tailLines := 0
		if tl, ok :=
			arguments["tail_lines"].(float64); ok {
			tailLines = int(tl)
		}

		// Retrieve log lines from the buffer.
		var lines []string
		if exec.LogBuffer != nil {
			if tailLines > 0 {
				lines = exec.LogBuffer.Tail(tailLines)
			} else {
				lines = exec.LogBuffer.Lines()
			}
		}

		output := strings.Join(lines, "\n")
		truncated := false
		if len(output) > maxOutputBytes {
			output = output[len(output)-maxOutputBytes:]
			truncated = true
		}

		result := OutputResult{
			ExecutionID: exec.ID,
			Status:      exec.Status,
			Lines:       len(lines),
			Output:      output,
			Truncated:   truncated,
		}

		return renderJSONResult(result, logger)
	}
}
