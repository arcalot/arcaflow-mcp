# Arcaflow MCP - AI Agent Behavioral Guidelines

Purpose: Permanent standards and behaviors for AI coding agents working on this project.  

## Project Overview

Arcaflow MCP Server - A Model Context Protocol (MCP) server enabling natural language conversations with AI agents to work with Arcaflow workflows.

Primary Capabilities:
- Skill 1: Build and validate inputs for existing Arcaflow workflows
- Skill 2: Analyze workflow results and suggest optimizations
- Future: Workflow execution, iterative optimization, workflow creation

Architecture: Hybrid Go (MCP server core) + Python (analysis engine)  
Repository: Monorepo structure with independent Go and Python modules  
License: Apache 2.0

## Essential References

- Arcaflow Core: https://github.com/arcalot/arcaflow-engine
- Arcaflow Documentation: https://arcalot.io/arcaflow
- Arcalot Community: https://github.com/arcalot/arcalot-round-table
- MCP Specification: https://modelcontextprotocol.io/specification/
- Reference Workflow: https://gitlab.com/redhat/edge/tests/perfscale/arcaflow-workflow-auto-perf

## Fundamental AI Behaviors

### ALWAYS Do

1. Write tests WITH code - Never after, always concurrently or first (TDD)
2. Update documentation immediately - Never defer or postpone
3. Ensure git hooks are installed - Run `scripts/dev-setup.sh` on first setup (hooks then run automatically on commit)
4. Verify MCP standards - Check spec compliance for protocol work
5. Explain "why" in comments - Not just "what"
6. Include regression tests - For every bug fix

### NEVER Do

1. Never defer tests or documentation - They are integral to development
2. Never skip hooks - Use `--no-verify` only when explicitly justified
3. Never commit without tests - Minimum 85% coverage required
4. Never accept PRs without docs - Documentation is not optional

## Code Standards

### Go
- Follow Go best practices and idioms
- Use `golangci-lint` for linting
- Comprehensive godoc comments for all exported symbols
- Maximum line length: 88 characters
- Error handling: Always check and handle errors appropriately
- Use structured logging (e.g., zap or zerolog)

### Python
- Use Poetry for dependency management
- Follow PEP 8 style guide
- Use Black formatter for code formatting
- Type hints required for all function signatures
- Comprehensive docstrings (NumPy style)
- Maximum line length: 88 characters
- Use structured logging (e.g., structlog)

### Version Control
- Commit messages: Verbose and descriptive
  - Include context and rationale for changes
  - Use conventional commits format (e.g., `feat:`, `fix:`, `docs:`)
  - Always include: `AI-assisted-by: <model name and version>`
- Branches: Feature branches, no direct commits to main
- PRs: Require review and passing CI before merge

### Commit in one try
- Review changes before committing: `git status --short` and `git diff`
- Ensure hook caches live in the workspace to avoid permission errors:
  - `XDG_CACHE_HOME="$PWD/.cache"`
  - `GOLANGCI_LINT_CACHE="$PWD/.cache/golangci-lint"`
  - `GOCACHE="$PWD/.cache/go-build"`
  - `GOMODCACHE="$PWD/.cache/go-mod"`
  - `POETRY_CACHE_DIR="$PWD/.cache/poetry"`
  - `POETRY_VIRTUALENVS_IN_PROJECT=true`
- Fix lint errors before commit (unused imports, import order, errcheck, etc.)

## Testing Standards

### Test Types and Scope
- Unit tests (`*_test.go`, `test_*.py`): Test functions/methods in isolation. Mock ALL external dependencies. Fast (<1s per module). Run on every commit.
- Integration tests (`test/integration/`): Test component interactions. Mock external services only. Test protocol compliance. Run in CI.
- E2E tests (`test/e2e/`): Test complete workflows with real dependencies. Test MCP client interactions. Run before merge.

### Requirements (All Types)
- Minimum >85% coverage (measured via unit tests)
- Deterministic and repeatable results
- Test error conditions and edge cases
- Include regression test for every bug fix
- Table-driven tests (Go), pytest fixtures (Python)

## Documentation Standards

- Format: Markdown for general docs, MkDocs for user documentation, Mermaid for diagrams
- API docs: godoc (Go), Sphinx-compatible docstrings (Python)
- User docs: `docs/arcaflow-mcp/` (MkDocs format for Arcaflow integration)
- Project docs: `docs/architecture/`, `docs/adr/`, `docs/development/`
- Code comments: Explain "why" not "what"
- All public APIs, configuration options, tools, and resources must be documented
- Include tested and verified examples
- Document architecture decisions in ADRs (`docs/adr/`)

## Security Standards

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
- Sandboxed execution where appropriate

### Security Practices
- Never commit secrets or credentials
- Use environment variables for sensitive configuration
- Validate and sanitize all user inputs
- Follow principle of least privilege
- Regular dependency updates for security patches
- Security review for new features

## MCP Protocol Compliance

### Standards Adherence
- Track and implement specific MCP spec version
- Verify compliance against latest MCP specification
- Test with official MCP clients (Claude Desktop, etc.)
- Document any deviations with justification

### Protocol Requirements
- Implement JSON-RPC 2.0 correctly
- Follow MCP capability negotiation protocol
- Use standard MCP error codes
- Support required transport layers (stdio, HTTP/SSE)
- Implement required protocol methods (`initialize`, `tools/list`, `tools/call`, etc.)

## Arcalot Alignment

### Community Standards
- Follow Arcalot community code of conduct
- Align with Arcalot governance and contribution standards
- Use Apache 2.0 License (Arcalot-compatible)
- Contribute back to Arcalot ecosystem where appropriate

### Arcaflow Integration
- Maintain compatibility with Arcaflow engine
- Follow Arcaflow terminology and conventions
- Integrate documentation with Arcaflow docs (https://arcalot.io/arcaflow/)
- Test against reference Arcaflow workflows
- Distinguish workflow schemas from plugin schemas:
  - Workflow schemas define the top-level `input` scope for the workflow itself.
  - Plugin schemas define step input/output contracts for each plugin.
  - MCP Skill 1 input validation must use the workflow `input` scope first.
  - Plugin schema validation is only required when validating step inputs.
- When validating workflow inputs, mirror engine behavior:
  - Use `go.flow.arcalot.io/pluginsdk/schema` and `schema.UnserializeScope`
    on the workflow `input` section.
  - Treat a successful `Unserialize` + `Serialize` roundtrip as a valid,
    deterministic payload.

## Development Workflow

1. Setup (first time): Run `scripts/dev-setup.sh` to install git hooks
2. While coding: Write tests concurrently, document as you go
3. Testing:
   - Go: `cd server && go test ./...` (or `./scripts/test-go.sh`)
   - Python: `cd analysis && poetry run pytest` (or `./scripts/test-python.sh`)
   - All: `./scripts/test-all.sh`
4. Architecture decisions: Document in `docs/adr/`
5. Commit: Hooks run automatically (formatting, linting, fast tests)
   - Optional: `./scripts/validate.sh` for manual pre-commit validation
   - Only bypass hooks with `--no-verify` when explicitly justified

CI Requirements: All checks must pass before merge (build, test, lint, security scan, documentation)
