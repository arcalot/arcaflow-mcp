#!/usr/bin/env bash
# Run integration validation with a real Arcaflow engine and workflow.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

engine_version="v0.20.0"
workflow_ref="84a7c4fdd55dcbd2c968bc13ad21c79a0c91e1f0"

engine_dir="${repo_root}/manual-validation/engine"
workflow_dir="${repo_root}/manual-validation/workflows/basic"
export_dir="${repo_root}/manual-validation/exports"
result_dir="${repo_root}/manual-validation/results"
cache_dir="${repo_root}/manual-validation/tmp"

mkdir -p "${engine_dir}" "${workflow_dir}" "${export_dir}" "${result_dir}" "${cache_dir}"

os_name="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch_name="$(uname -m)"
case "${arch_name}" in
  x86_64|amd64)
    arch_name="amd64"
    ;;
  aarch64|arm64)
    arch_name="arm64"
    ;;
  *)
    echo "unsupported architecture: ${arch_name}" >&2
    exit 1
    ;;
esac

case "${os_name}" in
  linux|darwin)
    ;;
  *)
    echo "unsupported operating system: ${os_name}" >&2
    exit 1
    ;;
esac

engine_version_trimmed="${engine_version#v}"
engine_tarball="arcaflow_${engine_version_trimmed}_${os_name}_${arch_name}.tar.gz"
engine_url="https://github.com/arcalot/arcaflow-engine/releases/download/${engine_version}/${engine_tarball}"
engine_bin="${engine_dir}/arcaflow"
engine_config="${repo_root}/manual-validation/engine-config.yaml"

if [[ ! -x "${engine_bin}" ]]; then
  curl -sSfL -o "${engine_dir}/${engine_tarball}" "${engine_url}"
  tar -xzf "${engine_dir}/${engine_tarball}" -C "${engine_dir}" --no-same-owner
  chmod +x "${engine_bin}"
fi

if [[ ! -f "${engine_config}" ]]; then
  cat > "${engine_config}" <<'EOF'
deployers:
  image:
    deployer_name: docker
log:
  level: info
logged_outputs:
  error:
    level: debug
EOF
fi

curl -sSfL -o "${workflow_dir}/workflow.yaml" \
  "https://raw.githubusercontent.com/arcalot/arcaflow-workflows/${workflow_ref}/basic-examples/basic/workflow.yaml"
curl -sSfL -o "${workflow_dir}/input.yaml" \
  "https://raw.githubusercontent.com/arcalot/arcaflow-workflows/${workflow_ref}/basic-examples/basic/input.yaml"

(
  cd "${repo_root}/server"
  GOCACHE="${repo_root}/.cache/go-build" \
  GOMODCACHE="${repo_root}/.cache/go-mod" \
  TMPDIR="${cache_dir}" \
  CCACHE_DISABLE=1 \
  CGO_ENABLED=0 \
  go run ./cmd/manual-validate \
    --workflow "${workflow_dir}/workflow.yaml" \
    --input "${workflow_dir}/input.yaml" \
    --output "${export_dir}/basic-input.json" \
    --format json
)

"${engine_bin}" \
  --workflow "${workflow_dir}/workflow.yaml" \
  --input "${export_dir}/basic-input.json" \
  --context "${workflow_dir}" \
  --config "${engine_config}" \
  > "${result_dir}/basic-result.yaml"

(
  cd "${repo_root}/analysis"
  POETRY_CACHE_DIR="${repo_root}/.cache/poetry" \
  POETRY_VIRTUALENVS_IN_PROJECT=true \
  poetry install --no-root
  POETRY_CACHE_DIR="${repo_root}/.cache/poetry" \
  POETRY_VIRTUALENVS_IN_PROJECT=true \
  poetry run python - <<'PY'
from arcaflow_analysis.analyzer.result_analyzer import ResultAnalyzer
from arcaflow_analysis.analyzer.result_comparator import ResultComparator
from arcaflow_analysis.parser.result_loader import ResultLoader
from arcaflow_analysis.parser.result_parser import ResultParser
from arcaflow_analysis.suggester.suggestion_engine import SuggestionEngine

result_path = "../manual-validation/results/basic-result.yaml"
loader = ResultLoader()
entry = loader.load_file(result_path)
payload = entry.payload or {}
if not isinstance(payload, dict) or payload.get("output_id") != "success":
    raise SystemExit("expected workflow output_id 'success'")

parser = ResultParser()
parsed = [parser.parse(entry)]
analysis = ResultAnalyzer().analyze(parsed)
comparison = ResultComparator().compare(parsed, {})
_ = SuggestionEngine().suggest(analysis, comparison)
PY
)
