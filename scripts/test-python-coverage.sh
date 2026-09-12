#!/usr/bin/env bash
# Run Python unit tests with coverage for the analysis module.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
analysis_dir="${repo_root}/analysis"
cache_dir="${repo_root}/.poetry-cache"

if [[ ! -f "${analysis_dir}/pyproject.toml" ]]; then
  echo "Python project not initialized in ${analysis_dir}." >&2
  exit 1
fi

cd "${analysis_dir}"
export PYTHONPATH="${analysis_dir}"
export POETRY_CACHE_DIR="${cache_dir}"
export POETRY_VIRTUALENVS_PATH="${cache_dir}/virtualenvs"
export VIRTUALENV_OVERRIDE_APP_DATA="${cache_dir}/virtualenv"
poetry install --no-root
poetry run pytest \
  --cov=arcaflow_analysis \
  --cov-report=term-missing \
  --cov-report=xml:coverage.xml
