package stdio

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

func TestServeHandlesInitialize(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{
		Name: "arcaflow-mcp",
	})

	var input bytes.Buffer
	writer := bufio.NewWriter(&input)

	initReq := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(protocol.InitializeParams{ProtocolVersion: protocol.ProtocolVersion}),
	}
	if err := WriteFrame(writer, mustMarshal(initReq)); err != nil {
		t.Fatalf("write initialize frame: %v", err)
	}

	initialized := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		Method:  "initialized",
	}
	if err := WriteFrame(writer, mustMarshal(initialized)); err != nil {
		t.Fatalf("write initialized frame: %v", err)
	}

	toolsList := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      rawID(2),
		Method:  "tools/list",
	}
	if err := WriteFrame(writer, mustMarshal(toolsList)); err != nil {
		t.Fatalf("write tools/list frame: %v", err)
	}

	var output bytes.Buffer
	server := NewServer(handler, &input, &output, nil)

	if err := server.Serve(context.Background()); err != nil {
		t.Fatalf("serve failed: %v", err)
	}

	reader := bufio.NewReader(&output)
	first, err := ReadFrame(reader)
	if err != nil {
		t.Fatalf("read initialize response: %v", err)
	}

	var initResp protocol.Response
	if err := json.Unmarshal(first, &initResp); err != nil {
		t.Fatalf("unmarshal initialize response: %v", err)
	}
	if initResp.Error != nil {
		t.Fatalf("unexpected initialize error: %#v", initResp.Error)
	}

	second, err := ReadFrame(reader)
	if err != nil {
		t.Fatalf("read tools/list response: %v", err)
	}

	var listResp protocol.Response
	if err := json.Unmarshal(second, &listResp); err != nil {
		t.Fatalf("unmarshal tools/list response: %v", err)
	}
	if listResp.Error != nil {
		t.Fatalf("unexpected tools/list error: %#v", listResp.Error)
	}
}

func TestServeReturnsReadError(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	input := bytes.NewBufferString("Content-Length: abc\r\n\r\n{}")
	var output bytes.Buffer
	server := NewServer(handler, input, &output, nil)
	if err := server.Serve(context.Background()); err == nil {
		t.Fatalf("expected read error")
	}
}

func TestServeHandlesJSONMode(t *testing.T) {
	t.Setenv("ARCAFLOW_MCP_STDIO_ALLOW_JSON", "1")

	handler := protocol.NewServer(nil, protocol.ServerInfo{
		Name: "arcaflow-mcp",
	})

	var input bytes.Buffer
	input.WriteString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"`)
	input.WriteString(protocol.ProtocolVersion)
	input.WriteString(`"}}` + "\n")
	input.WriteString(`{"jsonrpc":"2.0","method":"initialized"}` + "\n")
	input.WriteString(`{"jsonrpc":"2.0","id":2,"method":"ping"}` + "\n")

	var output bytes.Buffer
	server := NewServer(handler, &input, &output, nil)

	if err := server.Serve(context.Background()); err != nil {
		t.Fatalf("serve failed: %v", err)
	}
	if output.Len() == 0 {
		t.Fatalf("expected json output")
	}
}

func mustMarshal(req protocol.Request) []byte {
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
