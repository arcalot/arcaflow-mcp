package protocol

import (
	"context"
	"encoding/json"
	"testing"
)

func TestHandleInvalidJSON(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})

	responses, err := server.Handle(context.Background(), []byte("{invalid"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	var resp Response
	if err := json.Unmarshal(responses[0], &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != ErrParse {
		t.Fatalf("expected parse error, got %#v", resp.Error)
	}
	if string(resp.ID) != "null" {
		t.Fatalf("expected null id, got %s", resp.ID)
	}
}

func TestInitializeSequence(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})

	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})

	responses, err := server.Handle(context.Background(), initReq)
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	var initResp Response
	if err := json.Unmarshal(responses[0], &initResp); err != nil {
		t.Fatalf("unmarshal initialize response: %v", err)
	}
	if initResp.Error != nil {
		t.Fatalf("unexpected error: %#v", initResp.Error)
	}

	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		Method:  "initialized",
	})

	_, err = server.Handle(context.Background(), initialized)
	if err != nil {
		t.Fatalf("initialized failed: %v", err)
	}

	listReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "tools/list",
	})

	responses, err = server.Handle(context.Background(), listReq)
	if err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	var listResp Response
	if err := json.Unmarshal(responses[0], &listResp); err != nil {
		t.Fatalf("unmarshal tools/list response: %v", err)
	}
	if listResp.Error != nil {
		t.Fatalf("unexpected error: %#v", listResp.Error)
	}
}

func TestRejectBeforeInitialize(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})

	listReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(3),
		Method:  "tools/list",
	})

	responses, err := server.Handle(context.Background(), listReq)
	if err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	var listResp Response
	if err := json.Unmarshal(responses[0], &listResp); err != nil {
		t.Fatalf("unmarshal tools/list response: %v", err)
	}
	if listResp.Error == nil || listResp.Error.Code != ErrInvalidRequest {
		t.Fatalf("expected invalid request error, got %#v", listResp.Error)
	}
}

func TestBatchHandling(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})

	batch := []Request{
		{
			JSONRPC: JSONRPCVersion,
			ID:      rawID(1),
			Method:  "initialize",
			Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
		},
		{
			JSONRPC: JSONRPCVersion,
			Method:  "initialized",
		},
		{
			JSONRPC: JSONRPCVersion,
			ID:      rawID(2),
			Method:  "tools/list",
		},
	}

	payload, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("marshal batch: %v", err)
	}

	responses, err := server.Handle(context.Background(), payload)
	if err != nil {
		t.Fatalf("batch failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response payload, got %d", len(responses))
	}

	var respBatch []Response
	if err := json.Unmarshal(responses[0], &respBatch); err != nil {
		t.Fatalf("unmarshal batch responses: %v", err)
	}
	if len(respBatch) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(respBatch))
	}
}

func TestNotificationHasNoResponse(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})

	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		Method:  "initialized",
	})

	responses, err := server.Handle(context.Background(), initialized)
	if err != nil {
		t.Fatalf("initialized notification failed: %v", err)
	}
	if responses != nil {
		t.Fatalf("expected no response for notification")
	}
}

func TestNotificationsInitializedTransitionsState(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})

	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		Method:  "notifications/initialized",
	})
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("notifications/initialized failed: %v", err)
	}

	listReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "tools/list",
	})
	responses, err := server.Handle(context.Background(), listReq)
	if err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
}

func TestNotificationRejectedForRequestMethods(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})

	req := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		Method:  "ping",
	})

	responses, err := server.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	if responses != nil {
		t.Fatalf("expected no response for notification")
	}
}

func TestInvalidIDRejected(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})

	rawID := json.RawMessage(`{"bad":"id"}`)
	req := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      &rawID,
		Method:  "ping",
	})

	responses, err := server.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	var resp Response
	if err := json.Unmarshal(responses[0], &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != ErrInvalidRequest {
		t.Fatalf("expected invalid request error, got %#v", resp.Error)
	}
}

func TestNullIDRejected(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})

	rawID := json.RawMessage(`null`)
	req := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      &rawID,
		Method:  "ping",
	})

	responses, err := server.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	var resp Response
	if err := json.Unmarshal(responses[0], &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != ErrInvalidRequest {
		t.Fatalf("expected invalid request error, got %#v", resp.Error)
	}
}

func TestBatchNotificationsNoResponse(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})

	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	batch := []Request{
		{
			JSONRPC: JSONRPCVersion,
			Method:  "initialized",
		},
		{
			JSONRPC: JSONRPCVersion,
			Method:  "ping",
		},
	}

	payload, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("marshal batch: %v", err)
	}

	responses, err := server.Handle(context.Background(), payload)
	if err != nil {
		t.Fatalf("batch failed: %v", err)
	}
	if responses != nil {
		t.Fatalf("expected no responses for notifications batch")
	}
}

func TestInvalidParamsRejected(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})

	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		Method:  "initialized",
	})
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("initialized failed: %v", err)
	}

	cases := []Request{
		{
			JSONRPC: JSONRPCVersion,
			ID:      rawID(2),
			Method:  "tools/list",
			Params:  mustMarshalRaw([]string{"bad"}),
		},
		{
			JSONRPC: JSONRPCVersion,
			ID:      rawID(3),
			Method:  "tools/call",
		},
		{
			JSONRPC: JSONRPCVersion,
			ID:      rawID(4),
			Method:  "resources/read",
			Params:  mustMarshalRaw([]string{"bad"}),
		},
		{
			JSONRPC: JSONRPCVersion,
			ID:      rawID(5),
			Method:  "ping",
			Params:  mustMarshalRaw(map[string]interface{}{"bad": "value"}),
		},
	}

	for _, req := range cases {
		responses, err := server.Handle(context.Background(), mustMarshal(req))
		if err != nil {
			t.Fatalf("handle failed: %v", err)
		}
		if len(responses) != 1 {
			t.Fatalf("expected 1 response, got %d", len(responses))
		}

		var resp Response
		if err := json.Unmarshal(responses[0], &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.Error == nil || resp.Error.Code != ErrInvalidParams {
			t.Fatalf("expected invalid params error, got %#v", resp.Error)
		}
	}
}

func TestRegisterToolSkipsInvalid(t *testing.T) {
	server := NewServer(nil, ServerInfo{})
	server.RegisterTool(ToolRegistration{})
	if len(server.tools) != 0 {
		t.Fatalf("expected no tools registered")
	}
	server.RegisterTool(ToolRegistration{
		Definition: ToolDefinition{Name: "ok"},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (ToolsCallResult, *ErrorObject) {
			_ = ctx
			_ = arguments
			return ToolsCallResult{}, nil
		},
	})
	if len(server.tools) != 1 {
		t.Fatalf("expected tool registration")
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	server := NewServer(nil, ServerInfo{})
	request := Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "unknown/method",
		Params:  mustMarshalRaw(map[string]interface{}{}),
	}
	responses, err := server.Handle(context.Background(), mustMarshal(request))
	if err != nil {
		t.Fatalf("handle request: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response")
	}
	var response Response
	if err := json.Unmarshal(responses[0], &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != ErrMethodNotFound {
		t.Fatalf("expected method not found error")
	}
}

func TestHandleInitializedTransitions(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})

	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "initialized",
	})
	responses, err := server.Handle(context.Background(), initialized)
	if err != nil {
		t.Fatalf("initialized failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response")
	}
}

func TestHandleResourcesList(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})

	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "initialized",
	})
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("initialized failed: %v", err)
	}

	listReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(3),
		Method:  "resources/list",
	})
	responses, err := server.Handle(context.Background(), listReq)
	if err != nil {
		t.Fatalf("resources/list failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response")
	}
}

func TestHandleToolsCallSuccess(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	server.RegisterTool(ToolRegistration{
		Definition: ToolDefinition{Name: "echo"},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (ToolsCallResult, *ErrorObject) {
			_ = ctx
			return ToolsCallResult{
				Content: []ToolContent{{
					Type: "text",
					Text: arguments["message"].(string),
				}},
			}, nil
		},
	})

	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "initialized",
	})
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("initialized failed: %v", err)
	}

	call := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(3),
		Method:  "tools/call",
		Params: mustMarshalRaw(ToolsCallParams{
			Name:      "echo",
			Arguments: map[string]interface{}{"message": "hello"},
		}),
	})
	responses, err := server.Handle(context.Background(), call)
	if err != nil {
		t.Fatalf("tools/call failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response")
	}
}

func TestHandleToolsCallNotFound(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "initialized",
	})
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("initialized failed: %v", err)
	}

	call := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(3),
		Method:  "tools/call",
		Params: mustMarshalRaw(ToolsCallParams{
			Name: "missing",
		}),
	})
	responses, err := server.Handle(context.Background(), call)
	if err != nil {
		t.Fatalf("tools/call failed: %v", err)
	}
	var response Response
	if err := json.Unmarshal(responses[0], &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestHandleResourcesReadNotFound(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "initialized",
	})
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("initialized failed: %v", err)
	}

	read := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(3),
		Method:  "resources/read",
		Params: mustMarshalRaw(ResourcesReadParams{
			URI: "resource://missing",
		}),
	})
	responses, err := server.Handle(context.Background(), read)
	if err != nil {
		t.Fatalf("resources/read failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response")
	}
}

func TestHandleResourcesReadMissingParams(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "initialized",
	})
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("initialized failed: %v", err)
	}

	read := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(3),
		Method:  "resources/read",
	})
	responses, err := server.Handle(context.Background(), read)
	if err != nil {
		t.Fatalf("resources/read failed: %v", err)
	}
	var response Response
	if err := json.Unmarshal(responses[0], &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestInvalidRequestID(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	raw := json.RawMessage("true")
	req := Request{
		JSONRPC: JSONRPCVersion,
		ID:      &raw,
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
		IDSet:   true,
	}
	responses, err := server.Handle(context.Background(), mustMarshal(req))
	if err != nil {
		t.Fatalf("handle request failed: %v", err)
	}
	var response Response
	if err := json.Unmarshal(responses[0], &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != ErrInvalidRequest {
		t.Fatalf("expected invalid request error")
	}
}

func TestHandleResourcesListWithParams(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "initialized",
	})
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("initialized failed: %v", err)
	}

	listReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(3),
		Method:  "resources/list",
		Params: mustMarshalRaw(ResourcesListParams{
			Cursor: "next",
		}),
	})
	responses, err := server.Handle(context.Background(), listReq)
	if err != nil {
		t.Fatalf("resources/list failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response")
	}
}

func TestHandlePingRejectsParams(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "initialized",
	})
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("initialized failed: %v", err)
	}

	ping := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(3),
		Method:  "ping",
		Params:  mustMarshalRaw(map[string]interface{}{"value": "bad"}),
	})
	responses, err := server.Handle(context.Background(), ping)
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	var response Response
	if err := json.Unmarshal(responses[0], &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestHandleInitializeRejectsBadProtocol(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	req := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: "invalid"}),
	})
	responses, err := server.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	var response Response
	if err := json.Unmarshal(responses[0], &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestHandleInitializeMissingParams(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	req := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
	})
	responses, err := server.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	var response Response
	if err := json.Unmarshal(responses[0], &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestInitializedNotificationBeforeInit(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	req := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		Method:  "initialized",
	})
	responses, err := server.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("initialized failed: %v", err)
	}
	if responses != nil {
		t.Fatalf("expected no response for notification")
	}
}

func TestHandleInitializedBeforeInit(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	req := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialized",
	})
	responses, err := server.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("initialized failed: %v", err)
	}
	var response Response
	if err := json.Unmarshal(responses[0], &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil || response.Error.Code != ErrInvalidRequest {
		t.Fatalf("expected invalid request error")
	}
}

func TestHandleNotifications(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	requests := []Request{
		{JSONRPC: JSONRPCVersion, Method: "initialized"},
		{JSONRPC: JSONRPCVersion, Method: "notifications/initialized"},
		{JSONRPC: JSONRPCVersion, Method: "notifications/cancelled"},
		{JSONRPC: JSONRPCVersion, Method: "ping"},
		{JSONRPC: JSONRPCVersion, Method: "unknown"},
	}
	for _, req := range requests {
		responses, err := server.Handle(context.Background(), mustMarshal(req))
		if err != nil {
			t.Fatalf("notification failed: %v", err)
		}
		if responses != nil {
			t.Fatalf("expected no response for notification")
		}
	}
}

func TestListTools(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	server.RegisterTool(ToolRegistration{
		Definition: ToolDefinition{Name: "ping"},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (ToolsCallResult, *ErrorObject) {
			_ = ctx
			_ = arguments
			return ToolsCallResult{}, nil
		},
	})
	list := server.listTools()
	if len(list) != 1 || list[0].Name != "ping" {
		t.Fatalf("expected tool list to include ping")
	}
}

func TestHandlePingSuccess(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "initialized",
	})
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("initialized failed: %v", err)
	}

	ping := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(3),
		Method:  "ping",
	})
	responses, err := server.Handle(context.Background(), ping)
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response")
	}
}

func TestHandleInitializedTwice(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "initialized",
	})
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("initialized failed: %v", err)
	}
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("second initialized failed: %v", err)
	}
}

func TestHandleToolsCallPanicRecovery(t *testing.T) {
	server := NewServer(
		nil, ServerInfo{Name: "arcaflow-mcp"},
	)
	server.RegisterTool(ToolRegistration{
		Definition: ToolDefinition{Name: "panicker"},
		Handler: func(
			_ context.Context,
			_ map[string]interface{},
		) (ToolsCallResult, *ErrorObject) {
			panic("deliberate test panic")
		},
	})

	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params: mustMarshalRaw(InitializeParams{
			ProtocolVersion: ProtocolVersion,
		}),
	})
	if _, err := server.Handle(
		context.Background(), initReq,
	); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "initialized",
	})
	if _, err := server.Handle(
		context.Background(), initialized,
	); err != nil {
		t.Fatalf("initialized: %v", err)
	}

	// Call the panicking tool — server must not crash.
	call := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(3),
		Method:  "tools/call",
		Params: mustMarshalRaw(ToolsCallParams{
			Name: "panicker",
		}),
	})
	responses, err := server.Handle(
		context.Background(), call,
	)
	if err != nil {
		t.Fatalf("handle returned error: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d",
			len(responses))
	}

	// Verify the response is an error, not a result.
	var resp Response
	if err := json.Unmarshal(
		responses[0], &resp,
	); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error response after panic")
	}
	if resp.Error.Code != ErrInternal {
		t.Errorf(
			"error code = %d, want %d",
			resp.Error.Code, ErrInternal,
		)
	}

	// Verify the server still works after the panic.
	pingReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(4),
		Method:  "ping",
	})
	pingResp, err := server.Handle(
		context.Background(), pingReq,
	)
	if err != nil {
		t.Fatalf("ping after panic: %v", err)
	}
	if len(pingResp) != 1 {
		t.Fatal("expected ping response")
	}
}

func mustMarshal(req Request) []byte {
	data, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}
	return data
}

func mustMarshalRaw(value interface{}) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func rawID(id int) *json.RawMessage {
	data, err := json.Marshal(id)
	if err != nil {
		panic(err)
	}
	raw := json.RawMessage(data)
	return &raw
}
