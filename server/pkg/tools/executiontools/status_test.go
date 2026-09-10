package executiontools

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

func TestStatusRunning(t *testing.T) {
	t.Parallel()
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	exec, err := manager.Create("test-wf", cancel)
	if err != nil {
		t.Fatal(err)
	}

	tool := NewWorkflowExecutionStatusTool(
		manager, slog.Default(),
	)
	result, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"execution_id": exec.ID,
		},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %v", errObj)
	}

	var out StatusResult
	if err := json.Unmarshal(
		[]byte(result.Content[0].Text), &out,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Status != StatusRunning {
		t.Errorf("status = %q, want running", out.Status)
	}
	if out.ElapsedSeconds < 0 {
		t.Error("elapsed should be >= 0")
	}
}

func TestStatusCompleted(t *testing.T) {
	t.Parallel()
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	exec, err := manager.Create("test-wf", cancel)
	if err != nil {
		t.Fatal(err)
	}

	manager.Complete(
		exec.ID, "success",
		map[string]interface{}{"key": "value"},
		false, nil,
	)

	tool := NewWorkflowExecutionStatusTool(
		manager, slog.Default(),
	)
	result, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"execution_id": exec.ID,
		},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %v", errObj)
	}

	var out StatusResult
	if err := json.Unmarshal(
		[]byte(result.Content[0].Text), &out,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Status != StatusCompleted {
		t.Errorf(
			"status = %q, want completed", out.Status,
		)
	}
	if out.OutputID != "success" {
		t.Errorf("output_id = %q", out.OutputID)
	}
	if out.Output == nil {
		t.Error("expected output data")
	}
}

func TestStatusNotFound(t *testing.T) {
	t.Parallel()
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	tool := NewWorkflowExecutionStatusTool(
		manager, slog.Default(),
	)
	_, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"execution_id": "exec-nonexistent",
		},
	)
	if errObj == nil {
		t.Fatal("expected error for unknown execution")
	}
}

func TestStatusMissingID(t *testing.T) {
	t.Parallel()
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	tool := NewWorkflowExecutionStatusTool(
		manager, slog.Default(),
	)
	_, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{},
	)
	if errObj == nil {
		t.Fatal("expected error for missing id")
	}
}
