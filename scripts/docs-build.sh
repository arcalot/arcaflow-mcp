#!/usr/bin/env bash
# Build user documentation with MkDocs.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

mkdocs_arcaflow="${repo_root}/docs/mkdocs-arcaflow.yml"

if [[ ! -f "${mkdocs_arcaflow}" ]]; then
  echo "Missing ${mkdocs_arcaflow}." >&2
  exit 1
fi

mkdocs build -f "${mkdocs_arcaflow}"
