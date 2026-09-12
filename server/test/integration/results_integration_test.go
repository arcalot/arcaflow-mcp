package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/analysis"
	"github.com/arcalot/arcaflow-mcp/server/pkg/tools/workflowtools"
)

func TestResultsAnalysisIntegration(t *testing.T) {
	t.Parallel()

	fixtures := filepath.Join("..", "..", "..", "test", "fixtures")
	resultsPath := filepath.Join(fixtures, "result-basic.json")

	loadTool := workflowtools.NewWorkflowResultsLoadTool(nil)
	loadResult, errObj := loadTool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": resultsPath,
		},
	})
	if errObj != nil {
		t.Fatalf("load results: %v", errObj)
	}

	var loaded workflowtools.ResultsLoadResult
	if err := json.Unmarshal([]byte(loadResult.Content[0].Text), &loaded); err != nil {
		t.Fatalf("unmarshal load result: %v", err)
	}

	server := newAnalysisIntegrationServer(t)
	t.Cleanup(server.Close)
	client := analysis.NewClient(server.URL)

	args := buildResultsArgs(loaded.Format, loaded.Payload, 1)
	compareArgs := buildResultsArgs(loaded.Format, loaded.Payload, 2)

	parseTool := workflowtools.NewWorkflowResultsParseTool(client, nil)
	parseResult, errObj := parseTool.Handler(context.Background(), args)
	if errObj != nil {
		t.Fatalf("parse results: %v", errObj)
	}
	var parsePayload workflowtools.ResultsParseResult
	if err := json.Unmarshal([]byte(parseResult.Content[0].Text), &parsePayload); err != nil {
		t.Fatalf("unmarshal parse result: %v", err)
	}
	if parsePayload.Analysis.RecordCount == 0 {
		t.Fatalf("expected parse record count")
	}

	analyzeTool := workflowtools.NewWorkflowResultsAnalyzeTool(client, nil)
	analyzeResult, errObj := analyzeTool.Handler(context.Background(), args)
	if errObj != nil {
		t.Fatalf("analyze results: %v", errObj)
	}
	var analyzePayload workflowtools.ResultsAnalyzeResult
	if err := json.Unmarshal([]byte(analyzeResult.Content[0].Text), &analyzePayload); err != nil {
		t.Fatalf("unmarshal analyze result: %v", err)
	}
	if len(analyzePayload.Suggestions) == 0 {
		t.Fatalf("expected analysis suggestions")
	}

	compareTool := workflowtools.NewWorkflowResultsCompareTool(client, nil)
	compareResult, errObj := compareTool.Handler(context.Background(), compareArgs)
	if errObj != nil {
		t.Fatalf("compare results: %v", errObj)
	}
	var comparePayload workflowtools.ResultsCompareResult
	if err := json.Unmarshal([]byte(compareResult.Content[0].Text), &comparePayload); err != nil {
		t.Fatalf("unmarshal compare result: %v", err)
	}
	if comparePayload.Comparison == nil {
		t.Fatalf("expected comparison summary")
	}

	guideTool := workflowtools.NewWorkflowOptimizationGuideTool(client, nil)
	guideResult, errObj := guideTool.Handler(context.Background(), args)
	if errObj != nil {
		t.Fatalf("optimization guide: %v", errObj)
	}
	var guidePayload workflowtools.OptimizationGuideResult
	if err := json.Unmarshal([]byte(guideResult.Content[0].Text), &guidePayload); err != nil {
		t.Fatalf("unmarshal guide result: %v", err)
	}
	if guidePayload.Guidance == "" {
		t.Fatalf("expected guidance")
	}

	metricsTool := workflowtools.NewWorkflowResultsMetricsExtractTool(client, nil)
	metricsResult, errObj := metricsTool.Handler(context.Background(), args)
	if errObj != nil {
		t.Fatalf("metrics extract: %v", errObj)
	}
	var metricsPayload workflowtools.ResultsMetricsExtractResult
	if err := json.Unmarshal([]byte(metricsResult.Content[0].Text), &metricsPayload); err != nil {
		t.Fatalf("unmarshal metrics result: %v", err)
	}
	if len(metricsPayload.MetricStats) == 0 {
		t.Fatalf("expected metric stats")
	}
}

func buildResultsArgs(
	format string,
	payload interface{},
	count int,
) map[string]interface{} {
	results := make([]map[string]interface{}, 0, count)
	for i := 0; i < count; i++ {
		results = append(results, map[string]interface{}{
			"format":  format,
			"payload": payload,
		})
	}
	return map[string]interface{}{
		"results": results,
	}
}

func newAnalysisIntegrationServer(t *testing.T) *httptest.Server {
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
					"latency_ms": map[string]interface{}{
						"mean": 10.0,
					},
				},
				"record_count": 2,
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
					"rationale": "Reduce variance.",
				},
			},
		}
		if compare {
			response["comparison"] = map[string]interface{}{
				"metric_stats": map[string]interface{}{
					"latency_ms": map[string]interface{}{
						"mean": 10.0,
					},
				},
				"rankings": map[string]interface{}{
					"latency_ms": []interface{}{
						[]interface{}{"run-a", 10.0},
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
