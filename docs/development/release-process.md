# Release Process

**Creating and Publishing Arcaflow MCP Releases**

This guide covers the complete process for creating, testing, and publishing releases of Arcaflow MCP.

---

## Overview

Arcaflow MCP uses semantic versioning and follows a structured release process:

1. Prepare release (version bump, changelog, testing)
2. Create release branch
3. Final validation
4. Tag release
5. Build artifacts
6. Publish release
7. Announce

---

## Versioning

### Semantic Versioning

Arcaflow MCP follows [SemVer 2.0.0](https://semver.org/):

**Format:** `MAJOR.MINOR.PATCH`

**Examples:**
- `0.1.0` - First release (pre-1.0, no API stability guarantees)
- `0.2.0` - New features, backward compatible
- `0.2.1` - Bug fixes only
- `1.0.0` - First stable release (API stability commitment)
- `2.0.0` - Breaking changes

**When to increment:**

**MAJOR** (breaking changes):
- Incompatible MCP protocol changes
- Incompatible tool schema changes
- Removed tools or resources
- Changed command-line interface
- Breaking configuration changes

**MINOR** (new features, backward compatible):
- New MCP tools or resources
- New configuration options
- Enhanced capabilities
- Performance improvements
- New deployment modes

**PATCH** (bug fixes only):
- Bug fixes
- Security patches
- Documentation fixes
- Dependency updates (no new features)

---

## Release Preparation

### Step 1: Update Version Numbers

**Go server version:**

Edit `server/pkg/version/version.go`:
```go
package version

const (
    Version = "0.2.0"  // Update this
    Commit  = ""       // Filled by build
)
```

**Python analysis version:**

Edit `analysis/pyproject.toml`:
```toml
[tool.poetry]
name = "arcaflow-analysis"
version = "0.2.0"  # Update this
```

### Step 2: Update CHANGELOG

Edit `docs/CHANGELOG.md`:

```markdown
## [0.2.0] - 2026-01-30

### Added
- New `workflow_discover` tool for finding workflows in repositories
- SSE session binding for stateful server mode interactions
- Multi-run comparison in result analysis

### Changed
- Improved error messages for schema validation
- Enhanced troubleshooting documentation
- Updated MCP protocol to version 2025-11-25

### Fixed
- Fixed nil pointer in workflow loading (#42)
- Corrected schema validation for nested objects (#45)
- Fixed race condition in tenant workspace creation (#47)

### Security
- Added rate limiting per tenant
- Enhanced token validation
- Improved audit logging detail
```

### Step 3: Update Documentation

**Check documentation references versions:**
```bash
# Find version references
grep -r "0.1.0" docs/

# Update to new version where appropriate
```

**Update compatibility matrix in README.md:**
```markdown
| Arcaflow MCP | Arcaflow Engine | MCP Protocol |
|--------------|-----------------|--------------|
| 0.2.0        | 0.20.0+         | 2025-11-25   |
```

---

## Step 4: Pre-Release Testing

### Run Full Test Suite

```bash
# All tests with coverage
./scripts/test-all.sh

# Expected: All tests pass
```

### Integration Testing

**Test both deployment modes:**

**Local mode:**
```bash
cd server
go build -o arcaflow-mcp ./cmd/arcaflow-mcp

# Test with example workflow
./arcaflow-mcp --mode local
# (Send test requests via stdio)
```

**Server mode:**
```bash
# Set up test environment
export DATA_DIR="/tmp/release-test"
export ARCAFLOW_MCP_ADMIN_TOKEN="release-test-token"
export ARCAFLOW_MCP_TOKEN_STORE_PATH="$DATA_DIR/tokens.json"
export ARCAFLOW_MCP_TENANT_STORE_PATH="$DATA_DIR/tenants.json"
export ARCAFLOW_MCP_AUDIT_STORE_PATH="$DATA_DIR/audit.json"
export ARCAFLOW_MCP_USAGE_STORE_PATH="$DATA_DIR/usage.json"
export ARCAFLOW_MCP_TENANT_WORKSPACE_ROOT="$DATA_DIR/tenants"
mkdir -p "$DATA_DIR"

# Start server
./arcaflow-mcp --mode server --address :8080 &
PID=$!

# Test MCP handshake and tools
curl -X POST http://localhost:8080/mcp \
  -H "Authorization: Bearer release-test-token" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'

curl -X POST http://localhost:8080/mcp \
  -H "Authorization: Bearer release-test-token" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"initialized"}'

curl -X POST http://localhost:8080/mcp \
  -H "Authorization: Bearer release-test-token" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'

# Cleanup
kill $PID
rm -rf "$DATA_DIR"
```

### Example Workflows

Test all example workflows:
```bash
cd examples/workflows

# Hello World
cd hello-world
# Test workflow loads and validates
# Test input examples work

# Data Processing
cd ../data-processing
# Test workflow loads and validates
# Test input examples work

# Performance Test
cd ../perf-test
# Test workflow loads and validates
# Test input examples work
```

---

## Step 5: Create Release Branch

```bash
# Create release branch from main
git checkout main
git pull origin main
git checkout -b release/v0.2.0

# Push release branch
git push origin release/v0.2.0
```

---

## Step 6: Final Validation

### Security Scan

```bash
# Go security check
cd server
go run golang.org/x/vuln/cmd/govulncheck ./...

# Python security check
cd ../analysis
poetry run safety check

# Expected: No known vulnerabilities
```

### License Verification

```bash
# Verify license headers present
./scripts/check-licenses.sh  # If script exists

# Verify LICENSE file is current
cat LICENSE  # Should be Apache 2.0
```

### Documentation Build

```bash
# Build user documentation
cd docs
mkdocs build -f mkdocs-arcaflow.yml

# Expected: Build succeeds without errors
# Output: site-arcaflow/

# Verify generated site
cd ../site-arcaflow
python3 -m http.server 8000
# Visit http://localhost:8000 and verify docs render correctly
```

---

## Step 7: Tag Release

### Create Git Tag

```bash
# Return to repository root
cd ..

# Create annotated tag
git tag -a v0.2.0 -m "Release v0.2.0

## Highlights
- New workflow discovery tool
- Enhanced result analysis
- Improved server mode security

See CHANGELOG.md for complete release notes."

# Push tag
git push origin v0.2.0
```

---

## Step 8: Build Release Artifacts

### Go Binaries (Multi-Platform)

```bash
cd server

# Linux amd64
GOOS=linux GOARCH=amd64 go build -o arcaflow-mcp-v0.2.0-linux-amd64 \
  -ldflags "-X github.com/arcalot/arcaflow-mcp/server/pkg/version.Version=0.2.0 \
            -X github.com/arcalot/arcaflow-mcp/server/pkg/version.Commit=$(git rev-parse HEAD)" \
  ./cmd/arcaflow-mcp

# Linux arm64
GOOS=linux GOARCH=arm64 go build -o arcaflow-mcp-v0.2.0-linux-arm64 \
  -ldflags "-X github.com/arcalot/arcaflow-mcp/server/pkg/version.Version=0.2.0 \
            -X github.com/arcalot/arcaflow-mcp/server/pkg/version.Commit=$(git rev-parse HEAD)" \
  ./cmd/arcaflow-mcp

# macOS amd64
GOOS=darwin GOARCH=amd64 go build -o arcaflow-mcp-v0.2.0-darwin-amd64 \
  -ldflags "-X github.com/arcalot/arcaflow-mcp/server/pkg/version.Version=0.2.0 \
            -X github.com/arcalot/arcaflow-mcp/server/pkg/version.Commit=$(git rev-parse HEAD)" \
  ./cmd/arcaflow-mcp

# macOS arm64
GOOS=darwin GOARCH=arm64 go build -o arcaflow-mcp-v0.2.0-darwin-arm64 \
  -ldflags "-X github.com/arcalot/arcaflow-mcp/server/pkg/version.Version=0.2.0 \
            -X github.com/arcalot/arcaflow-mcp/server/pkg/version.Commit=$(git rev-parse HEAD)" \
  ./cmd/arcaflow-mcp

# Create checksums
sha256sum arcaflow-mcp-v0.2.0-* > checksums-v0.2.0.txt
```

### Python Package

```bash
cd ../analysis

# Build Python package
poetry build

# Output: dist/arcaflow_analysis-0.2.0.tar.gz
#         dist/arcaflow_analysis-0.2.0-py3-none-any.whl
```

### Container Images

```bash
# Build multi-arch container images
# (This will be part of Phase 8 - Deployment & Distribution)

# Placeholder for future container build process
# See Phase 8 tasks for container image publishing
```

---

## Step 9: Create GitHub Release

### Using GitHub CLI

```bash
# Create release with binaries
gh release create v0.2.0 \
  --title "Arcaflow MCP v0.2.0" \
  --notes-file release-notes-v0.2.0.md \
  server/arcaflow-mcp-v0.2.0-* \
  server/checksums-v0.2.0.txt \
  analysis/dist/arcaflow_analysis-0.2.0.tar.gz \
  analysis/dist/arcaflow_analysis-0.2.0-py3-none-any.whl
```

### Release Notes Template

Create `release-notes-v0.2.0.md`:

```markdown
# Arcaflow MCP v0.2.0

Natural language interface for Arcaflow workflows with enhanced discovery and analysis capabilities.

## Highlights

🔍 **Workflow Discovery** - New `workflow_discover` tool finds workflows in git repositories and local directories
📊 **Enhanced Analysis** - Improved result analysis with multi-metric optimization
🔒 **Security Improvements** - Rate limiting, enhanced audit logging, token rotation support

## What's New

### Features
- Workflow discovery with intelligent caching
- SSE session binding for stateful server interactions
- Multi-run comparison with weighted scoring
- Historical trend analysis

### Improvements
- Better error messages for schema validation
- Enhanced troubleshooting documentation
- Improved server mode setup guides
- Complete MCP protocol compliance (2025-11-25)

### Bug Fixes
- Fixed nil pointer in workflow loading (#42)
- Corrected schema validation for nested objects (#45)
- Fixed race condition in tenant workspace creation (#47)

## Installation

### Local Mode (Desktop AI)
```bash
# Download binary for your platform
wget https://github.com/arcalot/arcaflow-mcp/releases/download/v0.2.0/arcaflow-mcp-v0.2.0-linux-amd64
chmod +x arcaflow-mcp-v0.2.0-linux-amd64
mv arcaflow-mcp-v0.2.0-linux-amd64 /usr/local/bin/arcaflow-mcp
```

### Server Mode (Production)
See [Server Mode Setup Guide](https://arcalot.io/arcaflow/arcaflow-mcp/usage/server-mode/)

## Upgrade Notes

### From v0.1.0

**Breaking changes:** None

**Configuration changes:** 
- New rate limit configuration options (optional)
- New audit log format (backward compatible)

**Migration steps:**
1. Stop server
2. Backup data directory
3. Replace binary
4. Restart server
5. Verify with health check: `curl http://server:8080/healthz`

## Compatibility

- **Arcaflow Engine:** 0.20.0 or later
- **MCP Protocol:** 2025-11-25
- **Go:** 1.23.0 (exact version, for building from source)
- **Python:** 3.12 (exact version, for analysis engine)

## Contributors

Thank you to everyone who contributed to this release!

See [full changelog](https://github.com/arcalot/arcaflow-mcp/blob/v0.2.0/docs/CHANGELOG.md) for details.
```

### Using GitHub Web UI

1. Go to repository → Releases → Draft a new release
2. Choose tag: `v0.2.0`
3. Release title: `Arcaflow MCP v0.2.0`
4. Copy release notes from template above
5. Upload binaries and packages
6. Check "Create a discussion for this release"
7. Publish release

---

## Step 10: Post-Release Tasks

### Merge Release Branch

```bash
# Merge release branch back to main
git checkout main
git merge release/v0.2.0
git push origin main

# Delete release branch (optional)
git branch -d release/v0.2.0
git push origin --delete release/v0.2.0
```

### Update Development Branch

```bash
# Bump to next development version
# Edit version.go: Version = "0.3.0-dev"
# Edit pyproject.toml: version = "0.3.0-dev"

git add server/pkg/version/version.go analysis/pyproject.toml
git commit -m "chore: bump version to 0.3.0-dev for next release"
git push origin main
```

### Publish Python Package (Optional)

**To PyPI (when ready):**
```bash
cd analysis

# Publish to PyPI (requires credentials)
poetry publish --build

# Or to Test PyPI first
poetry config repositories.testpypi https://test.pypi.org/legacy/
poetry publish --build -r testpypi
```

**Note:** Python package publishing may be deferred until Phase 8.

---

## Step 11: Announcement

### Communication Channels

**GitHub:**
- Release notes published ✓
- Discussion thread created
- Pin release discussion

**Community:**
- Announce in [Arcalot Discussions](https://github.com/arcalot/arcalot-round-table)
- Update Arcaflow MCP entry in MCP server directory
- Post to relevant Slack/Discord channels

**Documentation:**
- Update arcalot.io documentation with new release
- Refresh MkDocs user documentation
- Update installation guides with new version

### Announcement Template

```markdown
# Arcaflow MCP v0.2.0 Released 🎉

We're excited to announce Arcaflow MCP v0.2.0, bringing enhanced workflow discovery and result analysis capabilities!

## Key Features

🔍 **Workflow Discovery** - Automatically find workflows in git repositories  
📊 **Advanced Analysis** - Multi-run comparison with AI-powered suggestions  
🔒 **Enterprise Ready** - Enhanced security and multi-tenancy

## Get Started

Download: https://github.com/arcalot/arcaflow-mcp/releases/tag/v0.2.0  
Docs: https://arcalot.io/arcaflow/arcaflow-mcp/  
Changelog: https://github.com/arcalot/arcaflow-mcp/blob/v0.2.0/docs/CHANGELOG.md

Try it out and let us know what you think!
```

---

## Release Checklist

### Pre-Release

- [ ] All tests pass (`./scripts/test-all.sh`)
- [ ] Version numbers updated (Go and Python)
- [ ] CHANGELOG.md updated with all changes
- [ ] Documentation references updated
- [ ] Security scan clean (no vulnerabilities)
- [ ] License headers verified
- [ ] Breaking changes documented (if any)
- [ ] Migration guide provided (if breaking changes)
- [ ] Example workflows tested
- [ ] Integration tests pass

### Release

- [ ] Release branch created (`release/vX.Y.Z`)
- [ ] Final validation complete
- [ ] Git tag created and pushed
- [ ] Binaries built for all platforms (Linux amd64/arm64, macOS amd64/arm64)
- [ ] Checksums generated
- [ ] GitHub release created
- [ ] Release notes published
- [ ] Artifacts uploaded

### Post-Release

- [ ] Release branch merged back to main
- [ ] Development version bumped (`X.Y+1.0-dev`)
- [ ] Announcement posted to GitHub Discussions
- [ ] Announced in Arcalot community
- [ ] Documentation site updated
- [ ] Python package published (if applicable)
- [ ] Container images built and published (Phase 8)

---

## Hotfix Releases

### For Critical Bugs or Security Issues

**Process:**
1. Create hotfix branch from release tag: `git checkout -b hotfix/v0.2.1 v0.2.0`
2. Fix the issue (minimal changes only)
3. Update version to patch: `0.2.0` → `0.2.1`
4. Update CHANGELOG with fix
5. Test thoroughly
6. Tag: `v0.2.1`
7. Build and release
8. Merge hotfix to both release branch and main

**Hotfix example:**
```bash
# Create hotfix branch
git checkout -b hotfix/v0.2.1 v0.2.0

# Fix the bug
# (Make minimal, targeted changes)

# Update version
# Edit version.go and pyproject.toml

# Update CHANGELOG
cat >> docs/CHANGELOG.md <<EOF

## [0.2.1] - 2026-02-05

### Fixed
- Critical: Fixed authentication bypass in server mode (#50)

### Security
- Patched token validation vulnerability (CVE-XXXX-XXXXX)
EOF

# Commit, tag, release
git commit -am "fix: patch authentication vulnerability"
git tag -a v0.2.1 -m "Hotfix release v0.2.1"
git push origin hotfix/v0.2.1 v0.2.1

# Merge back
git checkout main
git merge hotfix/v0.2.1
git push origin main
```

---

## Release Automation (Future)

### GitHub Actions (Phase 8)

Planned automation:
- Automated testing on PR
- Automated builds on tag push
- Automated GitHub release creation
- Automated container image builds
- Automated documentation deployment

**CI/CD pipeline will:**
1. Run all tests
2. Build multi-platform binaries
3. Generate checksums
4. Create GitHub release
5. Upload artifacts
6. Publish containers
7. Deploy documentation

See Phase 8 tasks for CI/CD implementation.

---

## Emergency Procedures

### Revoking a Release

If critical security issue discovered post-release:

**1. Mark release as vulnerable:**
```bash
# Add security advisory to release notes
gh release edit v0.2.0 --notes "⚠️ SECURITY: This release has a critical vulnerability. 
Upgrade to v0.2.1 immediately. See advisory: https://..."
```

**2. Create hotfix immediately:**
```bash
# Follow hotfix process above
```

**3. Notify users:**
- GitHub Security Advisory
- Email if user list available
- Pin announcement to repository

**4. Update documentation:**
- Mark version as vulnerable in compatibility matrix
- Add upgrade instructions
- Document mitigation if upgrade not immediately possible

---

## Related Documentation

- **[Contributing Guide](../../CONTRIBUTING.md)** - How to contribute
- **[Testing Guide](testing.md)** - Testing requirements
- **[CHANGELOG](../CHANGELOG.md)** - Version history
- **[Development Overview](README.md)** - Development workflow

---

[← Back to Development Documentation](README.md)
