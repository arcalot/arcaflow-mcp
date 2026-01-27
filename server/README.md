## Go MCP Server

This directory contains the Go MCP server core implementation.

## Components

- **Transport layer** - stdio (local mode) and HTTP/SSE (server mode)
- **MCP protocol** - JSON-RPC 2.0 implementation
- **Tool handlers** - Workflow loading, validation, input construction
- **Resource handlers** - Workflow schemas, examples, execution results
- **Authentication** - Bearer token auth and multi-tenant isolation
- **Arcaflow integration** - Schema extraction and validation

## Documentation

### For Users
- [Getting Started](../docs/arcaflow-mcp/getting-started.md) - Quick start guide
- [Local Mode Setup](../docs/arcaflow-mcp/usage/local-mode.md) - Claude Desktop configuration
- [Server Mode Setup](../docs/arcaflow-mcp/usage/server-mode.md) - Multi-user deployment
- [Configuration Reference](../docs/arcaflow-mcp/usage/configuration.md) - All options

### For Developers
- [Architecture Overview](../docs/architecture/go-server.md) - Go server internals
- [Development Setup](../docs/development/setup.md) - Environment setup
- [Testing Guide](../docs/development/testing.md) - Testing standards
- [API Documentation](../docs/api/go-server.md) - Go API reference

### Package Documentation
- [godoc](https://pkg.go.dev/github.com/arcalot/arcaflow-mcp/server) - Go package docs

## Building

```bash
# Build server binary
go build -o arcaflow-mcp ./cmd/arcaflow-mcp

# Run tests
go test ./...

# Run with coverage
go test ./... -coverprofile=coverage.out
```

## Running

```bash
# Local mode (stdio) - no configuration needed
./arcaflow-mcp --mode local

# Server mode (HTTP/SSE) - requires configuration
# Step 1: Set data directory location (change this path if needed)
export DATA_DIR="./data"

# Step 2: Create data directory
mkdir -p "$DATA_DIR"

# Step 3: Set admin token (copy-paste for testing)
export ARCAFLOW_MCP_ADMIN_TOKEN="dev-test-admin-token-12345"

# Step 4: Configure storage paths (all use $DATA_DIR)
export ARCAFLOW_MCP_TOKEN_STORE_PATH="$DATA_DIR/tokens.json"
export ARCAFLOW_MCP_TENANT_STORE_PATH="$DATA_DIR/tenants.json"
export ARCAFLOW_MCP_AUDIT_STORE_PATH="$DATA_DIR/audit.json"
export ARCAFLOW_MCP_USAGE_STORE_PATH="$DATA_DIR/usage.json"
export ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT="$DATA_DIR/tenants"

# Step 5: Start server
./arcaflow-mcp --mode server --address :8080
```

**Server Mode Requirements:**
- **Authentication**: Must set `ARCAFLOW_MCP_ADMIN_TOKEN` (example above is for testing only)
- **Data directory**: Defaults to `/var/lib/arcaflow-mcp` (requires root); override as shown for development
- **Production tokens**: Generate secure token with `openssl rand -hex 32`

See [Configuration Reference](../docs/arcaflow-mcp/usage/configuration.md) for all options and [Server Mode Setup](../docs/arcaflow-mcp/usage/server-mode.md) for complete deployment guide.

## Contributing

See [Contributing Guide](../CONTRIBUTING.md) and [Development Setup](../docs/development/setup.md).
