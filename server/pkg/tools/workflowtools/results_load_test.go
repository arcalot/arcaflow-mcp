package workflowtools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

func TestWorkflowResultsLoadFilesystemJSON(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "result.json")
	if err := os.WriteFile(path, []byte(`{"success":true}`), 0o644); err != nil {
		t.Fatalf("write result: %v", err)
	}

	tool := NewWorkflowResultsLoadTool(nil)
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": path,
		},
		"format": "json",
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload ResultsLoadResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Format != "json" {
		t.Fatalf("expected json format")
	}
}

func TestWorkflowResultsLoadURLYAML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("status: ok\n"))
	}))
	defer server.Close()

	tool := NewWorkflowResultsLoadTool(nil)
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "url",
			"location": server.URL,
		},
		"format": "yaml",
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload ResultsLoadResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Format != "yaml" {
		t.Fatalf("expected yaml format")
	}
}

func TestWorkflowResultsLoadMissingSource(t *testing.T) {
	tool := NewWorkflowResultsLoadTool(nil)
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
	if errObj == nil {
		t.Fatalf("expected missing source error")
	}
	// errObj is guaranteed non-nil here due to check above
	if errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestWorkflowResultsLoadAutoDetectsLog(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "result.log")
	if err := os.WriteFile(path, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write result: %v", err)
	}

	tool := NewWorkflowResultsLoadTool(nil)
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": path,
		},
		"format": "log",
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}
	var payload ResultsLoadResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Format != "log" {
		t.Fatalf("expected log format")
	}
}

func TestParseResultPayloadInvalidFormat(t *testing.T) {
	_, _, err := parseResultPayload("value", "xml")
	if err == nil {
		t.Fatalf("expected unsupported format error")
	}
}

func TestParseResultPayloadAutoDetectsJSONAndYAML(t *testing.T) {
	format, _, err := parseResultPayload(`{"ok":true}`, "")
	if err != nil {
		t.Fatalf("expected json parse, got %v", err)
	}
	if format != "json" {
		t.Fatalf("expected json format")
	}

	format, _, err = parseResultPayload("status: ok\n", "")
	if err != nil {
		t.Fatalf("expected yaml parse, got %v", err)
	}
	if format != "yaml" {
		t.Fatalf("expected yaml format")
	}
}

func TestWorkflowResultsLoadInvalidURLViaTool(t *testing.T) {
	tool := NewWorkflowResultsLoadTool(nil)
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "url",
			"location": "http://[::1",
		},
	})
	if errObj == nil {
		t.Fatalf("expected invalid url error")
	}
}

func TestLoadResultContentErrors(t *testing.T) {
	_, _, _, err := loadResultContent(context.Background(), ListSourceParams{
		Kind:     "unknown",
		Location: "/tmp/result.json",
	})
	if err == nil {
		t.Fatalf("expected unsupported kind error")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	_, _, _, err = loadResultContent(context.Background(), ListSourceParams{
		Kind:     "url",
		Location: server.URL,
	})
	if err == nil {
		t.Fatalf("expected url status error")
	}
}

func TestWorkflowResultsLoadIncludeRawToggle(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "result.json")
	if err := os.WriteFile(path, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("write result: %v", err)
	}

	tool := NewWorkflowResultsLoadTool(nil)
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": path,
		},
		"format":      "json",
		"include_raw": false,
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}
	var payload ResultsLoadResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.RawText != "" {
		t.Fatalf("expected raw text to be omitted")
	}
}
