package workflowtools

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/analysis"
)

func TestWorkflowResultsMetricsExtractTool(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsMetricsExtractTool(client, slog.Default())
	result, errObj := tool.Handler(context.Background(), analysisToolArgs())
	if errObj != nil {
		t.Fatalf("expected no error, got %v", errObj)
	}

	var payload ResultsMetricsExtractResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(payload.MetricStats) == 0 {
		t.Fatalf("expected metric stats")
	}
	if payload.RecordCount == 0 {
		t.Fatalf("expected record count")
	}
}

func TestWorkflowResultsMetricsExtractWithSource(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	resultPath := filepath.Join(root, "result.json")
	if err := os.WriteFile(
		resultPath,
		[]byte(`{"success":true,"metrics":{"latency_ms":12}}`),
		0o644,
	); err != nil {
		t.Fatalf("write result: %v", err)
	}

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsMetricsExtractTool(client, slog.Default())
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

	var payload ResultsMetricsExtractResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(payload.MetricStats) == 0 {
		t.Fatalf("expected metric stats")
	}
}

func TestWorkflowResultsMetricsMissingClient(t *testing.T) {
	t.Parallel()

	tool := NewWorkflowResultsMetricsExtractTool(nil, slog.Default())
	_, errObj := tool.Handler(context.Background(), analysisToolArgs())
	if errObj == nil {
		t.Fatalf("expected missing client error")
	}
}

func TestWorkflowResultsMetricsCanceledContext(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsMetricsExtractTool(client, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, errObj := tool.Handler(ctx, analysisToolArgs())
	if errObj == nil {
		t.Fatalf("expected context error")
	}
}

func TestWorkflowResultsMetricsInvalidArguments(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsMetricsExtractTool(client, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"results": make(chan int),
	})
	if errObj == nil {
		t.Fatalf("expected invalid arguments error")
	}
}

func TestWorkflowResultsMetricsRejectsResultsAndSource(t *testing.T) {
	t.Parallel()

	server := newAnalysisTestServer(t)
	t.Cleanup(server.Close)

	client := analysis.NewClient(server.URL)
	tool := NewWorkflowResultsMetricsExtractTool(client, slog.Default())
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
