package workflowtools

import (
	"encoding/json"
	"testing"
)

func TestToolInputSchemasAreValidJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		schema string
	}{
		{"workflow_list", workflowListInputSchema},
		{"workflow_load", workflowLoadInputSchema},
		{"workflow_schema_get", workflowSchemaGetInputSchema},
		{"workflow_describe", workflowDescribeInputSchema},
		{"workflow_input_build", workflowInputBuildInputSchema},
		{"workflow_input_validate", workflowInputValidateInputSchema},
		{"workflow_input_export", workflowInputExportInputSchema},
		{"workflow_input_examples_get", workflowInputExamplesInputSchema},
		{"workflow_results_load", workflowResultsLoadInputSchema},
		{"workflow_results_parse", workflowResultsAnalysisInputSchema},
		{"workflow_results_analyze", workflowResultsAnalysisInputSchema},
		{"workflow_results_compare", workflowResultsAnalysisInputSchema},
		{"workflow_inputs_suggest", workflowResultsAnalysisInputSchema},
		{"workflow_optimization_guide", workflowResultsAnalysisInputSchema},
		{"workflow_results_metrics_extract",
			workflowResultsMetricsExtractInputSchema,
		},
		{"workflow_history_load", workflowHistoryLoadInputSchema},
		{"plugin_schema_get", pluginSchemaGetInputSchema},
	}

	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if testCase.schema == "" {
				t.Fatalf("schema is empty")
			}
			if !json.Valid([]byte(testCase.schema)) {
				t.Fatalf("schema is not valid JSON")
			}
		})
	}
}
