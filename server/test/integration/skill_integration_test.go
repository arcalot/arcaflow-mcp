package integration

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/pluginschema"
	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
)

func TestSkill1WorkflowIntegration(t *testing.T) {
	t.Parallel()

	fixtures := filepath.Join("..", "..", "..", "test", "fixtures")
	workflowPath := filepath.Join(fixtures, "workflow-basic.yaml")

	loader := workflow.NewLoader()
	index, err := loader.LoadFromFilesystem(context.Background(), workflowPath)
	if err != nil {
		t.Fatalf("load workflow: %v", err)
	}
	if len(index.Workflows) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(index.Workflows))
	}

	parser := workflow.NewParser()
	parsed, err := parser.Parse(context.Background(), index.Workflows[0])
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	if len(parsed.InputSchema) == 0 || len(parsed.OutputSchema) == 0 {
		t.Fatalf("expected parsed schemas")
	}

	validator := workflow.NewInputValidator()
	input := []byte(`{"name":"integration"}`)
	result, err := validator.Validate(context.Background(), parsed.Workflow, input)
	if err != nil {
		t.Fatalf("validate input: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected input to be valid")
	}

	exported, err := workflow.GenerateInputFile(
		context.Background(),
		parsed,
		input,
		workflow.ExportFormatJSON,
	)
	if err != nil {
		t.Fatalf("export input: %v", err)
	}
	if !json.Valid(exported.Payload) {
		t.Fatalf("expected exported input to be valid json")
	}

	handler := pluginschema.NewHandler(nil)
	entries, err := handler.LoadFromWorkflow(
		context.Background(),
		index.Workflows[0].Content,
		index.Workflows[0].LocalPath,
	)
	if err != nil {
		t.Fatalf("load plugin schemas: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 plugin schema entry, got %d", len(entries))
	}
}

func TestSkill1ValidationBlocksInvalidInput(t *testing.T) {
	t.Parallel()

	fixtures := filepath.Join("..", "..", "..", "test", "fixtures")
	workflowPath := filepath.Join(fixtures, "workflow-basic.yaml")

	loader := workflow.NewLoader()
	index, err := loader.LoadFromFilesystem(context.Background(), workflowPath)
	if err != nil {
		t.Fatalf("load workflow: %v", err)
	}

	validator := workflow.NewInputValidator()
	result, err := validator.Validate(
		context.Background(),
		index.Workflows[0],
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("validate input: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid input to fail validation")
	}
	if len(result.Issues) == 0 {
		t.Fatalf("expected validation issues")
	}
}
