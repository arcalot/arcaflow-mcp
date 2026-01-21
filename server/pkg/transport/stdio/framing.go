// Package stdio implements MCP stdio framing.
package stdio

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

const maxContentLength = 10 * 1024 * 1024

type FrameMode string

const (
	ModeContentLength FrameMode = "content-length"
	ModeJSON          FrameMode = "json"
)

// ReadFrame reads a Content-Length framed JSON-RPC message.
func ReadFrame(reader *bufio.Reader) ([]byte, error) {
	payload, _, err := ReadFrameWithMode(reader)
	if err != nil {
		return nil, err
	}

	if len(payload) == 0 {
		return nil, fmt.Errorf("empty payload")
	}

	return payload, nil
}

// ReadFrameWithMode reads a JSON-RPC message and returns the framing mode.
func ReadFrameWithMode(reader *bufio.Reader) ([]byte, FrameMode, error) {
	if allowJSONFallback() {
		if payload, ok, err := readJSONFrame(reader); ok || err != nil {
			return payload, ModeJSON, err
		}
	}

	payload, err := readContentLengthFrame(reader)
	return payload, ModeContentLength, err
}

func readContentLengthFrame(reader *bufio.Reader) ([]byte, error) {
	length, err := readContentLength(reader)
	if err != nil {
		return nil, err
	}

	if length <= 0 {
		return nil, fmt.Errorf("content-length must be positive")
	}
	if length > maxContentLength {
		return nil, fmt.Errorf("content-length exceeds limit")
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}

	return payload, nil
}

// WriteFrame writes a Content-Length framed JSON-RPC message.
func WriteFrame(writer *bufio.Writer, payload []byte) error {
	if _, err := fmt.Fprintf(
		writer,
		"Content-Length: %d\r\n\r\n",
		len(payload),
	); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	if _, err := writer.Write(payload); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush writer: %w", err)
	}

	return nil
}

// WriteJSONFrame writes a JSON payload with a newline delimiter.
func WriteJSONFrame(writer *bufio.Writer, payload []byte) error {
	if _, err := writer.Write(payload); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}
	if _, err := writer.WriteString("\n"); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush writer: %w", err)
	}
	return nil
}

func readContentLength(reader *bufio.Reader) (int, error) {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			return 0, fmt.Errorf("missing Content-Length header")
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		if name != "content-length" {
			continue
		}

		length, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("invalid content-length: %w", err)
		}

		if err := discardHeaderTerminator(reader); err != nil {
			return 0, err
		}

		return length, nil
	}
}

func discardHeaderTerminator(reader *bufio.Reader) error {
	peek, err := reader.Peek(1)
	if err != nil {
		return err
	}
	if peek[0] != '\r' && peek[0] != '\n' {
		return nil
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			return nil
		}
	}
}

func allowJSONFallback() bool {
	value := strings.TrimSpace(os.Getenv("ARCAFLOW_MCP_STDIO_ALLOW_JSON"))
	if value == "" {
		return true
	}
	return strings.EqualFold(value, "true") || value == "1"
}

func readJSONFrame(reader *bufio.Reader) ([]byte, bool, error) {
	first, err := peekNextNonSpace(reader)
	if err != nil {
		return nil, true, err
	}
	if first != '{' && first != '[' {
		return nil, false, nil
	}
	payload, err := readJSONPayload(reader)
	if err != nil {
		return nil, true, err
	}
	logJSONFallback()
	return payload, true, nil
}

func peekNextNonSpace(reader *bufio.Reader) (byte, error) {
	for {
		b, err := reader.Peek(1)
		if err != nil {
			return 0, err
		}
		if len(b) == 0 {
			return 0, io.EOF
		}
		if unicode.IsSpace(rune(b[0])) {
			if _, err := reader.ReadByte(); err != nil {
				return 0, err
			}
			continue
		}
		return b[0], nil
	}
}

func readJSONPayload(reader *bufio.Reader) ([]byte, error) {
	line, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("read json payload: %w", err)
	}
	if len(line) == 0 {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("empty json payload")
	}
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty json payload")
	}
	var raw json.RawMessage
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return nil, fmt.Errorf("decode json payload: %w", err)
	}
	return bytes.TrimSpace(raw), nil
}

var jsonFallbackOnce sync.Once

func logJSONFallback() {
	jsonFallbackOnce.Do(func() {
		slog.Default().Debug(
			"stdio JSON fallback enabled; Content-Length header missing",
		)
	})
}
