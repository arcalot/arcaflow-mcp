## Tool overview

This section documents the MCP tools and resources exposed by Arcaflow MCP. Tool
descriptions are written to help LLM clients route natural-language requests
without users naming a specific tool.

Examples (natural language):
- "Analyze /path/results.json" → `workflow_results_analyze` with
  `source.kind=filesystem`.
- "Show me /path/results.json" → `workflow_results_load` with
  `source.kind=filesystem`.

### Input construction tools

Detailed schemas and examples are in `docs/arcaflow-mcp/tools/input-tools.md`.

- `workflow_discover` - discover workflows with selection guidance and timing
- `workflow_list` - list workflows across filesystem, URL, and git sources
- `workflow_load` - load a workflow document from a selected source
- `workflow_describe` - summarize workflow metadata and steps
- `workflow_schema_get` - resolve workflow input/output schemas
- `workflow_input_build` - build or update draft inputs
- `workflow_input_validate` - validate draft inputs against schemas
- `workflow_input_export` - export validated inputs to JSON/YAML
- `workflow_input_examples_get` - retrieve example inputs
- `plugin_schema_get` - fetch plugin schemas referenced by workflow steps

### Result analysis tools

Detailed schemas and examples are in `docs/arcaflow-mcp/tools/result-tools.md`.

- `workflow_results_load` - load result files from disk or URL
- `workflow_results_parse` - parse results into summary stats
- `workflow_results_analyze` - analyze results and suggest improvements
- `workflow_results_compare` - compare runs and rank metrics
- `workflow_inputs_suggest` - generate input modifications
- `workflow_optimization_guide` - provide strategic optimization guidance
- `workflow_results_metrics_extract` - extract metric statistics
- `workflow_history_load` - load historical analysis runs

### Resources

Resource URI schemes are documented in `docs/arcaflow-mcp/tools/resources.md`.

- `workflow://` - workflow definitions
- `workflow-schema://` - resolved workflow schemas
- `workflow-example://` - example input payloads
- `plugin-schema://` - plugin schema references
- `execution://` - execution results
- `execution-log://` - execution log payloads

Arcaflow MCP does not currently expose workflow execution tools. Execution
capabilities are planned for a future release.
