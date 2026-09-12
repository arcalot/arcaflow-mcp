// Package plugintools provides MCP tool handlers for
// discovering and inspecting Arcaflow plugins from
// Quay.io container registries.
package plugintools

import (
	"encoding/json"
	"log/slog"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

// renderJSONResult marshals the given value to JSON and
// wraps it in a ToolsCallResult with a single text
// content block.
func renderJSONResult(
	result interface{},
	logger *slog.Logger,
) (protocol.ToolsCallResult, *protocol.ErrorObject) {
	payload, err := json.Marshal(result)
	if err != nil {
		if logger != nil {
			logger.Error(
				"marshal tool result", "error", err,
			)
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

// toolError builds a protocol ErrorObject with the
// given code, human-readable message, and optional
// structured data.
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
