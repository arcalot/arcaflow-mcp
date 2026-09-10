package executiontools

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// --- ExecutionManager tests ---

func TestNewExecutionManager(t *testing.T) {
	t.Parallel()
	m := NewExecutionManager(0, 0, nil)
	if m.maxConcurrent != defaultMaxConcurrent {
		t.Errorf(
			"expected default maxConcurrent %d, got %d",
			defaultMaxConcurrent, m.maxConcurrent,
		)
	}
	if m.completedTTL != defaultCompletedTTL {
		t.Errorf(
			"expected default TTL %v, got %v",
			defaultCompletedTTL, m.completedTTL,
		)
	}
	if m.logger == nil {
		t.Error("expected non-nil logger")
	}
}

func TestCreateExecution(t *testing.T) {
	t.Parallel()
	m := NewExecutionManager(5, time.Hour, nil)

	exec, err := m.Create("test-workflow", func() {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != StatusRunning {
		t.Errorf(
			"expected status %s, got %s",
			StatusRunning, exec.Status,
		)
	}
	if exec.WorkflowName != "test-workflow" {
		t.Errorf(
			"expected workflow name test-workflow, got %s",
			exec.WorkflowName,
		)
	}
	if exec.ID == "" {
		t.Error("expected non-empty ID")
	}
	if exec.StartedAt.IsZero() {
		t.Error("expected non-zero StartedAt")
	}
	if exec.LogBuffer == nil {
		t.Error("expected non-nil LogBuffer")
	}
}

func TestCreateConcurrencyLimit(t *testing.T) {
	t.Parallel()
	m := NewExecutionManager(2, time.Hour, nil)

	_, err1 := m.Create("wf1", func() {})
	if err1 != nil {
		t.Fatalf("unexpected error: %v", err1)
	}
	_, err2 := m.Create("wf2", func() {})
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}

	_, err3 := m.Create("wf3", func() {})
	if err3 == nil {
		t.Fatal("expected error for exceeding limit")
	}
}

func TestGetExecution(t *testing.T) {
	t.Parallel()
	m := NewExecutionManager(5, time.Hour, nil)

	exec, _ := m.Create("wf", func() {})
	got := m.Get(exec.ID)
	if got == nil {
		t.Fatal("expected execution, got nil")
	}
	if got.ID != exec.ID {
		t.Errorf("expected ID %s, got %s", exec.ID, got.ID)
	}
}

func TestGetUnknown(t *testing.T) {
	t.Parallel()
	m := NewExecutionManager(5, time.Hour, nil)

	got := m.Get("nonexistent")
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestCompleteExecution(t *testing.T) {
	t.Parallel()
	m := NewExecutionManager(5, time.Hour, nil)

	exec, _ := m.Create("wf", func() {})
	m.Complete(exec.ID, "success", map[string]string{
		"key": "value",
	}, false, nil)

	got := m.Get(exec.ID)
	if got.Status != StatusCompleted {
		t.Errorf(
			"expected status %s, got %s",
			StatusCompleted, got.Status,
		)
	}
	if got.OutputID != "success" {
		t.Errorf(
			"expected outputID success, got %s",
			got.OutputID,
		)
	}
	if got.CompletedAt.IsZero() {
		t.Error("expected non-zero CompletedAt")
	}
	if got.Error != nil {
		t.Errorf("expected nil error, got %v", got.Error)
	}
}

func TestCompleteWithError(t *testing.T) {
	t.Parallel()
	m := NewExecutionManager(5, time.Hour, nil)

	exec, _ := m.Create("wf", func() {})
	testErr := errors.New("engine failure")
	m.Complete(exec.ID, "", nil, true, testErr)

	got := m.Get(exec.ID)
	if got.Status != StatusFailed {
		t.Errorf(
			"expected status %s, got %s",
			StatusFailed, got.Status,
		)
	}
	if got.Error != testErr {
		t.Errorf(
			"expected error %v, got %v",
			testErr, got.Error,
		)
	}
	if !got.OutputError {
		t.Error("expected OutputError to be true")
	}
}

func TestCancelExecution(t *testing.T) {
	t.Parallel()
	m := NewExecutionManager(5, time.Hour, nil)

	cancelled := false
	exec, _ := m.Create("wf", func() {
		cancelled = true
	})

	err := m.Cancel(exec.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := m.Get(exec.ID)
	if got.Status != StatusCancelled {
		t.Errorf(
			"expected status %s, got %s",
			StatusCancelled, got.Status,
		)
	}
	if !cancelled {
		t.Error("expected cancel func to be called")
	}
	if got.CompletedAt.IsZero() {
		t.Error("expected non-zero CompletedAt")
	}
}

func TestCancelUnknown(t *testing.T) {
	t.Parallel()
	m := NewExecutionManager(5, time.Hour, nil)

	err := m.Cancel("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown ID")
	}
}

func TestCancelCompleted(t *testing.T) {
	t.Parallel()
	m := NewExecutionManager(5, time.Hour, nil)

	exec, _ := m.Create("wf", func() {})
	m.Complete(exec.ID, "ok", nil, false, nil)

	err := m.Cancel(exec.ID)
	if err == nil {
		t.Fatal(
			"expected error cancelling completed exec",
		)
	}
}

func TestCleanup(t *testing.T) {
	t.Parallel()
	// Use a very short TTL so we can trigger cleanup.
	m := NewExecutionManager(
		5, 1*time.Millisecond, nil,
	)

	exec, _ := m.Create("wf", func() {})
	m.Complete(exec.ID, "ok", nil, false, nil)

	// Ensure enough time passes for TTL expiry.
	time.Sleep(5 * time.Millisecond)
	m.Cleanup()

	got := m.Get(exec.ID)
	if got != nil {
		t.Error("expected cleaned-up execution to be nil")
	}
}

func TestCleanupKeepsRunning(t *testing.T) {
	t.Parallel()
	m := NewExecutionManager(
		5, 1*time.Millisecond, nil,
	)

	exec, _ := m.Create("wf", func() {})

	time.Sleep(5 * time.Millisecond)
	m.Cleanup()

	got := m.Get(exec.ID)
	if got == nil {
		t.Error("running execution should not be removed")
	}
}

func TestRunningCount(t *testing.T) {
	t.Parallel()
	m := NewExecutionManager(10, time.Hour, nil)

	if m.RunningCount() != 0 {
		t.Errorf("expected 0 running, got %d",
			m.RunningCount())
	}

	_, _ = m.Create("wf1", func() {})
	_, _ = m.Create("wf2", func() {})
	if m.RunningCount() != 2 {
		t.Errorf("expected 2 running, got %d",
			m.RunningCount())
	}

	exec3, _ := m.Create("wf3", func() {})
	m.Complete(exec3.ID, "ok", nil, false, nil)
	if m.RunningCount() != 2 {
		t.Errorf("expected 2 running after complete, "+
			"got %d", m.RunningCount())
	}
}

// --- LogBuffer tests ---

func TestLogBufferWrite(t *testing.T) {
	t.Parallel()
	buf := NewLogBuffer(100)

	n, err := buf.Write([]byte("line1\nline2\nline3\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 18 {
		t.Errorf("expected 18 bytes written, got %d", n)
	}

	lines := buf.Lines()
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	expected := []string{"line1", "line2", "line3"}
	for i, want := range expected {
		if lines[i] != want {
			t.Errorf(
				"line[%d]: expected %q, got %q",
				i, want, lines[i],
			)
		}
	}
}

func TestLogBufferTail(t *testing.T) {
	t.Parallel()
	buf := NewLogBuffer(100)
	for i := 0; i < 10; i++ {
		_, _ = fmt.Fprintf(buf, "line-%d\n", i)
	}

	tail := buf.Tail(3)
	if len(tail) != 3 {
		t.Fatalf("expected 3 tail lines, got %d",
			len(tail))
	}
	expected := []string{"line-7", "line-8", "line-9"}
	for i, want := range expected {
		if tail[i] != want {
			t.Errorf(
				"tail[%d]: expected %q, got %q",
				i, want, tail[i],
			)
		}
	}

	// Tail larger than buffer returns all.
	all := buf.Tail(100)
	if len(all) != 10 {
		t.Errorf("expected 10, got %d", len(all))
	}

	// Tail(0) returns nil.
	empty := buf.Tail(0)
	if empty != nil {
		t.Errorf("expected nil, got %v", empty)
	}
}

func TestLogBufferMaxLines(t *testing.T) {
	t.Parallel()
	buf := NewLogBuffer(5)

	for i := 0; i < 10; i++ {
		_, _ = fmt.Fprintf(buf, "line-%d\n", i)
	}

	if buf.Len() != 5 {
		t.Errorf("expected 5 lines, got %d", buf.Len())
	}

	lines := buf.Lines()
	// Should retain lines 5-9 (oldest dropped).
	if lines[0] != "line-5" {
		t.Errorf(
			"expected oldest line to be line-5, got %s",
			lines[0],
		)
	}
	if lines[4] != "line-9" {
		t.Errorf(
			"expected newest line to be line-9, got %s",
			lines[4],
		)
	}
}

func TestLogBufferConcurrent(t *testing.T) {
	t.Parallel()
	buf := NewLogBuffer(1000)

	var wg sync.WaitGroup
	writers := 10
	linesPerWriter := 100

	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < linesPerWriter; i++ {
				_, _ = fmt.Fprintf(
					buf, "w%d-line-%d\n", id, i,
				)
			}
		}(w)
	}

	// Concurrent readers.
	for r := 0; r < 5; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				_ = buf.Lines()
				_ = buf.Tail(10)
				_ = buf.Len()
			}
		}()
	}

	wg.Wait()

	total := buf.Len()
	expected := writers * linesPerWriter
	if total != expected {
		t.Errorf(
			"expected %d lines, got %d",
			expected, total,
		)
	}
}
