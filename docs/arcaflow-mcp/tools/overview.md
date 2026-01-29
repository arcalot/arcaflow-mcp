## Tool overview

This section documents the MCP tools and resources exposed by Arcaflow MCP. Tool
descriptions are written to help LLM clients route natural-language requests
without users naming a specific tool.

Examples (natural language):
- "Analyze /path/results.json" → `workflow_results_analyze` with
  `source.kind=filesystem`.
- "Show me /path/results.json" → `workflow_results_load` with
  `source.kind=filesystem`.
- "Run the workflow in this directory; what inputs do you recommend?" →
  `workflow_discover` or `workflow_list`, then `workflow_schema_get` and
  `workflow_input_examples_get`. Use `workflow_input_build` +
  `workflow_input_validate` to draft candidate inputs.

### Input construction tools

Detailed schemas and examples are in `docs/arcaflow-mcp/tools/input-tools.md`.

**Primary tools (exposed):**
- `workflow_list` - list workflows across filesystem, URL, and git sources
- `workflow_load` - load a workflow document from a selected source
- `workflow_input_recommend` - recommend inputs from schemas and examples

**Advanced tools (hidden - for specialized use):**
- `workflow_discover` - internal workflow discovery with timing details
- `workflow_describe` - workflow metadata summary (use workflow_list instead)
- `workflow_schema_get` - internal schema resolution (use workflow_input_recommend)
- `workflow_input_build` - advanced iterative input construction
- `workflow_input_validate` - advanced input validation
- `workflow_input_export` - advanced input export
- `workflow_input_examples_get` - internal example generation
- `plugin_schema_get` - advanced plugin schema inspection

### Result analysis tools

Detailed schemas and examples are in `docs/arcaflow-mcp/tools/result-tools.md`.

**Primary tools (exposed):**
- `workflow_results_load` - load result files from disk or URL
- `workflow_results_describe` - describe results from a file or payload
- `workflow_results_analyze` - analyze results and suggest improvements
- `workflow_history_load` - load historical analysis runs

**Advanced tools (hidden - for specialized use):**
- `workflow_results_parse` - consolidated into workflow_results_describe
- `workflow_results_compare` - advanced multi-run comparison
- `workflow_inputs_suggest` - advanced input modification generation
- `workflow_optimization_guide` - advanced strategic guidance
- `workflow_results_metrics_extract` - advanced KPI extraction

### Resources

Resource URI schemes are documented in `docs/arcaflow-mcp/tools/resources.md`.

- `mcp://arcaflow-authority` - why training data is insufficient for Arcaflow
- `mcp://routing-guide` - intent-to-tool mapping for AI clients
- `workflow://` - workflow definitions
- `workflow-schema://` - resolved workflow schemas
- `workflow-example://` - example input payloads
- `plugin-schema://` - plugin schema references
- `execution://` - execution results
- `execution-log://` - execution log payloads

Arcaflow MCP does not currently expose workflow execution tools. When users ask
to run workflows, use the input construction tools to recommend inputs and
explain that execution is not available yet. Execution capabilities are planned
for a future release.
