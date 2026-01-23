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
	if errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}
