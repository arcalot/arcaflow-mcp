## Contributing to Arcaflow MCP

Thanks for contributing. This repository is a hybrid Go + Python MCP server
focused on Arcaflow workflow input construction and result analysis.

### Prerequisites

**Required Tools** (versions aligned with [Arcalot organization standards](https://github.com/arcalot)):
- Go (see `ARCALOT_GO_VERSION` org variable)
- Python (see `ARCALOT_PYTHON_SUPPORTED_VERSIONS` org variable)
- Poetry (Python dependency management)
- protoc with Go and Python gRPC plugins
- Podman/Buildah (for container workflows)

**Finding Current Version Requirements:**
- Check `.github/workflows/ci.yml` for exact versions used in CI
- See [Version Management](docs/development/version-management.md) for details
- GitHub Organization variables define authoritative requirements

### Developer setup

1. Clone the repository.
2. Run `scripts/dev-setup.sh` to install git hooks and tooling checks.
3. Follow the README for environment-specific setup.

### Workflow expectations

- Write tests and documentation alongside code changes.
- Keep new lines within 88 characters when practical.
- Avoid adding new dependencies without justification.
- Use structured logging and robust error handling.
- Do not add sleeps or waits without explicit rationale.

### Testing

Run the scoped scripts to match the component you changed:

- Go tests: `./scripts/test-go.sh`
- Python tests: `./scripts/test-python.sh`
- Full suite: `./scripts/test-all.sh`

### Formatting and linting

- Go uses `gofmt` and `golangci-lint`.
- Python uses `black` and `ruff`.

Use `./scripts/validate.sh` before opening a PR.

### Documentation

Documentation is required with every change. Update:

- User docs in `docs/arcaflow-mcp/`
- Project docs in `docs/architecture/`, `docs/adr/`, and `docs/development/`

### Security

If you find a security issue, follow `SECURITY.md`.
