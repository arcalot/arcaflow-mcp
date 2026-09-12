# Version Management

**Arcaflow MCP Version Management Strategy**

---

## Overview

Arcaflow MCP uses **GitHub Organization variables as the single source of truth** for Go and Python version requirements. This ensures consistency with the broader Arcalot ecosystem and eliminates version drift.

**Version Sources:**

1. **GitHub Organization Variables** - Authoritative source for Go/Python versions
2. **Runtime `VERSION` File** - Application version (repository root: `/VERSION`)
3. **Documentation** - Version requirements documented in `README.md` and `CONTRIBUTING.md`

---

## GitHub Organization Variables (Source of Truth)

### Variables

**Location:** Arcalot GitHub Organization settings → Actions → Variables

| Variable | Purpose |
|----------|---------|
| `ARCALOT_GO_VERSION` | Go version (aligned with `arcaflow-engine` main) |
| `ARCALOT_PYTHON_VERSION` | Primary Python version |
| `ARCALOT_PYTHON_SUPPORTED_VERSIONS` | All supported Python versions for testing |

**To see current values:** Check `.github/workflows/ci.yml` which shows both the variable names and their current fallback values.

### Usage

**In GitHub Actions workflows:**

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ vars.ARCALOT_GO_VERSION || '1.23' }}
      
      - uses: actions/setup-python@v5
        with:
          python-version: ${{ vars.ARCALOT_PYTHON_VERSION || '3.12' }}
```

**Benefits:**
- **Single source of truth** across all Arcalot projects
- **No drift possible** - one place to update
- **Automatic propagation** to all workflows
- **Ecosystem alignment** - all projects stay synchronized

---

## Unified Versioning

### Primary: Root VERSION File

**Location:** `/VERSION` (repository root)

**Format:** Single line with version string
```
0.1.0-dev
```

**Used By:**
- Go server runtime (`server/pkg/version/version.go`)
- Python analysis runtime (`analysis/arcaflow_analysis/server/app.py`)
- Container image builds (`server/Containerfile`)
- Documentation and CI/CD

### Secondary: Git Tags

**Format:** `v{MAJOR}.{MINOR}.{PATCH}` (e.g., `v1.0.0`)

**Used By:**
- GoReleaser for release automation
- GitHub releases
- Container image tags

**Requirement:** Git tag must match VERSION file for releases

### Metadata: Python Package Version

**Location:** `analysis/pyproject.toml`

**Purpose:** Python package metadata (not read at runtime)

**Requirement:** Should be kept in sync with VERSION file for consistency

---

## Version Resolution

### Go Server

**Code:** `server/pkg/version/version.go`

**Resolution Order:**
1. Environment variable: `ARCAFLOW_MCP_VERSION`
2. File in current directory: `./VERSION`
3. File relative to binary: `../VERSION`
4. Fallback: `"dev"`

### Python Analysis Engine

**Code:** `analysis/arcaflow_analysis/server/app.py`

**Resolution:**
```python
repo_root = Path(__file__).resolve().parents[3]
version_file = repo_root / "VERSION"
version = version_file.read_text(encoding="utf-8").strip()
```

**Fallback:** `"unknown"` if VERSION file not found

---

## Development Workflow

### Checking Your Environment

**Verify required tools are installed:**

```bash
# Check tools are present
go version
python3 --version
poetry --version
golangci-lint --version

# Compare with requirements in .github/workflows/ci.yml
# to ensure compatibility
```

**To see exact version requirements:** Check `.github/workflows/ci.yml` which uses organization variables with fallback values.

See [Development Setup](setup.md) for installation instructions.

---

## Release Process

### Update Version for Release

```bash
# 1. Update VERSION file
echo "1.0.0" > VERSION

# 2. Update Python package metadata (keep in sync)
cd analysis
poetry version 1.0.0
cd ..

# 3. Commit changes
git add VERSION analysis/pyproject.toml
git commit -m "chore: bump version to 1.0.0"

# 4. Create git tag
git tag -a v1.0.0 -m "Release v1.0.0"

# 5. Push tag (triggers release workflow)
git push origin v1.0.0
```

**Important:** VERSION file and git tag should always match for releases.

---

## CI/CD Integration

### Workflow Version Usage

All CI/CD workflows use organization variables:

```yaml
# .github/workflows/ci.yml
- uses: actions/setup-go@v5
  with:
    go-version: ${{ vars.ARCALOT_GO_VERSION || '1.23' }}
```

**Fallback values** provide compatibility if organization variables are not set.

### Container Image Tags

**For releases (git tags):**
- `quay.io/arcalot/arcaflow-mcp-server:1.0.0`
- `quay.io/arcalot/arcaflow-mcp-server:v1`
- `quay.io/arcalot/arcaflow-mcp-server:latest`

**For development (PRs/branches):**
- `quay.io/arcalot/arcaflow-mcp-server:dev`
- `quay.io/arcalot/arcaflow-mcp-server:pr-123`

---

## Updating Versions

### When to Update Go/Python Versions

**Organization-wide updates** (Arcalot maintainers only):

1. Update GitHub Organization variables
2. All Arcalot projects automatically use new versions in CI/CD
3. Update local development environments as needed

**Project-specific documentation:**

1. Update `README.md` prerequisites
2. Update `CONTRIBUTING.md` prerequisites  
3. Update `docs/development/setup.md`

### Keeping Documentation in Sync

**When organization variables change:**

```bash
# 1. Verify current org variable values (Arcalot maintainers)
# Check GitHub Organization settings → Actions → Variables

# 2. Update version-specific files if needed
# - server/go.mod (if Go version changed)
# - analysis/pyproject.toml (if Python version changed)

# 3. Commit version-specific file changes only
# Documentation automatically points to org variables
git commit -am "build: update to match ARCALOT_GO_VERSION org variable"
```

---

## Why Organization Variables?

### Benefits

1. **Single Source of Truth** - One place to update across all Arcalot projects
2. **No Drift** - Impossible for projects to get out of sync with ecosystem
3. **Automatic Propagation** - CI/CD workflows automatically use latest versions
4. **Ecosystem Alignment** - All Arcalot projects use consistent versions
5. **Simplified Maintenance** - No per-project version files to keep in sync

### Trade-offs

**Benefit:**
- Centralized version control
- Ecosystem consistency
- Reduced maintenance burden

**Limitation:**
- Requires organization admin access to change
- All projects must be compatible with organization versions

**Decision:** Benefits far outweigh limitations for Arcalot ecosystem projects.

---

## Troubleshooting

### My local Go/Python version doesn't match

**Cause:** Your environment uses a different version than organization standards

**Solution:**

```bash
# Check required versions
cat README.md | grep -A5 "Prerequisites"

# Install required versions
# Go: https://go.dev/dl/
# Python: https://www.python.org/downloads/
# or use version managers (pyenv, goenv, asdf)
```

### CI/CD workflow fails with wrong version

**Cause:** Organization variable not set or workflow has hardcoded fallback

**Solution:**

1. Check workflow file has correct org variable reference:
   ```yaml
   go-version: ${{ vars.ARCALOT_GO_VERSION || '1.23' }}
   ```

2. Verify organization variables are set (Arcalot admins)

3. Update fallback value if needed

### Version mismatch between VERSION file and pyproject.toml

**Cause:** Files updated independently

**Solution:**

```bash
VERSION=$(cat VERSION)
cd analysis && poetry version $VERSION
git add analysis/pyproject.toml
git commit -m "chore: sync pyproject.toml version to $VERSION"
```

---

## References

- [Development Setup](setup.md) - Installing required tools
- [Release Process](release-process.md) - Creating releases
- [Versioning Architecture](../architecture/versioning.md) - Technical details
- [Container Deployment](../arcaflow-mcp/deployment/container.md) - Container builds
