#!/usr/bin/env bash
# Run both Go and Python test suites.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

"${repo_root}/scripts/test-go.sh"
"${repo_root}/scripts/test-python.sh"
