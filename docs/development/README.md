# Development Documentation

Development guides for contributing to the Arcaflow MCP server.

[← Back to Main README](../../README.md)

## Overview

This directory contains development documentation for contributors and
maintainers. Start here if you want to contribute code, fix bugs, or understand
the development workflow.

## Quick Start for Developers

```bash
# 1. Clone repository
git clone https://github.com/arcalot/arcaflow-mcp.git
cd arcaflow-mcp

# 2. Install git hooks and setup environment
./scripts/dev-setup.sh

# 3. Install Python dependencies
cd analysis && poetry install && cd ..

# 4. Download Go dependencies
cd server && go mod download && cd ..

# 5. Verify setup
./scripts/validate.sh

# 6. Run tests
./scripts/test-all.sh
```

**Detailed Instructions:** [Development Setup](setup.md)

## Documentation Index

- **[Development Setup](setup.md)** - Complete environment setup (step-by-step)
- **[Testing Guide](testing.md)** - Testing standards and practices
- **[Debugging Guide](debugging.md)** - Common issues and debugging tools
- **[Release Process](release-process.md)** - Creating releases

## Essential Reading

Before contributing, please read:

1. **[CONTRIBUTING.md](../../CONTRIBUTING.md)** - Contribution guidelines and PR
   process
2. **[AGENTS.md](../../AGENTS.md)** - Coding standards and quality requirements
3. **[CODE_OF_CONDUCT.md](../../CODE_OF_CONDUCT.md)** - Community standards

## Development Workflow

### Standard Development Cycle

```mermaid
graph LR
    A[Pick Issue] --> B[Create Branch]
    B --> C[Write Code]
    C --> D[Write Tests]
    D --> E[Update Docs]
    E --> F[Run Validation]
    F --> G{Hooks Pass?}
    G -->|Yes| H[Commit]
    G -->|No| C
    H --> I[Push Branch]
    I --> J[Create PR]
    J --> K[CI Checks]
    K --> L{Approved?}
    L -->|Yes| M[Merge]
    L -->|No| C
```

### 1. Pick an Issue

- Check [open issues](https://github.com/arcalot/arcaflow-mcp/issues)
- Comment on issue to claim it
- For new features, discuss with maintainers first

### 2. Create Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-number-description
```

### 3. Write Code + Tests + Docs

**CRITICAL:** Tests and documentation are written WITH code, not after.

```bash
# Write your code
vim server/pkg/mypackage/myfile.go

# Write tests at the same time
vim server/pkg/mypackage/myfile_test.go

# Update documentation immediately
vim docs/relevant-doc.md
```

**Standards:**
- Maximum line length: 88 characters
- Go: Follow Go idioms, use golangci-lint
- Python: PEP 8, type hints, Black formatting
- Tests: >85% coverage required
- Docs: Update with code changes

See [AGENTS.md](../../AGENTS.md) for complete standards.

### 4. Run Validation

```bash
# Run all validation checks
./scripts/validate.sh

# Run all tests
./scripts/test-all.sh

# Or run individually:
./scripts/test-go.sh      # Go tests only
./scripts/test-python.sh  # Python tests only
```

### 5. Commit Changes

Git hooks run automatically on commit:
- Code formatting
- Linting
- Fast tests

```bash
# Stage changes
git add .

# Commit (hooks run automatically)
git commit -m "feat: add new feature

Detailed description of changes...

AI-assisted-by: Claude Sonnet 4.5"
```

**Commit Message Format:**
- Use conventional commits: `feat:`, `fix:`, `docs:`, `test:`, etc.
- Be descriptive and verbose
- Include AI assistance credit if applicable
- Reference related issues: `Fixes #123`

See [AGENTS.md](../../AGENTS.md) for commit standards.

### 6. Push and Create PR

```bash
# Push branch
git push -u origin feature/your-feature-name

# Create PR on GitHub
# Include:
# - Clear description of changes
# - Why the change is needed
# - How to test
# - Any breaking changes
# - Screenshots if UI changes
```

### 7. PR Review Process

- CI checks must pass (build, test, lint)
- At least one maintainer review required
- Address review feedback
- Keep PR updated with main branch
- Squash commits if requested

---

## Project Structure

```
arcaflow-mcp/
├── server/              # Go MCP server
│   ├── cmd/             # Binaries
│   ├── pkg/             # Go packages
│   └── test/            # Integration tests
├── analysis/            # Python analysis engine
│   ├── arcaflow_analysis/   # Python package
│   └── tests/           # Python tests
├── api/                 # Shared API definitions
├── docs/                # Documentation
├── examples/            # Example workflows
├── scripts/             # Development scripts
├── .github/             # GitHub workflows (CI/CD)
├── AGENTS.md            # AI agent guidelines
├── CONTRIBUTING.md      # Contribution guidelines
└── README.md            # Project overview
```

**Detailed Structure:** [Architecture Overview](../architecture/overview.md)

---

## Development Tools

### Required Tools

- **Go 1.23.0** - Go compiler and tools
- **Python 3.12** - Python interpreter
- **Poetry 1.8.3** - Python dependency management
- **protoc** - Protocol buffer compiler
- **golangci-lint** - Go linting
- **black** - Python code formatting
- **pre-commit** - Git hook management

**Installation:** [Development Setup](setup.md)

### Helper Scripts

All scripts in `scripts/` directory:

```bash
./scripts/dev-setup.sh       # Install git hooks
./scripts/validate.sh        # Run all validation
./scripts/test-all.sh        # Run all tests
./scripts/test-go.sh         # Run Go tests
./scripts/test-python.sh     # Run Python tests
./scripts/docs-serve.sh      # Serve user docs locally
./scripts/docs-build.sh      # Build user docs
```

### IDE Setup

Recommended IDE settings:

**VSCode/Cursor:**
```json
{
  "editor.rulers": [88],
  "editor.formatOnSave": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "python.formatting.provider": "black",
  "python.linting.enabled": true,
  "python.linting.pylintEnabled": true
}
```

**GoLand/PyCharm:**
- Set line length to 88
- Enable format on save
- Configure golangci-lint
- Configure Black formatter

**EditorConfig:**
- `.editorconfig` file included in repo
- Enable EditorConfig support in your IDE

---

## Testing Strategy

### Test Types

1. **Unit Tests** - Test functions/methods in isolation
   - Mock all external dependencies
   - Fast (<1s per module)
   - Run on every commit
   - Location: `*_test.go`, `test_*.py`

2. **Integration Tests** - Test component interactions
   - Mock external services only
   - Test protocol compliance
   - Run in CI
   - Location: `server/test/integration/`

3. **E2E Tests** - Test complete workflows
   - Real dependencies
   - Test MCP client interactions
   - Run before merge
   - Location: `server/test/e2e/`

**Detailed Guide:** [Testing Guide](testing.md)

### Running Tests

```bash
# All tests with coverage
./scripts/test-all.sh

# Go tests only
cd server && go test ./... -v -cover

# Python tests only
cd analysis && poetry run pytest

# Specific package
cd server && go test ./pkg/protocol/... -v

# Specific Python module
cd analysis && poetry run pytest tests/test_analyzer.py

# With coverage report
cd server && go test ./... -coverprofile=coverage.out
cd server && go tool cover -html=coverage.out

cd analysis && poetry run pytest --cov=arcaflow_analysis
```

### Writing Tests

**Go Test Example:**
```go
func TestWorkflowLoad(t *testing.T) {
    t.Parallel()
    
    tests := []struct {
        name    string
        source  Source
        want    *Workflow
        wantErr bool
    }{
        {
            name: "filesystem workflow",
            source: Source{
                Kind:     SourceKindFilesystem,
                Location: "testdata/workflow.yaml",
            },
            want: &Workflow{/* ... */},
            wantErr: false,
        },
        // more test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            
            loader := NewLoader()
            got, err := loader.Load(context.Background(), tt.source)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Load() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Python Test Example:**
```python
import pytest
from arcaflow_analysis.parser import ResultLoader

def test_result_loader_json():
    """Test loading JSON results."""
    loader = ResultLoader()
    results = loader.load_from_file("tests/fixtures/results.json")
    
    assert results is not None
    assert "output" in results
    assert results["output"]["status"] == "success"

def test_result_loader_invalid_format():
    """Test error handling for invalid format."""
    loader = ResultLoader()
    
    with pytest.raises(ValueError, match="Invalid result format"):
        loader.load_from_file("tests/fixtures/invalid.txt")
```

---

## Coding Standards

### Go Standards

- Follow Go best practices and idioms
- Comprehensive godoc comments for exported symbols
- Maximum line length: 88 characters
- Error handling: Always check and handle errors
- Use structured logging
- Run golangci-lint before committing

**Example:**
```go
// LoadWorkflow loads a workflow from the specified source.
//
// The source can be a filesystem path, URL, or git repository.
// Returns an error if the workflow cannot be loaded or parsed.
func LoadWorkflow(ctx context.Context, source Source) (*Workflow, error) {
    if err := source.Validate(); err != nil {
        return nil, fmt.Errorf("invalid source: %w", err)
    }
    
    // Implementation...
    
    return workflow, nil
}
```

### Python Standards

- PEP 8 compliance (enforced by Black)
- Type hints for all function signatures
- NumPy-style docstrings
- Maximum line length: 88 characters
- Use structured logging

**Example:**
```python
def analyze_results(
    results: dict[str, Any],
    goal: str,
    *,
    threshold: float = 0.95
) -> AnalysisResult:
    """
    Analyze workflow results against a defined goal.
    
    Parameters
    ----------
    results : dict[str, Any]
        Parsed workflow execution results
    goal : str
        User-defined goal statement
    threshold : float, optional
        Success threshold (default: 0.95)
        
    Returns
    -------
    AnalysisResult
        Analysis with metrics and recommendations
        
    Raises
    ------
    ValueError
        If results format is invalid
        
    Examples
    --------
    >>> results = {"output": {"success_rate": 0.80}}
    >>> analysis = analyze_results(results, "95% success rate")
    >>> print(analysis.recommendations)
    """
    # Implementation...
```

**Complete Standards:** [AGENTS.md](../../AGENTS.md)

---

## Debugging

Common debugging scenarios and tools:

**Local Mode (stdio):**
```bash
# Add debug logging
./server/arcaflow-mcp --mode local --log-level debug 2>server.log

# Test with manual JSON-RPC messages
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | \
  ./server/arcaflow-mcp --mode local
```

**Server Mode (HTTP/SSE):**
```bash
# Step 1: Set data directory location (change this path if needed)
export DATA_DIR="./data"

# Step 2: Create data directory (from repository root)
mkdir -p "$DATA_DIR"

# Step 3: Set admin token (copy-paste ready for development)
export ARCAFLOW_MCP_ADMIN_TOKEN="dev-test-token-12345"

# Step 4: Configure storage paths (all use $DATA_DIR)
export ARCAFLOW_MCP_TOKEN_STORE_PATH="$DATA_DIR/tokens.json"
export ARCAFLOW_MCP_TENANT_STORE_PATH="$DATA_DIR/tenants.json"
export ARCAFLOW_MCP_AUDIT_STORE_PATH="$DATA_DIR/audit.json"
export ARCAFLOW_MCP_USAGE_STORE_PATH="$DATA_DIR/usage.json"
export ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT="$DATA_DIR/tenants"

# Step 5: Start with debug logging
./server/arcaflow-mcp --mode server --address :8080 --log-level debug &

# Wait for startup
sleep 2

# Step 6: In another terminal, test with curl (token must match step 3)
# Part 1: Initialize request
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev-test-token-12345" \
  -d @- <<'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
EOF

# Part 2: Send initialized notification
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev-test-token-12345" \
  -d @- <<'EOF'
{"jsonrpc":"2.0","method":"initialized"}
EOF
```

**Detailed Guide:** [Debugging Guide](debugging.md)

---

## Documentation

### Documentation Types

1. **User Documentation** (`docs/arcaflow-mcp/`)
   - For end users and operators
   - MkDocs format
   - Will integrate with Arcaflow docs

2. **Project Documentation** (`docs/`)
   - Architecture, ADRs, API docs
   - For developers and maintainers
   - Standard Markdown

### Building User Docs

```bash
# Serve locally
./scripts/docs-serve.sh
# Visit: http://127.0.0.1:8000

# Build for deployment
./scripts/docs-build.sh
```

### Documentation Standards

- All code changes require documentation updates
- Use numbered steps for procedures
- Include expected outcomes
- Add cross-links to related docs
- Test all code examples
- Use consistent terminology

---

## AI-Assisted Development

This project supports AI-assisted development.

**For AI Coding Agents:**
- Read [AGENTS.md](../../AGENTS.md) for behavioral guidelines
- Write tests WITH code (not after)
- Update docs immediately (not deferred)
- Require explicit confirmation before scope changes

**For Human Developers:**
- AI assistance is optional but encouraged
- Same quality standards apply
- Credit AI assistance in commits
- All PRs reviewed by humans

---

## Release Process

See [Release Process](release-process.md) for details on:
- Version numbering (semantic versioning)
- Release checklist
- Changelog generation
- Artifact building
- Distribution

---

## Getting Help

### Resources

- **[Troubleshooting Guide](debugging.md)** - Common development issues
- **[Architecture Docs](../architecture/)** - System design
- **[API Docs](../api/)** - API reference
- **[GitHub Issues](https://github.com/arcalot/arcaflow-mcp/issues)** - Bug reports and features

### Community

- **[Arcalot Round Table](https://github.com/arcalot/arcalot-round-table)** - Community hub
- **[GitHub Discussions](https://github.com/arcalot/arcaflow-mcp/discussions)** - Q&A and ideas

---

## Related Documentation

### For Contributors
- [Contributing Guide](../../CONTRIBUTING.md)
- [Code of Conduct](../../CODE_OF_CONDUCT.md)
- [AGENTS.md](../../AGENTS.md)

### For Users
- [Getting Started](../arcaflow-mcp/getting-started.md)
- [User Documentation](../arcaflow-mcp/)

### For Maintainers
- [Release Process](release-process.md)
- [ADR Index](../adr/README.md)

---

[← Back to Main README](../../README.md)
