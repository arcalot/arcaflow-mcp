# Getting Started: Build from Source

Build and run Arcaflow MCP from source for development and customization.
Estimated time: **~15 minutes**.

---

## Prerequisites

- **Go** 1.24+ ([install](https://go.dev/doc/install))
- **Python** 3.12+ with **Poetry** 1.8.3+ ([install Poetry](https://python-poetry.org/docs/#installation))
- **Git**
- **Podman** or **Docker**

> [!IMPORTANT]
> A container runtime is required even when building from source. The
> Arcaflow engine resolves plugin schemas by pulling container images
> during workflow input validation. See
> [Configuration: Engine Deployer](usage/configuration.md#engine-deployer).

---

## Step 1: Clone and Set Up

```bash
git clone https://github.com/arcalot/arcaflow-mcp.git
cd arcaflow-mcp

# Install git hooks and verify dependencies
./scripts/dev-setup.sh
```

---

## Step 2: Build the Go MCP Server

```bash
cd server
go build -o arcaflow-mcp ./cmd/arcaflow-mcp
cd ..
```

Verify the build:

```bash
./server/arcaflow-mcp --version
```

---

## Step 3: Install the Python Analysis Engine

```bash
cd analysis
poetry install
cd ..
```

---

## Step 4: Start Both Components

You need two terminals. Start the analysis engine first.

**Terminal 1 -- Analysis Engine:**

```bash
cd analysis
poetry run python -m arcaflow_analysis.server.http_server
```

Wait for the startup message, then verify:

```bash
curl http://localhost:8081/healthz
# Expected: {"status":"healthy"}
```

**Terminal 2 -- MCP Server (local mode):**

```bash
export ARCAFLOW_MCP_ANALYSIS_HTTP_URL="http://localhost:8081"
./server/arcaflow-mcp --mode local
```

The server is now running on stdin/stdout, ready for an MCP client.

> [!TIP]
> To configure your MCP client to use the locally-built server, point it
> at the binary instead of a container:
> ```json
> {
>   "mcpServers": {
>     "arcaflow": {
>       "command": "/full/path/to/arcaflow-mcp/server/arcaflow-mcp",
>       "args": ["--mode", "local"],
>       "env": {
>         "ARCAFLOW_MCP_ANALYSIS_HTTP_URL": "http://localhost:8081"
>       }
>     }
>   }
> }
> ```

> [!NOTE]
> **For server mode** instead of local mode, set additional environment
> variables and use `--mode server`:
> ```bash
> export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)"
> export ARCAFLOW_MCP_TOKEN_STORE_PATH="./data/tokens.json"
> export ARCAFLOW_MCP_TENANT_STORE_PATH="./data/tenants.json"
> export ARCAFLOW_MCP_AUDIT_STORE_PATH="./data/audit.json"
> export ARCAFLOW_MCP_USAGE_STORE_PATH="./data/usage.json"
> export ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT="./data/tenants"
> mkdir -p ./data/tenants
> ./server/arcaflow-mcp --mode server --address 127.0.0.1:8080
> ```
> Then follow [Getting Started: Server Mode](getting-started-server.md)
> from Step 5 to create tenants.

---

## Try It Out

If using local mode with an MCP client, verify the setup by asking your
AI agent:

1. *"Can you see the Arcaflow MCP tools? List them."*
   - Expected: tool list including `workflow_list`, `workflow_input_validate`, etc.

2. *"List the available workflows from the filesystem at examples/workflows"*
   - Expected: hello-world, data-processing, perf-test

3. *"Show me the input template for the hello-world workflow"*
   - Expected: input schema with a `name` field

If using server mode, verify with health checks:

```bash
curl http://localhost:8080/healthz
# Expected: {"status":"healthy","version":"..."}
```

---

## What's Next

- [Development Setup](../development/setup.md) -- testing, linting, debugging
- [Contributing Guide](../../CONTRIBUTING.md) -- how to submit changes
- [Local Mode Usage](usage/local-mode.md) -- detailed client configuration
- [Server Mode Usage](usage/server-mode.md) -- authentication, tenants, SSE
- [Troubleshooting](troubleshooting.md) -- common issues and solutions
