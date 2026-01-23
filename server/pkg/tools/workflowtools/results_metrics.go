package workflowtools

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/arcalot/arcaflow-mcp/server/pkg/analysis"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

const workflowResultsMetricsExtractInputSchema = `{
  "type": "object",
  "properties": {
    "results": {
      "type": "array",
      "description": "Result payloads to analyze.",
      "items": {
        "type": "object",
        "properties": {
          "format": {
            "type": "string",
            "description": "Result format hint: json, yaml, yml, log, or txt."
          },
          "payload": {
            "description": "Parsed result payload or raw string."
          }
        },
        "required": ["payload"],
        "additionalProperties": false
      },
      "minItems": 1
    }
  },
  "required": ["results"],
  "additionalProperties": false
}`

// ResultsMetricsExtractResult reports extracted metric statistics.
type ResultsMetricsExtractResult struct {
	MetricStats map[string]map[string]float64 `json:"metric_stats"`
	RecordCount int                           `json:"record_count"`
}

// NewWorkflowResultsMetricsExtractTool registers workflow_results_metrics_extract.
func NewWorkflowResultsMetricsExtractTool(
	analysisClient *analysis.Client,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name:        "workflow_results_metrics_extract",
			Description: "Extract metrics and summary statistics from results.",
			InputSchema: json.RawMessage(workflowResultsMetricsExtractInputSchema),
		},
		Handler: func(
			ctx context.Context,
			arguments map[string]interface{},
		) (protocol.ToolsCallResult, *protocol.ErrorObject) {
			if analysisClient == nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"analysis service not configured",
					map[string]string{
						"hint": "set analysis.analysis_http_url in configuration",
					},
				)
			}
			if ctx.Err() != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"context cancelled",
					map[string]string{"error": ctx.Err().Error()},
				)
			}
			payload, err := json.Marshal(arguments)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"invalid tool arguments",
					map[string]string{"error": err.Error()},
				)
			}
			var params ResultsAnalysisParams
			if err := json.Unmarshal(payload, &params); err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"invalid tool arguments",
					map[string]string{"error": err.Error()},
				)
			}
			if len(params.Results) == 0 {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"results are required",
					nil,
				)
			}
			response, err := analysisClient.Analyze(ctx, analysis.AnalyzeRequest{
				Results: params.Results,
			})
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"analysis request failed",
					map[string]string{"error": err.Error()},
				)
			}
			result := ResultsMetricsExtractResult{
				MetricStats: response.Analysis.MetricStats,
				RecordCount: response.Analysis.RecordCount,
			}
			return renderJSONResult(result, logger)
		},
	}
}
