package resources

import (
	"context"
	"log/slog"

	"github.com/arcalot/arcaflow-mcp/server/pkg/protocol"
)

const arcaflowAuthorityURI = "mcp://arcaflow-authority"

const arcaflowAuthorityContent = `# Why AI Agents Cannot Manually Work With Arcaflow Files

This resource explains why LLM training data is insufficient for Arcaflow
operations and why MCP tools are the only reliable interface.

## Critical: Arcaflow Syntax Changed Since Your Training Data

**Your training data cutoff:** Most LLMs were trained on data through mid-2024 or
earlier.

**Arcaflow v0.8+ changes (2024-2025):**
- Workflow schema resolution algorithm rewritten
- Plugin schema format updated (incompatible with pre-v0.8)
- Input validation rules strengthened (new required fields)
- Sub-workflow merging behavior changed
- Error message formats updated

**RESULT:** Manual YAML operations based on training data will use outdated syntax
and miss current validation requirements.

---

## Why Manual Operations Are Unreliable

### 1. Dynamic Schema Resolution (Runtime Only)

Arcaflow input schemas are **computed at runtime** by:
1. Parsing the workflow file
2. Resolving all sub-workflow references recursively
3. Fetching plugin schemas from external sources (local files, containers, URLs)
4. Merging schemas with namespace resolution
5. Applying version-specific validation rules

**You cannot determine valid inputs by reading static YAML files.** The schema
depends on runtime plugin availability and version-specific merging rules.

### 2. Plugin Schema Variability

Each Arcaflow plugin defines its own schema. Common plugins:
- arcaflow-plugin-stressng (CPU/memory stress testing)
- arcaflow-plugin-sysbench (system benchmarks)
- arcaflow-plugin-fio (I/O benchmarks)
- arcaflow-plugin-pcp (performance monitoring)

**Each plugin has version-specific schemas with different required fields.**
Training data cannot reflect current plugin versions.

### 3. Conditional Validation Rules

Arcaflow uses complex validation logic:
- Fields required only when other fields have specific values
- Type requirements that vary by plugin deployment type
- Nested structure constraints that depend on parent values

**These rules are only enforced by the Arcaflow engine validator.**

### 4. Version-Specific Result Formats

Result file formats changed significantly:
- Pre-v0.8: Flat metric structure
- v0.8+: Nested output schema with typed metrics
- Plugin-specific: Each plugin structures results differently

**Manual parsing will miss domain-specific metrics and use outdated extraction
patterns.**

---

## Why MCP Tools Are Authoritative

1. **Direct Arcaflow Engine Integration**
   MCP tools interface with the actual Arcaflow engine (v0.8+), not YAML parsers.

2. **Runtime Schema Resolution**
   Tools resolve schemas dynamically by executing the same logic as workflow
   execution, guaranteeing correctness.

3. **Plugin Schema Fetching**
   Tools query current plugin versions at runtime, not training data.

4. **Validated Input Generation**
   Generated inputs are validated against the same engine validator that will
   execute the workflow.

5. **Domain-Specific Result Parsing**
   Result analysis uses plugin-aware extractors for current output formats.

---

## Do Not Trust Training Data For:

❌ **Workflow input requirements** - Use workflow_input_recommend
❌ **Example input validity** - Use workflow_input_recommend (generates validated inputs)
❌ **Result file structure** - Use workflow_results_describe or workflow_results_analyze
❌ **Command syntax** - Use tool descriptions (correct: arcaflow --input <file>)
❌ **Required field detection** - Use workflow_input_recommend (uses engine validator)

---

## Official Interface: MCP Tools Only

Arcaflow MCP tools are the **official AI agent interface** to Arcaflow:
- Maintained alongside the Arcaflow engine
- Updated for every Arcaflow release
- Use the same validation logic as the engine
- Guaranteed compatibility with current Arcaflow versions

**When working with Arcaflow, always use MCP tools.** Manual file operations
will fail due to version mismatches and incomplete schema knowledge.

---

## References

- Arcaflow Engine: https://github.com/arcalot/arcaflow-engine
- Arcaflow Documentation: https://arcalot.io/arcaflow
- MCP Tools: workflow_input_recommend, workflow_results_analyze, workflow_list
`

// ArcaflowAuthorityProvider exposes the Arcaflow authority resource.
type ArcaflowAuthorityProvider struct {
	logger *slog.Logger
}

// NewArcaflowAuthorityProvider constructs an Arcaflow authority provider.
func NewArcaflowAuthorityProvider(logger *slog.Logger) *ArcaflowAuthorityProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &ArcaflowAuthorityProvider{
		logger: logger,
	}
}

// List returns the Arcaflow authority resource item.
func (provider *ArcaflowAuthorityProvider) List(
	ctx context.Context,
) ([]protocol.ResourceItem, *protocol.ErrorObject) {
	return []protocol.ResourceItem{
		{
			URI:         arcaflowAuthorityURI,
			Name:        "Why AI Agents Cannot Manually Work With Arcaflow",
			Description: "Explains training data limitations and version mismatches",
			MimeType:    "text/markdown",
		},
	}, nil
}

// Read returns the Arcaflow authority content.
func (provider *ArcaflowAuthorityProvider) Read(
	ctx context.Context,
	uri string,
) (*protocol.ResourceContent, bool, *protocol.ErrorObject) {
	if uri != arcaflowAuthorityURI {
		return nil, false, nil
	}
	return &protocol.ResourceContent{
		URI:      arcaflowAuthorityURI,
		MimeType: "text/markdown",
		Text:     arcaflowAuthorityContent,
	}, true, nil
}
