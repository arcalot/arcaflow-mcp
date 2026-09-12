package workflowtools

import (
	"context"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

func TestToolHandlersRequireDependencies(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		call func() *protocol.ErrorObject
	}{
		{
			name: "workflow_schema_get",
			call: func() *protocol.ErrorObject {
				tool := NewWorkflowSchemaGetTool(nil, nil, nil)
				_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
				return errObj
			},
		},
		{
			name: "workflow_input_build",
			call: func() *protocol.ErrorObject {
				tool := NewWorkflowInputBuildTool(nil, nil, nil)
				_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
				return errObj
			},
		},
		{
			name: "workflow_input_validate",
			call: func() *protocol.ErrorObject {
				tool := NewWorkflowInputValidateTool(nil, nil, nil)
				_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
				return errObj
			},
		},
		{
			name: "workflow_input_export",
			call: func() *protocol.ErrorObject {
				tool := NewWorkflowInputExportTool(nil, nil, nil, nil)
				_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
				return errObj
			},
		},
		{
			name: "workflow_input_examples_get",
			call: func() *protocol.ErrorObject {
				tool := NewWorkflowInputExamplesTool(nil, nil, nil)
				_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
				return errObj
			},
		},
		{
			name: "workflow_input_template",
			call: func() *protocol.ErrorObject {
				tool := NewWorkflowInputTemplateTool(nil, nil, nil)
				_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
				return errObj
			},
		},
		{
			name: "plugin_schema_get",
			call: func() *protocol.ErrorObject {
				tool := NewPluginSchemaGetTool(nil, nil)
				_, errObj := tool.Handler(context.Background(), map[string]interface{}{})
				return errObj
			},
		},
	}

	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			errObj := testCase.call()
			if errObj == nil {
				t.Fatalf("expected error")
			}
			if errObj.Code != protocol.ErrInternal {
				t.Fatalf("expected internal error, got %d", errObj.Code)
			}
		})
	}
}
