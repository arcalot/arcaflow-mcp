package workflowtools

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/analysis"
)

func TestWorkflowHistoryLoadList(t *testing.T) {
	t.Parallel()

	server := newHistoryTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowHistoryLoadTool(client, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload HistoryLoadResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(payload.Runs) != 1 {
		t.Fatalf("expected 1 run summary")
	}
}

func TestWorkflowHistoryLoadRun(t *testing.T) {
	t.Parallel()

	server := newHistoryTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowHistoryLoadTool(client, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"run_id": "run-1",
	})
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload HistoryLoadResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Run == nil || payload.Run.RunID != "run-1" {
		t.Fatalf("expected run record")
	}
}

func newHistoryTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/analysis/history":
			response := map[string]interface{}{
				"runs": []map[string]interface{}{
					{
						"run_id":      "run-1",
						"workflow_id": "workflow-1",
						"created_at":  "2026-01-23T00:00:00Z",
						"metrics":     map[string]interface{}{"latency": 1.0},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		case r.Method == http.MethodGet && r.URL.Path == "/analysis/history/run-1":
			response := map[string]interface{}{
				"run": map[string]interface{}{
					"run_id":        "run-1",
					"workflow_id":   "workflow-1",
					"created_at":    "2026-01-23T00:00:00Z",
					"input_payload": map[string]interface{}{"name": "input"},
					"metrics":       map[string]interface{}{"latency": 1.0},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
