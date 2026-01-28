# Frequently Asked Questions (FAQ)

Common questions about the Arcaflow MCP server.

[← Back to Documentation Index](index.md) | [← Back to README](../../README.md)

## General Questions

### What is Arcaflow MCP?

Arcaflow MCP is a Model Context Protocol (MCP) server that enables natural
language conversations with AI agents (like Claude, Gemini) to build and
validate Arcaflow workflow inputs, and analyze workflow execution results.

It bridges the gap between conversational AI and machine-readable workflow
configurations, ensuring all inputs are deterministic and schema-validated.

**Learn more:** [Capabilities](concepts/capabilities.md)

---

### What can Arcaflow MCP do?

Current capabilities:

- **Input Construction**: Build workflow inputs through natural conversation
- **Schema Validation**: Ensure inputs match workflow requirements (100%)
- **Result Analysis**: Analyze workflow outputs and suggest optimizations
- **Multi-Run Comparison**: Compare multiple executions to find optimal configs
- **Workflow Discovery**: Find workflows in filesystem, URLs, and git repos

**Learn more:** [Capabilities](concepts/capabilities.md)

---

### What can't Arcaflow MCP do (yet)?

Not yet available:

- **Workflow Execution**: Run workflows (planned for future release)
- **Iterative Optimization**: Automated optimization loops (planned)
- **Workflow Creation**: Generate new workflow YAML (planned for future release)

Users currently run workflows externally and bring results back for analysis.

---

### How is this different from running Arcaflow directly?

Arcaflow MCP doesn't replace Arcaflow - it enhances it:

**Traditional Arcaflow:**
- Hand-write workflow YAML configuration
- Manually construct JSON/YAML input files
- Run workflow with `arcaflow` command
- Manually analyze results

**With Arcaflow MCP:**
- Describe your intent conversationally to AI
- AI helps build valid inputs through dialogue
- Export schema-validated inputs for Arcaflow
- AI analyzes results and suggests improvements
- Iterate based on AI recommendations

You still run Arcaflow workflows normally. MCP helps with input preparation and
result analysis.

**Learn more:** [Getting Started](getting-started.md)

---

## Deployment Questions

### Should I use local mode or server mode?

**Use Local Mode if:**
- Using desktop AI clients (Claude Desktop, Cursor, etc.)
- Single user
- Development and testing
- No network deployment needed

**Use Server Mode if:**
- Multiple users / teams
- Central deployment (Kubernetes, VMs)
- Need authentication and multi-tenancy
- Production usage
- Remote AI client access

Most users start with local mode.

**Learn more:** [Deployment Modes](concepts/deployment-modes.md)

---

### Can I use both local and server mode?

Yes! They're independent deployment options:
- Use local mode for personal development
- Connect to shared server mode for team workflows
- Run multiple instances with different configurations

**Learn more:** [Local Mode](usage/local-mode.md), [Server Mode](usage/server-mode.md)

---

### Why does server mode fail with "permission denied"?

**Problem:** Server fails to start with error about `/var/lib/arcaflow-mcp` permissions.

**Cause:** By default, the server uses `/var/lib/arcaflow-mcp` for data storage (tokens, tenants, audit logs, usage stats), which requires root permissions.

**Solutions:**

**Development/Testing** (use local data directory - copy-paste ready):
```bash
# Set data directory location (change this path if needed)
export DATA_DIR="./data"

# Create and configure
mkdir -p "$DATA_DIR"
export ARCAFLOW_MCP_ADMIN_TOKEN="dev-test-token-12345"
export ARCAFLOW_MCP_TOKEN_STORE_PATH="$DATA_DIR/tokens.json"
export ARCAFLOW_MCP_TENANT_STORE_PATH="$DATA_DIR/tenants.json"
export ARCAFLOW_MCP_AUDIT_STORE_PATH="$DATA_DIR/audit.json"
export ARCAFLOW_MCP_USAGE_STORE_PATH="$DATA_DIR/usage.json"
export ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT="$DATA_DIR/tenants"

# Start server
./server/arcaflow-mcp --mode server --address :8080
```

**Production** (configure proper permissions - commands will determine your user/group):
```bash
# Create directory with your ownership
sudo mkdir -p /var/lib/arcaflow-mcp
sudo chown $(whoami):$(id -gn) /var/lib/arcaflow-mcp
sudo chmod 750 /var/lib/arcaflow-mcp

# Generate secure token and start server
export ARCAFLOW_MCP_ADMIN_TOKEN="$(openssl rand -hex 32)"
./server/arcaflow-mcp --mode server --address :8080
```

**Learn more:** [Server Mode Setup](usage/server-mode.md), [Troubleshooting](troubleshooting.md#permission-denied-on-startup)

---

### How do I test the server is working?

After starting the server, test with curl following MCP protocol requirements:

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
    "clientInfo": {"name": "test", "version": "1.0"}
  }
}
EOF

# Expected: {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-11-25",...}}

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

# Step 3: List available tools (now this works!)
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

# Expected: {"jsonrpc":"2.0","id":2,"result":{"tools":[...]}}
```

**Important:**
- **MCP requires two-step handshake**: (1) `initialize` request, then (2) `initialized` notification
- **Session state**: The examples above establish protocol state for that connection only
- **Multi-step workflows**: Use SSE session binding to maintain state across multiple requests
- **AI clients**: Claude Desktop and other MCP clients automatically handle session binding

**Learn more:** [Server Mode Testing](usage/server-mode.md#quick-start), [Session Binding](usage/server-mode.md#session-binding), [Troubleshooting](troubleshooting.md#server-not-initialized-error)

---

### What AI clients are supported?

Any MCP-compatible client:

**Tested and Supported:**
- Claude Desktop (Anthropic)
- Cursor IDE
- Other MCP-compliant clients

**Requirements:**
- MCP 2025-11-25 specification support
- stdio transport (local mode) or HTTP/SSE (server mode)
- JSON-RPC 2.0

**Learn more:** [Local Mode Setup](usage/local-mode.md)

---

### Do I need Arcaflow Engine installed?

**For Input Construction Only:** No
- MCP server validates inputs without running workflows
- You can build and export inputs without Arcaflow installed

**For Actual Workflow Execution:** Yes
- You need Arcaflow Engine to run workflows
- MCP exports valid inputs that Arcaflow Engine consumes
- Install from: https://github.com/arcalot/arcaflow-engine

**For Result Analysis:** No
- MCP can analyze any workflow output files
- No Arcaflow Engine required for analysis

**Learn more:** [Getting Started](getting-started.md)

---

## Technical Questions

### What languages and versions are required?

**Runtime Requirements:**
- Go (see `ARCALOT_GO_VERSION` GitHub Organization variable)
- Python (see `ARCALOT_PYTHON_SUPPORTED_VERSIONS` GitHub Organization variable)
- Poetry (latest stable)

**Finding current versions:**
Check [project README](../../README.md#prerequisites) or `.github/workflows/ci.yml` for exact current versions.

**Why specific versions?**
- Aligns with Arcaflow project standards
- Ensures compatibility with Arcaflow workflows
- Reproducible builds

**Learn more:** [Prerequisites](../../README.md#prerequisites)

---

### What's the hybrid Go + Python architecture?

**Go MCP Server:**
- Handles MCP protocol (JSON-RPC 2.0)
- Manages transports (stdio, HTTP/SSE)
- Implements authentication and multi-tenancy
- Performs input construction and validation

**Python Analysis Engine:**
- Analyzes workflow results
- Generates optimization suggestions
- Pattern recognition and learning
- Statistical analysis

**Why hybrid?**
- Go excels at server infrastructure
- Python excels at data analysis and AI
- Each component uses the best tool for its job

**Learn more:** [Architecture Overview](concepts/architecture.md)

---

### How does authentication work?

**Local Mode:** No authentication
- Server launched on-demand by AI client
- Communication via stdio (local process)
- Isolated per user session

**Server Mode:** Bearer token authentication
- Each tenant has unique token
- Token passed in Authorization header
- Multi-tenant workspace isolation
- Rate limiting per tenant

**Learn more:** [Authentication Setup](deployment/authentication.md)

---

### Is it secure for multi-user deployments?

Server mode includes security features:

- Bearer token authentication
- Multi-tenant workspace isolation
- Rate limiting per tenant
- Audit logging for all operations
- TLS 1.3 support
- Input validation at all boundaries

**Important:** Follow security best practices:
- Use HTTPS/TLS in production
- Rotate tokens regularly
- Monitor audit logs
- Keep software updated

**Learn more:** [Multi-tenancy](concepts/multi-tenancy.md), [Security](../../SECURITY.md)

---

### What's the performance like?

**Measured Performance:**
- Protocol overhead: <100ms (local), <200ms (server)
- Schema extraction: <500ms
- Input validation: <100ms
- Result analysis: <3s (typical)
- Multi-run comparison: <5s (5 runs)

**Scalability (Server Mode):**
- 100+ concurrent tenant connections
- Handles high-throughput multi-tenant workloads
- Tenant isolation maintained under load

**Learn more:** [Configuration](usage/configuration.md)

---

## Workflow Questions

### What workflows are supported?

Any valid Arcaflow workflow YAML file:

- Single-step workflows
- Multi-step workflows  
- Workflows with sub-workflows
- Complex DAG workflows
- Workflows from any source (filesystem, URL, git)

**Requirements:**
- Valid Arcaflow workflow YAML format
- Workflow has `input` schema defined
- Plugins/steps are properly defined

**Learn more:** [Input Construction Guide](usage/input-construction.md)

---

### Can I use workflows from git repositories?

Yes! Workflows can be loaded from:

- Filesystem paths
- URLs (direct workflow YAML)
- Git repositories (with branch/tag/commit refs)

**Example:**
```json
{
  "source": {
    "kind": "git",
    "location": "https://github.com/org/workflows.git",
    "ref": "main",
    "subdir": "examples/"
  },
  "selector": {
    "path": "my-workflow.yaml"
  }
}
```

**Learn more:** [Input Construction Guide](usage/input-construction.md)

---

### How do I find workflows I haven't seen before?

Use the `workflow_discover` tool:

1. Point to a source (filesystem, git, URL)
2. MCP searches for workflow YAML files
3. Returns list of discovered workflows with metadata
4. Select one to load

**Learn more:** [Input Construction Guide](usage/input-construction.md)

---

### Do inputs always validate before export?

**Yes - 100% validation enforcement.**

Inputs CANNOT be exported without passing schema validation:
- All required fields must be present
- All field types must match schema
- All constraints must be satisfied
- Validation is deterministic and repeatable

This ensures exported inputs always work with Arcaflow.

**Learn more:** [Input Construction Guide](usage/input-construction.md)

---

## Result Analysis Questions

### What result formats are supported?

**Supported Formats:**
- JSON (workflow output)
- YAML (workflow output)
- Plain text logs (for error analysis)

**Result Source:**
- Local filesystem
- URLs (future)
- Data stores via MCP (future enhancement)

**Learn more:** [Result Analysis Guide](usage/result-analysis.md)

---

### Can I analyze results from multiple runs?

Yes! Use multi-run comparison:

1. Load multiple result files
2. MCP extracts metrics from each
3. Compares results across configurations
4. Identifies patterns and optimal settings
5. Recommends best configuration

**Learn more:** [Multi-Run Comparison Tutorial](examples/multi-run-comparison.md)

---

### How does AI generate optimization suggestions?

The Python analysis engine:

1. Parses workflow results
2. Extracts key metrics
3. Compares against user-defined goals
4. Analyzes patterns across multiple runs
5. Generates recommendations based on:
   - Metric trends
   - Goal achievement progress
   - Historical patterns
   - Best practices

Suggestions explain the rationale behind recommendations.

**Learn more:** [Result Analysis Guide](usage/result-analysis.md)

---

## Contributing Questions

### How can I contribute?

We welcome contributions!

**Getting Started:**
1. Read [CONTRIBUTING.md](../../CONTRIBUTING.md)
2. Review [AGENTS.md](../../AGENTS.md) for coding standards
3. Set up development environment: [Development Setup](../development/setup.md)
4. Pick an issue or propose a feature

**What to contribute:**
- Bug fixes
- New features (discuss first)
- Documentation improvements
- Test coverage
- Performance optimizations

**Learn more:** [Contributing Guide](../../CONTRIBUTING.md)

---

### Is AI-assisted development required?

No, but it's encouraged!

This project is developed with AI assistance (Claude, Gemini, etc.), but
human developers are welcome to contribute traditionally.

**For AI-assisted development:**
- Follow [AGENTS.md](../../AGENTS.md) guidelines
- Write tests WITH code (not after)
- Update docs immediately

**For traditional development:**
- Same quality standards apply
- Tests and docs still required
- Follow coding guidelines
- All PRs reviewed by maintainers

**Learn more:** [Contributing Guide](../../CONTRIBUTING.md)

---

### What's the release cycle?

**Current Status:** Pre-release (0.1.0 in development)

**Planned:**
- Semantic versioning (MAJOR.MINOR.PATCH)
- Regular releases after 0.1.0
- Security patches as needed
- Coordinated with Arcaflow Engine releases

**Learn more:** [Release Process](../development/release-process.md)

---

## Support Questions

### Where do I get help?

**Documentation:**
- [Troubleshooting Guide](troubleshooting.md)
- [Full Documentation](index.md)
- [Examples and Tutorials](examples/)

**Community:**
- [GitHub Issues](https://github.com/arcalot/arcaflow-mcp/issues)
- [Arcalot Round Table](https://github.com/arcalot/arcalot-round-table)
- [Arcaflow Documentation](https://arcalot.io/arcaflow)

**Security Issues:**
- Report privately per [SECURITY.md](../../SECURITY.md)

---

### How do I report bugs?

1. Check [existing issues](https://github.com/arcalot/arcaflow-mcp/issues)
2. Verify it's reproducible
3. Collect diagnostic info (see [Troubleshooting](troubleshooting.md))
4. Create detailed issue with:
   - Clear description
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details
   - Logs (sanitize sensitive data)

**Learn more:** [Troubleshooting Guide](troubleshooting.md#report-issues)

---

### Can I use this in production?

**Current Status:** Pre-release (under active development)

**Not recommended for production yet:**
- Still under active development
- API may change before 1.0 release
- Security audit pending
- Performance tuning ongoing

**Use for:**
- Development and testing
- Proof of concept
- Evaluation

**Production readiness planned:** Version 1.0 release

---

## More Questions?

Can't find your answer?

- Check [Troubleshooting Guide](troubleshooting.md)
- Review [Full Documentation](index.md)
- Ask in [GitHub Issues](https://github.com/arcalot/arcaflow-mcp/issues)
- Join [Arcalot Community](https://github.com/arcalot/arcalot-round-table)

---

[← Back to Documentation Index](index.md) | [← Back to README](../../README.md)
