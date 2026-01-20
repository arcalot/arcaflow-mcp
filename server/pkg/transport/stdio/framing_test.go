package stdio

import (
	"bufio"
	"bytes"
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
