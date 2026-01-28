#!/usr/bin/env bash
# Install git hooks and validate development environment.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "=== Arcaflow MCP Developer Setup ==="
echo ""

# Check required tools
echo "Step 1: Checking required tools..."
missing_tools=()

if ! command -v go &> /dev/null; then
    missing_tools+=("Go")
fi

if ! command -v python3 &> /dev/null; then
    missing_tools+=("Python 3")
fi

if ! command -v poetry &> /dev/null; then
    missing_tools+=("Poetry")
fi

if ! command -v git &> /dev/null; then
    missing_tools+=("Git")
fi

if [ ${#missing_tools[@]} -gt 0 ]; then
    echo "ERROR: Missing required tools: ${missing_tools[*]}"
    echo ""
    echo "Required tools (version requirements in .github/workflows/ci.yml):"
    echo "  - Go (see ARCALOT_GO_VERSION)"
    echo "  - Python (see ARCALOT_PYTHON_SUPPORTED_VERSIONS)"
    echo "  - Poetry"
    echo "  - Git"
    echo ""
    echo "Install instructions: docs/development/setup.md"
    exit 1
fi

echo "✓ All required tools found"
echo ""
echo "Step 2: Installing git hooks..."

hooks_setup="${repo_root}/.githooks/setup-hooks.sh"

if [[ ! -x "${hooks_setup}" ]]; then
  echo "ERROR: Missing ${hooks_setup}. Ensure .githooks is present." >&2
  exit 1
fi

"${hooks_setup}"

echo ""
echo "✓ Developer setup complete!"
echo ""
echo "Next steps:"
echo "  1. cd analysis && poetry install  # Install Python dependencies"
echo "  2. cd server && go mod download    # Download Go dependencies"
echo "  3. ./scripts/validate.sh          # Verify everything works"
