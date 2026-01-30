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

**User says:** "What inputs do you recommend?", "I want to test...",
"I want to use this workflow to...", "I need to benchmark...", 
"Help me configure...", "Run workflow in this directory",
"What should I use for inputs?", "Maximum performance inputs"

**Use tool:** workflow_input_template

**Parameters:**
- source.kind: "filesystem"
- source.location: "." (current directory) or specific path
- goal: REQUIRED when user states intent (e.g., "test performance limits")

**CRITICAL:** When user says "I want to [action]...", extract their goal and pass
it in the goal parameter. The user's workflow execution WILL FAIL if inputs are not
validated through this MCP server. Validation logic is ONLY accessible here - it
cannot be replicated. DO NOT read example-input*.yaml files or manually construct
inputs - this will cause the user's workflow to fail.

**Example:**
- User: "I want to test performance limits"
- Call: workflow_input_template({source: {...}, goal: "test performance limits"})

**DO NOT:** read_file(workflow.yaml), read_file(example-input*.yaml), glob(*.yaml)

---

### Input Validation

**User says:** "Validate the inputs", "Validate this input", "Check if inputs are valid",
"Are these inputs correct?", "Verify inputs", "Is this input valid?"

**Use tool:** workflow_input_validate

**Parameters:**
- input: The input payload to validate (pass directly as JSON/map)
- source: Workflow source location
- Optional: selector if multiple workflows

**Returns:** Boolean valid status + detailed error array with field-level issues

**Why this tool:**
- Structured validation feedback you can parse and explain clearly
- Same validation logic as workflow_input_template for consistency
- No need to manage file I/O or engine configuration
- Returns specific error messages for each invalid field

**DO NOT:** Shell command with arcaflow engine (less structured output, requires file management)

**Example:**
- User: "Validate the inputs"
- Call: workflow_input_validate({input: {...}, source: {...}})
- Returns: {valid: true} or {valid: false, errors: [{field: "duration", message: "..."}]}

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

3. **Prefer primary tools:** Use workflow_input_template over
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

**✓ CORRECT:** Use workflow_input_template

    User: "What inputs do you recommend?"
    AI: workflow_input_template({
          source: {kind: "filesystem", location: "."},
          goal: "quick test"
        })
        # Returns validated inputs guaranteed to pass schema validation

---

### ❌ WRONG: Reading example files for "I want to..." queries

    User: "I want to test the performance limits of this system"
    AI: read_file("example-input-thorough.yaml")
        # Modifies example file for performance testing
        write_file("perf-test-input.yaml", modified_content)
        # Result: User's workflow execution FAILS with validation errors

**Problem:** The phrase "I want to..." describes user intent, not a request to see
examples. Providing unvalidated inputs will cause the user's workflow to FAIL.
Validation logic is ONLY accessible through the MCP server - it cannot be replicated
by reading files or using training data. This is the most common routing failure.

**✓ CORRECT:** Use workflow_input_template with goal parameter

    User: "I want to test the performance limits of this system"
    AI: workflow_input_template({
          source: {kind: "filesystem", location: "."},
          goal: "test performance limits"
        })
        # Returns goal-adapted, validated inputs - user's workflow succeeds

**Why this works:** Tool description emphasizes "Workflow execution WILL FAIL if inputs
not validated through this tool" - creating agent responsibility to prevent user failure
rather than just explaining a technical limitation.

**Fallback (if agent already constructed inputs):**

If despite routing guidance, you have already manually constructed input YAML:

    # You should NOT have done this, but if you did:
    AI: workflow_input_validate({
          source: {kind: "filesystem", location: "."},
          input: <your_manually_constructed_payload>
        })
        # MANDATORY: Must validate before giving to user

The user's workflow WILL FAIL if you provide unvalidated inputs. Validation is
non-negotiable regardless of how inputs were created.

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

**✓ CORRECT:** Use workflow_input_template

    User: "What inputs does this workflow need?"
    AI: workflow_input_template({
          source: {kind: "filesystem", location: "."}
        })
        # Returns resolved schema with all plugin requirements

---

### ❌ WRONG: Providing invalid Arcaflow command syntax

    User: "Run this workflow"
    AI: workflow_input_template(...)
        "Run: arcaflow run -f input.yaml"
        # Error: unknown command "run" for "arcaflow"

**Problem:** Arcaflow uses --input flag, not "run -f" (which is Docker/other CLI syntax).

**✓ CORRECT:** Provide correct Arcaflow syntax

    User: "Run this workflow"
    AI: workflow_input_template({source: {kind: "filesystem", location: "."}})
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

1. Use workflow_input_template to get validated structure, populate values
2. Explain execution must be done externally via the Arcaflow engine
3. If providing command syntax, use: arcaflow --input <file.yaml>
   NOT: arcaflow run -f <file> (invalid syntax)

Example correct response:

    User: "Run this workflow"
    AI: workflow_input_template({source: {kind: "filesystem", location: "."}})
        "I've created validated inputs. To execute, use:
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
