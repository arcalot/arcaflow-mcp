package executiontools

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

func TestCancelToolRunning(t *testing.T) {
	t.Parallel()
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()
	exec, err := manager.Create("test-wf", cancel)
	if err != nil {
		t.Fatal(err)
	}

	tool := NewWorkflowExecutionCancelTool(
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

	var out CancelResult
	if err := json.Unmarshal(
		[]byte(result.Content[0].Text), &out,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Status != StatusCancelled {
		t.Errorf(
			"status = %q, want cancelled",
			out.Status,
		)
	}

	// Verify the context was cancelled.
	select {
	case <-ctx.Done():
		// expected
	default:
		t.Error("context should be cancelled")
	}
}

func TestCancelToolNotFound(t *testing.T) {
	t.Parallel()
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	tool := NewWorkflowExecutionCancelTool(
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

func TestCancelToolCompleted(t *testing.T) {
	t.Parallel()
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	_, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()
	exec, err := manager.Create("test-wf", cancel)
	if err != nil {
		t.Fatal(err)
	}
	manager.Complete(exec.ID, "success", nil, false, nil)

	tool := NewWorkflowExecutionCancelTool(
		manager, slog.Default(),
	)
	_, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"execution_id": exec.ID,
		},
	)
	if errObj == nil {
		t.Fatal(
			"expected error for completed execution",
		)
	}
}

func TestCancelToolMissingID(t *testing.T) {
	t.Parallel()
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	tool := NewWorkflowExecutionCancelTool(
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
