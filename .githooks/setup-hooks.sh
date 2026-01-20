#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if ! command -v git >/dev/null 2>&1; then
  echo "git is required to install hooks" >&2
  exit 1
fi

echo "Installing git hooks from ${repo_root}/.githooks"
git -C "${repo_root}" config core.hooksPath ".githooks"
echo "Hooks installed."
