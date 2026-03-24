# Getting Started with Arcaflow MCP

This guide will get you up and running with Arcaflow MCP in minutes.

## Understanding the Architecture

**Arcaflow MCP uses TWO components that work together:**

| Component | Purpose | Required For |
|-----------|---------|--------------|
| **Go MCP Server** | MCP protocol handler, workflow loading, input validation | **Always required** |
| **Python Analysis Engine** | Result parsing, analysis, optimization suggestions | **Only for result analysis** |

**Component Dependencies:**

```
Your Goal                     → Components Needed
─────────────────────────────────────────────────
Build workflow inputs         → Go Server ONLY
Analyze workflow results      → Go Server + Python Engine (both)
Complete workflow lifecycle   → Go Server + Python Engine (both)
```

**Why two components?**
- **Go server**: Fast, efficient protocol handling and workflow operations
- **Python engine**: Rich data analysis and ML libraries for result optimization

**Startup Order Matters:**
1. **Start Python Analysis Engine FIRST** (if needed for result analysis)
2. **Then start/configure Go MCP Server** (connects to analysis engine at http://localhost:8081)

## Choose Your Path

Pick the deployment method that best fits your needs:

- **🐳 [Containers](#quick-start-with-containers-recommended)** - Fastest, easiest, works everywhere
- **📦 [Pre-Compiled Binaries](#quick-start-with-pre-compiled-binaries)** - Native performance, no containers needed (after v0.1.0)
- **🔧 [Build from Source](#quick-start-from-source-development)** - For development and customization

**New to Arcaflow MCP?** Start with containers for the quickest experience.

---

## Quick Start with Containers (Recommended)

The fastest way to try Arcaflow MCP is using pre-built container images for **both components**.

> **🚨 Pre-Release Container Tags (Before v0.1.0)**
>
> We're currently in active development. Container tags change with each commit to main.
>
> **Get the current tag** (recommended - ensures version alignment):
> ```bash
> # Automatically matches your repo/docs version
> export TAG=$(curl -s https://raw.githubusercontent.com/arcalot/arcaflow-mcp/main/scripts/get-container-tag.sh | bash)
> echo "Current tag: $TAG"
> ```
>
> **Why use the helper script?**
> - **Version alignment**: If you cloned the repo, uses YOUR git commit (e.g., `main-abc1234`)
> - **Single source**: Uses GitHub commit as source of truth, not quay.io registry
> - **Predictable**: Always returns the tag that SHOULD match your documentation
>
> **Alternative - Query quay.io directly** (simpler but may mismatch your docs):
> ```bash
> # Gets latest available tag from quay.io (might be newer than your docs version)
> export TAG=$(curl -s 'https://quay.io/api/v1/repository/arcalot/arcaflow-mcp-server/tag/' | \
>   jq -r '.tags[] | select(.name | startswith("main-")) | .name' | sort -V | tail -1)
> ```
> 
> The script provides **version alignment**: your docs match your container. With quay.io, you get "latest available" which might be newer.
>
> **After v0.1.0 release**, use stable tags:
> - `:latest` - Latest stable build
> - `:v1.0.0` - Specific version releases

### Prerequisites

- **Container Runtime**: Podman or Docker
- **MCP Client** (for local mode): Claude Desktop, Cursor, or similar

### 1. Pull Pre-Built Images for BOTH Components

**Pull Latest Images:**

```bash
# Get current development tag (before v0.1.0)
export TAG=$(curl -s https://raw.githubusercontent.com/arcalot/arcaflow-mcp/main/scripts/get-container-tag.sh | bash)

# Pull COMPONENT 1: Go MCP Server
podman pull quay.io/arcalot/arcaflow-mcp-server:${TAG}

# Pull COMPONENT 2: Python Analysis Engine
podman pull quay.io/arcalot/arcaflow-mcp-analysis:${TAG}
```

**Finding Specific Version Tags:**

You can also query our Quay.io repositories to find specific version tags:

```bash
# List available tags using skopeo
skopeo list-tags docker://quay.io/arcalot/arcaflow-mcp-server
skopeo list-tags docker://quay.io/arcalot/arcaflow-mcp-analysis

# Or query Quay.io API directly
curl -s 'https://quay.io/api/v1/repository/arcalot/arcaflow-mcp-server/tag/?limit=100' | \
  jq -r '.tags[].name' | sort -V
curl -s 'https://quay.io/api/v1/repository/arcalot/arcaflow-mcp-analysis/tag/?limit=100' | \
  jq -r '.tags[].name' | sort -V

# Then pull a specific version
podman pull quay.io/arcalot/arcaflow-mcp-server:v1.0.0
podman pull quay.io/arcalot/arcaflow-mcp-analysis:v1.0.0
```

**Browse All Tags**: [quay.io/arcalot](https://quay.io/organization/arcalot)

**Note**: Both images use the same tag. Always pull both to ensure compatibility.

### 2. Choose Your Deployment Mode

#### Option A: Local Mode (Desktop AI Clients)

For use with Claude Desktop, Cursor, or other MCP-compatible clients.

**Setup: Start BOTH components**

**Component 1 - Start Python Analysis Engine FIRST (Background Service):**

```bash
# Get current tag (if not already set)
export TAG=${TAG:-$(curl -s https://raw.githubusercontent.com/arcalot/arcaflow-mcp/main/scripts/get-container-tag.sh | bash)}

# Create network for component communication
podman network create arcaflow 2>/dev/null || true

# Start analysis engine on port 8081
podman run -d \
  --name arcaflow-analysis \
  --network arcaflow \
  -p 8081:8081 \
  quay.io/arcalot/arcaflow-mcp-analysis:${TAG}

# Wait a moment for startup
sleep 2

# Verify it's running and healthy
curl http://localhost:8081/health
# Expected: {"status":"healthy"}
```

**⚠️ Important**: The analysis engine MUST be running before starting the Go MCP server.

**Component 2 - Configure MCP Client to Launch Go MCP Server:**

The AI client will start the Go MCP server on-demand. It connects to the analysis engine.

Add to your MCP client configuration (e.g., `~/.config/Claude/claude_desktop_config.json`):

**Note**: Replace `${TAG}` with the actual tag you pulled (e.g., `main-abc1234`):

```json
{
  "mcpServers": {
    "arcaflow": {
      "command": "podman",
      "args": [
        "run",
        "-i",
        "--rm",
        "--network", "arcaflow",
        "-e", "ARCAFLOW_MCP_ANALYSIS_HTTP_URL=http://arcaflow-analysis:8081",
        "quay.io/arcalot/arcaflow-mcp-server:main-abc1234",
        "--mode", "local"
      ]
    }
  }
}
```

**Alternative using host network** (if arcaflow network doesn't work):

```json
{
  "mcpServers": {
    "arcaflow": {
      "command": "podman",
      "args": [
        "run",
        "-i",
        "--rm",
        "--network", "host",
        "-e", "ARCAFLOW_MCP_ANALYSIS_HTTP_URL=http://localhost:8081",
        "quay.io/arcalot/arcaflow-mcp-server:main-abc1234",
        "--mode", "local"
      ]
    }
  }
}
```

**Both components are now connected!** The MCP server communicates with the analysis engine.

**Next Steps**: See [Local Mode Setup](usage/local-mode.md) for detailed configuration.

#### Option B: Server Mode (Multi-Tenant Deployments)

For team deployments with multiple users. Deploys BOTH components as services.

**What is multi-tenancy?** Each team or user group gets an isolated workspace
called a "tenant" with independent authentication tokens and data isolation.
After deploying containers, you'll create tenants and tokens for your users.

**Using Docker Compose or Podman Compose:**

The repository includes a production-ready compose file that deploys both components with proper health checks, volumes, and networking.

```bash
# Method 1: Clone repository
git clone https://github.com/arcalot/arcaflow-mcp.git
cd arcaflow-mcp/deploy/container

# Method 2: Download compose file directly
curl -O https://raw.githubusercontent.com/arcalot/arcaflow-mcp/main/deploy/container/compose.yml

# Generate secure admin token (REQUIRED)
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)"
echo "Save this token: $ARCAFLOW_MCP_ADMIN_TOKEN"

# Optional: Use development tag (before v0.1.0)
# export TAG="main-abc1234"  # Replace with actual commit SHA

# Start BOTH services together
docker compose up -d

# Or with Podman:
# podman-compose up -d
```

**What the compose file deploys:**
- **Component 1**: Python Analysis Engine on port 8081
- **Component 2**: Go MCP Server on port 8080 (connects to analysis engine)
- **Persistent volumes**: Data survives container restarts
- **Health checks**: Ensures both components are ready
- **Proper networking**: Components can communicate

**Verify BOTH Components Are Running:**

```bash
# Check service status
docker compose ps

# Check analysis engine health
curl http://localhost:8081/health
# Expected: {"status":"healthy"}

# Check MCP server health
curl http://localhost:8080/health
# Expected: {"status":"healthy","version":"..."}

# View logs
docker compose logs -f

# Both components are running and connected!
```

**Create Your First Tenant** (required for access):

```bash
# Create tenant
curl -X POST http://localhost:8080/admin/tenants \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"my-team","display_name":"My Team"}'

# Create tenant token (users will use this)
curl -X POST http://localhost:8080/admin/tenants/my-team/tokens \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' | jq -r '.token'
```

**Next Steps**: 
- 📖 **Complete tenant setup**: [Authentication Guide](deployment/authentication.md)
- 📖 **Configure clients**: [Server Mode Setup](usage/server-mode.md)
- 📖 **Advanced configuration**: [Container Deployment Guide](deployment/container.md)

---

## Quick Start with Pre-Compiled Binaries

Download pre-built binaries for your platform (available after v0.1.0 release).

### Prerequisites

- **Go MCP Server**: Download binary for your OS/architecture
- **Python Analysis Engine**: Python 3.12+ with pip
- **MCP Client** (for local mode): Claude Desktop, Cursor, or similar

### 1. Download Go MCP Server Binary

Choose your platform:

**Linux (x86_64):**
```bash
curl -L https://github.com/arcalot/arcaflow-mcp/releases/latest/download/arcaflow-mcp-linux-amd64 -o arcaflow-mcp
chmod +x arcaflow-mcp
sudo mv arcaflow-mcp /usr/local/bin/
```

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/arcalot/arcaflow-mcp/releases/latest/download/arcaflow-mcp-darwin-arm64 -o arcaflow-mcp
chmod +x arcaflow-mcp
sudo mv arcaflow-mcp /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -L https://github.com/arcalot/arcaflow-mcp/releases/latest/download/arcaflow-mcp-darwin-amd64 -o arcaflow-mcp
chmod +x arcaflow-mcp
sudo mv arcaflow-mcp /usr/local/bin/
```

**Windows (x86_64):**
```powershell
# Download from: https://github.com/arcalot/arcaflow-mcp/releases/latest
# Or use PowerShell:
Invoke-WebRequest -Uri "https://github.com/arcalot/arcaflow-mcp/releases/latest/download/arcaflow-mcp-windows-amd64.exe" -OutFile "arcaflow-mcp.exe"
# Move to a directory in your PATH
```

**Browse All Releases**: [github.com/arcalot/arcaflow-mcp/releases](https://github.com/arcalot/arcaflow-mcp/releases)

### 2. Install Python Analysis Engine

The Python analysis engine currently requires installation from source or containers. PyPI distribution planned for v0.1.0+.

**Current Method (From Source):**

```bash
# Clone repository
git clone https://github.com/arcalot/arcaflow-mcp.git
cd arcaflow-mcp/analysis

# Install with Poetry
poetry install

# Run the server
poetry run python -m arcaflow_analysis.server.http_server
```

**Future Method (After PyPI automation is complete):**

```bash
# Will be available in a future release:
pip install arcaflow-analysis
arcaflow-analysis-server
```

### 3. Run Components

**Local Mode:**

```bash
# Terminal 1: Start analysis engine (from source)
cd arcaflow-mcp/analysis
poetry run python -m arcaflow_analysis.server.http_server &

# Verify it's running
curl http://localhost:8081/health

# Terminal 2: Configure your MCP client to launch the Go binary
# Example: ./arcaflow-mcp
# with env: ARCAFLOW_MCP_ANALYSIS_HTTP_URL=http://localhost:8081
```

**Server Mode:**

```bash
# Set admin token
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)"

# Terminal 1: Start analysis engine (from source)
cd arcaflow-mcp/analysis
poetry run python -m arcaflow_analysis.server.http_server

# Terminal 2: Start MCP server (using downloaded binary)
./arcaflow-mcp --mode server
```

**Note**: Before the first release (v0.1.0), use containers or build from source instead.

---

## Quick Start from Source (Development)

For developers working on Arcaflow MCP itself. Requires building BOTH components.

### Prerequisites

- **Go** 1.23.0+ (see [project README](../../README.md#prerequisites))
- **Python** 3.12+ with Poetry
- **Git**
- **Podman** or **Docker** — required for workflow input validation (the engine resolves plugin schemas from container images)

### 1. Clone and Setup

```bash
# Clone repository
git clone https://github.com/arcalot/arcaflow-mcp.git
cd arcaflow-mcp

# Run development setup (installs git hooks, dependencies)
./scripts/dev-setup.sh
```

### 2. Build BOTH Components

```bash
# Build COMPONENT 1: Go MCP Server
cd server
go build -o arcaflow-mcp ./cmd/arcaflow-mcp
cd ..

# Install COMPONENT 2: Python Analysis Engine
cd analysis
poetry install
cd ..
```

### 3. Start BOTH Services

You need **two terminals** to run both components:

**Terminal 1 - Start Component 2 (Python Analysis Engine):**

```bash
cd analysis
poetry run python -m arcaflow_analysis.server.http_server
# Runs on http://localhost:8081
```

**Terminal 2 - Start Component 1 (Go MCP Server):**

```bash
# Local mode (for MCP clients)
export ARCAFLOW_MCP_ANALYSIS_HTTP_URL="http://localhost:8081"
./server/arcaflow-mcp --mode local

# OR Server mode (for HTTP/SSE)
export ARCAFLOW_MCP_ADMIN_TOKEN="dev-token-$(date +%s)"
export ARCAFLOW_MCP_ANALYSIS_HTTP_URL="http://localhost:8081"
./server/arcaflow-mcp --mode server --address :8080
```

**Both components are now running!** The MCP server connects to the analysis engine at http://localhost:8081.

**Next Steps**: See [Development Setup](../development/setup.md) for full development workflow.

---

## Verify Your Setup

After deployment, verify both components are working correctly.

### Quick Health Check

**For Container Deployments:**

```bash
# Check Analysis Engine (if using result analysis features)
curl http://localhost:8081/health
# Expected: {"status":"healthy"}

# Check MCP Server (server mode only)
curl http://localhost:8080/health
# Expected: {"status":"healthy","version":"..."}
```

**For Local Mode (Claude Desktop):**

Open your AI client and try:
```
"Can you see the Arcaflow MCP server?"
```

The AI should list Arcaflow tools if connection is successful.

### Full MCP Protocol Test (Server Mode)

Test the complete MCP handshake:

```bash
# Step 1: Initialize MCP session
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'

# Expected: {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-11-25",...}}

# Step 2: Complete initialization
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d '{"jsonrpc":"2.0","method":"initialized"}'

# Step 3: List available tools (verifies server is ready)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'

# Expected: {"jsonrpc":"2.0","id":2,"result":{"tools":[...]}}
```

---

## Your First Workflow

Now that your server is running, let's load and work with a real workflow.

### Using Built-In Example Workflows

The Arcaflow MCP repository includes example workflows in the `examples/workflows/` directory:

- **hello-world/** - Simple single-input workflow (perfect for first test)
- **data-processing/** - More complex workflow with multiple inputs
- **perf-test/** - Advanced workflow for performance testing

### Complete First-Time Workflow

**For Claude Desktop (Local Mode):**

1. **Load the example workflow:**
   ```
   "Load the hello-world workflow from <your-path>/arcaflow-mcp/examples/workflows/hello-world"
   ```

2. **Ask about the schema:**
   ```
   "What inputs does this workflow require?"
   ```

3. **Build inputs conversationally:**
   ```
   "Build inputs for this workflow. Use the name 'Alice'"
   ```

4. **Validate:**
   ```
   "Validate these inputs"
   ```

5. **Export for use:**
   ```
   "Export these inputs to /tmp/hello-inputs.yaml"
   ```

**You've just built your first workflow input!** The exported file is ready to use with Arcaflow Engine.

### For Server Mode (curl):

Complete example loading the hello-world workflow:

```bash
# Set workflow path (adjust to your installation location)
WORKFLOW_DIR="/path/to/arcaflow-mcp/examples/workflows/hello-world"

# Initialize MCP (required handshake)
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'

curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARCAFLOW_MCP_ADMIN_TOKEN" \
  -d '{"jsonrpc":"2.0","method":"initialized"}'

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
        "location": "$WORKFLOW_DIR"
      },
      "selector": {
        "path": "workflow.yaml"
      }
    }
  }
}
EOF
```

### Need More Workflows?

**Full Arcaflow Workflows Repository:**

```bash
# Clone the official workflows repository
git clone https://github.com/arcalot/arcaflow-workflows
cd arcaflow-workflows

# Explore available workflows
ls -la
```

Then load any workflow with your AI client or MCP server.

---

## Cleanup and Shutdown

### Stopping Services

**For Docker Compose:**
```bash
docker compose stop          # Stop containers (preserves data)
docker compose down          # Stop and remove containers
docker compose down -v       # Stop, remove containers AND volumes (deletes data)
```

**For Podman Compose:**
```bash
podman-compose stop
podman-compose down
```

**For Background Processes (built from source):**
```bash
# If you started analysis engine in background:
ps aux | grep "arcaflow_analysis"
kill <PID>

# Or if you saved the PID:
kill $ANALYSIS_PID
```

**For Individual Containers:**
```bash
podman stop arcaflow-analysis
podman stop arcaflow-mcp-server

# Remove containers
podman rm arcaflow-analysis arcaflow-mcp-server
```

---

## Troubleshooting

### Containers Won't Start

**Check logs:**

```bash
podman logs arcaflow-mcp-server
podman logs arcaflow-analysis
```

**Common issues:**
- Port 8080 or 8081 already in use
- Missing admin token (server mode)
- Analysis engine not reachable

### Can't Find Current Development Tag

**Using the helper script** (easiest):

```bash
# Clone repository (if not already)
git clone https://github.com/arcalot/arcaflow-mcp.git
cd arcaflow-mcp

# Get current tag
export TAG=$(./scripts/get-container-tag.sh)
echo "Using tag: $TAG"

# Or get latest main branch tag (without cloning)
export TAG=$(curl -s https://raw.githubusercontent.com/arcalot/arcaflow-mcp/main/scripts/get-container-tag.sh | bash -s -- --main)
```

**Manual method:**

```bash
# From repository
git rev-parse --short=7 HEAD

# From GitHub (no clone needed)
curl -s https://api.github.com/repos/arcalot/arcaflow-mcp/commits/main | \
  jq -r '.sha[:7]'
```

**Use as tag:**

```bash
export TAG="main-$(curl -s https://api.github.com/repos/arcalot/arcaflow-mcp/commits/main | jq -r '.sha[:7]')"
echo "Using tag: $TAG"
```

### More Help

- **[Troubleshooting Guide](troubleshooting.md)** - Common issues and solutions
- **[FAQ](faq.md)** - Frequently asked questions
- **[GitHub Issues](https://github.com/arcalot/arcaflow-mcp/issues)** - Report bugs

---

## Troubleshooting

### Common Issues

#### Port Conflicts

**Problem:** Error about ports 8080 or 8081 already in use

**Solution:**
```bash
# Check what's using the ports
sudo lsof -i :8080
sudo lsof -i :8081

# Option 1: Stop conflicting service
# Option 2: Use different ports with -p flag
podman run -d -p 8082:8081 quay.io/arcalot/arcaflow-mcp-analysis:latest
```

#### Missing Admin Token

**Problem:** Container compose fails with "ARCAFLOW_MCP_ADMIN_TOKEN is required"

**Solution:**
```bash
# Generate and export token before running compose
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -base64 32)"
echo "Save this token: $ARCAFLOW_MCP_ADMIN_TOKEN"
```

#### Components Can't Communicate

**Problem:** MCP server can't reach analysis engine

**Solution:**
```bash
# Verify analysis engine is healthy
curl http://localhost:8081/health

# Check container networking (Docker/Podman Compose)
docker compose ps  # Both should show "healthy"

# For manual containers, ensure same network or use --link
podman run --network container:arcaflow-analysis ...
```

#### Health Check Failures

**Problem:** Containers restart repeatedly, health checks failing

**Solution:**
```bash
# Check container logs
docker compose logs analysis
docker compose logs mcp-server

# Common causes:
# - Missing required environment variables
# - Volume permission issues (SELinux)
# - Insufficient container resources
```

#### Workflow Directory Mount Issues

**Problem:** Compose fails with "no such file or directory: ./workflows"

**Solution:**
```bash
# Create the directory
mkdir -p ./workflows

# Or comment out the volume mount in compose.yml if not needed
```

### Getting Help

- **Documentation:** Full guides at [arcalot.io/arcaflow](https://arcalot.io/arcaflow/)
- **Issues:** Report at [github.com/arcalot/arcaflow-mcp/issues](https://github.com/arcalot/arcaflow-mcp/issues)
- **Community:** Join discussions at [github.com/arcalot/arcalot-round-table](https://github.com/arcalot/arcalot-round-table)

---

## Next Steps

### Learn the Basics

- **[Concepts: Architecture](concepts/architecture.md)** - How Arcaflow MCP works
- **[Concepts: Capabilities](concepts/capabilities.md)** - Input construction and result analysis
- **[Concepts: Deployment Modes](concepts/deployment-modes.md)** - Local vs Server mode

### Dive Deeper

- **[Local Mode Guide](usage/local-mode.md)** - Configure desktop AI clients
- **[Server Mode Guide](usage/server-mode.md)** - Deploy for teams
- **[Container Deployment](deployment/container.md)** - Production container setup
- **[Kubernetes Deployment](deployment/kubernetes.md)** - Kubernetes manifests

### Try Tutorials

- **[Tutorial 1: Basic Workflow](examples/basic-workflow.md)** - Build your first input
- **[Tutorial 2: Iterative Optimization](examples/iterative-optimization.md)** - Analyze and optimize
- **[Tutorial 3: Multi-Run Comparison](examples/multi-run-comparison.md)** - Compare configurations

---

**Ready to explore?** Connect your AI client and start conversing about Arcaflow workflows!
