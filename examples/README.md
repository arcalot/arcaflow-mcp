# Examples and Tutorials

Example workflows, tutorials, and usage demonstrations for the Arcaflow MCP
server.

[← Back to Main README](../README.md)

## Overview

This directory contains working examples demonstrating the Arcaflow MCP server's
capabilities. All examples are tested and include step-by-step instructions.

## Quick Start Examples

### 1. Hello World Workflow

The simplest possible Arcaflow workflow for testing MCP server setup.

**Location:** `workflows/hello-world.yaml`

**Use Case:** Verify MCP server installation and basic functionality

**Complexity:** Beginner

**Tutorial:** See [hello-world/README.md](workflows/hello-world/README.md)

---

### 2. Data Processing Workflow

A workflow that transforms input data and produces output metrics.

**Location:** `workflows/data-processing.yaml`

**Use Case:** Learn input construction and basic result analysis

**Complexity:** Intermediate

**Tutorial:** See [data-processing/README.md](workflows/data-processing/README.md)

---

### 3. Performance Testing Workflow

A workflow that runs performance tests and produces detailed metrics.

**Location:** `workflows/perf-test.yaml`

**Use Case:** Multi-run comparison and optimization

**Complexity:** Advanced

**Tutorial:** See [perf-test/README.md](workflows/perf-test/README.md)

---

## Complete Tutorials

Full tutorials demonstrating end-to-end workflows:

### Tutorial 1: Conversational Input Construction

Learn to build workflow inputs through conversation with AI.

**What You'll Learn:**
- Discover workflows from git repositories
- Extract and understand workflow schemas
- Build inputs conversationally
- Validate inputs against schemas
- Export deterministic input files

**Location:** See [User Documentation - Basic Workflow](../docs/arcaflow-mcp/examples/basic-workflow.md)

**Duration:** ~15 minutes

---

### Tutorial 2: Result Analysis and Optimization

Learn to analyze workflow results and generate optimization suggestions.

**What You'll Learn:**
- Load workflow execution results
- Analyze results against defined goals
- Understand optimization recommendations
- Apply suggestions to improve results

**Location:** See [User Documentation - Iterative Optimization](../docs/arcaflow-mcp/examples/iterative-optimization.md)

**Duration:** ~20 minutes

---

### Tutorial 3: Multi-Run Comparison

Learn to compare multiple workflow executions to find optimal configurations.

**What You'll Learn:**
- Load results from multiple runs
- Compare metrics across configurations
- Identify patterns and trends
- Determine optimal settings

**Location:** See [User Documentation - Multi-Run Comparison](../docs/arcaflow-mcp/examples/multi-run-comparison.md)

**Duration:** ~25 minutes

---

## Example Workflows

All example workflows include:
- Complete workflow YAML file
- Example input files
- Expected output examples
- README with usage instructions
- Tested and verified working

### Available Workflows

| Workflow | Complexity | Purpose |
|----------|------------|---------|
| [hello-world](workflows/hello-world/) | Beginner | Basic MCP server testing |
| [data-processing](workflows/data-processing/) | Intermediate | Input construction & analysis |
| [perf-test](workflows/perf-test/) | Advanced | Multi-run optimization |

---

## Running Examples

### Prerequisites

- Arcaflow MCP server installed (see [Getting Started](../docs/arcaflow-mcp/getting-started.md))
- MCP client (Claude Desktop, etc.) or curl for testing
- Arcaflow Engine 0.20.0+ (for executing workflows)

### Using Examples with MCP Server

**Local Mode (Claude Desktop):**

1. Configure Claude Desktop to use Arcaflow MCP
2. Start conversation: "Load the hello-world workflow from examples/"
3. Follow AI's guidance to build inputs
4. Export inputs and run workflow

**Server Mode (curl):**

```bash
# Step 1: Set data and repo directory locations (change these paths if needed)
export DATA_DIR="/var/tmp/arcaflow-mcp-data"
export REPO_DIR="/opt/arcaflow-mcp"

# Step 2: Create data directory
mkdir -p "$DATA_DIR"

# Step 3: Set admin token (copy-paste for testing)
export ARCAFLOW_MCP_ADMIN_TOKEN="example-test-token-12345"

# Step 4: Configure storage paths (all use $DATA_DIR)
export ARCAFLOW_MCP_TOKEN_STORE_PATH="$DATA_DIR/tokens.json"
export ARCAFLOW_MCP_TENANT_STORE_PATH="$DATA_DIR/tenants.json"
export ARCAFLOW_MCP_AUDIT_STORE_PATH="$DATA_DIR/audit.json"
export ARCAFLOW_MCP_USAGE_STORE_PATH="$DATA_DIR/usage.json"
export ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT="$DATA_DIR/tenants"

# Step 5: Start Python analysis engine (required for result analysis features)
cd "$REPO_DIR/analysis"
poetry install
poetry run python -m arcaflow_analysis.server.http_server &
ANALYSIS_PID=$!
cd "$REPO_DIR"

# Wait for analysis engine to start
sleep 2

# Step 6: Start Go MCP server (connects to analysis engine)
export ARCAFLOW_MCP_ANALYSIS_HTTP_URL="http://localhost:8081"
arcaflow-mcp --mode server --address :8080 &
ARCAFLOW_MCP_PID=$!

# Wait for server to start
sleep 2

# Step 7: Initialize MCP session (part 1 - REQUIRED handshake)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-11-25",
    "capabilities": {},
    "clientInfo": {
      "name": "curl-test",
      "version": "1.0"
    }
  }
}
EOF

# Step 8: Complete initialization (part 2 - notification, no id field)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{
  "jsonrpc": "2.0",
  "method": "initialized"
}
EOF

# Step 9: Load workflow using absolute path (now this will work!)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<EOF
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "workflow_load",
    "arguments": {
      "source": {
        "kind": "filesystem",
        "location": "$REPO_DIR/examples/workflows/hello-world"
      },
      "selector": {
        "path": "workflow.yaml"
      }
    }
  }
}
EOF

# Step 10: Cleanup (kill both processes)
kill $ARCAFLOW_MCP_PID $ANALYSIS_PID
```

**Important Notes:**
- **Two components required**: Server mode needs both the Go MCP server and Python analysis engine running
- **MCP server in your executable path**: All commands assume the `arcaflow-mcp` binary is in your `$PATH`
- **Token must match**: The token in the curl command must match `ARCAFLOW_MCP_ADMIN_TOKEN`
- **Absolute paths**: Workflow locations must be absolute paths; `$PWD` expands to current directory
- **Production**: For production use, generate secure token with `openssl rand -hex 32`

See [Server Mode Setup](../docs/arcaflow-mcp/usage/server-mode.md) for complete configuration details.

### Executing Workflows

After building inputs with MCP server:

```bash
# Execute workflow with Arcaflow Engine
arcaflow -input inputs.yaml workflow.yaml

# Results will be written to output.yaml or specified location
```

---

## Example Directory Structure

```
examples/
├── README.md                           # This file
└── workflows/                          # Example Arcaflow workflows
    ├── hello-world/                    # Simple hello world
    │   ├── README.md                   # Usage guide
    │   ├── workflow.yaml               # Workflow definition
    │   ├── inputs/                     # Example inputs
    │   │   ├── example1.yaml
    │   │   └── example2.yaml
    │   └── outputs/                    # Example outputs
    │       ├── example1-output.yaml
    │       └── example2-output.yaml
    ├── data-processing/                # Data transformation
    │   ├── README.md
    │   ├── workflow.yaml
    │   ├── inputs/
    │   └── outputs/
    └── perf-test/                      # Performance testing
        ├── README.md
        ├── workflow.yaml
        ├── inputs/
        └── outputs/
```

---

## Contributing Examples

Have a useful example workflow? We'd love to include it!

**Requirements:**
- Working Arcaflow workflow YAML
- At least 2 example input files
- At least 2 example output files
- Comprehensive README with:
  - Purpose and use case
  - Step-by-step instructions
  - Expected results
  - Troubleshooting tips
- All files tested and verified working

**Contribution Process:**
1. Create example in `examples/workflows/your-example/`
2. Test thoroughly
3. Submit PR with clear description
4. See [Contributing Guide](../CONTRIBUTING.md)

---

## Troubleshooting

### Workflow Load Fails

**Problem:** MCP server can't find workflow file

**Solution:**
- Use absolute paths: `file:///absolute/path/to/workflow.yaml`
- Verify file exists: `ls -l path/to/workflow.yaml`
- Check file permissions

See [Troubleshooting Guide](../docs/arcaflow-mcp/troubleshooting.md)

---

### Input Validation Fails

**Problem:** Built inputs don't validate against schema

**Solution:**
- Review validation error messages
- Check example inputs in workflow directory
- Verify data types match schema
- Use `workflow_schema_get` to see requirements

See [Input Construction Guide](../docs/arcaflow-mcp/usage/input-construction.md)

---

### Workflow Execution Fails

**Problem:** Arcaflow Engine can't execute workflow

**Solution:**
- Verify Arcaflow Engine is installed
- Check plugin images are accessible
- Review workflow logs for specific errors
- Test with example inputs first

See [Arcaflow Documentation](https://arcalot.io/arcaflow)

---

## Additional Resources

### Documentation
- [Getting Started Guide](../docs/arcaflow-mcp/getting-started.md)
- [Input Construction Guide](../docs/arcaflow-mcp/usage/input-construction.md)
- [Result Analysis Guide](../docs/arcaflow-mcp/usage/result-analysis.md)
- [Tool Reference](../docs/arcaflow-mcp/tools/overview.md)

### Arcaflow Resources
- [Arcaflow Documentation](https://arcalot.io/arcaflow)
- [Arcaflow Engine](https://github.com/arcalot/arcaflow-engine)
- [Arcaflow Workflows Repository](https://github.com/arcalot/arcaflow-workflows)

### Community
- [Arcalot Round Table](https://github.com/arcalot/arcalot-round-table)
- [GitHub Issues](https://github.com/arcalot/arcaflow-mcp/issues)

---

[← Back to Main README](../README.md)
