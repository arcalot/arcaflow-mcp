// Package executiontools provides state management for
// Arcaflow workflow executions, including concurrency
// limits, TTL-based cleanup, and log capture.
package executiontools

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Status represents the lifecycle state of a workflow
// execution.
type Status string

const (
	// StatusRunning indicates the execution is in
	// progress.
	StatusRunning Status = "running"
	// StatusCompleted indicates the execution finished
	// successfully.
	StatusCompleted Status = "completed"
	// StatusFailed indicates the execution finished
	// with an error.
	StatusFailed Status = "failed"
	// StatusCancelled indicates the execution was
	// cancelled by the user.
	StatusCancelled Status = "cancelled"
)

const (
	defaultMaxConcurrent = 5
	defaultCompletedTTL  = 1 * time.Hour
	defaultLogMaxLines   = 1000
)

// Execution holds the state of a single workflow
// execution, including its lifecycle status, engine
// results, and captured log output.
type Execution struct {
	// ID is the unique identifier for the execution.
	ID string
	// Status is the current lifecycle state.
	Status Status
	// WorkflowName is the name of the workflow being
	// executed.
	WorkflowName string
	// StartedAt is when the execution began.
	StartedAt time.Time
	// CompletedAt is when the execution finished
	// (zero value if still running).
	CompletedAt time.Time

	// OutputID is the engine output ID populated on
	// completion.
	OutputID string
	// OutputData holds the engine output data populated
	// on completion.
	OutputData interface{}
	// OutputError indicates whether the engine output
	// represents an error path.
	OutputError bool
	// Error holds any error that caused execution
	// failure.
	Error error

	// Cancel is the context cancellation function for
	// this execution.
	Cancel context.CancelFunc

	// LogBuffer captures log output from the engine
	// execution.
	LogBuffer *LogBuffer
}

// LogBuffer captures log output from engine execution
// in a thread-safe ring buffer. It implements io.Writer
// so it can be used directly as a log destination.
type LogBuffer struct {
	mu    sync.RWMutex
	lines []string
	max   int
}

// NewLogBuffer creates a LogBuffer that retains at most
// maxLines lines. If maxLines is <= 0 it defaults to
// 1000.
func NewLogBuffer(maxLines int) *LogBuffer {
	if maxLines <= 0 {
		maxLines = defaultLogMaxLines
	}
	return &LogBuffer{
		lines: make([]string, 0, maxLines),
		max:   maxLines,
	}
}

// Write implements io.Writer. It splits the input on
// newlines and appends each non-empty line to the
// buffer, dropping the oldest lines when the maximum
// is exceeded.
func (b *LogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	parts := strings.Split(string(p), "\n")
	for _, line := range parts {
		if line == "" {
			continue
		}
		b.lines = append(b.lines, line)
	}
	// Trim to max by dropping oldest.
	if len(b.lines) > b.max {
		b.lines = b.lines[len(b.lines)-b.max:]
	}
	return len(p), nil
}

// Lines returns a copy of all captured log lines.
func (b *LogBuffer) Lines() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	out := make([]string, len(b.lines))
	copy(out, b.lines)
	return out
}

// Tail returns the last n captured log lines. If fewer
// than n lines exist, all lines are returned.
func (b *LogBuffer) Tail(n int) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if n <= 0 {
		return nil
	}
	start := len(b.lines) - n
	if start < 0 {
		start = 0
	}
	out := make([]string, len(b.lines)-start)
	copy(out, b.lines[start:])
	return out
}

// Len returns the number of lines currently stored.
func (b *LogBuffer) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.lines)
}

// ExecutionManager tracks running and completed workflow
// executions with concurrency limits and TTL cleanup.
type ExecutionManager struct {
	mu            sync.RWMutex
	executions    map[string]*Execution
	maxConcurrent int
	completedTTL  time.Duration
	logger        *slog.Logger
}

// NewExecutionManager creates a manager with the given
// concurrency limit and completed execution TTL.
// Zero maxConcurrent defaults to 5.
// Zero completedTTL defaults to 1 hour.
// A nil logger is replaced with slog.Default().
func NewExecutionManager(
	maxConcurrent int,
	completedTTL time.Duration,
	logger *slog.Logger,
) *ExecutionManager {
	if maxConcurrent <= 0 {
		maxConcurrent = defaultMaxConcurrent
	}
	if completedTTL <= 0 {
		completedTTL = defaultCompletedTTL
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &ExecutionManager{
		executions:    make(map[string]*Execution),
		maxConcurrent: maxConcurrent,
		completedTTL:  completedTTL,
		logger:        logger,
	}
}

// Create registers a new execution and returns it.
// It returns an error if the concurrent execution limit
// has been reached. Cleanup is called automatically
// before checking the limit.
func (m *ExecutionManager) Create(
	workflowName string,
	cancel context.CancelFunc,
) (*Execution, error) {
	m.Cleanup()

	m.mu.Lock()
	defer m.mu.Unlock()

	running := 0
	for _, e := range m.executions {
		if e.Status == StatusRunning {
			running++
		}
	}
	if running >= m.maxConcurrent {
		return nil, fmt.Errorf(
			"concurrent execution limit reached (%d)",
			m.maxConcurrent,
		)
	}

	id := generateID()
	exec := &Execution{
		ID:           id,
		Status:       StatusRunning,
		WorkflowName: workflowName,
		StartedAt:    time.Now(),
		Cancel:       cancel,
		LogBuffer:    NewLogBuffer(defaultLogMaxLines),
	}
	m.executions[id] = exec

	m.logger.Info("execution created",
		"id", id,
		"workflow", workflowName,
	)
	return exec, nil
}

// Get returns an execution by ID, or nil if not found.
func (m *ExecutionManager) Get(id string) *Execution {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.executions[id]
}

// Complete marks an execution as completed or failed.
// If err is non-nil the status is set to StatusFailed,
// otherwise StatusCompleted.
func (m *ExecutionManager) Complete(
	id string,
	outputID string,
	outputData interface{},
	outputError bool,
	err error,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	exec, ok := m.executions[id]
	if !ok {
		m.logger.Warn(
			"complete called for unknown execution",
			"id", id,
		)
		return
	}

	exec.CompletedAt = time.Now()
	exec.OutputID = outputID
	exec.OutputData = outputData
	exec.OutputError = outputError
	exec.Error = err

	if err != nil {
		exec.Status = StatusFailed
	} else {
		exec.Status = StatusCompleted
	}

	m.logger.Info("execution completed",
		"id", id,
		"status", exec.Status,
	)
}

// Cancel marks an execution as cancelled and calls
// the context cancel function. It returns an error if
// the execution is not found or is not in a running
// state.
func (m *ExecutionManager) Cancel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	exec, ok := m.executions[id]
	if !ok {
		return fmt.Errorf(
			"execution not found: %s", id,
		)
	}
	if exec.Status != StatusRunning {
		return fmt.Errorf(
			"cannot cancel execution in %s state",
			exec.Status,
		)
	}

	exec.Status = StatusCancelled
	exec.CompletedAt = time.Now()
	if exec.Cancel != nil {
		exec.Cancel()
	}

	m.logger.Info("execution cancelled", "id", id)
	return nil
}

// RunningCount returns the number of currently running
// executions.
func (m *ExecutionManager) RunningCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, e := range m.executions {
		if e.Status == StatusRunning {
			count++
		}
	}
	return count
}

// Cleanup removes completed, failed, and cancelled
// executions whose CompletedAt timestamp is older than
// the configured TTL.
func (m *ExecutionManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-m.completedTTL)
	for id, e := range m.executions {
		if e.Status == StatusRunning {
			continue
		}
		if !e.CompletedAt.IsZero() &&
			e.CompletedAt.Before(cutoff) {
			delete(m.executions, id)
			m.logger.Debug(
				"cleaned up execution", "id", id,
			)
		}
	}
}

// generateID returns a unique execution identifier in
// the format "exec-<8 hex chars>" using crypto/rand.
func generateID() string {
	b := make([]byte, 4)
	//nolint:errcheck // crypto/rand.Read never errors
	_, _ = rand.Read(b)
	return fmt.Sprintf("exec-%x", b)
}
