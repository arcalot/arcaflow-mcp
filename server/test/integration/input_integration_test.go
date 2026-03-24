package integration

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/pluginschema"
	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
	"github.com/arcalot/arcaflow-mcp/server/pkg/config"
)

// engineConfig builds an engine config that works in CI and
// restricted environments by disabling container networking.
func engineConfig() *config.EngineConfig {
	return &config.EngineConfig{
		Deployer: "podman",
		DeploymentConfig: map[string]any{
			"host": map[string]any{
				"NetworkMode": "none",
			},
		},
	}
}

func TestInputWorkflowIntegration(t *testing.T) {
	t.Parallel()

	fixtures := filepath.Join("..", "..", "..", "test", "fixtures")
	workflowPath := filepath.Join(fixtures, "workflow-basic.yaml")

	loader := workflow.NewLoader()
	index, err := loader.LoadFromFilesystem(
		context.Background(), workflowPath,
	)
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

	ec := engineConfig()
	engineCfg, err := ec.BuildEngineConfig()
	if err != nil {
		t.Fatalf("build engine config: %v", err)
	}
	validator := workflow.NewInputValidator(
		workflow.WithInputValidatorConfig(engineCfg),
	)

	// Test 1: Valid input should pass
	input := []byte(`{"nickname":"integration"}`)
	result, err := validator.Validate(
		context.Background(), parsed.Workflow, input,
	)
	if err != nil {
		t.Fatalf("validate input: %v", err)
	}
	if !result.Valid {
		t.Fatalf(
			"expected input to be valid, issues: %v",
			result.Issues,
		)
	}
	if len(result.NormalizedInput) == 0 {
		t.Fatalf("expected normalized input")
	}

	// Verify normalized output is valid JSON containing input
	var normalized map[string]any
	if err := json.Unmarshal(result.NormalizedInput, &normalized); err != nil {
		t.Fatalf("unmarshal normalized: %v", err)
	}
	if normalized["nickname"] != "integration" {
		t.Fatalf(
			"expected nickname=integration, got %v",
			normalized["nickname"],
		)
	}

	// Test 2: Export should produce valid JSON
	exported, err := workflow.GenerateInputFile(
		context.Background(),
		parsed,
		input,
		workflow.ExportFormatJSON,
		workflow.WithInputValidator(validator),
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
	if len(entries) != 0 {
		t.Fatalf(
			"expected no plugin schema entries, got %d",
			len(entries),
		)
	}
}

func TestInputValidationBlocksInvalidInput(t *testing.T) {
	t.Parallel()

	fixtures := filepath.Join("..", "..", "..", "test", "fixtures")
	workflowPath := filepath.Join(fixtures, "workflow-basic.yaml")

	loader := workflow.NewLoader()
	index, err := loader.LoadFromFilesystem(
		context.Background(), workflowPath,
	)
	if err != nil {
		t.Fatalf("load workflow: %v", err)
	}

	ec := engineConfig()
	engineCfg, err := ec.BuildEngineConfig()
	if err != nil {
		t.Fatalf("build engine config: %v", err)
	}
	validator := workflow.NewInputValidator(
		workflow.WithInputValidatorConfig(engineCfg),
	)

	// Empty input should fail — missing required "nickname"
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

// NOTE: Workflows using !expr YAML tags (e.g. hello-world) cannot be loaded
// by the MCP loader's generic YAML parser. The Arcaflow engine has its own
// parser for !expr. This is a known limitation — the engine-based validator
// works once the workflow content reaches engine.Parse(), but the MCP loader's
// isWorkflowDocument check rejects !expr tags during initial loading.
