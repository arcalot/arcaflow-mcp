#!/usr/bin/env bash
# Install git hooks for local development.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

hooks_setup="${repo_root}/.githooks/setup-hooks.sh"

if [[ ! -x "${hooks_setup}" ]]; then
  echo "Missing ${hooks_setup}. Ensure .githooks is present." >&2
  exit 1
fi

"${hooks_setup}"

echo "Developer setup complete."
