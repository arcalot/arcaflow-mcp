// Command manual-validate exports inputs for manual Arcaflow validation.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
)

func main() {
	workflowPath := flag.String("workflow", "", "path to workflow YAML/JSON")
	inputPath := flag.String("input", "", "path to input JSON/YAML")
	outputPath := flag.String("output", "", "path to write exported input")
	format := flag.String("format", "json", "export format: json or yaml")
	flag.Parse()

	if *workflowPath == "" || *inputPath == "" || *outputPath == "" {
		fmt.Fprintln(os.Stderr, "workflow, input, and output flags are required")
		os.Exit(1)
	}

	workflowContent, err := os.ReadFile(*workflowPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read workflow: %v\n", err)
		os.Exit(1)
	}
	inputContent, err := os.ReadFile(*inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		os.Exit(1)
	}

	parser := workflow.NewParser()
	parsed, err := parser.Parse(context.Background(), workflow.Workflow{
		ID:        "manual-validation",
		LocalPath: *workflowPath,
		Content:   workflowContent,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse workflow: %v\n", err)
		os.Exit(1)
	}

	output, err := workflow.GenerateInputFile(
		context.Background(),
		parsed,
		inputContent,
		workflow.ExportFormat(*format),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate input: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputPath, output.Payload, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}
}
