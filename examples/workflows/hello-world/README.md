# Hello World Workflow Example

A minimal Arcaflow workflow for testing the Arcaflow MCP server.

[← Back to Examples](../../README.md)

## Purpose

This is the simplest possible Arcaflow workflow. Use it to:
- Verify Arcaflow MCP server installation
- Test basic workflow loading and schema extraction
- Learn input construction with minimal complexity
- Understand workflow execution basics

## Workflow Description

The hello-world workflow:
1. Accepts a `name` input parameter (string)
2. Generates a greeting message
3. Returns the message as output

**Complexity:** Beginner  
**Estimated Time:** 5-10 minutes

## Files

- `workflow.yaml` - Workflow definition
- `inputs/example1.yaml` - Basic input example
- `inputs/example2.yaml` - Alternative input example
- `outputs/example1-output.yaml` - Expected output for example1
- `outputs/example2-output.yaml` - Expected output for example2

## Prerequisites

- Arcaflow MCP server installed and running
- For curl examples: Server must be running with authentication configured
  - Set `ARCAFLOW_MCP_ADMIN_TOKEN` environment variable
  - Configure data storage paths (see [Server Mode Setup](../../docs/arcaflow-mcp/usage/server-mode.md))
- MCP client (Claude Desktop, etc.) or curl
- Arcaflow Engine 0.20.0+ (optional, for execution)

**Note:** All curl examples below include the complete MCP handshake (`initialize` + `initialized`). Each example is self-contained and can be run independently.

## Usage with MCP Server

### Step 1: Load Workflow

**Using Claude Desktop:**
```
"Load the hello-world workflow from 
/path/to/arcaflow-mcp/examples/workflows/hello-world/workflow.yaml"
```

**Using curl (assumes server running with token from server setup):**
```bash
# Set the workflow location (run from repository root)
WORKFLOW_PATH="$PWD/examples/workflows/hello-world/workflow.yaml"

# Initialize MCP session (required before tool calls)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"curl","version":"1.0"}}}
EOF

# Complete initialization
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{"jsonrpc":"2.0","method":"initialized"}
EOF

# Load workflow
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
        "location": "$WORKFLOW_PATH"
      }
    }
  }
}
EOF
```

### Step 2: Get Workflow Schema

**Using Claude Desktop:**
```
"What inputs does this workflow require?"
```

**Expected Schema:**
```yaml
input:
  name:
    type: string
    required: true
    description: Name to greet
```

### Step 3: Build Inputs Conversationally

**Using Claude Desktop:**
```
"Build inputs for this workflow. Use the name 'Alice'"
```

AI will help construct:
```yaml
name: "Alice"
```

### Step 4: Validate Inputs

**Using Claude Desktop:**
```
"Validate these inputs"
```

MCP server will confirm inputs are valid and match schema.

### Step 5: Export Inputs

**Using Claude Desktop:**
```
"Export these inputs to hello-inputs.yaml"
```

**Using curl (use your actual admin token):**
```bash
# Initialize MCP session (if not already done)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"curl","version":"1.0"}}}
EOF

curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{"jsonrpc":"2.0","method":"initialized"}
EOF

# Export inputs
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "workflow_input_export",
    "arguments": {
      "input": {"name": "Alice"},
      "format": "yaml",
      "output_path": "/path/to/hello-inputs.yaml"
    }
  }
}
EOF
```

### Step 6: Execute Workflow (Optional)

If you have Arcaflow Engine installed:

```bash
# Execute workflow
arcaflow -input hello-inputs.yaml \
  examples/workflows/hello-world/workflow.yaml

# Check output
cat output.yaml
```

Expected output:
```yaml
output:
  message: "Hello, Alice!"
  status: "success"
```

## Example Inputs

### Example 1: Basic Greeting

File: `inputs/example1.yaml`

```yaml
name: "World"
```

Expected Output: `outputs/example1-output.yaml`

```yaml
output:
  message: "Hello, World!"
  status: "success"
```

### Example 2: Named Greeting

File: `inputs/example2.yaml`

```yaml
name: "Arcaflow User"
```

Expected Output: `outputs/example2-output.yaml`

```yaml
output:
  message: "Hello, Arcaflow User!"
  status: "success"
```

## Learning Objectives

After completing this example, you should understand:

1. **Workflow Loading:**
   - How to load workflows from filesystem
   - How MCP server parses workflow files
   - How to reference workflows by path

2. **Schema Extraction:**
   - How MCP server extracts input schemas
   - Required vs optional fields
   - Field types and descriptions

3. **Input Construction:**
   - How to build simple inputs
   - How validation works
   - How to export validated inputs

4. **Workflow Execution:**
   - How to run workflows with Arcaflow Engine
   - How inputs map to outputs
   - How to interpret results

## Next Steps

After mastering hello-world, try:

1. **[Data Processing Example](../data-processing/)** - Learn about complex
   inputs and result analysis
2. **[Performance Testing Example](../perf-test/)** - Learn about multi-run
   comparison
3. **[User Tutorials](../../../docs/arcaflow-mcp/examples/)** - Complete
   end-to-end tutorials

## Troubleshooting

### Workflow Not Found

**Problem:** MCP server can't find workflow.yaml

**Solution:**
```bash
# Verify file exists
ls -l examples/workflows/hello-world/workflow.yaml

# Use absolute path
realpath examples/workflows/hello-world/workflow.yaml
```

### Schema Validation Fails

**Problem:** Inputs don't validate

**Solution:**
- Verify `name` field is a string
- Ensure `name` field is present (required)
- Check YAML syntax is valid

### Execution Fails

**Problem:** Arcaflow Engine can't execute workflow

**Solution:**
- Verify Arcaflow Engine is installed: `arcaflow --version`
- Check workflow syntax: `arcaflow --validate workflow.yaml`
- Ensure plugin images are accessible

## Related Documentation

- [Getting Started Guide](../../../docs/arcaflow-mcp/getting-started.md)
- [Input Construction Guide](../../../docs/arcaflow-mcp/usage/input-construction.md)
- [Troubleshooting](../../../docs/arcaflow-mcp/troubleshooting.md)

---

[← Back to Examples](../../README.md)
