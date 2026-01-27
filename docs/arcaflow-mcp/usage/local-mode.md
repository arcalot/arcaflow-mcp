## Local mode

Local mode runs the MCP server over stdio using MCP Content-Length framing. It
is intended for desktop MCP clients that launch the server as a subprocess.

### Start the server

Most desktop MCP clients launch local servers on-demand. In that case, the
client configuration should point to this command and the client will manage
the lifecycle (spawn on connect, terminate on disconnect):

```
./server/arcaflow-mcp --mode local
```

### Configure an MCP client

Most MCP clients have a similar configuration flow for local servers. The
exact UI and file format vary, but the required information is usually the
same:

- **Name/ID:** A label like `arcaflow-mcp`.
- **Command:** The executable path, for example `./server/arcaflow-mcp`.
- **Arguments:** `--mode local` (plus `--config` if you use a config file).
- **Working directory:** The repo root so relative paths resolve correctly.
- **Environment:** Optional overrides such as `ARCAFLOW_MCP_LOG_LEVEL=debug`.

After saving the configuration, the client should start the server on demand
and complete MCP initialization automatically. If the client expects a JSON
config file, look for an MCP or "local servers" section and provide the command
and arguments there.

Example JSON-style entry (field names may vary by client):

```json
{
  "name": "arcaflow-mcp",
  "command": "./server/arcaflow-mcp",
  "args": ["--mode", "local"],
  "cwd": "/path/to/arcaflow-mcp",
  "env": {
    "ARCAFLOW_MCP_LOG_LEVEL": "info"
  }
}
```

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
        "ARCAFLOW_MCP_LOG_LEVEL": "info"
      },
      "trust": true
    }
  }
}
```
