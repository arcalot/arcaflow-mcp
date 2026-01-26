package workflowtools

import (
	"context"
	"log/slog"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/analysis"
	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
)

func TestRenderJSONResultError(t *testing.T) {
	t.Parallel()

	_, errObj := renderJSONResult(map[string]interface{}{
		"bad": make(chan int),
	}, slog.Default())
	if errObj == nil {
		t.Fatalf("expected render error")
	}
}

func TestLoadIndexRejectsUnknownKind(t *testing.T) {
	t.Parallel()

	loader := workflow.NewLoader()
	_, err := loadIndex(context.Background(), loader, ListSourceParams{
		Kind:     "unknown",
		Location: "/tmp",
	})
	if err == nil {
		t.Fatalf("expected unknown kind error")
	}
}

func TestSelectWorkflowErrors(t *testing.T) {
	t.Parallel()

	workflowItem := workflow.Workflow{ID: "id-1", Path: "path.yaml", Name: "name"}

	_, err := selectWorkflow([]workflow.Workflow{workflowItem}, LoadSelectorParams{
		ID: "missing",
	})
	if err == nil {
		t.Fatalf("expected missing id error")
	}

	_, err = selectWorkflow([]workflow.Workflow{workflowItem}, LoadSelectorParams{
		Path: "missing.yaml",
	})
	if err == nil {
		t.Fatalf("expected missing path error")
	}
}

func TestBuildOptimizationGuidancePaths(t *testing.T) {
	t.Parallel()

	empty := buildOptimizationGuidance(analysis.AnalyzeResponse{
		Analysis: analysis.AnalysisSummary{},
	})
	if empty == "" {
		t.Fatalf("expected guidance")
	}

	withFindings := buildOptimizationGuidance(analysis.AnalyzeResponse{
		Analysis: analysis.AnalysisSummary{
			Findings: []analysis.AnalysisFinding{
				{Severity: "info", Message: "ok"},
			},
		},
	})
	if withFindings == "" || withFindings == empty {
		t.Fatalf("expected findings guidance")
	}
}
