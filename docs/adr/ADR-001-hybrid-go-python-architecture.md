## ADR 001: Hybrid Go + Python Architecture

### Status

Accepted - 2026-01-20

### Context

The MCP server requires robust protocol handling and concurrency (Go), while
workflow result analysis benefits from Python's data analysis ecosystem.
We need to support input construction and result analysis with clear separation
of concerns and independent scaling.

### Decision

Adopt a hybrid architecture:

- Go implements the MCP server core, transport, and input construction schema handling
- Python implements the analysis engine and result analysis logic
- gRPC is the preferred inter-service protocol

### Alternatives considered

1. Go-only implementation
2. Python-only implementation

### Consequences

- Two runtimes to deploy and manage
- Clear ownership boundaries for protocol vs analysis
- Better long-term flexibility for advanced analysis features
