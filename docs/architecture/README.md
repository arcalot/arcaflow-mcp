# Architecture Documentation

Technical architecture documentation for the Arcaflow MCP server.

[← Back to Main README](../../README.md)

## Overview

This directory contains detailed technical architecture documentation for
developers and maintainers. For user-oriented architecture information, see
[User Architecture Guide](../arcaflow-mcp/concepts/architecture.md).

## Documentation Index

### System Architecture

- **[Architecture Overview](overview.md)** - Complete system design, component
  interaction, and technology choices
- **[Data Flow](data-flow.md)** - Request processing, data flow sequences, and
  state management
- **[Persistence Architecture](persistence.md)** - Database design, storage
  strategies, and data lifecycle

### Component Architecture

- **[Go MCP Server](go-server.md)** - Go server internals, package structure,
  and data flow
- **[Python Analysis Engine](python-engine.md)** - Python engine internals,
  modules, and algorithms
- **[Inter-service Communication](inter-service.md)** - Go-Python communication
  via gRPC/REST

## Architecture Diagrams

### High-Level System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     AI Agents / Clients                     │
│           (Claude Desktop, API clients, etc.)               │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ stdio (local) / HTTP+SSE (server)
                     │
┌────────────────────▼────────────────────────────────────────┐
│               MCP Server Core (Go)                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Transport Layer (stdio + HTTP/SSE)                  │   │
│  │ Authentication & Multi-tenancy                      │   │
│  │ Protocol Handler (JSON-RPC 2.0)                     │   │
│  │ Workflow Tools (load, validate, export)             │   │
│  └─────────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ gRPC / REST
                     │
┌────────────────────▼────────────────────────────────────────┐
│            Analysis Engine (Python)                         │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Result Parser & Metrics Extractor                   │   │
│  │ Pattern Analyzer & Suggestion Generator             │   │
│  │ Historical Database (SQLite/PostgreSQL)             │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

See [Architecture Overview](overview.md) for detailed diagrams.

## Key Architectural Decisions

Major architectural decisions are documented as ADRs (Architecture Decision
Records). See [ADR Index](../adr/README.md) for all decisions.

**Notable ADRs:**
- [ADR-001: Hybrid Go + Python Architecture](../adr/ADR-001-hybrid-go-python-architecture.md)
- [ADR-002: Monorepo Structure](../adr/README.md)
- [ADR-003: Deployment Modes](../adr/README.md)

## Package Structure

### Go Server (`server/`)

```
server/
├── cmd/                     # Binaries
│   ├── arcaflow-mcp/       # Main MCP server
│   └── manual-validate/    # Validation utilities
├── pkg/                     # Go packages
│   ├── analysis/           # Analysis service client
│   ├── arcaflow/           # Arcaflow integration
│   │   ├── pluginschema/   # Plugin schema handling
│   │   └── workflow/       # Workflow loading & parsing
│   ├── audit/              # Audit logging
│   ├── auth/               # Authentication
│   ├── config/             # Configuration
│   ├── protocol/           # MCP protocol (JSON-RPC)
│   ├── ratelimit/          # Rate limiting
│   ├── resources/          # MCP resources
│   ├── state/              # State management
│   ├── tenant/             # Multi-tenancy
│   ├── tools/              # MCP tools
│   │   └── workflowtools/  # Workflow tool implementations
│   ├── transport/          # Transport layers
│   │   ├── httpserver/     # HTTP/SSE transport
│   │   └── stdio/          # stdio transport
│   └── version/            # Version info
└── test/                    # Integration tests
    └── integration/         # E2E integration tests
```

See [Go Server Architecture](go-server.md) for details.

### Python Analysis Engine (`analysis/`)

```
analysis/
├── arcaflow_analysis/       # Python package
│   ├── analyzer/           # Result analysis
│   │   ├── result_analyzer.py      # Metrics & goal analysis
│   │   └── result_comparator.py    # Multi-run comparison
│   ├── db/                 # Database layer
│   │   ├── models.py       # SQLAlchemy models
│   │   ├── repository.py   # Data access
│   │   └── session.py      # Session management
│   ├── parser/             # Result parsing
│   │   ├── result_loader.py        # Load from files/URLs
│   │   └── result_parser.py        # Parse and extract
│   ├── server/             # HTTP server
│   │   ├── app.py          # FastAPI application
│   │   ├── http_api.py     # HTTP API routes
│   │   └── http_server.py  # Server startup
│   └── suggester/          # Suggestions
│       └── suggestion_engine.py    # Optimization suggestions
└── tests/                   # Python tests
```

See [Python Engine Architecture](python-engine.md) for details.

## Design Principles

### 1. Hybrid Language Strategy

- **Go**: Server infrastructure (protocol, auth, transport)
- **Python**: Data analysis and AI (results, suggestions, ML)
- Each component uses the best tool for its purpose

**Rationale:** [ADR-001](../adr/ADR-001-hybrid-go-python-architecture.md)

### 2. Deterministic Inputs

All workflow inputs must be:
- Schema-validated before export (100% enforcement)
- Deterministic (reproducible)
- Machine-readable (JSON/YAML)
- Compatible with Arcaflow Engine

**Why:** Guarantees exported inputs always work with Arcaflow.

### 3. Multi-Tenant Isolation

Server mode provides complete tenant isolation:
- Workspace separation
- Token-based authentication
- Rate limiting per tenant
- Audit logging for compliance

### 4. Modular Design

Clear separation of concerns:
- Transport → Protocol → Tools/Resources → Domain logic
- Each layer independently testable
- Minimal coupling between components

### 5. Schema-First

All data validated against schemas:
- Workflow inputs validated against workflow schemas
- MCP messages validated against MCP protocol schema
- API requests validated against API schemas

## Performance Characteristics

**Measured Performance:**
- Protocol overhead: <100ms (local), <200ms (server)
- Schema extraction: <500ms
- Input validation: <100ms
- Result analysis: <3s (typical)
- Multi-run comparison: <5s (5 runs)

**Scalability:**
- 100+ concurrent tenant connections (server mode)
- Handles high-throughput multi-tenant workloads
- Tenant isolation maintained under load

See [Architecture Overview](overview.md) for performance details.

## Testing Strategy

### Unit Tests
- Go: `*_test.go` files alongside code
- Python: `tests/test_*.py` in analysis directory
- Mock all external dependencies
- Target: >85% coverage

### Integration Tests
- Located in `server/test/integration/`
- Test component interactions
- Mock external services only
- Verify MCP protocol compliance

### E2E Tests
- Test complete workflows
- Real Arcaflow engine integration
- Verify exported inputs work with Arcaflow

See [Testing Guide](../development/testing.md) for details.

## Security Architecture

### Threat Model

- Tool/skill poisoning or unauthorized execution
- Prompt injection attacks via workflow inputs
- Data leakage between tenants
- Supply chain compromise via dependencies
- Unauthorized workflow execution

### Security Controls

- Authentication and authorization per user/tenant
- Scoped permissions for tools and resources
- Input validation at all boundaries
- Audit logging of all operations
- Regular security scans of dependencies
- TLS for all network communications (server mode)

See [SECURITY.md](../../SECURITY.md) for security policy.

## Related Documentation

### For Users
- [User Architecture Guide](../arcaflow-mcp/concepts/architecture.md)
- [Getting Started](../arcaflow-mcp/getting-started.md)

### For Developers
- [API Documentation](../api/)
- [Development Setup](../development/setup.md)
- [Contributing Guide](../../CONTRIBUTING.md)

### For Maintainers
- [ADR Index](../adr/README.md)
- [Release Process](../development/release-process.md)

---

[← Back to Main README](../../README.md)
