package workflowtools

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestWorkflowResultsDescribeTool(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsDescribeTool(client, slog.Default())
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
	assertSuggestionActionable(t, payload.Suggestions[0])
}

func TestWorkflowResultsAnalyzeWithSource(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	resultPath := filepath.Join(root, "result.json")
	if err := os.WriteFile(
		resultPath,
		[]byte(`{"success":false,"metrics":{"latency_ms":12}}`),
		0o644,
	); err != nil {
		t.Fatalf("write result: %v", err)
	}

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsAnalyzeTool(client, slog.Default())
	result, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": resultPath,
		},
		"format": "json",
	})
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

func TestWorkflowResultsCompareRequiresTwoResults(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsCompareTool(client, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"results": []interface{}{
			map[string]interface{}{
				"format":  "json",
				"payload": map[string]interface{}{"success": true},
			},
		},
	})
	if errObj == nil {
		t.Fatalf("expected comparison error")
	}
}

func TestWorkflowResultsCompareRejectsSourceOnly(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsCompareTool(client, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": "/tmp/results.json",
		},
	})
	if errObj == nil {
		t.Fatalf("expected comparison error")
	}
}

func TestWorkflowResultsAnalyzeMissingResults(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsAnalyzeTool(client, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
	if errObj == nil {
		t.Fatalf("expected missing results error")
	}
}

func TestWorkflowResultsAnalyzeRejectsResultsAndSource(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsAnalyzeTool(client, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"results": []interface{}{
			map[string]interface{}{
				"format":  "json",
				"payload": map[string]interface{}{"success": true},
			},
		},
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": "/tmp/results.json",
		},
	})
	if errObj == nil {
		t.Fatalf("expected results and source error")
	}
}

func TestWorkflowResultsAnalyzeMissingClient(t *testing.T) {
	t.Parallel()

	tool := NewWorkflowResultsAnalyzeTool(nil, slog.Default())
	_, errObj := tool.Handler(context.Background(), analysisToolArgs())
	if errObj == nil {
		t.Fatalf("expected missing analysis client error")
	}
}

func TestWorkflowResultsDescribeMissingClient(t *testing.T) {
	t.Parallel()

	tool := NewWorkflowResultsDescribeTool(nil, slog.Default())
	_, errObj := tool.Handler(context.Background(), analysisToolArgs())
	if errObj == nil {
		t.Fatalf("expected missing analysis client error")
	}
}

func TestWorkflowResultsAnalyzeCanceledContext(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)
	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsAnalyzeTool(client, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, errObj := tool.Handler(ctx, analysisToolArgs())
	if errObj == nil {
		t.Fatalf("expected cancelled context error")
	}
}

func TestWorkflowResultsAnalyzeProvidesHintOnFailure(t *testing.T) {
	t.Parallel()

	client := analysis.NewClient("http://[::1")
	tool := NewWorkflowResultsAnalyzeTool(client, slog.Default())
	_, errObj := tool.Handler(context.Background(), analysisToolArgs())
	if errObj == nil {
		t.Fatalf("expected analysis error")
	}
	if !errorDataHasHint(errObj.Data) {
		t.Fatalf("expected hint in error data")
	}
}

func TestWorkflowResultsCompareMissingClient(t *testing.T) {
	t.Parallel()

	tool := NewWorkflowResultsCompareTool(nil, slog.Default())
	_, errObj := tool.Handler(context.Background(), analysisToolArgs())
	if errObj == nil {
		t.Fatalf("expected missing analysis client error")
	}
}

func TestWorkflowResultsCompareInvalidArguments(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsCompareTool(client, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"results": make(chan int),
	})
	if errObj == nil {
		t.Fatalf("expected invalid arguments error")
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
	assertSuggestionActionable(t, payload.Suggestions[0])
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

func errorDataHasHint(data interface{}) bool {
	switch value := data.(type) {
	case map[string]string:
		return value["hint"] != ""
	case map[string]interface{}:
		hint, ok := value["hint"].(string)
		return ok && hint != ""
	default:
		return false
	}
}

func assertSuggestionActionable(
	t *testing.T,
	suggestion map[string]interface{},
) {
	t.Helper()

	title, _ := suggestion["title"].(string)
	rationale, _ := suggestion["rationale"].(string)
	priority, _ := suggestion["priority"].(string)
	if title == "" || rationale == "" || priority == "" {
		t.Fatalf("expected actionable suggestion fields")
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
