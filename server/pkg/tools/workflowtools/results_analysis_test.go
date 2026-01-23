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

func TestWorkflowResultsParseTool(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsParseTool(client, slog.Default())
	result, errObj := tool.Handler(context.Background(), analysisCompareArgs())
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload ResultsParseResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Analysis.RecordCount == 0 {
		t.Fatalf("expected record count")
	}
}

func TestWorkflowResultsAnalyzeTool(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsAnalyzeTool(client, slog.Default())
	result, errObj := tool.Handler(context.Background(), analysisToolArgs())
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload ResultsAnalyzeResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(payload.Suggestions) == 0 {
		t.Fatalf("expected suggestions")
	}
}

func TestWorkflowResultsCompareTool(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsCompareTool(client, slog.Default())
	result, errObj := tool.Handler(context.Background(), analysisCompareArgs())
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload ResultsCompareResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Comparison == nil {
		t.Fatalf("expected comparison summary")
	}
}

func TestWorkflowInputsSuggestTool(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowInputsSuggestTool(client, slog.Default())
	result, errObj := tool.Handler(context.Background(), analysisToolArgs())
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload InputsSuggestResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(payload.Suggestions) == 0 {
		t.Fatalf("expected suggestions")
	}
}

func TestWorkflowOptimizationGuideTool(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowOptimizationGuideTool(client, slog.Default())
	result, errObj := tool.Handler(context.Background(), analysisToolArgs())
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload OptimizationGuideResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload.Guidance == "" {
		t.Fatalf("expected guidance")
	}
}

func analysisToolArgs() map[string]interface{} {
	return map[string]interface{}{
		"results": []interface{}{
			map[string]interface{}{
				"format":  "json",
				"payload": map[string]interface{}{"success": true},
			},
		},
	}
}

func analysisCompareArgs() map[string]interface{} {
	return map[string]interface{}{
		"results": []interface{}{
			map[string]interface{}{
				"format":  "json",
				"payload": map[string]interface{}{"success": true},
			},
			map[string]interface{}{
				"format":  "json",
				"payload": map[string]interface{}{"success": true},
			},
		},
	}
}

func newAnalysisTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analysis/summary" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, compare := body["compare"].(bool)
		response := map[string]interface{}{
			"analysis": map[string]interface{}{
				"success_rate": 1.0,
				"metric_stats": map[string]interface{}{
					"latency": map[string]interface{}{
						"mean": 1.0,
					},
				},
				"record_count": 1,
				"findings": []map[string]interface{}{
					{
						"severity": "info",
						"message":  "ok",
					},
				},
			},
			"suggestions": []map[string]interface{}{
				{
					"title":     "Tune input",
					"priority":  "high",
					"rationale": "Improve throughput.",
				},
			},
		}
		if compare {
			response["comparison"] = map[string]interface{}{
				"metric_stats": map[string]interface{}{
					"latency": map[string]interface{}{
						"mean": 1.0,
					},
				},
				"rankings": map[string]interface{}{
					"latency": []interface{}{
						[]interface{}{"run-a", 1.0},
					},
				},
				"findings": []map[string]interface{}{},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
}
