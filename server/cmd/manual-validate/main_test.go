package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"os"
)

func TestRunManualValidateRequiresFlags(t *testing.T) {
	var stderr bytes.Buffer
	if err := runManualValidate([]string{}, &stderr); err == nil {
		t.Fatalf("expected missing flags error")
	}
}

func TestRunManualValidateWritesOutput(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	inputPath := filepath.Join(root, "input.json")
	outputPath := filepath.Join(root, "output.json")

	workflowContent := []byte(
		"version: v0.1\n" +
			"input:\n" +
			"  root: InputParams\n" +
			"  objects:\n" +
			"    InputParams:\n" +
			"      id: InputParams\n" +
			"      properties:\n" +
			"        nickname:\n" +
			"          required: true\n" +
			"          type:\n" +
			"            type_id: string\n" +
			"outputs:\n" +
			"  success: {}\n",
	)
	if err := os.WriteFile(workflowPath, workflowContent, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(inputPath, []byte(`{"nickname":"test"}`), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	args := []string{
		"-workflow", workflowPath,
		"-input", inputPath,
		"-output", outputPath,
		"-format", "json",
	}
	var stderr bytes.Buffer
	if err := runManualValidate(args, &stderr); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output file: %v", err)
	}
}

func TestRunManualValidateMissingWorkflowFile(t *testing.T) {
	root := t.TempDir()
	inputPath := filepath.Join(root, "input.json")
	outputPath := filepath.Join(root, "output.json")
	if err := os.WriteFile(inputPath, []byte(`{"nickname":"test"}`), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	args := []string{
		"-workflow", filepath.Join(root, "missing.yaml"),
		"-input", inputPath,
		"-output", outputPath,
	}
	var stderr bytes.Buffer
	if err := runManualValidate(args, &stderr); err == nil {
		t.Fatalf("expected read workflow error")
	}
}

func TestRunManualValidateInvalidWorkflow(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	inputPath := filepath.Join(root, "input.json")
	outputPath := filepath.Join(root, "output.json")

	if err := os.WriteFile(workflowPath, []byte("name: invalid"), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(inputPath, []byte(`{"nickname":"test"}`), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	args := []string{
		"-workflow", workflowPath,
		"-input", inputPath,
		"-output", outputPath,
	}
	var stderr bytes.Buffer
	if err := runManualValidate(args, &stderr); err == nil {
		t.Fatalf("expected parse workflow error")
	}
}

func TestRunManualValidateMissingInputFile(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	outputPath := filepath.Join(root, "output.json")

	workflowContent := []byte(
		"version: v0.1\n" +
			"input:\n" +
			"  root: InputParams\n" +
			"  objects:\n" +
			"    InputParams:\n" +
			"      id: InputParams\n" +
			"      properties:\n" +
			"        nickname:\n" +
			"          required: true\n" +
			"          type:\n" +
			"            type_id: string\n",
	)
	if err := os.WriteFile(workflowPath, workflowContent, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	args := []string{
		"-workflow", workflowPath,
		"-input", filepath.Join(root, "missing.json"),
		"-output", outputPath,
	}
	var stderr bytes.Buffer
	if err := runManualValidate(args, &stderr); err == nil {
		t.Fatalf("expected read input error")
	}
}

func TestRunManualValidateWriteOutputError(t *testing.T) {
	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.yaml")
	inputPath := filepath.Join(root, "input.json")

	workflowContent := []byte(
		"version: v0.1\n" +
			"input:\n" +
			"  root: InputParams\n" +
			"  objects:\n" +
			"    InputParams:\n" +
			"      id: InputParams\n" +
			"      properties:\n" +
			"        nickname:\n" +
			"          required: true\n" +
			"          type:\n" +
			"            type_id: string\n" +
			"outputs:\n" +
			"  success: {}\n",
	)
	if err := os.WriteFile(workflowPath, workflowContent, 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(inputPath, []byte(`{"nickname":"test"}`), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	outputPath := filepath.Join(root, "missing-dir", "output.json")
	args := []string{
		"-workflow", workflowPath,
		"-input", inputPath,
		"-output", outputPath,
	}
	var stderr bytes.Buffer
	if err := runManualValidate(args, &stderr); err == nil {
		t.Fatalf("expected write output error")
	}
}
