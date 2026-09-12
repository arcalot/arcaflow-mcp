#!/usr/bin/env bash
# Run Python unit tests for the analysis module.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
analysis_dir="${repo_root}/analysis"

if [[ ! -f "${analysis_dir}/pyproject.toml" ]]; then
  echo "Python project not initialized in ${analysis_dir}." >&2
  exit 1
fi

cd "${analysis_dir}"
export PYTHONPATH="${analysis_dir}"
poetry run pytest
