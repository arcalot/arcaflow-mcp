# Getting Started: Local Mode (Containers)

Set up Arcaflow MCP for use with Claude Desktop, Cursor, Claude Code, or other
MCP-compatible AI clients. Estimated time: **~5 minutes**.

Arcaflow MCP has two components that work together:

| Component | Container Image | Purpose |
|-----------|----------------|---------|
| **Python Analysis Engine** | `arcaflow-mcp-analysis` | Result parsing and optimization suggestions |
| **Go MCP Server** | `arcaflow-mcp-server` | MCP protocol, workflow loading, input validation |

You will start the analysis engine as a background container, then configure
your AI client to launch the MCP server on demand.

---

## Prerequisites

- **Podman** or **Docker** installed and running
- An **MCP-compatible AI client** (Claude Desktop, Cursor, Claude Code, etc.)

> [!IMPORTANT]
> A container runtime (Podman or Docker) is also required for workflow input
> validation. The Arcaflow engine resolves plugin schemas by pulling container
> images. See [Configuration: Engine Deployer](usage/configuration.md#engine-deployer)
> for details.

---

## Step 1: Get the Container Tag

> [!WARNING]
> **Pre-release (before v0.1.0):** Container tags change with each commit.
> After v0.1.0, use `:latest` or version tags like `:v1.0.0`.

```bash
# If you have the repo cloned (uses your current branch and commit):
export TAG=$(./scripts/get-container-tag.sh)

# If you do NOT have the repo cloned (uses latest main branch commit):
export TAG=main-$(curl -s https://api.github.com/repos/arcalot/arcaflow-mcp/commits/main \
  | grep -m1 '"sha"' | cut -d'"' -f4 | cut -c1-7)

echo "Using tag: $TAG"
```

> [!TIP]
> After v0.1.0, skip this step and use `:latest` in the commands below.

---

## Step 2: Pull Container Images

```bash
podman pull quay.io/arcalot/arcaflow-mcp-server:${TAG}
podman pull quay.io/arcalot/arcaflow-mcp-analysis:${TAG}
```

Both images must use the same tag to ensure compatibility.

---

## Step 3: Start the Analysis Engine

Create a shared container network so the MCP server container can reach the
analysis engine by name (`arcaflow-analysis`) instead of relying on host port
mapping:

```bash
podman network create arcaflow 2>/dev/null || true
```

Start the analysis engine:

```bash
podman run -d \
  --name arcaflow-analysis \
  --network arcaflow \
  -p 8081:8081 \
  quay.io/arcalot/arcaflow-mcp-analysis:${TAG}
```

Verify it started successfully:

```bash
sleep 2
curl http://localhost:8081/healthz
```

Expected output: `{"status":"healthy"}`

> [!IMPORTANT]
> The analysis engine must be running before your AI client launches the MCP
> server, because the server connects to it at startup.

---

## Step 4: Configure Your MCP Client

Add the MCP server to your AI client configuration. The client will launch
the server container on demand and communicate with it over stdin/stdout.

**Claude Desktop** (`~/.config/Claude/claude_desktop_config.json` on Linux,
`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "arcaflow": {
      "command": "podman",
      "args": [
        "run", "-i", "--rm",
        "--network", "arcaflow",
        "-v", "/path/to/your/workflows:/workflows:ro",
        "-e", "ARCAFLOW_MCP_ANALYSIS_HTTP_URL=http://arcaflow-analysis:8081",
        "quay.io/arcalot/arcaflow-mcp-server:TAG_HERE",
        "--mode", "local"
      ]
    }
  }
}
```

Replace:
- `TAG_HERE` with your actual tag (the value of `echo $TAG`)
- `/path/to/your/workflows` with the directory containing your Arcaflow
  workflow files (e.g., the `examples/workflows` directory in this repo)

> [!IMPORTANT]
> The MCP server runs inside a container and cannot access host files
> without a volume mount. The `-v /host/path:/workflows:ro` flag maps
> a host directory into the container at `/workflows`. Use `/workflows`
> as the path when asking the AI to discover workflows.

> [!TIP]
> **If the `arcaflow` network doesn't work** (e.g., networking issues with
> Podman), use `--network host` instead and change the analysis URL to
> `http://localhost:8081`:
> ```json
> {
>   "mcpServers": {
>     "arcaflow": {
>       "command": "podman",
>       "args": [
>         "run", "-i", "--rm",
>         "--network", "host",
>         "-v", "/path/to/your/workflows:/workflows:ro",
>         "-e", "ARCAFLOW_MCP_ANALYSIS_HTTP_URL=http://localhost:8081",
>         "quay.io/arcalot/arcaflow-mcp-server:TAG_HERE",
>         "--mode", "local"
>       ]
>     }
>   }
> }
> ```

For other MCP clients (Cursor, Claude Code, etc.), see
[Local Mode Usage](usage/local-mode.md) for client-specific configuration.

Restart your AI client after saving the configuration.

---

## Try It Out

Verify the setup by asking your AI agent these questions in order:

1. **Check connection:** *"Can you see the Arcaflow MCP tools? List them."*
   - Expected: The agent lists tools like `workflow_list`,
     `workflow_input_validate`, `workflow_input_template`, etc.

2. **Discover workflows:** *"List the available workflows from the filesystem at /workflows"*
   - This uses the volume-mounted directory from Step 4.
   - Expected: The agent calls `workflow_list` and finds your workflow
     files. If you mounted `examples/workflows`, you'll see hello-world,
     data-processing, and perf-test.

3. **Get a workflow template:** *"Show me the input template for the hello-world workflow"*
   - Expected: The agent calls `workflow_input_template` and shows the
     input schema with a `name` field.

> [!NOTE]
> In local mode, there is no HTTP health endpoint for the MCP server — it
> communicates over stdin/stdout with your AI client. If the steps above
> don't work, check the [Troubleshooting Guide](troubleshooting.md).

---

## What's Next

- [Local Mode Usage](usage/local-mode.md) -- detailed configuration and all client options
- [Tutorial: Basic Workflow](examples/basic-workflow.md) -- complete input construction walkthrough
- [Configuration Reference](usage/configuration.md) -- engine deployer, logging, and more
- [Troubleshooting](troubleshooting.md) -- common issues and solutions
