package protocol

import (
	"bytes"
	"context"
	"encoding/json"
)

// Request is a JSON-RPC request or notification.
type Request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
	IDSet   bool             `json:"-"`
}

// UnmarshalJSON captures whether an id was provided (including null).
func (r *Request) UnmarshalJSON(data []byte) error {
	type wireRequest struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var wire wireRequest
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	r.JSONRPC = wire.JSONRPC
	r.Method = wire.Method
	r.Params = wire.Params

	rawID, ok := raw["id"]
	if !ok {
		r.IDSet = false
		r.ID = nil
		return nil
	}

	r.IDSet = true
	if bytes.Equal(bytes.TrimSpace(rawID), []byte("null")) {
		r.ID = nil
		return nil
	}

	rawCopy := make(json.RawMessage, len(rawID))
	copy(rawCopy, rawID)
	r.ID = &rawCopy
	return nil
}

// Response is a JSON-RPC response payload.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *ErrorObject    `json:"error,omitempty"`
}

// ClientInfo describes a connecting MCP client.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ServerInfo describes the MCP server identity.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ClientCapabilities represent client-side features in initialize.
type ClientCapabilities struct{}

// ToolsCapability describes tool-related features.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability describes resource-related features.
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerCapabilities represent server-side features in initialize response.
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
}

// InitializeParams holds the initialize request payload.
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities,omitempty"`
	ClientInfo      *ClientInfo        `json:"clientInfo,omitempty"`
}

// InitializeResult is the initialize response payload.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// ToolsListParams provides paging for tools/list.
type ToolsListParams struct {
	Cursor string `json:"cursor,omitempty"`
}

// ToolDefinition describes an MCP tool.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// ToolsListResult is returned by tools/list.
type ToolsListResult struct {
	Tools      []ToolDefinition `json:"tools"`
	NextCursor string           `json:"nextCursor,omitempty"`
}

// ToolsCallParams identifies a tool invocation.
type ToolsCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolContent is output from a tool call.
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ToolsCallResult is returned by tools/call.
type ToolsCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ResourcesListParams provides paging for resources/list.
type ResourcesListParams struct {
	Cursor string `json:"cursor,omitempty"`
}

// ResourceItem describes a resource entry.
type ResourceItem struct {
	URI         string `json:"uri"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourcesListResult is returned by resources/list.
type ResourcesListResult struct {
	Resources  []ResourceItem `json:"resources"`
	NextCursor string         `json:"nextCursor,omitempty"`
}

// ResourcesReadParams identifies a resource to read.
type ResourcesReadParams struct {
	URI string `json:"uri"`
}

// ResourceContent describes a resource payload.
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// ResourcesReadResult is returned by resources/read.
type ResourcesReadResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ResourceProvider supplies resources for resources/list and resources/read.
type ResourceProvider interface {
	List(ctx context.Context) ([]ResourceItem, *ErrorObject)
	Read(
		ctx context.Context,
		uri string,
	) (*ResourceContent, bool, *ErrorObject)
}
