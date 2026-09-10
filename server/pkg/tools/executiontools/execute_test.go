package executiontools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"go.flow.arcalot.io/engine"
	"go.flow.arcalot.io/engine/loadfile"
)

// mockEngine implements engine.WorkflowEngine for tests.
type mockEngine struct {
	runDelay   time.Duration
	outputID   string
	outputData interface{}
	outputErr  bool
	err        error
}

func (m *mockEngine) RunWorkflow(
	ctx context.Context,
	_ []byte,
	_ loadfile.FileCache,
	_ string,
) (string, interface{}, bool, error) {
	if m.runDelay > 0 {
		select {
		case <-time.After(m.runDelay):
		case <-ctx.Done():
			return "", nil, true, ctx.Err()
		}
	}
	return m.outputID, m.outputData, m.outputErr, m.err
}

func (m *mockEngine) Parse(
	_ loadfile.FileCache,
	_ string,
) (engine.Workflow, error) {
	return nil, fmt.Errorf("not implemented")
}

// mockEngineFactory returns a pre-configured mock.
type mockEngineFactory struct {
	engine *mockEngine
	err    error
}

func (f *mockEngineFactory) Create(
	_ map[string]interface{},
) (engine.WorkflowEngine, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.engine, nil
}

// writeWorkflow writes a minimal workflow YAML to a
// temp directory and returns the directory path.
func writeWorkflow(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	content := []byte(`version: v0.2.0
input:
  root: Root
  objects:
    Root:
      id: Root
      properties:
        name:
          required: true
          type:
            type_id: string
steps: {}
outputs:
  success:
    status: ok
`)
	path := filepath.Join(dir, "workflow.yaml")
	if err := os.WriteFile(
		path, content, 0o644,
	); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	return dir
}

func TestWorkflowExecuteSuccess(t *testing.T) {
	t.Parallel()
	eng := &mockEngine{
		outputID: "success",
		outputData: map[string]interface{}{
			"result": "ok",
		},
	}
	factory := &mockEngineFactory{engine: eng}
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	dir := writeWorkflow(t)

	tool := NewWorkflowExecuteTool(
		workflow.NewLoader(), manager,
		factory, slog.Default(),
	)

	result, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"source": map[string]interface{}{
				"kind":     "filesystem",
				"location": dir,
			},
			"deployer_config": map[string]interface{}{
				"deployers": map[string]interface{}{
					"image": map[string]interface{}{
						"deployer_name": "podman",
					},
				},
			},
			"input": map[string]interface{}{
				"name": "test",
			},
		},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %v", errObj)
	}

	var out ExecuteResult
	if err := json.Unmarshal(
		[]byte(result.Content[0].Text), &out,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Status != StatusRunning {
		t.Errorf("status = %q, want running", out.Status)
	}
	if out.ExecutionID == "" {
		t.Error("expected execution_id")
	}

	// Wait for async execution to complete.
	time.Sleep(200 * time.Millisecond)

	exec := manager.Get(out.ExecutionID)
	if exec == nil {
		t.Fatal("execution not found")
	}
	if exec.Status != StatusCompleted {
		t.Errorf(
			"status = %q, want completed",
			exec.Status,
		)
	}
	if exec.OutputID != "success" {
		t.Errorf(
			"output_id = %q, want success",
			exec.OutputID,
		)
	}
}

func TestWorkflowExecuteMissingSource(t *testing.T) {
	t.Parallel()
	factory := &mockEngineFactory{
		engine: &mockEngine{},
	}
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	tool := NewWorkflowExecuteTool(
		workflow.NewLoader(), manager, factory, nil,
	)

	_, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"deployer_config": map[string]interface{}{},
		},
	)
	if errObj == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestWorkflowExecuteMissingDeployer(t *testing.T) {
	t.Parallel()
	factory := &mockEngineFactory{
		engine: &mockEngine{},
	}
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	tool := NewWorkflowExecuteTool(
		workflow.NewLoader(), manager, factory, nil,
	)

	_, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"source": map[string]interface{}{
				"kind":     "filesystem",
				"location": "/tmp",
			},
		},
	)
	if errObj == nil {
		t.Fatal("expected error for missing deployer")
	}
}

func TestWorkflowExecuteEngineCreateError(t *testing.T) {
	t.Parallel()
	factory := &mockEngineFactory{
		err: fmt.Errorf("bad deployer config"),
	}
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	dir := writeWorkflow(t)

	tool := NewWorkflowExecuteTool(
		workflow.NewLoader(), manager, factory, nil,
	)

	_, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"source": map[string]interface{}{
				"kind":     "filesystem",
				"location": dir,
			},
			"deployer_config": map[string]interface{}{},
		},
	)
	if errObj == nil {
		t.Fatal("expected error for engine failure")
	}
}

func TestWorkflowExecuteConcurrencyLimit(t *testing.T) {
	t.Parallel()
	eng := &mockEngine{runDelay: 5 * time.Second}
	factory := &mockEngineFactory{engine: eng}
	manager := NewExecutionManager(
		1, time.Hour, slog.Default(),
	)
	dir := writeWorkflow(t)

	tool := NewWorkflowExecuteTool(
		workflow.NewLoader(), manager, factory, nil,
	)

	args := map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "filesystem",
			"location": dir,
		},
		"deployer_config": map[string]interface{}{},
	}

	// First execution — should succeed.
	_, errObj := tool.Handler(
		context.Background(), args,
	)
	if errObj != nil {
		t.Fatalf("first exec: %v", errObj)
	}

	// Second — should hit the limit.
	_, errObj = tool.Handler(
		context.Background(), args,
	)
	if errObj == nil {
		t.Fatal("expected concurrency limit error")
	}
}

func TestWorkflowExecuteTimeout(t *testing.T) {
	t.Parallel()
	eng := &mockEngine{runDelay: 10 * time.Second}
	factory := &mockEngineFactory{engine: eng}
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	dir := writeWorkflow(t)

	tool := NewWorkflowExecuteTool(
		workflow.NewLoader(), manager, factory, nil,
	)

	result, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"source": map[string]interface{}{
				"kind":     "filesystem",
				"location": dir,
			},
			"deployer_config": map[string]interface{}{},
			"timeout_seconds": float64(1),
		},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %v", errObj)
	}

	var out ExecuteResult
	if err := json.Unmarshal(
		[]byte(result.Content[0].Text), &out,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Wait for timeout to trigger.
	time.Sleep(2 * time.Second)

	exec := manager.Get(out.ExecutionID)
	if exec == nil {
		t.Fatal("execution not found")
	}
	if exec.Status != StatusFailed {
		t.Errorf(
			"status = %q, want failed",
			exec.Status,
		)
	}
}

func TestWorkflowExecuteEngineFailure(t *testing.T) {
	t.Parallel()
	eng := &mockEngine{
		err: fmt.Errorf("plugin connection failed"),
	}
	factory := &mockEngineFactory{engine: eng}
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	dir := writeWorkflow(t)

	tool := NewWorkflowExecuteTool(
		workflow.NewLoader(), manager, factory, nil,
	)

	result, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"source": map[string]interface{}{
				"kind":     "filesystem",
				"location": dir,
			},
			"deployer_config": map[string]interface{}{},
		},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %v", errObj)
	}

	var out ExecuteResult
	if err := json.Unmarshal(
		[]byte(result.Content[0].Text), &out,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Wait for async execution.
	time.Sleep(200 * time.Millisecond)

	exec := manager.Get(out.ExecutionID)
	if exec == nil {
		t.Fatal("execution not found")
	}
	if exec.Status != StatusFailed {
		t.Errorf(
			"status = %q, want failed",
			exec.Status,
		)
	}
}

func TestSelectWorkflow(t *testing.T) {
	t.Parallel()
	wfs := []workflow.Workflow{
		{Path: "a.yaml", Name: "a"},
		{Path: "b.yaml", Name: "b"},
	}

	// Select by path.
	wf, err := selectWorkflow(wfs, "b.yaml")
	if err != nil {
		t.Fatalf("select b: %v", err)
	}
	if wf.Name != "b" {
		t.Errorf("name = %q, want b", wf.Name)
	}

	// No path, multiple → error.
	_, err = selectWorkflow(wfs, "")
	if err == nil {
		t.Fatal("expected error for ambiguous")
	}

	// Single, no path → auto-select.
	wf, err = selectWorkflow(wfs[:1], "")
	if err != nil {
		t.Fatalf("auto-select: %v", err)
	}
	if wf.Name != "a" {
		t.Errorf("name = %q, want a", wf.Name)
	}

	// Empty.
	_, err = selectWorkflow(nil, "")
	if err == nil {
		t.Fatal("expected error for empty")
	}

	// Missing path.
	_, err = selectWorkflow(wfs, "missing.yaml")
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}
