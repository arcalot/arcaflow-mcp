#!/usr/bin/env bash
# Run Go and Python linters for tracked modules.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
server_dir="${repo_root}/server"
analysis_dir="${repo_root}/analysis"

if [[ -f "${server_dir}/go.mod" ]]; then
  if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "golangci-lint is required for Go linting." >&2
    exit 1
  fi
  (cd "${server_dir}" && golangci-lint run ./...)
fi

if [[ -f "${analysis_dir}/pyproject.toml" ]]; then
  (cd "${analysis_dir}" && poetry run ruff check .)
fi
