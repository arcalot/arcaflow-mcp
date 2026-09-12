## Tool overview

This section documents the MCP tools and resources exposed by Arcaflow MCP. Tool
descriptions are written to help LLM clients route natural-language requests
without users naming a specific tool.

Examples (natural language):
- "Analyze /path/results.json" → `workflow_results_analyze` with
  `source.kind=filesystem`.
- "Show me /path/results.json" → `workflow_results_load` with
  `source.kind=filesystem`.
- "What Arcaflow plugins are available?" → `plugin_list`
- "Show me the schema for the fio plugin" → `plugin_describe`
  with `plugin=arcaflow-plugin-fio`
- "Run the workflow in this directory; what inputs do you recommend?" →
  `workflow_discover` or `workflow_list`, then `workflow_schema_get` and
  `workflow_input_examples_get`. Use `workflow_input_build` +
  `workflow_input_validate` to draft candidate inputs.

### Input construction tools

Detailed schemas and examples are in `docs/arcaflow-mcp/tools/input-tools.md`.

**Primary tools (registered and available to AI agents):**
- `workflow_list` - list workflows across filesystem, URL, and git sources
- `workflow_load` - load a workflow document from a selected source
- `workflow_input_template` - get validated input structure/template from schemas
- `workflow_input_validate` - validate inputs against workflow schema (fallback safety net)
- `workflow_input_export` - export validated inputs to JSON or YAML files

**Hidden tools (code exists but NOT registered):**

These tools exist in the codebase but are intentionally not exposed to AI agents to avoid tool selection confusion and maintain a focused, high-quality tool surface area. Reasons for keeping them hidden:

- **Internal dependencies**: Some tools provide shared types or helper functions used by registered tools (e.g., `workflow_discover` provides `DiscoveryResult` and `loadDetails` types)
- **Redundancy**: Tool capabilities fully covered by simpler registered tools (e.g., `workflow_describe` duplicates `workflow_list` + `workflow_load`)
- **Deferred features**: Advanced capabilities not yet needed by typical workflows (e.g., `workflow_input_build` for multi-step iterative construction)
- **Too specialized**: Edge case functionality better handled through other means (e.g., `plugin_schema_get` for deep schema inspection)

Hidden tools list:
- `workflow_discover` - shared types/functions (DiscoveryResult, loadDetails). Use workflow_list instead.
- `workflow_describe` - redundant with workflow_list + workflow_load
- `workflow_schema_get` - internal schema resolution used by workflow_input_template
- `workflow_input_build` - advanced iterative construction (deferred for future)
- `workflow_input_examples_get` - internal example generation used by workflow_input_template
- `plugin_schema_get` - plugin-level schema inspection (too specialized)

### Result analysis tools

Detailed schemas and examples are in `docs/arcaflow-mcp/tools/result-tools.md`.

**Primary tools (registered and available to AI agents):**
- `workflow_results_load` - load result files from disk or URL
- `workflow_results_describe` - describe results from a file or payload
- `workflow_results_analyze` - analyze results and suggest strategic optimization patterns
- `workflow_results_compare` - compare multiple runs and rank metrics
- `workflow_history_load` - load historical analysis runs

**Hidden tools (code exists but NOT registered):**

See input tools section above for explanation of why tools are kept hidden. Result analysis hidden tools:
- `workflow_results_parse` - consolidated into workflow_results_describe
- `workflow_optimization_guide` - specialized narrative output. Use workflow_results_analyze instead.
- `workflow_results_metrics_extract` - KPI-only extraction. Use workflow_results_describe instead.

### Plugin discovery tools

Detailed schemas and examples are in
`docs/arcaflow-mcp/tools/plugin-tools.md`.

**Primary tools (registered and available to AI agents):**
- `plugin_list` - list available Arcaflow plugins with metadata,
  keywords, and categories
- `plugin_describe` - get detailed plugin information including
  step schemas

**Usage pattern:**
1. Call `plugin_list` to browse the catalog (optionally filter by
   category or architecture)
2. Call `plugin_describe` with a specific plugin name to get full
   step schemas

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
