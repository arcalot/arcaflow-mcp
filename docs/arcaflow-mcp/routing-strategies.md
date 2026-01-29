# Comprehensive MCP Routing Strategies

This document summarizes all server-side strategies implemented to improve AI client
routing to Arcaflow MCP tools.

## Problem Statement

AI clients (Gemini, Claude, etc.) were consistently choosing generic file operations
(ReadFile, WriteFile, Glob, Shell) instead of specialized MCP tools, resulting in:

1. **Validation errors** - Missing required fields when manually editing inputs
2. **Incomplete parsing** - Missing metrics when manually reading results
3. **Invalid syntax** - Wrong Arcaflow command syntax (`arcaflow run -f` instead of `arcaflow --input`)
4. **Missed optimizations** - Manual analysis missing performance improvement opportunities

## Implemented Strategies

### 1. Tool Consolidation (Reduce Decision Space)

**Strategy:** Expose only 7 primary tools by default, hide 12 advanced tools.

**Rationale:** Fewer choices reduce decision paralysis and improve routing accuracy.

**Primary tools:**
- `workflow_list` - discover workflows
- `workflow_load` - inspect workflow content
- `workflow_input_recommend` - generate validated inputs
- `workflow_results_load` - load result files
- `workflow_results_describe` - summarize results
- `workflow_results_analyze` - analyze and suggest input improvements
- `workflow_history_load` - load historical runs

**Hidden tools:** workflow_discover, workflow_describe, workflow_schema_get,
workflow_input_examples_get, workflow_input_build, workflow_input_validate,
workflow_input_export, plugin_schema_get, workflow_results_parse,
workflow_results_compare, workflow_inputs_suggest, workflow_optimization_guide,
workflow_results_metrics_extract

### 2. Concrete User Phrase Matching

**Strategy:** Include 3-5 exact user phrases in each tool description.

**Example:**
```
workflow_results_analyze: "USE THIS when user says: 'Analyze results at @file',
'How can I improve performance?', 'Output is at @file, what new inputs should I use?',
'Results are at @file, suggest inputs'"
```

**Rationale:** Direct phrase matching creates semantic anchors for LLM routing.

### 3. Explicit Benefit Statements (PREVENTS:)

**Strategy:** State consequences of NOT using the tool.

**Example:**
```
workflow_input_recommend: "PREVENTS: Validation errors from missing required fields
or incorrect types"
```

**Rationale:** LLMs may avoid anti-patterns when consequences are explicit.

### 4. Inline Parameter Examples

**Strategy:** Show exact parameter structure in descriptions.

**Example:**
```
workflow_results_analyze: "EXAMPLE: {source: {kind: 'filesystem',
location: 'results.yaml'}}"
```

**Rationale:** Reduces uncertainty about how to call the tool.

### 5. Aggressive Negative Hints

**Strategy:** Explicitly discourage file operations.

**Example:**
```
workflow_input_recommend: "DO NOT read workflow.yaml or example files - this tool
uses the schema internally"

workflow_results_analyze: "DO NOT read the file yourself with read_file - this tool
does it internally"
```

**Rationale:** Explicit prohibition may override default file-reading patterns.

### 6. File-Handling Transparency

**Strategy:** Clarify that MCP tools handle file I/O internally.

**Example:**
```
workflow_results_analyze: "THIS TOOL READS FILES - just provide source.kind=filesystem
+ location"
```

**Rationale:** Eliminates assumption that files must be read separately.

### 7. Schema-Level Use-Case Hints

**Strategy:** Add use-case guidance to parameter descriptions.

**Example:**
```
"source": {
  "description": "Result file source. USE THIS when user provides @file or file path
  instead of reading the file with read_file."
}
```

**Rationale:** Provides routing hints at the parameter level.

### 8. Machine-Readable Routing Guide Resource

**Strategy:** Expose `mcp://routing-guide` resource with intent-to-tool mapping.

**Contents:**
- User phrase patterns for each tool
- Parameter examples
- Explicit "DO NOT" guidance
- Anti-patterns with side-by-side comparisons

**Rationale:** Clients can fetch and cache routing rules at initialization.

### 9. Anti-Pattern Documentation

**Strategy:** Document 5 concrete failure patterns with corrections.

**Anti-patterns:**
1. Reading/editing example inputs manually → validation errors
2. Reading result files manually → incomplete parsing
3. Reading workflow files manually → missed schema resolution
4. Invalid command syntax → execution failures
5. Manually suggesting inputs → validation errors, missed optimizations

**Rationale:** Explicit failure examples may train LLM pattern recognition.

### 10. Real-World Failure Case Examples

**Strategy:** Document actual routing failures from user transcripts.

**Example:**
```
User: "Output is at @mcp-test-out-1.yaml. What new inputs should I use?"
Wrong: ReadFile → manual analysis → WriteFile
Correct: workflow_results_analyze
```

**Rationale:** Concrete examples matching real queries improve routing accuracy.

### 11. Execution Syntax Guidance

**Strategy:** Include correct Arcaflow command syntax in descriptions.

**Example:**
```
workflow_input_recommend: "NOTE: MCP does not execute workflows. For execution, user
runs: arcaflow --input <file.yaml> (NOT arcaflow run -f)"
```

**Rationale:** Prevents syntax errors when AI suggests execution commands.

## Measurement Strategy

To verify routing effectiveness:

1. **Test with known-bad queries** from user transcripts
2. **Monitor tool selection** - should match expected tools
3. **Track file operation fallbacks** - should be zero for workflow/result queries
4. **Collect routing failures** - refine descriptions for failed patterns

## Expected Routing Behavior

### Query: "What inputs do you recommend?"

- ✅ Expected: `workflow_input_recommend`
- ❌ Not expected: ReadFile workflow.yaml, Glob *.yaml

### Query: "Describe results at @file.yaml"

- ✅ Expected: `workflow_results_describe`
- ❌ Not expected: ReadFile file.yaml

### Query: "Output is at @file, what new inputs should I use?"

- ✅ Expected: `workflow_results_analyze`
- ❌ Not expected: ReadFile, WriteFile, manual analysis

### Query: "List workflows in this directory"

- ✅ Expected: `workflow_list`
- ❌ Not expected: Glob *.yaml, ReadFolder

## Limitations

These strategies are **server-side only** and rely on:

1. LLMs respecting tool descriptions
2. Semantic matching between user queries and descriptions
3. No conflicting client-side routing policies

If routing issues persist, additional client-side tuning may be required (e.g., system
prompts, tool whitelisting, routing policy configuration).

## Future Improvements

1. **Tool metadata extensions** - Add priority/weight fields to tool definitions
2. **Routing policy resource** - Expose machine-readable routing rules
3. **Usage telemetry** - Track which tools are selected for which query patterns
4. **A/B testing** - Compare routing accuracy with different description strategies
5. **Client initialization hint** - Recommend fetching routing guide at startup
