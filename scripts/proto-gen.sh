#!/usr/bin/env bash
# Generate Go and Python gRPC sources from proto files.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
proto_dir="${repo_root}/api/proto"
gen_dir="${repo_root}/api/generated"

if [[ ! -d "${proto_dir}" ]]; then
  echo "No proto directory found at ${proto_dir}." >&2
  exit 1
fi

mkdir -p "${gen_dir}/go" "${gen_dir}/python"

if ! command -v protoc >/dev/null 2>&1; then
  echo "protoc is required to generate gRPC code." >&2
  exit 1
fi

shopt -s nullglob
proto_files=("${proto_dir}"/*.proto)
if [[ ${#proto_files[@]} -eq 0 ]]; then
  echo "No proto files found in ${proto_dir}."
  exit 0
fi

if ! command -v protoc-gen-go >/dev/null 2>&1; then
  echo "protoc-gen-go is required." >&2
  exit 1
fi

if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
  echo "protoc-gen-go-grpc is required." >&2
  exit 1
fi

if ! command -v python >/dev/null 2>&1; then
  echo "python is required for Python gRPC generation." >&2
  exit 1
fi

python -m grpc_tools.protoc \
  -I "${proto_dir}" \
  --python_out "${gen_dir}/python" \
  --grpc_python_out "${gen_dir}/python" \
  "${proto_files[@]}"

protoc \
  -I "${proto_dir}" \
  --go_out "${gen_dir}/go" \
  --go-grpc_out "${gen_dir}/go" \
  "${proto_files[@]}"
