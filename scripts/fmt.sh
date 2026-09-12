#!/usr/bin/env bash
# Format Go and Python sources using configured formatters.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
server_dir="${repo_root}/server"
analysis_dir="${repo_root}/analysis"

if [[ -d "${server_dir}" ]]; then
  find "${server_dir}" -name "*.go" -print0 | xargs -0 gofmt -w
fi

if [[ -f "${analysis_dir}/pyproject.toml" ]]; then
  (cd "${analysis_dir}" && poetry run black .)
fi
