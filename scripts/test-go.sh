#!/usr/bin/env bash
# Run Go unit tests for the server module.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
server_dir="${repo_root}/server"

if [[ ! -f "${server_dir}/go.mod" ]]; then
  echo "Go module not initialized in ${server_dir}." >&2
  exit 1
fi

cd "${server_dir}"
go test ./...
