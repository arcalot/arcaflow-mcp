#!/usr/bin/env bash
# get-container-tag.sh - Get current container image tag
#
# This script determines the appropriate container tag to use:
# - If in a git repository: uses current commit SHA
# - If not in a repository: fetches latest main branch SHA from GitHub
# - Supports both development tags (main-<sha>) and release tags (v1.0.0)

set -euo pipefail

REPO="arcalot/arcaflow-mcp"
GITHUB_API="https://api.github.com/repos/${REPO}"

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Get the appropriate container image tag for Arcaflow MCP.

OPTIONS:
    -h, --help          Show this help message
    -l, --latest        Use 'latest' tag (stable releases only)
    -m, --main          Force main branch tag (development)
    -v, --version VER   Use specific version (e.g., '1.0.0' or 'v1.0.0')
    
EXAMPLES:
    # Get current commit tag (if in repo) or latest main tag
    $(basename "$0")
    
    # Use latest stable release
    $(basename "$0") --latest
    
    # Get latest main branch development tag
    $(basename "$0") --main
    
    # Use specific version
    $(basename "$0") --version 1.0.0

OUTPUT:
    Prints the tag to use (e.g., 'main-abc1234' or 'latest')
    Sets TAG environment variable for easy reuse:
        export TAG=\$($(basename "$0"))
        podman pull quay.io/arcalot/arcaflow-mcp-server:\${TAG}
EOF
}

# Parse arguments
USE_LATEST=false
FORCE_MAIN=false
VERSION=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -l|--latest)
            USE_LATEST=true
            shift
            ;;
        -m|--main)
            FORCE_MAIN=true
            shift
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        *)
            echo "Error: Unknown option: $1" >&2
            usage >&2
            exit 1
            ;;
    esac
done

# If specific version requested
if [ -n "$VERSION" ]; then
    # Add 'v' prefix if not present
    if [[ ! "$VERSION" =~ ^v ]]; then
        echo "v${VERSION}"
    else
        echo "$VERSION"
    fi
    exit 0
fi

# If latest stable requested
if [ "$USE_LATEST" = true ]; then
    echo "latest"
    exit 0
fi

# Determine SHA
if [ "$FORCE_MAIN" = true ] || ! git rev-parse --git-dir > /dev/null 2>&1; then
    # Not in a git repo or forced main - fetch from GitHub
    BRANCH="main"
    SHA=$(curl -s "${GITHUB_API}/commits/${BRANCH}" | \
        grep -m1 '"sha"' | \
        cut -d'"' -f4 | \
        cut -c1-7)

    if [ -z "$SHA" ]; then
        echo "Error: Failed to fetch latest commit SHA from GitHub" >&2
        exit 1
    fi
else
    # In a git repo - use current commit and branch
    SHA=$(git rev-parse --short=7 HEAD)
    BRANCH=$(git rev-parse --abbrev-ref HEAD)
fi

echo "${BRANCH}-${SHA}"
