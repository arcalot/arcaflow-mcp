package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"github.com/arcalot/arcaflow-mcp/server/pkg/transport/httpserver"
)

func TestHTTPSSessionBinding(t *testing.T) {
	handler := protocol.NewServer(nil, protocol.ServerInfo{Name: "arcaflow-mcp"})
	server := httpserver.NewServer(
		httpserver.Config{Address: "127.0.0.1:0"},
		handler,
		nil,
	)
	httpHandler := server.Handler()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sseReq := httptest.NewRequest(http.MethodGet, "/mcp/events", nil)
	sseReq = sseReq.WithContext(ctx)
	sseRecorder := newStreamingResponse()
	go func() {
		httpHandler.ServeHTTP(sseRecorder, sseReq)
		close(sseRecorder.writes)
	}()

	reader := bufio.NewReader(newChannelReader(sseRecorder.writes))
	event, data := readSSEEvent(t, ctx, reader)
	sessionID := sseRecorder.Header().Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatalf("missing Mcp-Session-Id header")
	}
	if event != "session" {
		t.Fatalf("expected session event, got %q", event)
	}
	if data != sessionID {
		t.Fatalf("expected session id %q, got %q", sessionID, data)
	}

	initReq := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(protocol.InitializeParams{ProtocolVersion: protocol.ProtocolVersion}),
	}
	postJSON(t, httpHandler, "/mcp", sessionID, initReq, http.StatusOK)

	event, data = readSSEEvent(t, ctx, reader)
	if event != "message" {
		t.Fatalf("expected message event, got %q", event)
	}
	var initResp protocol.Response
	if err := json.Unmarshal([]byte(data), &initResp); err != nil {
		t.Fatalf("unmarshal init response: %v", err)
	}
	if initResp.Error != nil {
		t.Fatalf("initialize error: %#v", initResp.Error)
	}

	initialized := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		Method:  "initialized",
	}
	postJSON(t, httpHandler, "/mcp", sessionID, initialized, http.StatusNoContent)

	pingReq := protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      rawID(2),
		Method:  "ping",
	}
	postJSON(t, httpHandler, "/mcp", sessionID, pingReq, http.StatusOK)

	event, data = readSSEEvent(t, ctx, reader)
	if event != "message" {
		t.Fatalf("expected message event, got %q", event)
	}
	var pingResp protocol.Response
	if err := json.Unmarshal([]byte(data), &pingResp); err != nil {
		t.Fatalf("unmarshal ping response: %v", err)
	}
	if pingResp.Error != nil {
		t.Fatalf("ping error: %#v", pingResp.Error)
	}
}

func postJSON(
	t *testing.T,
	handler http.Handler,
	path string,
	sessionID string,
	req protocol.Request,
	expected int,
) {
	t.Helper()
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	httpReq.Header.Set("Content-Type", "application/json")
	if sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", sessionID)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httpReq)
	if recorder.Code != expected {
		t.Fatalf("expected %d, got %d", expected, recorder.Code)
	}
}

func readSSEEvent(
	t *testing.T,
	ctx context.Context,
	reader *bufio.Reader,
) (string, string) {
	t.Helper()
	type result struct {
		event string
		data  string
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		var (
			event string
			data  []string
		)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				ch <- result{err: err}
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				ch <- result{event: event, data: strings.Join(data, "\n")}
				return
			}
			if strings.HasPrefix(line, "event:") {
				event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}
			if strings.HasPrefix(line, "data:") {
				data = append(data, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
	}()

	select {
	case <-ctx.Done():
		t.Fatalf("timeout waiting for SSE event: %v", ctx.Err())
		return "", ""
	case res := <-ch:
		if res.err != nil {
			t.Fatalf("read SSE event: %v", res.err)
			return "", ""
		}
		if res.event == "" {
			t.Fatalf("missing SSE event type")
			return "", ""
		}
		return res.event, res.data
	}
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

type streamingResponse struct {
	header http.Header
	status int
	writes chan []byte
	mu     sync.Mutex
}

func newStreamingResponse() *streamingResponse {
	return &streamingResponse{
		header: make(http.Header),
		writes: make(chan []byte, 16),
	}
}

func (s *streamingResponse) Header() http.Header {
	return s.header
}

func (s *streamingResponse) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	copied := make([]byte, len(data))
	copy(copied, data)
	s.writes <- copied
	return len(data), nil
}

func (s *streamingResponse) WriteHeader(statusCode int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = statusCode
}

func (s *streamingResponse) Flush() {}

type channelReader struct {
	ch  <-chan []byte
	buf []byte
}

func newChannelReader(ch <-chan []byte) *channelReader {
	return &channelReader{ch: ch}
}

func (c *channelReader) Read(p []byte) (int, error) {
	for len(c.buf) == 0 {
		next, ok := <-c.ch
		if !ok {
			return 0, io.EOF
		}
		c.buf = next
	}
	n := copy(p, c.buf)
	c.buf = c.buf[n:]
	return n, nil
}
