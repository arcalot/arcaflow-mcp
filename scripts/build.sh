#!/usr/bin/env bash
# Build Go and Python artifacts when modules are present.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
server_dir="${repo_root}/server"
analysis_dir="${repo_root}/analysis"

if [[ -f "${server_dir}/go.mod" ]]; then
  (cd "${server_dir}" && go build ./...)
fi

if [[ -f "${analysis_dir}/pyproject.toml" ]]; then
  (cd "${analysis_dir}" && poetry build)
fi
