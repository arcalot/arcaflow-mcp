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
    },
    "source": {
      "type": "object",
      "description": "Optional results file source for large payloads or when a file path is provided. Prefer this over read_file for huge files.",
      "properties": {
        "kind": {
          "type": "string",
          "description": "Result source kind: filesystem or url (use filesystem for local paths)."
        },
        "location": {
          "type": "string",
          "description": "Filesystem path or URL for the result file (absolute paths preferred)."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
    },
    "format": {
      "type": "string",
      "description": "Optional format hint for source: json, yaml, yml, log, or txt."
    }
  },
  "anyOf": [
    {"required": ["results"]},
    {"required": ["source"]}
  ],
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
			Description: "Extract metrics and summary statistics from results. Use " +
				"when the user wants KPIs only. Example: \"KPIs from " +
				"/path/results.json\". If a file path is provided, " +
				"use source.kind=filesystem and do not call read_file.",
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
			if params.Source != nil && len(params.Results) > 0 {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"provide results or source, not both",
					map[string]string{
						"hint": "use source for large files or results for inline data",
					},
				)
			}
			if params.Source == nil && len(params.Results) == 0 {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"results or source are required",
					nil,
				)
			}
			if params.Source != nil {
				if params.Source.Kind == "" || params.Source.Location == "" {
					return protocol.ToolsCallResult{}, toolError(
						protocol.ErrInvalidParams,
						"source.kind and source.location are required",
						nil,
					)
				}
				resultPayload, format, sizeBytes, err := loadAnalysisResultFromSource(
					ctx,
					*params.Source,
					params.FormatHint,
				)
				if err != nil {
					logger.Warn(
						"analysis source load failed",
						"tool",
						"workflow_results_metrics_extract",
						"source_kind",
						params.Source.Kind,
						"source_location",
						params.Source.Location,
						"error",
						err,
					)
					return protocol.ToolsCallResult{}, toolError(
						protocol.ErrInvalidParams,
						"result load failed",
						map[string]string{"error": err.Error()},
					)
				}
				logger.Info(
					"analysis source loaded",
					"tool",
					"workflow_results_metrics_extract",
					"source_kind",
					params.Source.Kind,
					"source_location",
					params.Source.Location,
					"format",
					format,
					"size_bytes",
					sizeBytes,
				)
				params.Results = []analysis.ResultPayload{resultPayload}
			}
			response, err := analysisClient.Analyze(ctx, analysis.AnalyzeRequest{
				Results: params.Results,
			})
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"analysis request failed",
					analysisRequestErrorData(err),
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
