# Arcaflow MCP Feature Status

This document provides a comprehensive overview of implemented, planned, and explicitly out-of-scope features for Arcaflow MCP. Use this to understand current capabilities and future direction.

**Quick Links:**
- [Roadmap](ROADMAP.md) - Development timeline and priorities
- [Capabilities Documentation](docs/arcaflow-mcp/concepts/capabilities.md) - Detailed feature descriptions
- [Request a Feature](https://github.com/arcalot/arcaflow-mcp/issues/new/choose) - Suggest new capabilities

---

## Feature Matrix

Legend:
- ✅ **Implemented** - Available in current release
- 🚧 **In Progress** - Actively being developed
- 📋 **Planned** - Scheduled for upcoming release
- 🔮 **Future** - Under consideration, not yet scheduled
- ❌ **Out of Scope** - Not planned for this project

---

## Core Capabilities

### Input Construction

| Feature | Status | Component | Notes |
|---------|--------|-----------|-------|
| Workflow discovery (filesystem) | ✅ Implemented | Go Server | Local and git repositories |
| Workflow discovery (git) | ✅ Implemented | Go Server | Full clone and shallow clone support |
| Workflow discovery (URL) | ✅ Implemented | Go Server | HTTP/HTTPS URLs |
| Schema extraction | ✅ Implemented | Go Server | Arcaflow schema format |
| Input validation (real-time) | ✅ Implemented | Go Server | Against workflow schemas |
| Conversational input building | ✅ Implemented | Go Server | Natural language interaction |
| Input export (YAML) | ✅ Implemented | Go Server | Deterministic serialization |
| Input export (JSON) | ✅ Implemented | Go Server | Deterministic serialization |
| Example input loading | ✅ Implemented | Go Server | From workflow directories |
| Complex type support (objects) | ✅ Implemented | Go Server | Nested objects |
| Complex type support (arrays) | ✅ Implemented | Go Server | Lists and sequences |
| Complex type support (maps) | ✅ Implemented | Go Server | Key-value dictionaries |
| Complex type support (unions) | ✅ Implemented | Go Server | Tagged unions |
| Plugin schema discovery | ✅ Implemented | Go Server | From workflow step definitions |
| Automatic path resolution | ✅ Implemented | Go Server | Relative→absolute path conversion |
| LLM routing optimization | ✅ Implemented | Go Server | Uncertainty injection, routing guides |
| Input templates and presets | 📋 Planned | Go Server | v0.2.0 |
| Input versioning | 🔮 Future | Go Server | Track input evolution |
| Input diffing | 🔮 Future | Go Server | Compare input versions |

### Result Analysis

| Feature | Status | Component | Notes |
|---------|--------|-----------|-------|
| Result loading (YAML/JSON) | ✅ Implemented | Python Engine | Structured results |
| Result loading (logs) | ✅ Implemented | Python Engine | Semi-structured logs |
| Result loading (filesystem) | ✅ Implemented | Python Engine | Local files |
| Result loading (URL) | ✅ Implemented | Python Engine | HTTP/HTTPS URLs |
| Metric extraction | ✅ Implemented | Python Engine | Automatic discovery |
| Statistical analysis (basic) | ✅ Implemented | Python Engine | Mean, median, std dev |
| Multi-run comparison | ✅ Implemented | Python Engine | Side-by-side analysis |
| AI optimization suggestions | ✅ Implemented | Python Engine | Natural language recommendations |
| Result ranking | ✅ Implemented | Python Engine | Multi-criteria ranking |
| Goal-based analysis | ✅ Implemented | Python Engine | User-defined success criteria |
| Historical result storage | ✅ Implemented | Python Engine | SQLite backend |
| Historical result retrieval | ✅ Implemented | Python Engine | Query past runs |
| Statistical analysis (advanced) | 📋 Planned | Python Engine | Trend detection, anomalies (v0.2.0) |
| Baseline comparison | 📋 Planned | Python Engine | Regression detection (v0.2.0) |
| Custom metric definitions | 📋 Planned | Python Engine | User-defined metrics (v0.2.0) |
| Text-based visualization | 📋 Planned | Python Engine | Charts and graphs (v0.2.0) |
| Database integration (PostgreSQL) | 🔮 Future | Python Engine | Multi-tenant production |
| Real-time streaming analysis | 🔮 Future | Python Engine | Live execution monitoring |
| Machine learning predictions | 🔮 Future | Python Engine | Performance forecasting |
| Graphical visualizations | ❌ Out of Scope | - | Use external tools (Grafana, etc.) |

### Workflow Execution

| Feature | Status | Component | Notes |
|---------|--------|-----------|-------|
| Workflow execution | 📋 Planned | Go Server | v0.2.0 - via Arcaflow Engine |
| Execution progress streaming | 📋 Planned | Go Server | Real-time status (v0.2.0) |
| Log integration | 📋 Planned | Go Server | Stream execution logs (v0.2.0) |
| Error handling and retries | 📋 Planned | Go Server | Configurable retry logic (v0.2.0) |
| Iterative optimization loops | 📋 Planned | Both | Execute → analyze → refine (v0.2.0) |
| Parallel execution | 🔮 Future | Go Server | Multiple concurrent runs |
| Execution scheduling | 🔮 Future | Go Server | Cron-like scheduling |
| Resource management | 🔮 Future | Go Server | CPU/memory limits |

### Workflow Creation

| Feature | Status | Component | Notes |
|---------|--------|-----------|-------|
| Workflow generation from NL | 🔮 Future | Go Server | v0.3.0 consideration |
| Plugin recommendation | 🔮 Future | Go Server | Suggest relevant plugins |
| Step composition assistance | 🔮 Future | Go Server | Build workflows interactively |
| Workflow validation | 🔮 Future | Go Server | Validate before execution |
| Workflow templates | 🔮 Future | Go Server | Reusable patterns |
| Direct workflow editing | ❌ Out of Scope | - | Use Arcaflow tools or text editors |

---

## MCP Protocol Features

### Transports

| Feature | Status | Component | Notes |
|---------|--------|-----------|-------|
| Stdio transport | ✅ Implemented | Go Server | Local mode |
| HTTP/SSE transport | ✅ Implemented | Go Server | Server mode |
| WebSocket transport | ❌ Out of Scope | - | Not in MCP spec we target |

### Protocol Features

| Feature | Status | Component | Notes |
|---------|--------|-----------|-------|
| Tools capability | ✅ Implemented | Go Server | All workflow tools |
| Resources capability | ✅ Implemented | Go Server | Workflow and schema resources |
| Routing guidance resources | ✅ Implemented | Go Server | mcp://routing-guide, mcp://arcaflow-authority |
| Prompts capability | ❌ Out of Scope | - | Not needed for our use case |
| Sampling capability | ❌ Out of Scope | - | Not needed for our use case |
| Logging capability | ✅ Implemented | Go Server | Server-side logging |
| MCP 2025-11-25 compliance | ✅ Implemented | Go Server | Full spec compliance |

---

## Deployment and Operations

### Deployment Modes

| Feature | Status | Component | Notes |
|---------|--------|-----------|-------|
| Local mode (stdio) | ✅ Implemented | Go Server | Desktop AI clients |
| Server mode (HTTP/SSE) | ✅ Implemented | Go Server | Multi-tenant deployments |
| Container deployment | ✅ Implemented | Both | Podman/Docker |
| Kubernetes deployment | 🚧 In Progress | Both | Manifests and docs |
| Binary distribution | 📋 Planned | Go Server | Pre-compiled binaries (v0.1.0) |
| PyPI distribution | 📋 Planned | Python Engine | pip install (v0.1.0) |
| Helm charts | 🔮 Future | Both | Kubernetes deployment |
| Operator pattern | 🔮 Future | Both | Kubernetes CRDs |

### Multi-Tenancy

| Feature | Status | Component | Notes |
|---------|--------|-----------|-------|
| Workspace isolation | ✅ Implemented | Go Server | Separate directories |
| Tenant authentication | ✅ Implemented | Go Server | Bearer tokens |
| Admin API | ✅ Implemented | Go Server | Tenant management |
| Rate limiting (per tenant) | ✅ Implemented | Go Server | Configurable limits |
| Usage tracking | ✅ Implemented | Go Server | Per-tenant metrics |
| Audit logging | ✅ Implemented | Go Server | All operations logged |
| Resource quotas | ✅ Implemented | Go Server | CPU/memory/storage limits |
| Fine-grained RBAC | 🔮 Future | Go Server | Role-based access control |
| SSO integration | 🔮 Future | Go Server | OIDC/SAML |

### Security

| Feature | Status | Component | Notes |
|---------|--------|-----------|-------|
| Bearer token authentication | ✅ Implemented | Go Server | Server mode |
| TLS support | ✅ Implemented | Go Server | HTTPS/TLS 1.3 |
| Input validation | ✅ Implemented | Go Server | All boundaries |
| Audit logging | ✅ Implemented | Go Server | Security events |
| Secret management | 🔮 Future | Go Server | Vault integration |
| Plugin sandboxing | 🔮 Future | Go Server | Execution isolation |

### Observability

| Feature | Status | Component | Notes |
|---------|--------|-----------|-------|
| Structured logging | ✅ Implemented | Both | JSON logs |
| Health endpoints | ✅ Implemented | Both | /healthz checks |
| Version endpoints | ✅ Implemented | Go Server | Version info |
| Prometheus metrics | 🔮 Future | Both | Monitoring |
| OpenTelemetry tracing | 🔮 Future | Both | Distributed tracing |
| Performance profiling | 🔮 Future | Both | pprof endpoints |

---

## Integration and Compatibility

### AI Clients

| Client | Status | Notes |
|--------|--------|-------|
| Claude Desktop | ✅ Supported | Tested and documented |
| Cursor IDE | ✅ Supported | Tested and documented |
| Custom MCP clients | ✅ Supported | Standard protocol |
| OpenAI integration | 🔮 Future | When OpenAI supports MCP |

### Arcaflow Ecosystem

| Feature | Status | Notes |
|---------|--------|-------|
| Arcaflow Engine 0.20+ | ✅ Compatible | Input validation |
| Arcaflow plugins | ✅ Compatible | Schema discovery |
| Arcaflow workflows | ✅ Compatible | No modifications needed |
| Arcaflow documentation site | 🚧 In Progress | MkDocs integration |

### CI/CD Integration

| Feature | Status | Notes |
|---------|--------|-------|
| GitHub Actions examples | 📋 Planned | v0.2.0 |
| GitLab CI examples | 📋 Planned | v0.2.0 |
| Jenkins integration | 🔮 Future | Pipeline examples |
| ArgoCD integration | 🔮 Future | GitOps workflows |

---

## Platform Support

### Operating Systems

| Platform | Go Server | Python Engine | Notes |
|----------|-----------|---------------|-------|
| Linux (x86_64) | ✅ Supported | ✅ Supported | Primary platform |
| Linux (arm64) | ✅ Supported | ✅ Supported | Tested on Fedora |
| macOS (Intel) | ✅ Supported | ✅ Supported | Tested on macOS 14+ |
| macOS (Apple Silicon) | ✅ Supported | ✅ Supported | Tested on macOS 14+ |
| Windows (x86_64) | 📋 Planned | ✅ Supported | Go binary planned v0.1.0 |
| FreeBSD | 🔮 Future | 🔮 Future | Community request |

### Container Platforms

| Platform | Status | Notes |
|----------|--------|-------|
| Podman | ✅ Supported | Primary development platform |
| Docker | ✅ Supported | Tested and documented |
| Podman Compose | ✅ Supported | Multi-container deployments |
| Docker Compose | ✅ Supported | Multi-container deployments |
| Kubernetes | 🚧 In Progress | Manifests and documentation |
| OpenShift | 🔮 Future | Enterprise Kubernetes |

---

## Explicitly Out of Scope

These features are **not planned** for Arcaflow MCP, as they are better handled by other tools:

### Direct Workflow Editing
**Why**: Text editors and IDEs are optimized for YAML editing. Arcaflow MCP focuses on input construction, not workflow development.

**Alternatives**: Use VS Code, vim, or any text editor with the Arcaflow workflows.

### Graphical Visualizations
**Why**: Dashboarding tools (Grafana, Kibana) provide rich visualization capabilities. Arcaflow MCP provides data, not dashboards.

**Alternatives**: Export results and use Grafana, Kibana, or custom dashboards.

### Workflow Repository Hosting
**Why**: Git platforms (GitHub, GitLab) provide excellent version control and collaboration features.

**Alternatives**: Store workflows in git repositories and reference them via MCP.

### Plugin Development
**Why**: Arcaflow already has excellent plugin development tools and documentation.

**Alternatives**: Use Arcaflow SDK to develop plugins, then reference them in workflows.

### Database-as-a-Service
**Why**: Arcaflow MCP is a workflow tool, not a database platform.

**Alternatives**: Use PostgreSQL, SQLite, or your preferred database for result storage.

---

## How to Use This Document

### Finding Feature Status

1. **Check the matrix** for your feature of interest
2. **Read the notes** for additional context
3. **Check the roadmap** for planned features' timeline
4. **Request missing features** via GitHub issues

### Understanding Status Codes

- **✅ Implemented**: Ready to use in current release
- **🚧 In Progress**: Being actively developed, may be in alpha/beta
- **📋 Planned**: Committed for an upcoming release (see roadmap)
- **🔮 Future**: Under consideration but not yet scheduled
- **❌ Out of Scope**: Not planned; use alternative tools

### Requesting Features

1. **Search this document** to ensure it's not already planned
2. **Check the roadmap** for timeline information
3. **Open a feature request** with details about your use case
4. **Engage with the community** in discussions

---

## Questions and Feedback

- **Feature requests**: [Open an issue](https://github.com/arcalot/arcaflow-mcp/issues/new/choose)
- **Status questions**: [Start a discussion](https://github.com/arcalot/arcalot-round-table/discussions)
- **Documentation improvements**: [Submit a PR](CONTRIBUTING.md)

**Last Updated**: January 2026  
**Next Review**: Periodically (see [Roadmap](ROADMAP.md))
