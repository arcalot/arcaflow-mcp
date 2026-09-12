package workflowtools

import (
	"context"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/analysis"
	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

func TestWorkflowSchemaGetInvalidArguments(t *testing.T) {
	loader := workflow.NewLoader()
	parser := workflow.NewParser()
	tool := NewWorkflowSchemaGetTool(loader, parser, nil)
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"bad": make(chan int),
	})
	if errObj == nil || errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestPluginSchemaGetInvalidArguments(t *testing.T) {
	loader := workflow.NewLoader()
	tool := NewPluginSchemaGetTool(loader, nil)
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"bad": make(chan int),
	})
	if errObj == nil || errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected invalid params error")
	}
}

func TestWorkflowResultsLoadUnsupportedKind(t *testing.T) {
	tool := NewWorkflowResultsLoadTool(nil)
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "invalid",
			"location": "/tmp",
		},
	})
	if errObj == nil {
		t.Fatalf("expected invalid params error")
	}
}

func TestWorkflowResultsLoadInvalidURL(t *testing.T) {
	tool := NewWorkflowResultsLoadTool(nil)
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{
		"source": map[string]interface{}{
			"kind":     "url",
			"location": "ht!tp://bad",
		},
	})
	if errObj == nil {
		t.Fatalf("expected invalid url error")
	}
}

func TestWorkflowHistoryLoadMissingClient(t *testing.T) {
	tool := NewWorkflowHistoryLoadTool(nil, nil)
	_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
	if errObj == nil || errObj.Code != protocol.ErrInvalidParams {
		t.Fatalf("expected missing client error")
	}
}

func TestWorkflowHistoryLoadCanceledContext(t *testing.T) {
	tool := NewWorkflowHistoryLoadTool(
		analysis.NewClient("http://example.com"),
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, errObj := tool.Handler(ctx, map[string]interface{}{})
	if errObj == nil {
		t.Fatalf("expected context error")
	}
}
