#!/usr/bin/env bash
# Run Go and Python linters for tracked modules.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
server_dir="${repo_root}/server"
analysis_dir="${repo_root}/analysis"
cache_root="${repo_root}/.gocache"
mod_cache="${repo_root}/.gomodcache"

if [[ -f "${server_dir}/go.mod" ]]; then
  if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "golangci-lint is required for Go linting." >&2
    exit 1
  fi
  mkdir -p "${cache_root}" "${mod_cache}"
  export GOCACHE="${cache_root}"
  export GOMODCACHE="${mod_cache}"
  (cd "${server_dir}" && golangci-lint run ./...)
fi

if [[ -f "${analysis_dir}/pyproject.toml" ]]; then
  echo "Running Python linters (black, flake8)..."
  (cd "${analysis_dir}" && poetry run black --check .)
  (cd "${analysis_dir}" && poetry run flake8 .)
fi
