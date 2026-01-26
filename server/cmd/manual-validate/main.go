// Command manual-validate exports inputs for manual Arcaflow validation.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
)

func main() {
	if err := runManualValidate(os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runManualValidate(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("manual-validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	workflowPath := flags.String("workflow", "", "path to workflow YAML/JSON")
	inputPath := flags.String("input", "", "path to input JSON/YAML")
	outputPath := flags.String("output", "", "path to write exported input")
	format := flags.String("format", "json", "export format: json or yaml")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *workflowPath == "" || *inputPath == "" || *outputPath == "" {
		return fmt.Errorf("workflow, input, and output flags are required")
	}

	workflowContent, err := os.ReadFile(*workflowPath)
	if err != nil {
		return fmt.Errorf("read workflow: %w", err)
	}
	inputContent, err := os.ReadFile(*inputPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	parser := workflow.NewParser()
	parsed, err := parser.Parse(context.Background(), workflow.Workflow{
		ID:        "manual-validation",
		LocalPath: *workflowPath,
		Content:   workflowContent,
	})
	if err != nil {
		return fmt.Errorf("parse workflow: %w", err)
	}

	output, err := workflow.GenerateInputFile(
		context.Background(),
		parsed,
		inputContent,
		workflow.ExportFormat(*format),
	)
	if err != nil {
		return fmt.Errorf("generate input: %w", err)
	}

	if err := os.WriteFile(*outputPath, output.Payload, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
