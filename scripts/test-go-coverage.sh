#!/usr/bin/env bash
# Run Go tests with coverage for the server module.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
server_dir="${repo_root}/server"
cache_root="${repo_root}/.gocache"
mod_cache="${repo_root}/.gomodcache"

if [[ ! -f "${server_dir}/go.mod" ]]; then
  echo "Go module not initialized in ${server_dir}." >&2
  exit 1
fi

cd "${server_dir}"
mkdir -p "${cache_root}" "${mod_cache}"
export GOCACHE="${cache_root}"
export GOMODCACHE="${mod_cache}"
go test ./... -coverprofile="${server_dir}/coverage.out"
