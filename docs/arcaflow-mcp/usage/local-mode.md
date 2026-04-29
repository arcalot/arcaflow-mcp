## Local mode

Local mode runs the MCP server over stdio using MCP Content-Length framing. It
is intended for desktop MCP clients that launch the server as a subprocess.

### Prerequisites

**Using Containers:**
- Podman or Docker
- Container images from quay.io (see [Getting Started](../getting-started.md))

**Building from Source:**

**Component requirements based on your needs:**

| Your Goal | Components Required |
|-----------|-------------------|
| Build workflow inputs only | Go MCP server |
| Analyze workflow results | Go MCP server + Python analysis engine |
| Complete workflow lifecycle | Go MCP server + Python analysis engine |

**Startup Order:** Python analysis engine MUST start BEFORE the Go MCP server (if using result analysis).

---

## Setup Steps

### Step 1: Start the Python Analysis Engine (If Needed)

**Required for:** Result analysis, optimization suggestions, multi-run comparison

**Skip if:** You only need to build workflow inputs

The Python analysis engine provides result analysis, comparison, and optimization features.

**Using Containers:**

```bash
# Get current development tag (before v0.1.0 release)
# With repo: export TAG=$(./scripts/get-container-tag.sh)
export TAG=main-$(curl -s https://api.github.com/repos/arcalot/arcaflow-mcp/commits/main | grep -m1 '"sha"' | cut -d'"' -f4 | cut -c1-7)
echo "Using tag: $TAG"

# Pull image
podman pull quay.io/arcalot/arcaflow-mcp-analysis:${TAG}

# Start analysis engine
podman run -d \
  --name arcaflow-analysis \
  -p 8081:8081 \
  quay.io/arcalot/arcaflow-mcp-analysis:${TAG}

# Wait for startup
sleep 2

# Verify it's running and healthy
curl http://localhost:8081/health
# Expected: {"status":"healthy"}
```

**Building from Source:**

```bash
cd analysis
poetry install
poetry run python -m arcaflow_analysis.server.http_server &

# Save PID for cleanup later
ANALYSIS_PID=$!
echo "Analysis engine PID: $ANALYSIS_PID"

# Wait for startup
sleep 2

# Verify it's running
curl http://localhost:8081/health
# Expected: {"status":"healthy"}
```

The engine listens on `http://localhost:8081` by default.

**Troubleshooting:**
- If health check fails, check logs: `podman logs arcaflow-analysis`
- If port 8081 is in use: `sudo lsof -i :8081`
- For source build: Check for Python errors in terminal

**Note:** If you skip this step, input construction features will still work, but result analysis tools will return errors.

---

### Step 2: Configure the MCP Client to Launch Go MCP Server

Most desktop MCP clients launch local servers on-demand.

**Using Containers:**

Configure your client to run the MCP server in a container (example for Podman):

```
podman run -i --rm \
  --network host \
  -e ARCAFLOW_MCP_ANALYSIS_HTTP_URL=http://localhost:8081 \
  quay.io/arcalot/arcaflow-mcp-server:${TAG} \
  --mode local
```

**Building from Source:**

Point to the built binary:

```
./server/arcaflow-mcp --mode local
```

The client will manage the lifecycle (spawn on connect, terminate on disconnect).

### Configure an MCP client

Most MCP clients have a similar configuration flow for local servers. The
exact UI and file format vary, but the required information is usually the
same:

- **Name/ID:** A label like `arcaflow-mcp`.
- **Command:** Container command or executable path
- **Arguments:** Container args or `--mode local`
- **Working directory:** (containers don't need this)
- **Environment:** **Required** `ARCAFLOW_MCP_ANALYSIS_HTTP_URL=http://localhost:8081` for result analysis features

After saving the configuration, the client should start the server on demand
and complete MCP initialization automatically.

**Example: Using Containers (Podman)**

```json
{
  "name": "arcaflow-mcp",
  "command": "podman",
  "args": [
    "run", "-i", "--rm",
    "--network", "host",
    "-e", "ARCAFLOW_MCP_ANALYSIS_HTTP_URL=http://localhost:8081",
    "-e", "ARCAFLOW_MCP_LOG_LEVEL=info",
    "quay.io/arcalot/arcaflow-mcp-server:main-abc1234",
    "--mode", "local"
  ]
}
```

**Example: Using Containers (Docker)**

```json
{
  "name": "arcaflow-mcp",
  "command": "docker",
  "args": [
    "run", "-i", "--rm",
    "--network", "host",
    "-e", "ARCAFLOW_MCP_ANALYSIS_HTTP_URL=http://localhost:8081",
    "-e", "ARCAFLOW_MCP_LOG_LEVEL=info",
    "quay.io/arcalot/arcaflow-mcp-server:main-abc1234",
    "--mode", "local"
  ]
}
```

**Example: Built from Source**

```json
{
  "name": "arcaflow-mcp",
  "command": "/path/to/arcaflow-mcp/server/arcaflow-mcp",
  "args": ["--mode", "local"],
  "cwd": "/path/to/arcaflow-mcp",
  "env": {
    "ARCAFLOW_MCP_ANALYSIS_HTTP_URL": "http://localhost:8081",
    "ARCAFLOW_MCP_LOG_LEVEL": "info"
  }
}
```

**Note:** Replace `main-abc1234` with the actual current tag. Use `./scripts/get-container-tag.sh` to find it.

### Supported MCP methods

The server exposes core MCP protocol methods with workflow management tools:

- `initialize` and `initialized` for capability negotiation
- `tools/list` (includes `ping`)
- `tools/call` (supports `ping`)
- `resources/list` (returns cached resources, initially empty)
- `resources/read` (supports workflow, schema, example, plugin schema, and execution/log URIs)
- `ping`

### Notes

- The server enforces the `2025-11-25` MCP protocol version.
- After `initialize`, the client must send `initialized` before invoking other
  methods.
- Local mode logs JSON to stderr to keep MCP stdio framing intact.
- JSON-RPC notifications (no `id`) never receive a response.
- Method params must be JSON objects when provided; arrays are rejected.
- For compatibility with Cursor and Gemini, stdio accepts either MCP
  Content-Length framing or newline-delimited JSON.

### Gemini CLI and Cursor example

This configuration works for both Gemini CLI (`~/.gemini/settings.json`) and
Cursor (`~/.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "arcaflow-mcp": {
      "command": "/path/to/arcaflow-mcp/server/arcaflow-mcp",
      "args": ["--mode", "local"],
      "cwd": "/path/to/arcaflow-mcp",
      "env": {
        "ARCAFLOW_MCP_ANALYSIS_HTTP_URL": "http://localhost:8081",
        "ARCAFLOW_MCP_LOG_LEVEL": "info"
      },
      "trust": true
    }
  }
}
```

**Important:** Make sure the Python analysis engine is running on `localhost:8081` before starting your MCP client, or result analysis features will fail.

---

## Verify Your Setup

### Test the Connection

After configuring your MCP client:

1. **Start your AI client** (Claude Desktop, Cursor, etc.)

2. **Test basic connectivity:**
   ```
   "Can you see the Arcaflow MCP server? List the available tools."
   ```

3. **Expected response:** AI should list Arcaflow tools including:
   - `workflow_load`
   - `workflow_schema_get`
   - `workflow_input_build`
   - `workflow_input_validate`
   - `workflow_input_export`
   - And more...

4. **Test with example workflow:**
   ```
   "Load the hello-world workflow from /path/to/arcaflow-mcp/examples/workflows/hello-world"
   ```

### Common Connection Issues

**"MCP server not responding"**
- Verify binary/container path in configuration is correct
- Check that analysis engine is running (if using result analysis)
- Restart AI client after configuration changes
- Check AI client logs for error messages

**"Analysis engine not reachable"**
```bash
# Verify analysis engine is running
curl http://localhost:8081/health

# Check what's using port 8081
sudo lsof -i :8081

# Check container status (if using containers)
podman ps | grep arcaflow-analysis
```

---

## Cleanup and Shutdown

### Stop the Analysis Engine

**If using containers:**
```bash
podman stop arcaflow-analysis
podman rm arcaflow-analysis
```

**If built from source:**
```bash
# If you saved the PID
kill $ANALYSIS_PID

# Or find and kill the process
ps aux | grep "arcaflow_analysis"
kill <PID>
```

### Stop the MCP Server

The MCP server is launched and managed by your AI client. When you close the AI client, the server stops automatically.

For manual testing, press Ctrl+C to stop the server.
