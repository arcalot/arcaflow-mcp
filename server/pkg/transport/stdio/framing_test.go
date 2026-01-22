package stdio

import (
	"bufio"
	"bytes"
	"errors"
	"strconv"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)

	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)
	if err := WriteFrame(writer, payload); err != nil {
		t.Fatalf("write frame: %v", err)
	}

	reader := bufio.NewReader(&buffer)
	readPayload, err := ReadFrame(reader)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}

	if string(readPayload) != string(payload) {
		t.Fatalf("payload mismatch: %s", readPayload)
	}
}

func TestMissingContentLength(t *testing.T) {
	data := []byte("X-Header: test\r\n\r\n{}")
	reader := bufio.NewReader(bytes.NewReader(data))

	if _, err := ReadFrame(reader); err == nil {
		t.Fatalf("expected error for missing content-length")
	}
}

func TestReadFrameWithoutHeaderTerminator(t *testing.T) {
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	header := []byte("Content-Length: ")
	header = strconv.AppendInt(header, int64(len(payload)), 10)
	header = append(header, '\n')
	data := append(header, payload...)
	reader := bufio.NewReader(bytes.NewReader(data))

	readPayload, mode, err := ReadFrameWithMode(reader)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if string(readPayload) != string(payload) {
		t.Fatalf("payload mismatch: %s", readPayload)
	}
	if mode != ModeContentLength {
		t.Fatalf("expected content-length mode, got %s", mode)
	}
}

func TestReadFrameJSONPayloadWithFallback(t *testing.T) {
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	reader := bufio.NewReader(bytes.NewReader(payload))

	readPayload, mode, err := ReadFrameWithMode(reader)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if string(readPayload) != string(payload) {
		t.Fatalf("payload mismatch: %s", readPayload)
	}
	if mode != ModeJSON {
		t.Fatalf("expected json mode, got %s", mode)
	}
}

func TestReadFrameJSONPayloadMultipleLines(t *testing.T) {
	first := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	second := []byte(`{"jsonrpc":"2.0","id":2,"method":"ping"}`)
	payload := append(append(first, '\n'), append(second, '\n')...)
	reader := bufio.NewReader(bytes.NewReader(payload))

	readPayload, mode, err := ReadFrameWithMode(reader)
	if err != nil {
		t.Fatalf("read first frame: %v", err)
	}
	if string(readPayload) != string(first) {
		t.Fatalf("first payload mismatch: %s", readPayload)
	}
	if mode != ModeJSON {
		t.Fatalf("expected json mode, got %s", mode)
	}

	readPayload, mode, err = ReadFrameWithMode(reader)
	if err != nil {
		t.Fatalf("read second frame: %v", err)
	}
	if string(readPayload) != string(second) {
		t.Fatalf("second payload mismatch: %s", readPayload)
	}
	if mode != ModeJSON {
		t.Fatalf("expected json mode, got %s", mode)
	}
}

func TestWriteJSONFrame(t *testing.T) {
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)

	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)
	if err := WriteJSONFrame(writer, payload); err != nil {
		t.Fatalf("write json frame: %v", err)
	}
	if buffer.Len() == 0 {
		t.Fatalf("expected output")
	}
	if buffer.Bytes()[buffer.Len()-1] != '\n' {
		t.Fatalf("expected newline terminator")
	}
}

func TestReadJSONPayloadErrors(t *testing.T) {
	reader := bufio.NewReader(bytes.NewBufferString("\n"))
	if _, err := readJSONPayload(reader); err == nil {
		t.Fatalf("expected error for empty payload")
	}
	reader = bufio.NewReader(bytes.NewBufferString("{invalid}\n"))
	if _, err := readJSONPayload(reader); err == nil {
		t.Fatalf("expected error for invalid json")
	}
}

func TestReadJSONPayloadEOF(t *testing.T) {
	reader := bufio.NewReader(bytes.NewBufferString(`{"jsonrpc":"2.0"}`))
	payload, err := readJSONPayload(reader)
	if err != nil {
		t.Fatalf("expected payload on eof: %v", err)
	}
	if string(payload) != `{"jsonrpc":"2.0"}` {
		t.Fatalf("unexpected payload: %s", payload)
	}
}

func TestPeekNextNonSpace(t *testing.T) {
	reader := bufio.NewReader(bytes.NewBufferString("  \n\t{"))
	next, err := peekNextNonSpace(reader)
	if err != nil {
		t.Fatalf("peek failed: %v", err)
	}
	if next != '{' {
		t.Fatalf("expected '{' after whitespace")
	}
}

func TestReadFrameJSONFallbackDisabled(t *testing.T) {
	t.Setenv("ARCAFLOW_MCP_STDIO_ALLOW_JSON", "0")
	reader := bufio.NewReader(bytes.NewBufferString(`{"jsonrpc":"2.0"}`))
	if _, _, err := ReadFrameWithMode(reader); err == nil {
		t.Fatalf("expected content-length error when fallback disabled")
	}
}

func TestReadFrameInvalidContentLength(t *testing.T) {
	data := []byte("Content-Length: abc\r\n\r\n{}")
	reader := bufio.NewReader(bytes.NewReader(data))
	if _, err := ReadFrame(reader); err == nil {
		t.Fatalf("expected invalid content-length error")
	}
}

func TestReadFrameTooLargeContentLength(t *testing.T) {
	header := []byte("Content-Length: ")
	header = strconv.AppendInt(header, int64(maxContentLength+1), 10)
	header = append(header, []byte("\r\n\r\n")...)
	reader := bufio.NewReader(bytes.NewReader(header))
	if _, err := ReadFrame(reader); err == nil {
		t.Fatalf("expected size limit error")
	}
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write(_ []byte) (int, error) {
	return 0, writer.err
}

func TestWriteFrameHeaderError(t *testing.T) {
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	writer := bufio.NewWriterSize(errorWriter{err: errTest}, 1)
	if err := WriteFrame(writer, payload); err == nil {
		t.Fatalf("expected write header error")
	}
}

func TestWriteFrameFlushError(t *testing.T) {
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	writer := bufio.NewWriterSize(errorWriter{err: errTest}, 1024)
	if err := WriteFrame(writer, payload); err == nil {
		t.Fatalf("expected flush error")
	}
}

func TestWriteJSONFramePayloadError(t *testing.T) {
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	writer := bufio.NewWriterSize(errorWriter{err: errTest}, 1)
	if err := WriteJSONFrame(writer, payload); err == nil {
		t.Fatalf("expected write payload error")
	}
}

func TestWriteJSONFrameFlushError(t *testing.T) {
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	writer := bufio.NewWriterSize(errorWriter{err: errTest}, 1024)
	if err := WriteJSONFrame(writer, payload); err == nil {
		t.Fatalf("expected flush error")
	}
}

var errTest = errors.New("test error")
