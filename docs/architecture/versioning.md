# Versioning Architecture

**Status:** Implemented  
**Last Updated:** 2026-01-27

---

## Overview

Arcaflow MCP uses a **unified versioning strategy** for the monorepo, ensuring both the Go MCP server and Python analysis engine share the same version number. This reflects the tightly coupled nature of these components.

---

## Version Sources

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

**Example:**
```go
import "github.com/arcalot/arcaflow-mcp/server/pkg/version"

v := version.Current() // Returns version string
```

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

## Container Image Builds

### Go Server Container

**Containerfile:** `server/Containerfile`

**Build Context:** Repository root (to access VERSION file)

**Build Command:**
```bash
podman build -t arcaflow-mcp-server:latest -f server/Containerfile .
```

**Version Injection:**
```dockerfile
# Copy VERSION from root
COPY VERSION ./

# Inject into binary at build time
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-X main.version=$(cat VERSION)" \
    -o arcaflow-mcp ./cmd/arcaflow-mcp
```

### Python Analysis Container

**Containerfile:** `analysis/Containerfile`

**Build Context:** `analysis/` directory

**Note:** Python engine reads VERSION at runtime (no build-time injection)

---

## Semantic Versioning

**Format:** `MAJOR.MINOR.PATCH[-SUFFIX]`

**Examples:**
- `0.1.0-dev` - Development version
- `0.1.0-rc1` - Release candidate
- `0.1.0` - Release version
- `1.0.0` - Stable API

### Increment Rules

**MAJOR** (breaking changes):
- Incompatible MCP protocol changes
- Incompatible tool schema changes
- Removed tools or resources

**MINOR** (new features, backward compatible):
- New MCP tools or resources
- New configuration options
- Enhanced capabilities

**PATCH** (bug fixes only):
- Bug fixes
- Security patches
- Documentation fixes

---

## Release Process

### 1. Update VERSION File

```bash
# Update to release version
echo "1.0.0" > VERSION

# Commit
git add VERSION analysis/pyproject.toml
git commit -m "chore: bump version to 1.0.0"
```

### 2. Update Python Package Version

```bash
cd analysis
poetry version 1.0.0  # Keep in sync
```

### 3. Create Git Tag

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

### 4. GoReleaser Automation

GoReleaser workflow triggers on tag push and:
- Builds multi-platform binaries
- Creates container images (tagged with version)
- Generates changelog
- Creates GitHub release

---

## Version Consistency

### Check Version Consistency

```bash
# Compare VERSION file and pyproject.toml
VERSION_FILE=$(cat VERSION)
PYPROJECT_VERSION=$(grep '^version = ' analysis/pyproject.toml | cut -d'"' -f2)

if [ "$VERSION_FILE" != "$PYPROJECT_VERSION" ]; then
    echo "WARNING: Version mismatch!"
    echo "  VERSION file: $VERSION_FILE"
    echo "  pyproject.toml: $PYPROJECT_VERSION"
fi
```

### Pre-commit Hook (Recommended)

Add to `.git/hooks/pre-commit`:
```bash
#!/bin/bash
# Verify VERSION and pyproject.toml are in sync

VERSION_FILE=$(cat VERSION)
PYPROJECT_VERSION=$(grep '^version = ' analysis/pyproject.toml | cut -d'"' -f2)

if [ "$VERSION_FILE" != "$PYPROJECT_VERSION" ]; then
    echo "ERROR: Version mismatch detected"
    echo "  VERSION: $VERSION_FILE"
    echo "  pyproject.toml: $PYPROJECT_VERSION"
    echo ""
    echo "Fix with: cd analysis && poetry version $VERSION_FILE"
    exit 1
fi
```

---

## Development Workflow

### Development Versions

Use `-dev` suffix for in-progress work:
```
0.2.0-dev
```

### Release Candidates

Use `-rc#` suffix for testing:
```
1.0.0-rc1
1.0.0-rc2
```

### Post-Release

Immediately bump to next dev version:
```bash
# After releasing 1.0.0
echo "1.0.1-dev" > VERSION
cd analysis && poetry version 1.0.1-dev
git commit -am "chore: bump version to 1.0.1-dev"
```

---

## CI/CD Integration

### GitHub Actions Variables

Container tags are derived from git context:

```yaml
- name: Extract version
  id: version
  run: |
    if [[ "${GITHUB_REF}" =~ ^refs/tags/v(.+)$ ]]; then
      VERSION="${BASH_REMATCH[1]}"
    else
      VERSION="dev"
    fi
    echo "version=${VERSION}" >> "$GITHUB_OUTPUT"
```

### Container Image Tags

**For releases (git tags):**
- `quay.io/arcalot/arcaflow-mcp-server:1.0.0`
- `quay.io/arcalot/arcaflow-mcp-server:v1`
- `quay.io/arcalot/arcaflow-mcp-server:latest`

**For development (PRs/branches):**
- `quay.io/arcalot/arcaflow-mcp-server:dev`
- `quay.io/arcalot/arcaflow-mcp-server:pr-123`

---

## Why Unified Versioning?

### Rationale

1. **Tight Coupling**: MCP server and analysis engine work together as a single system
2. **Simplified Communication**: Single version to communicate to users
3. **Release Coordination**: Components are always released together
4. **Compatibility Clarity**: Version match guarantees compatibility

### Trade-offs

**Benefits:**
- Simple version management
- No compatibility matrix needed
- Clear release boundaries
- Enforces coordinated development

**Limitations:**
- Can't version components independently
- Bug fix in one requires version bump for both

**Decision:** Benefits outweigh limitations for this tightly coupled architecture.

---

## Troubleshooting

### Version Shows as "dev" or "unknown"

**Cause:** VERSION file not found at runtime

**Solutions:**
1. Ensure VERSION exists in repository root
2. Check working directory at runtime
3. Set `ARCAFLOW_MCP_VERSION` environment variable

### Container Build Can't Find VERSION

**Cause:** Build context doesn't include VERSION file

**Solution:** Build from repository root:
```bash
# Correct
podman build -f server/Containerfile .

# Wrong (VERSION not in context)
cd server && podman build -f Containerfile .
```

### Version Mismatch Between Components

**Cause:** VERSION and pyproject.toml out of sync

**Solution:**
```bash
VERSION=$(cat VERSION)
cd analysis && poetry version $VERSION
git add analysis/pyproject.toml
git commit -m "chore: sync pyproject.toml version to $VERSION"
```

---

## References

- [Semantic Versioning 2.0.0](https://semver.org/)
- [Release Process](../development/release-process.md)
- [Version Management](../development/version-management.md)
- [Container Deployment](../arcaflow-mcp/deployment/container.md)
