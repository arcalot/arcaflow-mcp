# Arcaflow MCP Project Documentation

This directory contains technical documentation for contributors and maintainers working on the Arcaflow MCP codebase.

## Documentation Structure

This repository maintains **two separate documentation sets**:

1. **Project Documentation** (this document) - Internal technical docs for developers
   - `architecture/` - System design and implementation details
   - `adr/` - Architecture decision records
   - `api/` - API reference documentation
   - `development/` - Contributor guides and workflows

2. **User Documentation** - External docs for arcalot.io integration (see note below)
   - `arcaflow-mcp/` - MkDocs-based documentation for end users
   - Built separately and published to [arcalot.io](https://arcalot.io/arcaflow)

**For contributors:** Focus on the project documentation sections below when working on the codebase. User documentation is maintained separately for external consumption.

---

## Table of Contents

### Architecture Documentation (`architecture/`)

Deep technical architecture for understanding system internals and design.

- **[Architecture Index](architecture/README.md)** - Navigation for architecture documentation
- **[System Overview](architecture/overview.md)** - High-level architecture and design rationale
- **[Go Server](architecture/go-server.md)** - MCP server implementation details
- **[Python Engine](architecture/python-engine.md)** - Analysis engine design and components
- **[Data Flow](architecture/data-flow.md)** - Request/response flow through the system
- **[Inter-Service Communication](architecture/inter-service.md)** - Go-Python communication patterns
- **[Persistence](architecture/persistence.md)** - Data storage and state management

---

### Architecture Decision Records (`adr/`)

Design decisions and their rationale, following the ADR format.

- **[ADR Index](adr/README.md)** - All architecture decisions with context and consequences
- **[ADR-001: Hybrid Go+Python Architecture](adr/ADR-001-hybrid-go-python-architecture.md)** - Rationale for Go+Python split

---

### API Documentation (`api/`)

API reference for developers working on or integrating with the system.

- **[API Index](api/README.md)** - Overview of all APIs and their purposes
- **[Go Server API](api/go-server.md)** - Go MCP server API reference (godoc)
- **[Python Engine API](api/python-engine.md)** - Python analysis engine API reference
- **[gRPC Protocol](api/grpc-protocol.md)** - Inter-service protocol specification

---

### Development Documentation (`development/`)

Practical guides for contributors and maintainers.

- **[Development Overview](development/README.md)** - Quick start and workflow for developers
- **[Setup Guide](development/setup.md)** - Development environment setup (Go, Python, tools)
- **[Testing Guide](development/testing.md)** - Testing standards, procedures, and coverage requirements
- **[Debugging Guide](development/debugging.md)** - Debugging tips, techniques, and common issues
- **[Release Process](development/release-process.md)** - How to create and publish releases

---

## Quick Links

**Project Components:**
- **[Repository README](../README.md)** - Main project overview and quick start
- **[Server README](../server/README.md)** - Go MCP server component
- **[Analysis README](../analysis/README.md)** - Python analysis engine component
- **[Examples](../examples/README.md)** - Working workflow examples and tutorials
- **[Changelog](CHANGELOG.md)** - Project changelog

**Development Tools:**
- **[Git Hooks](../scripts/dev-setup.sh)** - Pre-commit hooks for linting and testing
- **[Test Scripts](../scripts/)** - Validation and testing utilities
- **[Contributing Guide](../CONTRIBUTING.md)** - How to contribute

---

## User Documentation (External)

User-facing documentation is maintained separately in `arcaflow-mcp/` for integration with the main Arcaflow documentation site at [arcalot.io](https://arcalot.io/arcaflow).

**User docs location:** `docs/arcaflow-mcp/` (built with MkDocs)  
**Build config:** `docs/mkdocs-arcaflow.yml`  
**Output:** `site-arcaflow/` (static HTML, git-ignored)

User documentation covers installation, usage, deployment, and troubleshooting from an end-user perspective. It is built independently and does not appear in repository navigation.

**To build user docs:**
```bash
cd docs
mkdocs build -f mkdocs-arcaflow.yml
```

---

## Documentation Standards

All project documentation follows these standards:

- **Format**: Markdown for easy viewing on GitHub
- **Style**: Technical depth, assumes programming knowledge
- **Structure**: Clear hierarchy with READMEs as navigation hubs
- **Cross-linking**: Links primarily within project docs; external links clearly labeled
- **Code examples**: Tested and verified (per git hooks)
- **Maintenance**: Updated with code changes; documentation is not optional

**For documentation contributions:** See [Development Guide](development/README.md) for workflow and standards.

---

## Contributing

To contribute to project documentation:

1. Follow the [Development Setup](development/setup.md) guide
2. Read the [Contributing Guide](../CONTRIBUTING.md)
3. Make changes and verify locally
4. Submit a pull request with documentation updates alongside code changes

Documentation quality gates:
- All code examples must be tested
- Cross-links must be valid
- Markdown must be properly formatted
- Changes must pass pre-commit hooks
