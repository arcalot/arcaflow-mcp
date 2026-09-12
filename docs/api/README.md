# API Documentation

API documentation for the Arcaflow MCP server components.

[← Back to Main README](../../README.md)

## Overview

This directory contains API documentation for both the Go MCP server and Python
analysis engine. For usage guides, see [User Documentation](../arcaflow-mcp/).

## Documentation Index

- **[Go Server API](go-server.md)** - Go API reference with links to godoc
- **[Python Engine API](python-engine.md)** - Python API reference with module
  documentation
- **[gRPC Protocol](grpc-protocol.md)** - gRPC service contracts and message
  formats

## API Types

### 1. MCP Protocol API (External)

The external API exposed to MCP clients (AI agents):

**Protocol:** JSON-RPC 2.0 over MCP  
**Transports:** stdio (local mode), HTTP/SSE (server mode)  
**Specification:** [MCP 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25/)

**Key Methods:**
- `initialize` - Initialize MCP session
- `tools/list` - List available MCP tools
- `tools/call` - Invoke MCP tool
- `resources/list` - List available MCP resources
- `resources/read` - Read MCP resource

**Documentation:**
- User Guide: [MCP Tools Overview](../arcaflow-mcp/tools/overview.md)
- Tool Reference: [Input Tools](../arcaflow-mcp/tools/input-tools.md),
  [Result Tools](../arcaflow-mcp/tools/result-tools.md)
- Protocol Details: [Go Server API](go-server.md)

---

### 2. Go Internal API

Internal Go APIs for server components:

**Language:** Go 1.23.0  
**Style:** Standard Go idioms and patterns  
**Documentation:** godoc comments inline with code

**Key Packages:**
- `pkg/protocol` - MCP protocol handler (JSON-RPC 2.0)
- `pkg/transport` - stdio and HTTP/SSE transports
- `pkg/tools/workflowtools` - Workflow tool implementations
- `pkg/resources` - MCP resource handlers
- `pkg/arcaflow/workflow` - Workflow loading and parsing
- `pkg/arcaflow/pluginschema` - Plugin schema handling
- `pkg/auth` - Authentication
- `pkg/tenant` - Multi-tenancy
- `pkg/state` - State management

**Documentation:**
- [Go Server API Reference](go-server.md)
- [Architecture](../architecture/go-server.md)

**Generate godoc locally:**
```bash
cd server
go doc -all ./pkg/...

# Or start godoc server:
godoc -http=:6060
# Then visit: http://localhost:6060/pkg/github.com/arcalot/arcaflow-mcp/server/pkg/
```

---

### 3. Python Internal API

Internal Python APIs for analysis engine:

**Language:** Python 3.12  
**Style:** Type-annotated, NumPy-style docstrings  
**Documentation:** Inline docstrings

**Key Modules:**
- `arcaflow_analysis.analyzer` - Result analysis and comparison
- `arcaflow_analysis.parser` - Result loading and parsing
- `arcaflow_analysis.suggester` - Optimization suggestions
- `arcaflow_analysis.db` - Historical database
- `arcaflow_analysis.server` - HTTP API server

**Documentation:**
- [Python Engine API Reference](python-engine.md)
- [Architecture](../architecture/python-engine.md)

**Generate Python docs locally:**
```bash
cd analysis
poetry run python -m pydoc -b
# Or install Sphinx and build:
poetry add --group dev sphinx sphinx-autodoc-typehints
poetry run sphinx-quickstart docs
poetry run sphinx-build -b html docs docs/_build
```

---

### 4. Inter-Service API (gRPC/REST)

API for communication between Go server and Python analysis engine:

**Protocol:** gRPC (primary) or REST (fallback)  
**Transport:** HTTP/2 (gRPC) or HTTP/1.1 (REST)  
**Format:** Protocol Buffers (gRPC) or JSON (REST)

**Service Methods:**
- `AnalyzeResults` - Analyze workflow results
- `CompareResults` - Compare multiple runs
- `SuggestInputs` - Generate optimization suggestions
- `ExtractMetrics` - Extract metrics from results

**Documentation:**
- [gRPC Protocol Reference](grpc-protocol.md)
- [Inter-service Architecture](../architecture/inter-service.md)

---

## API Documentation by Audience

### For MCP Client Developers

Building MCP clients that connect to Arcaflow MCP:

**Start Here:**
- [MCP Protocol Basics](https://modelcontextprotocol.io/specification/)
- [Tool Overview](../arcaflow-mcp/tools/overview.md)
- [Local Mode Setup](../arcaflow-mcp/usage/local-mode.md)
- [Server Mode Setup](../arcaflow-mcp/usage/server-mode.md)

**API Reference:**
- [Input Tools API](../arcaflow-mcp/tools/input-tools.md)
- [Result Tools API](../arcaflow-mcp/tools/result-tools.md)
- [Resources API](../arcaflow-mcp/tools/resources.md)

### For Go Developers

Contributing to the Go MCP server:

**Start Here:**
- [Development Setup](../development/setup.md)
- [Go Server Architecture](../architecture/go-server.md)
- [Testing Guide](../development/testing.md)

**API Reference:**
- [Go Server API](go-server.md)
- [godoc in browser](#generate-godoc-locally)

### For Python Developers

Contributing to the Python analysis engine:

**Start Here:**
- [Development Setup](../development/setup.md)
- [Python Engine Architecture](../architecture/python-engine.md)
- [Testing Guide](../development/testing.md)

**API Reference:**
- [Python Engine API](python-engine.md)
- [pydoc in browser](#generate-python-docs-locally)

### For Integration Developers

Integrating Arcaflow MCP with other systems:

**Start Here:**
- [Architecture Overview](../architecture/overview.md)
- [Inter-service Communication](../architecture/inter-service.md)

**API Reference:**
- [gRPC Protocol](grpc-protocol.md)
- [Go Server API](go-server.md)
- [Python Engine API](python-engine.md)

---

## API Versioning

### MCP Protocol Version

Currently implements: **MCP 2025-11-25**

Breaking changes to MCP protocol require:
- Major version bump (e.g., 0.1.0 → 1.0.0)
- Migration guide
- Deprecation warnings for 2 versions

### Internal API Versioning

**Go APIs:**
- Follow Go module versioning
- Breaking changes avoided in minor versions
- Use of go.mod for dependency management

**Python APIs:**
- Follow semantic versioning
- Poetry for dependency management
- Type hints for API stability

**gRPC Protocol:**
- Protocol buffer versioning
- Backward compatibility maintained
- Deprecation cycle for breaking changes

---

## API Standards

### Go API Standards

- Comprehensive godoc comments for all exported symbols
- Follow Go best practices and idioms
- Maximum line length: 88 characters
- Error handling: Always return error values
- Use structured logging (zap/zerolog)

See [AGENTS.md](../../AGENTS.md) for complete standards.

### Python API Standards

- Type hints for all function signatures
- NumPy-style docstrings
- Maximum line length: 88 characters
- PEP 8 compliance (enforced by Black)
- Use structured logging (structlog)

See [AGENTS.md](../../AGENTS.md) for complete standards.

### API Documentation Standards

- All public APIs must be documented
- Include parameter descriptions
- Include return value descriptions
- Include example usage
- Document error conditions
- Include version compatibility notes

---

## Examples

### MCP Tool Call Example

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "workflow_load",
    "arguments": {
      "source": {
        "kind": "git",
        "location": "https://github.com/arcalot/workflows.git",
        "ref": "main"
      },
      "selector": {
        "path": "example-workflow.yaml"
      }
    }
  }
}
```

### Go API Usage Example

```go
import (
    "github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"
)

// Load workflow from filesystem
loader := workflow.NewLoader()
wf, err := loader.Load(ctx, workflow.Source{
    Kind:     workflow.SourceKindFilesystem,
    Location: "/path/to/workflow.yaml",
})
if err != nil {
    return fmt.Errorf("failed to load workflow: %w", err)
}

// Extract input schema
schema := wf.InputSchema()
```

### Python API Usage Example

```python
from arcaflow_analysis.parser import ResultLoader
from arcaflow_analysis.analyzer import ResultAnalyzer

# Load and analyze results
loader = ResultLoader()
results = loader.load_from_file("/path/to/results.json")

analyzer = ResultAnalyzer()
analysis = analyzer.analyze(
    results=results,
    goal="Achieve 95% success rate"
)

print(analysis.summary)
print(analysis.recommendations)
```

---

## API Testing

### Unit Testing

All APIs have comprehensive unit tests:
- Go: `*_test.go` files (>85% coverage)
- Python: `test_*.py` files (>85% coverage)

Run tests:
```bash
# Go tests
./scripts/test-go.sh

# Python tests
./scripts/test-python.sh

# All tests
./scripts/test-all.sh
```

### Integration Testing

Integration tests in `server/test/integration/`:
- Test Go-Python communication
- Test MCP protocol compliance
- Test end-to-end workflows

### API Contract Testing

- gRPC contracts tested with protocol buffer validation
- MCP protocol tested against specification
- JSON schema validation for all inputs/outputs

---

## Related Documentation

### User Documentation
- [Getting Started](../arcaflow-mcp/getting-started.md)
- [Tool Reference](../arcaflow-mcp/tools/overview.md)
- [Configuration](../arcaflow-mcp/usage/configuration.md)

### Developer Documentation
- [Architecture Overview](../architecture/)
- [Development Setup](../development/setup.md)
- [Contributing Guide](../../CONTRIBUTING.md)

### External References
- [MCP Specification](https://modelcontextprotocol.io/specification/2025-11-25/)
- [Arcaflow Documentation](https://arcalot.io/arcaflow)
- [Go Documentation](https://go.dev/doc/)
- [Python Documentation](https://docs.python.org/3.12/)

---

[← Back to Main README](../../README.md)
