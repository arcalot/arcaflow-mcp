// Package protocol implements MCP JSON-RPC handling and method routing.
package protocol

import "fmt"

const (
	// JSONRPCVersion is the only supported JSON-RPC version.
	JSONRPCVersion = "2.0"
	// ProtocolVersion is the MCP protocol version implemented by the server.
	ProtocolVersion = "2025-11-25"
)

const (
	ErrParse          = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternal       = -32603
)

// ErrorObject is a JSON-RPC error payload.
type ErrorObject struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (err *ErrorObject) Error() string {
	return fmt.Sprintf("json-rpc error %d: %s", err.Code, err.Message)
}

func newError(code int, message string, data interface{}) *ErrorObject {
	return &ErrorObject{
		Code:    code,
		Message: message,
		Data:    data,
	}
}
