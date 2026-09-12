package executiontools

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestOutputRunning(t *testing.T) {
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

	// Write some log lines.
	_, _ = exec.LogBuffer.Write(
		[]byte("line 1\nline 2\nline 3\n"),
	)

	tool := NewWorkflowExecutionOutputTool(
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

	var out OutputResult
	if err := json.Unmarshal(
		[]byte(result.Content[0].Text), &out,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Lines != 3 {
		t.Errorf("lines = %d, want 3", out.Lines)
	}
	if !strings.Contains(out.Output, "line 1") {
		t.Error("expected output to contain line 1")
	}
}

func TestOutputTail(t *testing.T) {
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

	_, _ = exec.LogBuffer.Write(
		[]byte("a\nb\nc\nd\ne\n"),
	)

	tool := NewWorkflowExecutionOutputTool(
		manager, slog.Default(),
	)
	result, errObj := tool.Handler(
		context.Background(),
		map[string]interface{}{
			"execution_id": exec.ID,
			"tail_lines":   float64(2),
		},
	)
	if errObj != nil {
		t.Fatalf("unexpected error: %v", errObj)
	}

	var out OutputResult
	if err := json.Unmarshal(
		[]byte(result.Content[0].Text), &out,
	); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Lines != 2 {
		t.Errorf("lines = %d, want 2", out.Lines)
	}
	if !strings.Contains(out.Output, "d") {
		t.Error("expected last 2 lines")
	}
	if strings.Contains(out.Output, "a") {
		t.Error("should not contain early lines")
	}
}

func TestOutputNotFound(t *testing.T) {
	t.Parallel()
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	tool := NewWorkflowExecutionOutputTool(
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

func TestOutputMissingID(t *testing.T) {
	t.Parallel()
	manager := NewExecutionManager(
		5, time.Hour, slog.Default(),
	)
	tool := NewWorkflowExecutionOutputTool(
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
