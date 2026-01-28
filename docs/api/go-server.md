# Go Server API Reference

API documentation for the Go MCP server packages.

## Overview

The Go server provides the MCP protocol implementation, workflow tools, and multi-tenancy features. All packages are documented using standard Go documentation conventions.

## Viewing Documentation

### Online (pkg.go.dev)

Once published, the API documentation will be available at:

```
https://pkg.go.dev/github.com/arcalot/arcaflow-mcp/server
```

### Local (godoc)

Generate and view documentation locally:

```bash
# Install godoc
go install golang.org/x/tools/cmd/godoc@latest

# Start godoc server
cd server
godoc -http=:6060

# Open in browser
open http://localhost:6060/pkg/github.com/arcalot/arcaflow-mcp/server/
```

### IDE Integration

**VS Code:**

- Hover over symbols for inline documentation
- Use `Go to Definition` (F12) to view source
- Install `Go` extension for enhanced features

**GoLand:**

- Quick Documentation: Ctrl+Q (Cmd+J on macOS)
- Parameter Info: Ctrl+P (Cmd+P on macOS)
- External Documentation: Shift+F1

## Package Structure

```
server/pkg/
├── analysis/            # Analysis engine HTTP client
├── arcaflow/            # Arcaflow integration
│   ├── pluginschema/    # Plugin schema handling
│   └── workflow/        # Workflow loading and validation
├── audit/               # Audit logging
├── auth/                # Authentication
├── config/              # Configuration management
├── protocol/            # MCP protocol implementation
├── ratelimit/           # Rate limiting
├── resources/           # MCP resources
├── state/               # Session state management
├── tenant/              # Multi-tenancy
├── tools/               # MCP tools
│   └── workflowtools/   # Workflow-specific tools
├── transport/           # Transport implementations
│   ├── httpserver/      # HTTP/SSE transport
│   └── stdio/           # stdio transport
└── version/             # Version information
```

## Core Packages

### protocol

**Import Path:** `github.com/arcalot/arcaflow-mcp/server/pkg/protocol`

**Purpose:** MCP JSON-RPC 2.0 protocol implementation

**Key Types:**

- `Server` - MCP protocol server
- `Session` - MCP session state
- `Tool` - MCP tool definition
- `Resource` - MCP resource definition

**Example:**

```go
import "github.com/arcalot/arcaflow-mcp/server/pkg/protocol"

// Create MCP server
server := protocol.NewServer(
    protocol.ServerInfo{
        Name:            "Arcaflow MCP",
        Version:         "1.0.0",
        ProtocolVersion: "2025-11-25",
    },
)

// Register tools
server.AddTool(tool, handler)
```

### workflow

**Import Path:** `github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow`

**Purpose:** Workflow loading, parsing, and validation

**Key Types:**

- `Loader` - Workflow loader interface
- `Workflow` - Parsed workflow representation
- `Validator` - Input validator
- `Source` - Workflow source (filesystem, git, URL)

**Example:**

```go
import "github.com/arcalot/arcaflow-mcp/server/pkg/arcaflow/workflow"

// Load workflow
loader := workflow.NewLoader()
wf, err := loader.Load(ctx, source, selector)
if err != nil {
    return fmt.Errorf("failed to load workflow: %w", err)
}

// Validate input
validator := workflow.NewValidator()
err = validator.Validate(wf, input)
if err != nil {
    return fmt.Errorf("validation failed: %w", err)
}
```

### tenant

**Import Path:** `github.com/arcalot/arcaflow-mcp/server/pkg/tenant`

**Purpose:** Multi-tenancy and resource isolation

**Key Types:**

- `Tenant` - Tenant definition
- `Context` - Tenant context for requests
- `Store` - Tenant persistence
- `Quota` - Resource quotas

**Example:**

```go
import "github.com/arcalot/arcaflow-mcp/server/pkg/tenant"

// Get tenant context from request
tenantCtx, err := auth.GetTenantContext(token)
if err != nil {
    return fmt.Errorf("authentication failed: %w", err)
}

// Check quota
if err := tenantCtx.CheckQuota("workflows", 1); err != nil {
    return fmt.Errorf("quota exceeded: %w", err)
}
```

### transport/httpserver

**Import Path:** `github.com/arcalot/arcaflow-mcp/server/pkg/transport/httpserver`

**Purpose:** HTTP/SSE transport for server mode

**Key Types:**

- `Server` - HTTP server
- `Handler` - Request handler
- `SSEWriter` - Server-Sent Events writer

**Example:**

```go
import "github.com/arcalot/arcaflow-mcp/server/pkg/transport/httpserver"

// Create HTTP server
httpServer := httpserver.New(
    httpserver.Config{
        ListenAddr: ":8080",
        Protocol:   protocolServer,
        Auth:       authenticator,
    },
)

// Start server
if err := httpServer.Run(ctx); err != nil {
    log.Fatal(err)
}
```

### tools/workflowtools

**Import Path:** `github.com/arcalot/arcaflow-mcp/server/pkg/tools/workflowtools`

**Purpose:** MCP tool implementations for workflows

**Key Functions:**

- `RegisterTools()` - Register all workflow tools
- Tool handlers for each MCP tool operation

**Example:**

```go
import "github.com/arcalot/arcaflow-mcp/server/pkg/tools/workflowtools"

// Register all workflow tools
workflowtools.RegisterTools(protocolServer)
```

## Generating Documentation

### HTML Documentation

```bash
cd server
godoc -http=:6060
```

### Markdown Documentation

```bash
# Install godoc-md
go install github.com/pieterclaerhout/go-doc-md@latest

# Generate markdown docs
cd server/pkg
godoc-md . > ../../docs/api/go-server-godoc.md
```

### PDF Documentation

```bash
# Generate HTML first
godoc -html . > api.html

# Convert to PDF (requires wkhtmltopdf)
wkhtmltopdf api.html api.pdf
```

## Documentation Standards

All exported symbols (functions, types, methods, constants) must have documentation comments following Go conventions:

```go
// Server implements the MCP protocol server.
//
// It manages tool registration, session state, and request handling.
// The server supports both stdio and HTTP/SSE transports.
//
// Example:
//
//	server := protocol.NewServer(info)
//	server.AddTool(tool, handler)
//	server.Run(ctx)
type Server struct {
    // ...
}

// HandleRequest processes an MCP JSON-RPC request.
//
// The request is validated, routed to the appropriate handler,
// and the result is returned. If the request is invalid or
// the handler returns an error, an MCP error response is returned.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - req: JSON-RPC request to process
//
// Returns:
//   - response: JSON-RPC response
//   - error: Error if processing fails
func (s *Server) HandleRequest(
    ctx context.Context,
    req *protocol.Request,
) (*protocol.Response, error) {
    // ...
}
```

## Related Documentation

- [Go Server Architecture](../architecture/go-server.md) - Architectural overview
- [Development Setup](../development/setup.md) - Setting up development environment
- [Testing Guide](../development/testing.md) - Writing and running tests

---

*For Python analysis engine API, see [python-engine.md](python-engine.md).*
