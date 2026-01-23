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
  "required": ["results"],
  "additionalProperties": false
}`

// ResultsAnalysisParams defines inputs shared by analysis tools.
type ResultsAnalysisParams struct {
	Results          []analysis.ResultPayload `json:"results"`
	Compare          bool                     `json:"compare,omitempty"`
	MetricDirections map[string]string        `json:"metric_directions,omitempty"`
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
		"Parse workflow results and extract summary statistics.",
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
		"Analyze results and generate input improvement suggestions.",
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
		"Compare multiple result payloads and rank metrics.",
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

// NewWorkflowInputsSuggestTool registers the workflow_inputs_suggest tool.
func NewWorkflowInputsSuggestTool(
	analysisClient *analysis.Client,
	logger *slog.Logger,
) protocol.ToolRegistration {
	return newResultsAnalysisTool(
		"workflow_inputs_suggest",
		"Generate input change suggestions from analysis.",
		analysisClient,
		logger,
		func(response analysis.AnalyzeResponse) (interface{}, error) {
			return InputsSuggestResult{Suggestions: response.Suggestions}, nil
		},
		false,
	)
}

// NewWorkflowOptimizationGuideTool registers the workflow_optimization_guide tool.
func NewWorkflowOptimizationGuideTool(
	analysisClient *analysis.Client,
	logger *slog.Logger,
) protocol.ToolRegistration {
	return newResultsAnalysisTool(
		"workflow_optimization_guide",
		"Summarize strategic optimization guidance from results.",
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
			if len(params.Results) == 0 {
				return protocol.ToolsCallResult{}, toolError(
					protocol.ErrInvalidParams,
					"results are required",
					nil,
				)
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
					map[string]string{"error": err.Error()},
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

func buildOptimizationGuidance(response analysis.AnalyzeResponse) string {
	if len(response.Suggestions) == 0 && len(response.Analysis.Findings) == 0 {
		return "No optimization guidance available; review results for new signals."
	}
	if len(response.Suggestions) == 0 {
		return "Review analysis findings to identify bottlenecks and missing data."
	}
	return "Prioritize the highest impact suggestions, then validate improvements."
}
