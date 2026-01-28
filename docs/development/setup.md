# Development Setup

**Complete Environment Setup for Contributing to Arcaflow MCP**

This guide walks you through setting up a complete development environment for both the Go MCP server and Python analysis engine components.

---

## Prerequisites

### Required Tools

**Go Development:**
- Go 1.23.0 (exact version, Arcaflow standard)
- git
- make (optional, for build automation)

**Python Development:**
- Python 3.12 (exact version, Arcaflow standard)
- Poetry (Python dependency management)
- pip

**Code Quality:**
- golangci-lint (for Go linting)
- Black (for Python formatting, installed via Poetry)
- pytest (for Python testing, installed via Poetry)

### Install Prerequisites

**Fedora/RHEL:**
```bash
# Go and development tools
sudo dnf install golang git make

# Verify Go version (must be 1.23.0)
go version

# Python and Poetry
sudo dnf install python3.12 python3-pip
curl -sSL https://install.python-poetry.org | python3 -

# golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
  sh -s -- -b $(go env GOPATH)/bin
```

**Ubuntu/Debian:**
```bash
# Go and development tools (may need manual install for 1.23.0)
# Check: https://go.dev/dl/ for Go 1.23.0 tarball if package manager has older version

# Python and Poetry
sudo apt install python3.12 python3-pip
curl -sSL https://install.python-poetry.org | python3 -

# golangci-lint  
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
  sh -s -- -b $(go env GOPATH)/bin
```

**macOS:**
```bash
# Using Homebrew
brew install go python@3.12 poetry golangci-lint git

# Verify Go version
go version  # Must show go1.23.0
```

---

## Step 1: Clone Repository

```bash
# Clone the repository
git clone https://github.com/arcalot/arcaflow-mcp.git
cd arcaflow-mcp

# Verify you're on the correct branch
git status
```

---

## Step 2: Install Git Hooks

The project uses pre-commit hooks for code quality checks.

```bash
# Run the setup script (installs hooks and configures cache directories)
./scripts/dev-setup.sh
```

**What this does:**
- Installs pre-commit hooks for linting and formatting
- Configures cache directories in workspace (`.cache/`)
- Sets up Poetry virtual environment
- Verifies all required tools are installed

**Hooks installed:**
- Go formatting (gofmt)
- Go linting (golangci-lint)
- Python formatting (Black)
- Fast tests (unit tests only)

---

## Step 3: Install Go Dependencies

```bash
cd server

# Download Go module dependencies
go mod download

# Verify dependencies
go mod verify

# Build the server to test setup
go build -o arcaflow-mcp ./cmd/arcaflow-mcp

# Verify build
./arcaflow-mcp --version
```

**Expected output:**
```
arcaflow-mcp version 0.1.0-dev
```

---

## Step 4: Install Python Dependencies

```bash
cd ../analysis

# Install dependencies with Poetry
poetry install

# Verify installation
poetry run python --version
poetry run pytest --version

# Verify dependencies
poetry show
```

**Expected:** List of all installed packages including pytest, black, structlog, etc.

---

## Step 5: Verify Setup

Run the validation script to ensure everything is configured correctly:

```bash
cd ..  # Return to repository root

# Run comprehensive validation
./scripts/validate.sh
```

**This checks:**
- Go dependencies installed
- Python dependencies installed
- Linters available and working
- Formatters configured correctly
- Cache directories writable

**Expected output:**
```
✓ Go environment ready
✓ Python environment ready
✓ Git hooks installed
✓ Linters configured
✓ All validations passed
```

---

## Step 6: Run Tests

Verify your development environment by running the test suites:

### Go Tests

```bash
cd server

# Run all Go tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with verbose output
go test -v ./...
```

**Expected:** All tests pass with >80% coverage.

### Python Tests

```bash
cd ../analysis

# Run all Python tests
poetry run pytest

# Run with coverage report
poetry run pytest --cov=arcaflow_analysis --cov-report=term

# Run with verbose output
poetry run pytest -v
```

**Expected:** All tests pass with >84% coverage.

### All Tests (Convenience Script)

```bash
cd ..  # Return to repository root

# Run both Go and Python tests
./scripts/test-all.sh
```

---

## Step 7: Configure Your Editor

### VS Code / Cursor

Create or update `.vscode/settings.json`:

```json
{
  "go.testFlags": ["-v"],
  "go.buildFlags": ["-v"],
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "python.defaultInterpreterPath": "${workspaceFolder}/analysis/.venv/bin/python",
  "python.formatting.provider": "black",
  "python.formatting.blackArgs": ["--line-length", "88"],
  "python.linting.enabled": true,
  "python.linting.pylintEnabled": false,
  "python.linting.flake8Enabled": true,
  "python.testing.pytestEnabled": true,
  "python.testing.pytestArgs": [
    "analysis/tests"
  ],
  "editor.formatOnSave": true,
  "files.trimTrailingWhitespace": true,
  "files.insertFinalNewline": true
}
```

### GoLand / PyCharm

**For Go (GoLand):**
- File → Settings → Go → Go Modules → Enable Go modules integration
- File → Settings → Tools → File Watchers → Add golangci-lint
- File → Settings → Editor → Code Style → Go → Set from project .editorconfig

**For Python (PyCharm):**
- File → Settings → Project → Python Interpreter → Select Poetry environment
- File → Settings → Tools → Python Integrated Tools → Testing → Default test runner: pytest
- File → Settings → Tools → Black → Enable Black formatter

---

## Development Workflow

### Making Changes

**1. Create a feature branch:**
```bash
git checkout -b feature/my-feature
```

**2. Make your changes**

**3. Run tests locally:**
```bash
./scripts/test-all.sh
```

**4. Commit (hooks will run automatically):**
```bash
git add <files>
git commit -m "feat: your feature description

Detailed description...

AI-assisted-by: <model name if applicable>"
```

**5. Push and create PR:**
```bash
git push origin feature/my-feature
# Create PR via GitHub UI or gh CLI
```

### Git Hooks Behavior

Pre-commit hooks run automatically on `git commit`:

**Formatting:**
- Go code: `gofmt` reformats if needed
- Python code: `black` reformats if needed

**Linting:**
- Go: `golangci-lint` checks code quality
- Python: `flake8` checks style

**Testing:**
- Fast unit tests run (< 5 seconds typically)
- Integration tests run in CI only

**If hooks fail:**
1. Review the error messages
2. Fix the issues
3. Re-stage the changes: `git add <fixed-files>`
4. Commit again

**Skip hooks only when necessary:**
```bash
# Only use --no-verify when you have explicit justification
git commit --no-verify -m "..."  # Strongly discouraged
```

---

## Common Setup Issues

### "Go version mismatch"

**Error:** `go: go.mod requires go >= 1.23.0`

**Solution:**
```bash
# Check current version
go version

# If not 1.23.0, download exact version from https://go.dev/dl/
# Arcaflow projects require exact Go version for consistency

# Linux
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz

# Verify
go version  # Should show go1.23.0
```

### "Poetry not found"

**Error:** `poetry: command not found`

**Solution:**
```bash
# Install Poetry
curl -sSL https://install.python-poetry.org | python3 -

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/.local/bin:$PATH"

# Reload shell
source ~/.bashrc
```

### "golangci-lint not found"

**Error:** Git hooks fail with `golangci-lint: command not found`

**Solution:**
```bash
# Install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
  sh -s -- -b $(go env GOPATH)/bin

# Verify installation
golangci-lint --version
```

### "Permission denied" on .cache directories

**Error:** Cannot write to cache directories during commit

**Solution:**
```bash
# Ensure cache directories are in workspace
ls -la .cache/

# If missing, re-run setup
./scripts/dev-setup.sh

# Verify environment variables are set (hooks set these automatically)
echo $GOLANGCI_LINT_CACHE  # Should be $PWD/.cache/golangci-lint
```

### "Python module not found"

**Error:** `ModuleNotFoundError` during tests

**Solution:**
```bash
cd analysis

# Reinstall dependencies
poetry install

# Verify virtual environment
poetry env info

# Activate virtual environment (if needed)
poetry shell
```

---

## Environment Variables

### Development Cache Locations

The git hooks automatically set these to keep caches in the workspace:

```bash
export XDG_CACHE_HOME="$PWD/.cache"
export GOLANGCI_LINT_CACHE="$PWD/.cache/golangci-lint"
export GOCACHE="$PWD/.cache/go-build"
export GOMODCACHE="$PWD/.cache/go-mod"
export POETRY_CACHE_DIR="$PWD/.cache/poetry"
export POETRY_VIRTUALENVS_IN_PROJECT=true
```

**Why?** Prevents permission issues with system-wide caches.

### Server Development

For testing server mode during development:

```bash
# Minimal development server configuration
export DATA_DIR="$PWD/.dev-data"
export ARCAFLOW_MCP_ADMIN_TOKEN="dev-test-token-$(date +%s)"
export ARCAFLOW_MCP_TOKEN_STORE_PATH="$DATA_DIR/tokens.json"
export ARCAFLOW_MCP_TENANT_STORE_PATH="$DATA_DIR/tenants.json"
export ARCAFLOW_MCP_AUDIT_STORE_PATH="$DATA_DIR/audit.json"
export ARCAFLOW_MCP_USAGE_STORE_PATH="$DATA_DIR/usage.json"
export ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT="$DATA_DIR/tenants"

# Create data directory
mkdir -p "$DATA_DIR"

# Run server
cd server
./arcaflow-mcp --mode server --address :8080
```

---

## Optional Tools

### Recommended

**Testing:**
- `gotestsum` - Better test output formatting
- `pytest-watch` - Auto-run tests on file changes

**Debugging:**
- `delve` - Go debugger
- `ipdb` - Python debugger (included in Poetry dev dependencies)

**Documentation:**
- `mkdocs` - For building user documentation
- `godoc` - For viewing Go documentation locally

### Installation

```bash
# gotestsum
go install gotest.tools/gotestsum@latest

# delve
go install github.com/go-delve/delve/cmd/dlv@latest

# mkdocs (for building user docs)
pip install mkdocs-material

# godoc (Go documentation server)
go install golang.org/x/tools/cmd/godoc@latest
```

---

## Verifying Your Setup

### Checklist

- [ ] Repository cloned successfully
- [ ] Git hooks installed (`./scripts/dev-setup.sh`)
- [ ] Go dependencies downloaded (`cd server && go mod download`)
- [ ] Python dependencies installed (`cd analysis && poetry install`)
- [ ] Go tests pass (`cd server && go test ./...`)
- [ ] Python tests pass (`cd analysis && poetry run pytest`)
- [ ] Validation script passes (`./scripts/validate.sh`)
- [ ] Can build server binary (`cd server && go build ./cmd/arcaflow-mcp`)
- [ ] Editor configured with Go and Python support
- [ ] Git hooks run on commit (test by making a dummy commit)

### Test Your Setup

```bash
# Make a trivial change to test hooks
echo "# Test change" >> README.md
git add README.md
git commit -m "test: verify hooks work"

# Expected: Hooks run, commit succeeds
# Revert test commit
git reset HEAD~1
git checkout README.md
```

---

## Next Steps

**Start Contributing:**
- Read [Contributing Guide](../../CONTRIBUTING.md) for workflow
- Check [open issues](https://github.com/arcalot/arcaflow-mcp/issues) for tasks
- Review [Testing Guide](testing.md) for test requirements
- See [Architecture Overview](../architecture/overview.md) to understand the codebase

**Start Development:**
- Pick an issue or feature
- Create feature branch
- Write tests first (TDD)
- Implement feature
- Ensure tests pass
- Update documentation
- Submit PR

---

## Related Documentation

- **[Contributing Guide](../../CONTRIBUTING.md)** - Contribution workflow
- **[Testing Guide](testing.md)** - Testing standards and procedures
- **[Debugging Guide](debugging.md)** - Debugging techniques
- **[Architecture Overview](../architecture/overview.md)** - System design

---

[← Back to Development Documentation](README.md)
