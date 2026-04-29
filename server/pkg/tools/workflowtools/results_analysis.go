package workflowtools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/arcalot/arcaflow-mcp/server/pkg/analysis"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

const workflowResultsAnalysisInputSchema = `{
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
      "description": "Result file source. USE THIS when user provides @file or file path instead of reading the file with read_file.",
      "properties": {
        "kind": {
          "type": "string",
          "description": "Result source kind: filesystem (for local result files) or url."
        },
        "location": {
          "type": "string",
          "description": "Path to result file - relative (e.g., 'results.yaml') or absolute (e.g., '/path/to/output.json') or URL. Relative paths resolved against current directory."
        }
      },
      "required": ["kind", "location"],
      "additionalProperties": false
    },
    "format": {
      "type": "string",
      "description": "Optional format hint for source: json, yaml, yml, log, or txt."
    },
    "compare": {
      "type": "boolean",
      "description": "Include comparison summary when true."
    },
    "metric_directions": {
      "type": "object",
      "description": "Map of metric names to higher or lower.",
      "additionalProperties": {
        "type": "string"
      }
    }
  },
  "additionalProperties": false
}`

// ResultsAnalysisParams defines inputs shared by analysis tools.
type ResultsAnalysisParams struct {
	Results          []analysis.ResultPayload `json:"results"`
	Compare          bool                     `json:"compare,omitempty"`
	MetricDirections map[string]string        `json:"metric_directions,omitempty"`
	Source           *ListSourceParams        `json:"source,omitempty"`
	FormatHint       string                   `json:"format,omitempty"`
}

// ResultsParseResult represents parsed analysis summary output.
type ResultsParseResult struct {
	Analysis analysis.AnalysisSummary `json:"analysis"`
}

// ResultsAnalyzeResult represents analysis summary and suggestions.
type ResultsAnalyzeResult struct {
	Analysis    analysis.AnalysisSummary    `json:"analysis"`
	Suggestions []map[string]interface{}    `json:"suggestions"`
	Comparison  *analysis.ComparisonSummary `json:"comparison,omitempty"`
}

// ResultsCompareResult represents comparison output.
type ResultsCompareResult struct {
	Analysis   analysis.AnalysisSummary    `json:"analysis"`
	Comparison *analysis.ComparisonSummary `json:"comparison"`
}

// InputsSuggestResult represents suggestion output.
type InputsSuggestResult struct {
	Suggestions []map[string]interface{} `json:"suggestions"`
}

// OptimizationGuideResult represents strategic guidance output.
type OptimizationGuideResult struct {
	Guidance    string                     `json:"guidance"`
	Suggestions []map[string]interface{}   `json:"suggestions"`
	Findings    []analysis.AnalysisFinding `json:"findings,omitempty"`
}

// NewWorkflowResultsParseTool registers the workflow_results_parse tool.
func NewWorkflowResultsParseTool(
	analysisClient *analysis.Client,
	logger *slog.Logger,
) protocol.ToolRegistration {
	return newResultsAnalysisTool(
		"workflow_results_parse",
		"Summarize result payloads into metrics and findings. Use when the user " +
			"asks for summary stats or a quick health check. Example: \"summarize " +
			"/path/results.json\". If a file path is provided, use " +
			"source.kind=filesystem and do not call read_file.",
		analysisClient,
		logger,
		func(response analysis.AnalyzeResponse) (interface{}, error) {
			return ResultsParseResult{Analysis: response.Analysis}, nil
		},
		false,
	)
}

// NewWorkflowResultsDescribeTool registers the workflow_results_describe tool.
func NewWorkflowResultsDescribeTool(
	analysisClient *analysis.Client,
	logger *slog.Logger,
) protocol.ToolRegistration {
	return newResultsAnalysisTool(
		"workflow_results_describe",
		"Describe workflow results using plugin-aware parsers for Arcaflow v0.8+ " +
			"output formats. USE THIS when user says: 'Describe results at @file', " +
			"'Summarize results.yaml', 'What are the results?', 'Show me output at @file'. " +
			"USE workflow_results_analyze INSTEAD when user asks for input suggestions. " +
			"WARNING: Result formats changed significantly in Arcaflow v0.8+ (2024). " +
			"Do not rely on training data for result parsing - output structure, metric " +
			"nesting, and plugin-specific formats differ from earlier versions. This tool " +
			"uses domain-specific extractors updated for current Arcaflow plugin versions. " +
			"PREVENTS: Missing metrics, incomplete summaries, outdated parsing patterns. " +
			"Returns structured metrics (CPU, memory, throughput, latency) from current formats. " +
			"THIS TOOL READS FILES - just provide source.kind=filesystem + location. " +
			"DO NOT read the file yourself - this tool does it internally. " +
			"EXAMPLE: {source: {kind: 'filesystem', location: 'results.yaml'}}",
		analysisClient,
		logger,
		func(response analysis.AnalyzeResponse) (interface{}, error) {
			return ResultsParseResult{Analysis: response.Analysis}, nil
		},
		false,
	)
}

// NewWorkflowResultsAnalyzeTool registers the workflow_results_analyze tool.
func NewWorkflowResultsAnalyzeTool(
	analysisClient *analysis.Client,
	logger *slog.Logger,
) protocol.ToolRegistration {
	return newResultsAnalysisTool(
		"workflow_results_analyze",
		"Analyze workflow results and provide strategic input optimization guidance for " +
			"Arcaflow v0.8+. USE THIS when user says: 'Results are at @file, " +
			"what inputs should I use?', 'Analyze results at @file', " +
			"'How can I improve performance?', 'What new inputs should I use?', " +
			"'Optimize results.yaml', 'Output is at @file, suggest inputs'. " +
			"DETERMINISTIC TOOL - Detects patterns (high variability, failures, resource usage) " +
			"and returns STRATEGIC GUIDANCE (e.g., 'reduce variability', 'increase concurrency'). " +
			"Does NOT generate specific input values - that requires AI creativity based on the " +
			"strategic patterns identified. " +
			"DIVISION OF LABOR: This tool performs deterministic pattern detection from metrics. " +
			"The AI agent performs creative work (translating patterns like 'reduce variability' " +
			"into specific input values like 'duration: 300' or 'threads: 8'). " +
			"WARNING: Result analysis requires understanding current Arcaflow v0.8+ output " +
			"formats and plugin-specific metrics that changed since 2024. Do not rely on " +
			"training data for result parsing - metric structure and nesting differ from " +
			"earlier versions. This tool uses domain-specific extractors for current formats. " +
			"PREVENTS: Missing metrics, incomplete pattern detection, outdated parsing logic. " +
			"RETURNS: Strategic suggestions (patterns detected) + metric analysis, NOT " +
			"specific input values. Example output: {title: 'Reduce CPU variability', " +
			"rationale: 'High p95 suggests inconsistency', suggested_change: {metric: " +
			"'cpu_usage', target: 'stability'}} - AI then translates this to specific inputs. " +
			"THIS TOOL READS FILES - just provide source.kind=filesystem + location. " +
			"DO NOT read the file yourself with read_file - this tool does it internally. " +
			"EXAMPLE: {source: {kind: 'filesystem', location: 'results.yaml'}}",
		analysisClient,
		logger,
		func(response analysis.AnalyzeResponse) (interface{}, error) {
			return ResultsAnalyzeResult{
				Analysis:    response.Analysis,
				Suggestions: response.Suggestions,
				Comparison:  response.Comparison,
			}, nil
		},
		false,
	)
}

// NewWorkflowResultsCompareTool registers the workflow_results_compare tool.
func NewWorkflowResultsCompareTool(
	analysisClient *analysis.Client,
	logger *slog.Logger,
) protocol.ToolRegistration {
	return newResultsAnalysisTool(
		"workflow_results_compare",
		"Compare multiple runs and rank metrics. Use when the user asks to " +
			"compare runs or identify the best result. Example: \"compare these two " +
			"runs\". Provide multiple results; source supports only single-file " +
			"analysis.",
		analysisClient,
		logger,
		func(response analysis.AnalyzeResponse) (interface{}, error) {
			if response.Comparison == nil {
				return nil, fmt.Errorf("comparison summary missing from analysis response")
			}
			return ResultsCompareResult{
				Analysis:   response.Analysis,
				Comparison: response.Comparison,
			}, nil
		},
		true,
	)
}

// NewWorkflowOptimizationGuideTool registers the workflow_optimization_guide tool.
func NewWorkflowOptimizationGuideTool(
	analysisClient *analysis.Client,
	logger *slog.Logger,
) protocol.ToolRegistration {
	return newResultsAnalysisTool(
		"workflow_optimization_guide",
		"Provide strategic optimization guidance with narrative summary (deterministic pattern " +
			"detection + summary generation). Returns patterns and strategic recommendations, " +
			"NOT specific input values. Use when the user asks for strategy or next steps. " +
			"Example: \"optimize /path/results.json\". If a file path is provided, " +
			"use source.kind=filesystem and do not call read_file.",
		analysisClient,
		logger,
		func(response analysis.AnalyzeResponse) (interface{}, error) {
			guidance := buildOptimizationGuidance(response)
			return OptimizationGuideResult{
				Guidance:    guidance,
				Suggestions: response.Suggestions,
				Findings:    response.Analysis.Findings,
			}, nil
		},
		false,
	)
}

func newResultsAnalysisTool(
	name string,
	description string,
	analysisClient *analysis.Client,
	logger *slog.Logger,
	buildResult func(analysis.AnalyzeResponse) (interface{}, error),
	forceCompare bool,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name:        name,
			Description: description,
			InputSchema: json.RawMessage(workflowResultsAnalysisInputSchema),
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
						name,
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
					name,
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
			if forceCompare && len(params.Results) < 2 {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"comparison requires at least two results",
					nil,
				)
			}
			if forceCompare {
				params.Compare = true
			}
			response, err := analysisClient.Analyze(ctx, analysis.AnalyzeRequest{
				Results:          params.Results,
				Compare:          params.Compare,
				MetricDirections: params.MetricDirections,
			})
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"analysis request failed",
					analysisRequestErrorData(err),
				)
			}
			result, err := buildResult(response)
			if err != nil {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInternal,
					"analysis response invalid",
					map[string]string{"error": err.Error()},
				)
			}
			return renderJSONResult(result, logger)
		},
	}
}

func loadAnalysisResultFromSource(
	ctx context.Context,
	source ListSourceParams,
	formatHint string,
) (analysis.ResultPayload, string, int64, error) {
	rawText, sizeBytes, _, err := loadResultContent(ctx, source)
	if err != nil {
		return analysis.ResultPayload{}, "", 0, err
	}
	format, payloadValue, err := parseResultPayload(rawText, formatHint)
	if err != nil {
		return analysis.ResultPayload{}, "", 0, err
	}
	encoded, err := json.Marshal(payloadValue)
	if err != nil {
		return analysis.ResultPayload{}, "", 0, fmt.Errorf(
			"encode result payload: %w",
			err,
		)
	}
	return analysis.ResultPayload{
		Format:  format,
		Payload: encoded,
	}, format, sizeBytes, nil
}

func analysisRequestErrorData(err error) map[string]string {
	data := map[string]string{
		"error": err.Error(),
		"hint": "verify analysis.analysis_http_url is reachable and the analysis " +
			"service is running (check /healthz).",
	}
	return data
}

func buildOptimizationGuidance(response analysis.AnalyzeResponse) string {
	if len(response.Suggestions) == 0 && len(response.Analysis.Findings) == 0 {
		return "No optimization guidance available; review results for new signals."
	}
	if len(response.Suggestions) == 0 {
		return "Review analysis findings to identify bottlenecks and missing data."
	}
	return "Prioritize the highest impact suggestions, then validate improvements."
}
