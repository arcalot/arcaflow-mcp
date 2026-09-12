# Troubleshooting Guide

This guide helps resolve common issues when using the Arcaflow MCP server.

[← Back to Documentation Index](index.md) | [← Back to README](../../README.md)

## Table of Contents

- [Installation Issues](#installation-issues) (including [container runtime](#container-runtime-not-found-validation-fails))
- [Connection Issues](#connection-issues)
- [Workflow Discovery Issues](#workflow-discovery-issues)
- [Input Construction Issues](#input-construction-issues)
- [Result Analysis Issues](#result-analysis-issues)
- [Server Mode Issues](#server-mode-issues)
- [Performance Issues](#performance-issues)
- [Getting Help](#getting-help)

---

## Installation Issues

### Go Build Fails

**Problem:** `go build` fails with module errors

**Solution:**
1. Verify Go version: `go version` (must be 1.23.0)
2. Download dependencies: `cd server && go mod download`
3. Clear module cache: `go clean -modcache`
4. Retry build

**Related:** [Development Setup](../development/setup.md)

---

### Python Poetry Install Fails

**Problem:** `poetry install` fails in analysis directory

**Solution:**
1. Verify Python version: `python --version` (must be 3.12.x)
2. Verify Poetry version: `poetry --version` (must be 1.8.3)
3. Clear Poetry cache: `poetry cache clear --all pypi`
4. Retry installation: `poetry install`

**Related:** [Development Setup](../development/setup.md)

---

### Git Hooks Not Working

**Problem:** Pre-commit hooks don't run or fail

**Solution:**
1. Reinstall hooks: `./scripts/dev-setup.sh`
2. Verify hook permissions: `ls -l .git/hooks/`
3. Set cache directories:
   ```bash
   export XDG_CACHE_HOME="$PWD/.cache"
   export GOLANGCI_LINT_CACHE="$PWD/.cache/golangci-lint"
   export GOCACHE="$PWD/.cache/go-build"
   export GOMODCACHE="$PWD/.cache/go-mod"
   export POETRY_CACHE_DIR="$PWD/.cache/poetry"
   ```
4. Test hooks manually: `./scripts/validate.sh`

**Related:** [Development Setup](../development/setup.md)

---

### Container Runtime Not Found (Validation Fails)

**Problem:** Workflow input validation fails with deployer or plugin errors

**Solution:**
1. Verify a container runtime is installed:
   ```bash
   podman --version   # Preferred
   docker --version   # Alternative
   ```
2. If using Docker instead of Podman, set the deployer:
   ```bash
   export ARCAFLOW_MCP_DEPLOYER=docker
   ```
   Or in your config YAML:
   ```yaml
   engine:
     deployer: docker
   ```
3. Verify the runtime can pull images:
   ```bash
   podman pull quay.io/arcalot/arcaflow-plugin-utilities:latest
   ```
4. Check permissions — rootless Podman may need `loginctl enable-linger $USER`

**Why is a container runtime needed?**
Arcaflow workflows reference plugins as container images. The engine
resolves plugin schemas by pulling and inspecting these images during
validation. Without a container runtime, the MCP server cannot validate
inputs against the full workflow schema.

**Related:** [Configuration - Engine Deployer](usage/configuration.md#engine-deployer)

---

## Connection Issues

### Claude Desktop Can't Connect (Local Mode)

**Problem:** Claude Desktop shows "MCP server not responding"

**Solution:**
1. Verify server binary exists and is executable:
   ```bash
   ls -l server/arcaflow-mcp
   ```
2. Test server manually:
   ```bash
   ./server/arcaflow-mcp --mode local
   ```
   Type `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` and press Enter
3. Check Claude Desktop MCP configuration file location:
   - macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
   - Linux: `~/.config/Claude/claude_desktop_config.json`
   - Windows: `%APPDATA%\Claude\claude_desktop_config.json`
4. Verify configuration format:
   ```json
   {
     "mcpServers": {
       "arcaflow": {
         "command": "/absolute/path/to/server/arcaflow-mcp",
         "args": ["--mode", "local"]
       }
     }
   }
   ```
5. Restart Claude Desktop after configuration changes

**Related:** [Local Mode Setup](usage/local-mode.md)

---

### Binary Not Found - Wrong Directory

**Problem:** Getting `No such file or directory` when trying to start server

**Error Examples:**
```bash
# From server/ directory:
./server/arcaflow-mcp --mode server --address :8080
# Error: bash: ./server/arcaflow-mcp: No such file or directory

# From repository root:
./arcaflow-mcp --mode server --address :8080
# Error: bash: ./arcaflow-mcp: No such file or directory
```

**Cause:** Binary path depends on your current directory location.

**Solution:**

**Check current location:**
```bash
# See where you are
pwd

# Look for the binary
ls server/arcaflow-mcp  # If in repository root
ls arcaflow-mcp         # If in server/ directory
```

**Use correct path based on location:**
```bash
# If in repository root (e.g., /path/to/arcaflow-mcp/):
./server/arcaflow-mcp --mode server --address :8080

# If in server/ directory (e.g., /path/to/arcaflow-mcp/server/):
./arcaflow-mcp --mode server --address :8080
```

**Recommended:** Always run from repository root for consistency with documentation examples.

---

### Server Mode Connection Refused

**Problem:** HTTP client can't connect to server mode

**Solution:**
1. Verify server is running:
   ```bash
   ps aux | grep arcaflow-mcp
   ```
2. Check server logs for errors
3. Verify address/port:
   ```bash
   netstat -an | grep 8080
   ```
4. Test with curl (complete handshake):
   ```bash
   # Initialize
   curl -X POST http://localhost:8080/mcp \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
     -d @- <<'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
EOF

   # Complete initialization
   curl -X POST http://localhost:8080/mcp \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
     -d @- <<'EOF'
{"jsonrpc":"2.0","method":"initialized"}
EOF
   ```
5. Check firewall rules if accessing remotely

**Related:** [Server Mode Setup](usage/server-mode.md)

---

### Permission Denied on Startup

**Problem:** Server fails to start with "permission denied" error for `/var/lib/arcaflow-mcp`

**Error Message:**
```
{"level":"ERROR","msg":"arcaflow-mcp failed","error":"mkdir /var/lib/arcaflow-mcp: permission denied"}
```

**Cause:** Server tries to create data directory at `/var/lib/arcaflow-mcp` by default, which requires root permissions.

**Solution:**

**For Development/Testing** (use local data directory):
```bash
# Step 1: Set data directory location (change this path if needed)
export DATA_DIR="./data"

# Step 2: Create local data directory
mkdir -p "$DATA_DIR"

# Step 3: Set admin token for testing
export ARCAFLOW_MCP_ADMIN_TOKEN="dev-test-token-12345"

# Step 4: Configure all store paths (all use $DATA_DIR)
export ARCAFLOW_MCP_TOKEN_STORE_PATH="$DATA_DIR/tokens.json"
export ARCAFLOW_MCP_TENANT_STORE_PATH="$DATA_DIR/tenants.json"
export ARCAFLOW_MCP_AUDIT_STORE_PATH="$DATA_DIR/audit.json"
export ARCAFLOW_MCP_USAGE_STORE_PATH="$DATA_DIR/usage.json"
export ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT="$DATA_DIR/tenants"

# Step 5: Start server
./server/arcaflow-mcp --mode server --address :8080
```

**For Production** (use system directory with proper permissions):
```bash
# Option 1: Run as root (not recommended)
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -hex 32)"
sudo -E ./server/arcaflow-mcp --mode server --address :8080

# Option 2: Create directory and set ownership (recommended)
# Determine your user and group
echo "Current user: $(whoami)"
echo "Primary group: $(id -gn)"

# Create directory with your ownership (replace USER:GROUP if needed)
sudo mkdir -p /var/lib/arcaflow-mcp
sudo chown $(whoami):$(id -gn) /var/lib/arcaflow-mcp
sudo chmod 750 /var/lib/arcaflow-mcp

# Start server as your user
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -hex 32)"
./server/arcaflow-mcp --mode server --address :8080

# Option 3: Use systemd service with proper user (best practice)
# See: deployment/systemd.md
```

**Related:** [Server Mode Setup](usage/server-mode.md), [Configuration Reference](usage/configuration.md)

---

### Server Not Initialized Error

**Problem:** Getting `{"error":{"code":-32600,"message":"server not initialized"}}` when making tool calls

**Error Context:**
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call",...}'

# Response: {"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"server not initialized"}}
```

**Cause:** MCP protocol requires a **two-step initialization handshake** before any operations:
1. Client sends `initialize` request → Server responds with capabilities
2. Client sends `initialized` notification → Server becomes ready for tool calls

**Common causes:**
- Missing step 2 (`initialized` notification) - most common
- Separate curl commands without SSE session binding - each connection has independent state
- Restarting server between commands - clears all session state

**Solution:**

Complete both steps of the initialization handshake:
```bash
# Step 1: Initialize request
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
      "name": "my-client",
      "version": "1.0"
    }
  }
}
EOF

# Step 2: Send initialized notification (completes handshake)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{
  "jsonrpc": "2.0",
  "method": "initialized"
}
EOF

# Step 3: Now you can make tool calls
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
EOF
```

**Important Notes:**
- **MCP Protocol**: The two-step handshake is required by the MCP specification. All MCP clients (Claude Desktop, etc.) automatically perform it.
- **Session State Without SSE**: Each curl example above works independently because it includes the complete handshake. However, state is **not** maintained across separate curl invocations.
- **For Multi-Step Workflows**: If you need to maintain state across multiple tool calls (like loading a workflow, then building inputs from it), use SSE session binding. See [Session Binding](usage/server-mode.md#session-binding) for details.
- **AI Clients**: MCP clients automatically handle session binding, so this is only a concern for manual curl testing.

**Related:** [MCP Specification](https://modelcontextprotocol.io/specification/), [Server Mode Setup](usage/server-mode.md), [Session Binding](usage/server-mode.md#session-binding)

---

### Unsupported Protocol Version Error

**Problem:** Getting `unsupported protocolVersion` error when initializing

**Error Message:**
```json
{
  "jsonrpc":"2.0",
  "id":1,
  "error":{
    "code":-32602,
    "message":"unsupported protocolVersion",
    "data":{"supported":"2025-11-25"}
  }
}
```

**Cause:** Client is using wrong MCP protocol version. The server supports `2025-11-25`.

**Solution:**

Use the correct protocol version and complete the handshake:
```bash
# Part 1: Initialize with correct version
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
    "clientInfo": {"name": "my-client", "version": "1.0"}
  }
}
EOF

# Part 2: Complete initialization
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d @- <<'EOF'
{
  "jsonrpc": "2.0",
  "method": "initialized"
}
EOF
```

**Note:** Always check the error message `data.supported` field to see what version the server expects.

**Related:** [MCP Protocol Versions](https://modelcontextprotocol.io/specification/), [Server Mode Setup](usage/server-mode.md)

---

### Authentication Failures

**Problem:** "Unauthorized" or "Invalid token" errors

**Solution:**
1. Verify token is configured:
   ```bash
   cat /path/to/tokens.json
   ```
2. Check token format in request header:
   ```
   Authorization: Bearer <token>
   ```
3. Verify token hasn't expired (if using time-based tokens)
4. Check server logs for authentication errors
5. Create new token if necessary

**Related:** [Authentication Setup](deployment/authentication.md)

---

## Workflow Discovery Issues

### Workflow Not Found

**Problem:** `workflow_discover` or `workflow_load` fails to find workflow

**Solution:**
1. Verify workflow path or URL is correct
2. For filesystem paths, use absolute paths:
   ```
   file:///absolute/path/to/workflow.yaml
   ```
3. For git repositories, verify:
   - Repository URL is accessible
   - Branch/tag/commit ref exists
   - Workflow file exists in specified path
4. Check permissions on filesystem paths
5. Use `workflow_discover` to search for workflows first

**Related:** [Input Construction Guide](usage/input-construction.md)

---

### Git Discovery Timeout

**Problem:** Git repository discovery times out or hangs

**Solution:**
1. Use specific branch/tag instead of default branch:
   ```json
   {
     "source": {
       "kind": "git",
       "location": "https://github.com/org/repo.git",
       "ref": "main"
     }
   }
   ```
2. Use subdirectory to narrow search:
   ```json
   {
     "source": {
       "kind": "git",
       "location": "https://github.com/org/repo.git",
       "ref": "main",
       "subdir": "workflows/"
     }
   }
   ```
3. Clone repository locally and use filesystem mode:
   ```bash
   git clone https://github.com/org/repo.git
   # Then use: file:///path/to/repo/workflow.yaml
   ```

**Related:** [Input Construction Guide](usage/input-construction.md)

---

### Multiple Workflows Found

**Problem:** `workflow_load` returns multiple matching workflows

**Solution:**
1. Use the `workflow_discover` response to get specific workflow IDs:
   ```json
   {
     "selector": {
       "id": "specific-workflow-id"
     }
   }
   ```
2. Use `selector.path` to specify exact file path
3. Narrow search with subdirectory in git sources

**Related:** [Input Construction Guide](usage/input-construction.md)

---

## Input Construction Issues

### Schema Validation Fails

**Problem:** Input validation fails with "does not match schema" errors

**Solution:**
1. Review the validation error message for specific field issues
2. Use `workflow_schema_get` to see required fields and types
3. Check example inputs with `workflow_input_examples_get`
4. Verify data types match schema (string vs number, etc.)
5. Check for required fields that are missing
6. Validate nested object structure matches schema

**Example Fix:**
```yaml
# Error: "expected number, got string"
# Incorrect:
parallel: "5"

# Correct:
parallel: 5
```

**Related:** [Input Construction Guide](usage/input-construction.md)

---

### Plugin Schema Not Found

**Problem:** Error loading plugin schemas for steps

**Solution:**
1. Verify workflow uses valid plugin references
2. For MCP usage, ensure `plugin_schema_ref` is set correctly
3. Check that plugin container images are accessible
4. For filesystem workflows, verify plugin deployment configurations
5. Use workflows with local or embedded schemas when possible

**Related:** [Input Construction Guide](usage/input-construction.md)

---

### Input Export Fails

**Problem:** `workflow_input_export` fails to write file

**Solution:**
1. Verify input validation passes first (export requires valid input)
2. Check filesystem permissions on target directory
3. Use absolute path for output file
4. Verify disk space available
5. Check for special characters in filename

**Related:** [Input Construction Guide](usage/input-construction.md)

---

## Result Analysis Issues

### Results File Not Found

**Problem:** `workflow_results_load` fails to find results file

**Solution:**
1. Use absolute path to results file:
   ```json
   {
     "source": {
       "kind": "filesystem",
       "location": "/absolute/path/to/results.json"
     }
   }
   ```
2. Verify file exists and is readable:
   ```bash
   ls -l /path/to/results.json
   ```
3. Check file format (JSON or YAML)
4. Verify results file contains valid workflow output

**Related:** [Result Analysis Guide](usage/result-analysis.md)

---

### Parse Errors on Results

**Problem:** Results file loads but parsing fails

**Solution:**
1. Verify file is valid JSON/YAML:
   ```bash
   # For JSON:
   jq . results.json
   
   # For YAML:
   python -c "import yaml; yaml.safe_load(open('results.yaml'))"
   ```
2. Check that file contains Arcaflow workflow output format
3. Try manually fixing JSON/YAML syntax errors
4. Use `workflow_results_load` with format hint:
   ```json
   {
     "format": "json"
   }
   ```

**Related:** [Result Analysis Guide](usage/result-analysis.md)

---

### No Metrics Extracted

**Problem:** Result analysis completes but no metrics found

**Solution:**
1. Verify workflow actually produced output metrics
2. Check results file contains `output` or `outputs` section
3. Review workflow output schema - metrics may be nested
4. Use `workflow_results_parse` to see parsed structure
5. Manually inspect results file for expected metric fields

**Related:** [Result Analysis Guide](usage/result-analysis.md)

---

## Server Mode Issues

### High Memory Usage

**Problem:** Server mode consumes excessive memory

**Solution:**
1. Check number of active sessions:
   - Review audit logs for concurrent connections
   - Implement session cleanup policies
2. Reduce cache sizes in configuration
3. Enable cache eviction for old workflow schemas
4. Monitor for memory leaks (check logs for growth patterns)
5. Restart server periodically if memory continues to grow
6. Consider horizontal scaling for high load

**Related:** [Server Mode Setup](usage/server-mode.md), [Configuration Reference](usage/configuration.md)

---

### Rate Limiting Errors

**Problem:** Requests fail with "rate limit exceeded" errors

**Solution:**
1. Check current rate limit configuration
2. Verify tenant is using correct token
3. Spread requests over time (avoid bursts)
4. Request rate limit increase from administrator
5. For administrators: adjust rate limits in configuration:
   ```yaml
   rateLimit:
     requestsPerMinute: 100
     burstSize: 20
   ```

**Related:** [Server Mode Setup](usage/server-mode.md), [Configuration Reference](usage/configuration.md)

---

### Multi-Tenant Isolation Issues

**Problem:** Users seeing other tenants' data

**Solution:**
1. **SECURITY ISSUE** - Report immediately to maintainers
2. Verify each tenant has unique token
3. Check audit logs for cross-tenant access
4. Restart server to clear any cached state
5. Review tenant configuration in tokens file
6. Check server logs for isolation errors

**Related:** [Multi-tenancy Concepts](concepts/multi-tenancy.md), [Authentication Setup](deployment/authentication.md)

---

## Performance Issues

### Slow Workflow Loading

**Problem:** `workflow_load` takes too long

**Solution:**
1. Use workflow discovery cache when re-loading same workflow
2. Clone git repositories locally for frequent access
3. Use specific refs/branches (not "HEAD" or default)
4. Enable workflow schema caching (check configuration)
5. Consider using workflow filesystem access instead of git for repeated access

**Expected Performance:**
- Filesystem: <500ms
- URL: <2s (depending on network)
- Git: <10s for first load, <2s cached

**Related:** [Configuration Reference](usage/configuration.md)

---

### Slow Result Analysis

**Problem:** Result analysis takes too long

**Solution:**
1. Check size of results file (large files take longer)
2. Verify Python analysis service is running (server mode)
3. Check system resources (CPU, memory)
4. Reduce number of runs in multi-run comparison
5. Use metric extraction instead of full analysis for large datasets

**Expected Performance:**
- Small results (<1MB): <2s
- Medium results (1-10MB): <5s
- Large results (>10MB): <15s

**Related:** [Result Analysis Guide](usage/result-analysis.md)

---

## Getting Help

### Check Logs

**Local Mode:**
- Server logs printed to stderr
- Redirect to file: `./server/arcaflow-mcp --mode local 2>server.log`

**Server Mode:**
- Configure log file in server configuration
- Check system logs: `journalctl -u arcaflow-mcp`
- Container logs: `podman logs <container-id>`

---

### Enable Debug Logging

Add to server command line:
```bash
./server/arcaflow-mcp --mode local --log-level debug
```

Or in configuration file:
```yaml
logging:
  level: debug
```

---

### Verify Installation

Run full validation:
```bash
./scripts/validate.sh
./scripts/test-all.sh
```

---

### Report Issues

If you can't resolve the issue:

1. Check [existing issues](https://github.com/arcalot/arcaflow-mcp/issues)
2. Collect diagnostic information:
   - Server version: `./server/arcaflow-mcp --version`
   - Go version: `go version`
   - Python version: `python --version`
   - Operating system and architecture
   - Relevant log excerpts (sanitize sensitive data)
   - Steps to reproduce
3. Create a new issue with:
   - Clear problem description
   - Expected vs actual behavior
   - Reproduction steps
   - Diagnostic information
   - Configuration (sanitize tokens/secrets)

---

### Community Support

- **Arcalot Community**: [arcalot-round-table](https://github.com/arcalot/arcalot-round-table)
- **Documentation**: [Full documentation index](index.md)
- **Examples**: [Tutorials and examples](examples/)

---

## Related Documentation

- [Getting Started Guide](getting-started.md)
- [Local Mode Setup](usage/local-mode.md)
- [Server Mode Setup](usage/server-mode.md)
- [Configuration Reference](usage/configuration.md)
- [Development Debugging Guide](../development/debugging.md)

---

[← Back to Documentation Index](index.md) | [← Back to README](../../README.md)
