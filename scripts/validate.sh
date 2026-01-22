#!/usr/bin/env bash
# Run lint and full test suite for validation.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

"${repo_root}/scripts/lint.sh"
"${repo_root}/scripts/test-all.sh"
"${repo_root}/scripts/test-go-coverage.sh"
"${repo_root}/scripts/test-python-coverage.sh"
