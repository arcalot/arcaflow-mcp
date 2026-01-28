# Arcaflow MCP

**Natural Language Interface for Arcaflow Workflows**

Arcaflow MCP is a Model Context Protocol (MCP) server that enables conversational workflow input construction and intelligent result analysis for Arcaflow workflows.

## What is Arcaflow MCP?

Arcaflow MCP bridges the gap between natural language AI assistants (like Claude) and Arcaflow's powerful workflow engine. Instead of manually writing YAML configuration files, you can:

- **Build workflow inputs conversationally** - Describe what you want to test, and the AI helps construct valid, schema-compliant inputs
- **Analyze results intelligently** - Get AI-powered insights on workflow results, optimization suggestions, and comparative analysis
- **Iterate faster** - Quickly refine inputs based on results, without editing YAML by hand

## Key Features

**🎯 Conversational Input Construction**
- Discover workflows from local files, git repositories, or URLs
- Extract and understand complex workflow schemas automatically
- Build inputs through natural conversation with validation
- Export ready-to-use YAML/JSON inputs

**📊 Intelligent Result Analysis**
- Parse workflow execution results automatically
- Compare multiple runs to identify optimal configurations
- Generate actionable optimization suggestions
- Track performance trends across iterations

**🚀 Two Deployment Modes**
- **Local Mode**: Desktop AI clients (Claude Desktop, Cursor) with personal workflows
- **Server Mode**: Multi-tenant HTTP/SSE server for team deployments

**🔒 Enterprise-Ready**
- Multi-tenancy with workspace isolation
- Authentication and rate limiting
- Audit logging and usage tracking
- Podman/Docker and Kubernetes deployment

## Quick Start

### For Desktop AI Users (Local Mode)

```bash
# 1. Build the Go MCP server
cd server && go build -o arcaflow-mcp ./cmd/arcaflow-mcp

# 2. Start Python analysis engine (for result analysis features)
cd ../analysis
poetry install
poetry run python -m arcaflow_analysis.server.http_server &

# 3. Configure your AI client (e.g., Claude Desktop)
# Add to MCP configuration (~/.config/Claude/claude_desktop_config.json):
{
  "mcpServers": {
    "arcaflow": {
      "command": "/path/to/arcaflow-mcp",
      "args": ["--mode", "local"],
      "env": {
        "ARCAFLOW_MCP_ANALYSIS_HTTP_URL": "http://localhost:8081"
      }
    }
  }
}

# 4. Start conversing with your AI about Arcaflow workflows!
```

**Note:** The Python analysis engine is required for result analysis features. Input construction works without it.

**Next Steps:** [Local Mode Setup Guide](usage/local-mode.md)

### For Team/Production Deployments (Server Mode)

```bash
# Quick test deployment
export ARCAFLOW_MCP_ADMIN_TOKEN="your-secure-token"
export DATA_DIR="./data"

# Configure storage
export ARCAFLOW_MCP_TOKEN_STORE_PATH="$DATA_DIR/tokens.json"
export ARCAFLOW_MCP_TENANT_STORE_PATH="$DATA_DIR/tenants.json"
export ARCAFLOW_MCP_AUDIT_STORE_PATH="$DATA_DIR/audit.json"
export ARCAFLOW_MCP_USAGE_STORE_PATH="$DATA_DIR/usage.json"
export ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT="$DATA_DIR/tenants"

# Start Python analysis engine
cd analysis
poetry install
poetry run python -m arcaflow_analysis.server.http_server &
cd ..

# Start Go MCP server (connects to analysis engine)
export ARCAFLOW_MCP_ANALYSIS_HTTP_URL="http://localhost:8081"
./arcaflow-mcp --mode server --address :8080
```

**Next Steps:** [Server Mode Setup Guide](usage/server-mode.md)

## Documentation Structure

### 🎓 Learning Path

**New to Arcaflow MCP?** Follow this path:

1. **[Getting Started](getting-started.md)** - Installation and first workflow
2. **[Tutorial: Basic Workflow](examples/basic-workflow.md)** - Build your first workflow input
3. **[Tutorial: Result Analysis](examples/iterative-optimization.md)** - Analyze and optimize results
4. **[Tutorial: Multi-Run Comparison](examples/multi-run-comparison.md)** - Compare configurations

### 📖 Core Guides

**Understanding the System:**
- [Architecture](concepts/architecture.md) - How Arcaflow MCP works
- [Capabilities](concepts/capabilities.md) - What the server can do
- [Deployment Modes](concepts/deployment-modes.md) - Local vs Server mode
- [Multi-Tenancy](concepts/multi-tenancy.md) - Team and enterprise usage

**Using Arcaflow MCP:**
- [Local Mode](usage/local-mode.md) - Desktop AI client setup
- [Server Mode](usage/server-mode.md) - Team deployment and API usage
- [Input Construction](usage/input-construction.md) - Building workflow inputs
- [Result Analysis](usage/result-analysis.md) - Analyzing and optimizing results
- [Configuration Reference](usage/configuration.md) - All configuration options

### 🚀 Deployment

**Production Deployment:**
- [Container Deployment](deployment/container.md) - Podman/Docker setup
- [Kubernetes Deployment](deployment/kubernetes.md) - K8s manifests and configuration
- [Authentication Setup](deployment/authentication.md) - Securing your deployment
- [TLS Configuration](deployment/tls.md) - HTTPS and certificate management

### 🛠️ Reference

**Tools and APIs:**
- [Tool Overview](tools/overview.md) - All available MCP tools
- [Input Construction Tools](tools/input-tools.md) - Schema and validation tools
- [Result Analysis Tools](tools/result-tools.md) - Analysis and comparison tools
- [MCP Resources](tools/resources.md) - Workflow discovery and metadata

### 🆘 Help and Support

**Troubleshooting:**
- [FAQ](faq.md) - Frequently asked questions
- [Troubleshooting Guide](troubleshooting.md) - Common issues and solutions
- [GitHub Issues](https://github.com/arcalot/arcaflow-mcp/issues) - Report bugs or request features
- [Arcalot Community](https://github.com/arcalot/arcalot-round-table) - Join the discussion

## Use Cases

### Performance Testing and Optimization
Build and iterate on performance test configurations, analyze results, and optimize based on AI-powered suggestions. See [Tutorial: Iterative Optimization](examples/iterative-optimization.md).

### Configuration Exploration
Quickly explore different configuration combinations, compare results, and identify optimal settings. See [Tutorial: Multi-Run Comparison](examples/multi-run-comparison.md).

### Workflow Development
Develop and test workflows interactively, ensuring inputs are valid before execution. See [Getting Started](getting-started.md).

### Team Collaboration
Deploy as a shared service for teams to build inputs and analyze results together. See [Server Mode](usage/server-mode.md) and [Multi-Tenancy](concepts/multi-tenancy.md).

## Requirements

- **Local Mode:**
  - Go 1.23.0 (for building from source)
  - MCP-compatible AI client (Claude Desktop, Cursor, etc.)
  - Arcaflow Engine 0.20.0+ (optional, for workflow execution)

- **Server Mode:**
  - Go 1.23.0 (for building from source)
  - Python 3.12 with Poetry (for analysis engine)
  - Linux/macOS (tested on Fedora, Ubuntu, macOS)
  - Podman or Docker (optional, for containerized deployment)

## License

Apache License 2.0 - See [LICENSE](https://github.com/arcalot/arcaflow-mcp/blob/main/LICENSE)

## Contributing

Contributions welcome! See our [Contributing Guide](https://github.com/arcalot/arcaflow-mcp/blob/main/CONTRIBUTING.md).

---

**Part of the [Arcalot Project](https://arcalot.io)** - Tools for scalable, reproducible testing and analysis.
