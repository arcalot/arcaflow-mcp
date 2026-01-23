package workflowtools

import (
	"context"
	"encoding/json"
	"log/slog"
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
