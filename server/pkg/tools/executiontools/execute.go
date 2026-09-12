package executiontools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
	"go.flow.arcalot.io/engine"
	engineconfig "go.flow.arcalot.io/engine/config"
	"go.flow.arcalot.io/engine/loadfile"
	"gopkg.in/yaml.v3"
)

const defaultTimeoutSeconds = 3600

const workflowExecuteInputSchema = `{
  "type": "object",
  "properties": {
    "source": {
      "type": "object",
      "description": "Workflow source (same as workflow_load).",
      "properties": {
        "kind": { "type": "string" },
        "location": { "type": "string" },
        "ref": { "type": "string" },
        "subdir": { "type": "string" }
      },
      "required": ["kind", "location"]
    },
    "selector": {
      "type": "object",
      "description": "Workflow selector within the source.",
      "properties": {
        "path": { "type": "string" }
      }
    },
    "input": {
      "type": "object",
      "description": "Workflow input as inline JSON."
    },
    "deployer_config": {
      "type": "object",
      "description": "Arcaflow deployer configuration including remote host connection details."
    },
    "timeout_seconds": {
      "type": "integer",
      "description": "Maximum execution time in seconds. Default 3600.",
      "default": 3600
    }
  },
  "required": ["source", "deployer_config"],
  "additionalProperties": false
}`

// ExecuteResult is the workflow_execute tool output.
type ExecuteResult struct {
	ExecutionID  string `json:"execution_id"`
	Status       Status `json:"status"`
	WorkflowName string `json:"workflow_name"`
	StartedAt    string `json:"started_at"`
}

// EngineFactory creates an engine from deployer config.
// This interface enables testing without real deployers.
type EngineFactory interface {
	Create(
		deployerConfig map[string]interface{},
	) (engine.WorkflowEngine, error)
}

// DefaultEngineFactory creates engines using the real
// Arcaflow engine with the provided deployer config.
type DefaultEngineFactory struct{}

// Create builds an engine config from the deployer map
// and returns a new engine instance.
func (f *DefaultEngineFactory) Create(
	deployerConfig map[string]interface{},
) (engine.WorkflowEngine, error) {
	cfg, err := engineconfig.Load(deployerConfig)
	if err != nil {
		return nil, fmt.Errorf(
			"load deployer config: %w", err,
		)
	}
	return engine.New(cfg)
}

// NewWorkflowExecuteTool registers the workflow_execute
// MCP tool. It starts workflow executions asynchronously
// using the Arcaflow engine as a library. The serverCtx
// is used as the parent context for executions so they
// are cancelled when the server shuts down.
func NewWorkflowExecuteTool(
	serverCtx context.Context,
	loader *workflow.Loader,
	manager *ExecutionManager,
	engineFactory EngineFactory,
	logger *slog.Logger,
) protocol.ToolRegistration {
	if logger == nil {
		logger = slog.Default()
	}
	return protocol.ToolRegistration{
		Definition: protocol.ToolDefinition{
			Name: "workflow_execute",
			Description: "Start a workflow execution. " +
				"Returns immediately with an " +
				"execution ID. Use " +
				"workflow_execution_status to poll " +
				"for completion. Requires source, " +
				"deployer_config, and optionally " +
				"input and timeout_seconds.",
			InputSchema: json.RawMessage(
				workflowExecuteInputSchema,
			),
		},
		Handler: handleExecute(
			serverCtx, loader, manager,
			engineFactory, logger,
		),
	}
}

func handleExecute(
	serverCtx context.Context,
	loader *workflow.Loader,
	manager *ExecutionManager,
	engineFactory EngineFactory,
	logger *slog.Logger,
) protocol.ToolHandler {
	return func(
		ctx context.Context,
		arguments map[string]interface{},
	) (protocol.ToolsCallResult, *protocol.ErrorObject) {
		// Parse source
		sourceRaw, ok := arguments["source"]
		if !ok {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInvalidParams,
					"source is required", nil,
				)
		}
		sourceMap, ok := sourceRaw.(map[string]interface{})
		if !ok {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInvalidParams,
					"source must be an object", nil,
				)
		}
		kind, _ := sourceMap["kind"].(string)
		location, _ := sourceMap["location"].(string)
		if kind == "" || location == "" {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInvalidParams,
					"source.kind and source.location "+
						"are required",
					nil,
				)
		}

		// Parse deployer config
		deployerRaw, ok := arguments["deployer_config"]
		if !ok {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInvalidParams,
					"deployer_config is required", nil,
				)
		}
		deployerConfig, ok :=
			deployerRaw.(map[string]interface{})
		if !ok {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInvalidParams,
					"deployer_config must be an object",
					nil,
				)
		}

		// Parse optional fields
		selectorPath := ""
		if sel, ok :=
			arguments["selector"].(map[string]interface{}); ok {
			selectorPath, _ = sel["path"].(string)
		}

		timeoutSec := defaultTimeoutSeconds
		if ts, ok := arguments["timeout_seconds"].(float64); ok {
			timeoutSec = int(ts)
		}

		// Load the workflow
		ref, _ := sourceMap["ref"].(string)
		subdir, _ := sourceMap["subdir"].(string)
		details, err := loader.LoadWithDetails(
			ctx,
			workflow.SourceMetadata{
				Kind:     workflow.SourceKind(kind),
				Location: location,
				Ref:      ref,
				Subdir:   subdir,
			},
		)
		if err != nil {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInvalidParams,
					fmt.Sprintf(
						"workflow load failed: %s",
						err.Error(),
					),
					nil,
				)
		}

		// Select workflow
		wf, err := selectWorkflow(
			details.Index.Workflows, selectorPath,
		)
		if err != nil {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInvalidParams,
					err.Error(), nil,
				)
		}

		// Marshal input to YAML for the engine
		var inputBytes []byte
		if inputRaw, ok := arguments["input"]; ok &&
			inputRaw != nil {
			inputBytes, err = yaml.Marshal(inputRaw)
			if err != nil {
				return protocol.ToolsCallResult{},
					toolError(
						protocol.ErrInvalidParams,
						"failed to marshal input",
						map[string]string{
							"error": err.Error(),
						},
					)
			}
		}

		// Create engine
		eng, err := engineFactory.Create(deployerConfig)
		if err != nil {
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInternal,
					fmt.Sprintf(
						"engine creation failed: %s",
						err.Error(),
					),
					nil,
				)
		}

		// Derive from server context so executions
		// stop when the server shuts down. The request
		// ctx is short-lived (tool call duration) so
		// we must not use it as the parent.
		execCtx, cancel := context.WithTimeout(
			serverCtx,
			time.Duration(timeoutSec)*time.Second,
		)

		// Register execution
		exec, err := manager.Create(wf.Name, cancel)
		if err != nil {
			cancel()
			return protocol.ToolsCallResult{},
				toolError(
					protocol.ErrInternal,
					err.Error(), nil,
				)
		}

		// Run in background goroutine
		go runExecution(
			execCtx, exec.ID, manager, eng,
			wf, inputBytes, logger,
		)

		result := ExecuteResult{
			ExecutionID:  exec.ID,
			Status:       StatusRunning,
			WorkflowName: wf.Name,
			StartedAt: exec.StartedAt.UTC().Format(
				time.RFC3339,
			),
		}

		// Do NOT log deployer_config — it contains
		// credentials.
		logger.Info("workflow execution started",
			"execution_id", exec.ID,
			"workflow", wf.Name,
			"timeout_seconds", timeoutSec,
		)

		return renderJSONResult(result, logger)
	}
}

// runExecution runs the engine workflow in a goroutine
// and stores results in the execution manager.
func runExecution(
	ctx context.Context,
	execID string,
	manager *ExecutionManager,
	eng engine.WorkflowEngine,
	wf workflow.Workflow,
	inputBytes []byte,
	logger *slog.Logger,
) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("execution panicked",
				"execution_id", execID,
				"panic", fmt.Sprintf("%v", r),
			)
			manager.Complete(
				execID, "", nil, true,
				fmt.Errorf("execution panicked: %v", r),
			)
		}
	}()

	// Build file cache from workflow content
	fileName := "workflow.yaml"
	if wf.Path != "" {
		fileName = wf.Path
	}
	fc := loadfile.NewFileCache("", map[string][]byte{
		fileName: wf.Content,
	})

	outputID, outputData, outputError, err :=
		eng.RunWorkflow(ctx, inputBytes, fc, fileName)

	// Complete is safe to call even if Cancel() already
	// set the status — it's a no-op for non-running
	// executions.
	if ctx.Err() == context.DeadlineExceeded {
		manager.Complete(
			execID, "", nil, true,
			fmt.Errorf("execution timed out"),
		)
		return
	}
	if ctx.Err() == context.Canceled {
		manager.Complete(
			execID, "", nil, true,
			fmt.Errorf("execution cancelled"),
		)
		return
	}

	manager.Complete(
		execID, outputID, outputData, outputError, err,
	)
}

// selectWorkflow picks a workflow by path or returns the
// only one available.
func selectWorkflow(
	workflows []workflow.Workflow,
	path string,
) (workflow.Workflow, error) {
	if len(workflows) == 0 {
		return workflow.Workflow{},
			fmt.Errorf("no workflows found in source")
	}
	if path != "" {
		for _, wf := range workflows {
			if wf.Path == path {
				return wf, nil
			}
		}
		return workflow.Workflow{},
			fmt.Errorf(
				"workflow not found: %s", path,
			)
	}
	if len(workflows) == 1 {
		return workflows[0], nil
	}
	return workflow.Workflow{},
		fmt.Errorf(
			"multiple workflows found, use selector "+
				"to choose one (%d available)",
			len(workflows),
		)
}
