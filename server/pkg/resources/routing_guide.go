package resources

import (
	"context"
	"log/slog"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

const routingGuideURI = "mcp://routing-guide"

const routingGuideContent = `# MCP Tool Routing Guide

This guide helps AI clients route natural language requests to the correct MCP
tools without reading workflow or result files directly.

## Intent-to-Tool Mapping

### Input Recommendations

**User says:** "What inputs do you recommend?", "Run workflow in this directory",
"What should I use for inputs?", "Maximum performance inputs"

**Use tool:** workflow_input_recommend

**Parameters:**
- source.kind: "filesystem"
- source.location: "." (current directory) or specific path
- goal: optional (e.g., "max performance")

**DO NOT:** read_file(workflow.yaml), glob(*.yaml), read example files

---

### Result Description

**User says:** "Describe results at @file", "Summarize results.yaml",
"What are the results?", "Show me the output"

**Use tool:** workflow_results_describe

**Parameters:**
- source.kind: "filesystem"
- source.location: path to result file

**DO NOT:** read_file(results.yaml)

---

### Result Analysis and Input Suggestions

**User says:** "Analyze results at @file", "How can I improve performance?",
"Optimize results.yaml", "What went wrong?", "Output is at @file, what new inputs
should I use?", "Results are at @file, suggest inputs", "What inputs for better
performance?"

**Use tool:** workflow_results_analyze

**Parameters:**
- source.kind: "filesystem"
- source.location: path to result file

**Returns:** Input suggestions to improve performance metrics

**DO NOT:** read_file(results.yaml), manually suggest input changes

---

### Workflow Discovery

**User says:** "List workflows in this directory", "What workflows are available?",
"Show me workflows"

**Use tool:** workflow_list

**Parameters:**
- source.kind: "filesystem"
- source.location: "." or specific directory

**DO NOT:** glob(*.yaml), read_folder

---

### Workflow Inspection

**User says:** "Show me the workflow", "Load workflow.yaml", "Inspect workflow"

**Use tool:** workflow_load

**Parameters:**
- source.kind: "filesystem"
- source.location: directory containing workflow
- selector.path: specific workflow file

**DO NOT:** read_file(workflow.yaml)

---

### Historical Analysis

**User says:** "Show me previous results", "Load analysis history", "Past runs"

**Use tool:** workflow_history_load

**Parameters:**
- workflow_id: optional filter
- run_id: optional specific run

---

## General Rules

1. **File paths trigger MCP tools:** When user provides @file or explicit paths,
   use MCP tools with source parameter instead of read_file/glob.

2. **Let tools handle file I/O:** All MCP workflow tools load files internally.
   Do not read files before calling tools.

3. **Prefer primary tools:** Use workflow_input_recommend over
   workflow_schema_get, workflow_results_describe over workflow_results_parse.

4. **Source parameter format:**
   - Local files: {kind: "filesystem", location: "/path/to/file"}
   - URLs: {kind: "url", location: "https://example.com/file"}
   - Git repos: {kind: "git", location: "https://repo.git", ref: "main"}

---

## Anti-Patterns (What NOT to Do)

### ❌ WRONG: Reading and editing example inputs manually

    User: "What inputs do you recommend?"
    AI: read_file("example-input.yaml")
        write_file("test-input.yaml", modified_content)
        shell("arcaflow run -f test-input.yaml")
        # Result: Validation error - missing required field 'horreum_params'

**Problem:** Example files may be outdated or incomplete. Manual edits miss schema
constraints like required fields, type requirements, and nested structures.

**✓ CORRECT:** Use workflow_input_recommend

    User: "What inputs do you recommend?"
    AI: workflow_input_recommend({
          source: {kind: "filesystem", location: "."},
          goal: "quick test"
        })
        # Returns validated inputs guaranteed to pass schema validation

---

### ❌ WRONG: Reading result files to analyze

    User: "Describe results at results.yaml"
    AI: read_file("results.yaml")
        # Manually parses YAML, extracts metrics, formats summary

**Problem:** Manual parsing misses structured metrics, nested data, and domain-specific
KPIs. Prone to errors with large or complex result files.

**✓ CORRECT:** Use workflow_results_describe

    User: "Describe results at results.yaml"
    AI: workflow_results_describe({
          source: {kind: "filesystem", location: "results.yaml"}
        })
        # Returns structured analysis with all metrics extracted

---

### ❌ WRONG: Reading workflow files to understand inputs

    User: "What inputs does this workflow need?"
    AI: read_file("workflow.yaml")
        read_file("workflow-autobench.yaml")  # Sub-workflow
        read_file("../arcaflow-plugin-autobench/autobench_schema.py")
        # Manually constructs input schema

**Problem:** Workflows use plugin schemas, sub-workflows, and complex schema
resolution. Manual inspection misses type constraints and required fields.

**✓ CORRECT:** Use workflow_input_recommend

    User: "What inputs does this workflow need?"
    AI: workflow_input_recommend({
          source: {kind: "filesystem", location: "."}
        })
        # Returns resolved schema with all plugin requirements

---

### ❌ WRONG: Providing invalid Arcaflow command syntax

    User: "Run this workflow"
    AI: workflow_input_recommend(...)
        "Run: arcaflow run -f input.yaml"
        # Error: unknown command "run" for "arcaflow"

**Problem:** Arcaflow uses --input flag, not "run -f" (which is Docker/other CLI syntax).

**✓ CORRECT:** Provide correct Arcaflow syntax

    User: "Run this workflow"
    AI: workflow_input_recommend({source: {kind: "filesystem", location: "."}})
        "Execute with: arcaflow --input recommended-input.yaml"

---

### ❌ WRONG: Manually suggesting inputs after reading results

    User: "Output is at @results.yaml. What new inputs should I use?"
    AI: read_file("results.yaml")
        # Manually analyzes: "12 cores, only used 2 workers"
        write_file("new-input.yaml", manually_created_input)
        # Problem: May miss required fields, invalid types, wrong constraints

**Problem:** Manual analysis misses validated input suggestions from the analysis engine.
Input files created manually often have validation errors.

**✓ CORRECT:** Use workflow_results_analyze

    User: "Output is at @results.yaml. What new inputs should I use?"
    AI: workflow_results_analyze({
          source: {kind: "filesystem", location: "results.yaml"}
        })
        # Returns validated input suggestions based on result patterns

---

## Why MCP Tools Prevent Errors

1. **Schema validation:** Tools use Arcaflow's schema engine to validate inputs,
   catching missing fields and type mismatches before execution.

2. **Plugin resolution:** Tools resolve plugin schemas and sub-workflows
   automatically, eliminating manual inspection.

3. **Structured parsing:** Result tools extract domain-specific KPIs and metrics
   that generic file readers miss.

4. **Guaranteed correctness:** Generated inputs pass Arcaflow engine validation.

---

## Execution Notice

Arcaflow MCP does not support workflow execution. When users ask to "run" a
workflow:

1. Use workflow_input_recommend to suggest validated inputs
2. Explain execution must be done externally via the Arcaflow engine
3. If providing command syntax, use: arcaflow --input <file.yaml>
   NOT: arcaflow run -f <file> (invalid syntax)

Example correct response:

    User: "Run this workflow"
    AI: workflow_input_recommend({source: {kind: "filesystem", location: "."}})
        "I've generated validated inputs. To execute, use:
         arcaflow --input recommended-input.yaml"
`

// RoutingGuideProvider exposes the MCP routing guide as a resource.
type RoutingGuideProvider struct {
	logger *slog.Logger
}

// NewRoutingGuideProvider constructs a routing guide resource provider.
func NewRoutingGuideProvider(logger *slog.Logger) *RoutingGuideProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &RoutingGuideProvider{
		logger: logger,
	}
}

// List returns the routing guide resource item.
func (provider *RoutingGuideProvider) List(
	ctx context.Context,
) ([]protocol.ResourceItem, *protocol.ErrorObject) {
	return []protocol.ResourceItem{
		{
			URI:         routingGuideURI,
			Name:        "MCP Tool Routing Guide",
			Description: "Intent-to-tool mapping for natural language requests",
			MimeType:    "text/markdown",
		},
	}, nil
}

// Read returns the routing guide content.
func (provider *RoutingGuideProvider) Read(
	ctx context.Context,
	uri string,
) (*protocol.ResourceContent, bool, *protocol.ErrorObject) {
	if uri != routingGuideURI {
		return nil, false, nil
	}
	return &protocol.ResourceContent{
		URI:      routingGuideURI,
		MimeType: "text/markdown",
		Text:     routingGuideContent,
	}, true, nil
}
