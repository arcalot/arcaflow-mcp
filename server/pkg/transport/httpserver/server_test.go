package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

func TestRoutes(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	server := NewServer(Config{Address: "127.0.0.1:0"}, handler, nil)
	if server.server == nil {
		t.Fatalf("expected server to be initialized")
	}

	tests := []struct {
		name   string
		method string
		path   string
		body   []byte
		cancel bool
		code   int
	}{
		{
			name:   "healthz",
			method: http.MethodGet,
			path:   "/healthz",
			code:   http.StatusOK,
		},
		{
			name:   "mcp-post-missing-body",
			method: http.MethodPost,
			path:   "/mcp",
			code:   http.StatusBadRequest,
		},
		{
			name:   "mcp-sse",
			method: http.MethodGet,
			path:   "/mcp/events",
			cancel: true,
			code:   http.StatusOK,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(
				testCase.method,
				testCase.path,
				bytes.NewReader(testCase.body),
			)
			if testCase.cancel {
				ctx, cancel := context.WithCancel(request.Context())
				cancel()
				request = request.WithContext(ctx)
			}
			server.server.Handler.ServeHTTP(recorder, request)
			if recorder.Code != testCase.code {
				t.Fatalf("expected %d, got %d", testCase.code, recorder.Code)
			}
		})
	}
}

func TestMCPPostFlow(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	server := NewServer(Config{Address: "127.0.0.1:0"}, handler, nil)

	initReq := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(protocol.InitializeParams{ProtocolVersion: protocol.ProtocolVersion}),
	}
	initResp := sendRequest(t, server, initReq)
	if initResp.Error != nil {
		t.Fatalf("unexpected initialize error: %#v", initResp.Error)
	}

	initialized := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		Method:  "initialized",
	}
	recorder := sendNotification(t, server, initialized)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}

	pingReq := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      rawID(2),
		Method:  "ping",
	}
	pingResp := sendRequest(t, server, pingReq)
	if pingResp.Error != nil {
		t.Fatalf("unexpected ping error: %#v", pingResp.Error)
	}
}

func TestSessionHeaderRequiredWhenActive(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	server := NewServer(Config{Address: "127.0.0.1:0"}, handler, nil)

	session := server.sessions.newSession()
	defer server.sessions.remove(session.id)

	req := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      rawID(1),
		Method:  "ping",
	}

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	request.Header.Set(sessionHeader, "missing")
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	request.Header.Set(sessionHeader, session.id)
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func sendRequest(
	t *testing.T,
	server *Server,
	req protocol.Request,
) protocol.Response {
	t.Helper()
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	server.server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var resp protocol.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return resp
}

func sendNotification(
	t *testing.T,
	server *Server,
	req protocol.Request,
) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
	server.server.Handler.ServeHTTP(recorder, request)
	return recorder
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
