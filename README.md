# Arcaflow MCP

Arcaflow MCP is a Model Context Protocol (MCP) server for working with
Arcaflow workflows. The initial focus is:

- Skill 1: Build and validate workflow inputs
- Skill 2: Analyze workflow outputs and suggest improvements

Workflow execution is intentionally out of scope for Phase 1.

## Status

Phase 1 scaffolding is underway. The server and analysis engine are not yet
implemented. See `DEVELOPMENT_PLAN.md` for the full roadmap.

## AI-assisted development

This project is developed with the assistance of AI code agents. Agent behavior
and quality standards are defined in `AGENTS.md`, and the authoritative roadmap
and phase gates are tracked in `DEVELOPMENT_PLAN.md`. Contributors should
consult both files before making changes or advancing phases.

Basic guidance when working with an AI coding agent:

- Point the agent to `AGENTS.md` and `DEVELOPMENT_PLAN.md` early in a session.
- Ask the agent to keep tests and documentation in lockstep with code changes.
- Require explicit confirmation before phase transitions or scope changes.
- Request a short summary of changes and any follow-up commands to run.

## Prerequisites

- Go 1.23.0
- Python 3.12
- Poetry 1.8.3
- protoc with Go and Python gRPC plugins
- Podman/Buildah (for container workflows)
- golangci-lint (for Go linting in `scripts/validate.sh`)

## Quick start (development)

1. Run `scripts/dev-setup.sh` to install git hooks.
2. Install Python dependencies: `cd analysis && poetry install`.
3. Download Go dependencies: `cd server && go mod download`.
4. Run `./scripts/validate.sh` to confirm tooling.
5. Use `./scripts/test-all.sh` to run the full test suite.

## Repository layout

- `server/` Go MCP server core (transport, protocol, tools)
- `analysis/` Python analysis engine and gRPC service
- `api/` Shared API definitions (protobuf)
- `docs/` User and project documentation
- `scripts/` Developer tooling and CI helper scripts

## Development workflow

- Write tests and documentation with code changes.
- Keep line lengths within 88 characters where practical.
- Use structured logging and robust error handling.
- Avoid new dependencies without justification.
- Do not bypass git hooks unless explicitly required.
- Ensure EditorConfig support is enabled in your editor.

## Documentation

- User docs: `docs/arcaflow-mcp/`
- Project docs: `docs/architecture/`, `docs/adr/`, `docs/development/`

## License

Apache 2.0. See `LICENSE`.
