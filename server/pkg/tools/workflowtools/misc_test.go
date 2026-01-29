package workflowtools

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

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
	_, err := loadDetails(context.Background(), loader, ListSourceParams{
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

func TestLoadErrorDataIncludesTimeoutGuidance(t *testing.T) {
	t.Parallel()

	err := loadTimeoutError{
		timeout: 5 * time.Second,
		err:     context.DeadlineExceeded,
	}
	data := loadErrorData(err)
	if data["timeout_seconds"] != 5 {
		t.Fatalf("expected timeout_seconds to be 5")
	}
	if guidance, ok := data["retry_guidance"].(string); !ok || guidance == "" {
		t.Fatalf("expected retry guidance")
	}
}

func TestWorkflowLoadFailureIncludesDetails(t *testing.T) {
	t.Parallel()

	loader := workflow.NewLoader()
	tool := NewWorkflowListTool(loader, slog.Default())
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "unknown",
			"location": "/tmp",
		},
	})
	if errObj == nil {
		t.Fatalf("expected load error")
	}
	if !strings.Contains(errObj.Message, "workflow source load failed:") {
		t.Fatalf("expected load failure prefix in message")
	}
	if !strings.Contains(errObj.Message, "unsupported source kind") {
		t.Fatalf("expected load failure detail in message")
	}
}

func TestSelectWorkflowAutoSelectsWorkflowYaml(t *testing.T) {
	t.Parallel()

	workflows := []workflow.Workflow{
		{ID: "id-1", Path: "workflow.yaml", Name: "primary"},
		{ID: "id-2", Path: "extra.yaml", Name: "extra"},
	}
	selected, err := selectWorkflow(workflows, LoadSelectorParams{})
	if err != nil {
		t.Fatalf("expected auto-select, got %v", err)
	}
	if selected.Path != "workflow.yaml" {
		t.Fatalf("expected workflow.yaml, got %s", selected.Path)
	}
}

func TestSelectWorkflowPrefersShallowWorkflowYaml(t *testing.T) {
	t.Parallel()

	workflows := []workflow.Workflow{
		{ID: "id-1", Path: "archive/workflow.yaml", Name: "archived"},
		{ID: "id-2", Path: "workflow.yaml", Name: "primary"},
	}
	selected, err := selectWorkflow(workflows, LoadSelectorParams{})
	if err != nil {
		t.Fatalf("expected auto-select, got %v", err)
	}
	if selected.Path != "workflow.yaml" {
		t.Fatalf("expected workflow.yaml, got %s", selected.Path)
	}
}

func TestSelectWorkflowAutoSelectsParent(t *testing.T) {
	t.Parallel()

	parent := workflow.Workflow{
		ID:   "parent",
		Path: "parent.yaml",
		Content: []byte(
			"version: v0.2.0\n" +
				"steps:\n" +
				"  child:\n" +
				"    workflow: child.yaml\n",
		),
	}
	child := workflow.Workflow{
		ID:      "child",
		Path:    "child.yaml",
		Content: []byte("version: v0.2.0\nsteps: {}\n"),
	}
	selected, err := selectWorkflow([]workflow.Workflow{child, parent}, LoadSelectorParams{})
	if err != nil {
		t.Fatalf("expected auto-select, got %v", err)
	}
	if selected.Path != "parent.yaml" {
		t.Fatalf("expected parent.yaml, got %s", selected.Path)
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
