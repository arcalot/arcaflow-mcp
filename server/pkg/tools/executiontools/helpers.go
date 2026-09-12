package executiontools

import (
	"encoding/json"
	"log/slog"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

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
			{Type: "text", Text: string(payload)},
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
