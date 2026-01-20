#!/usr/bin/env bash
# Build multi-arch container images with buildah/podman.
set -euo pipefail

# Requirements:
# - buildah, podman, skopeo (optional), qemu-user-static recommended
#
# Env vars:
#   IMAGE_REPO      Base repo (e.g., quay.io/org) [optional]
#   REGISTRY_USERNAME / REGISTRY_PASSWORD  Registry credentials (optional)
#   QUAY_USERNAME / QUAY_PASSWORD          Back-compat for Quay (optional)
#   OCI_REVISION    Revision label value (optional)
#
# Options:
#   -t, --tag <tag>           Image tag (default: git short SHA or timestamp)
#   -e, --expires <period>    Expiration label value (default: 90d)
#   --expires-label <key>     Expiration label key (default: quay.expires-after)
#   --push                    Push manifest to registry
#   --push-main               Also push/alias to :main
#   --push-alias <tag>        Push additional alias tag (repeatable)
#   -f, --file <path>         Containerfile path (default: Containerfile)
#   -c, --context <path>      Build context path (default: .)
#   -n, --name <name>         Image name (default: arcaflow-mcp)
#   -h, --help                Show help

show_help() {
  cat <<EOF
Usage: IMAGE_REPO=quay.io/org $(basename "$0") [options]

Options:
  -t, --tag <tag>           Image tag (default: git short SHA or timestamp)
  -e, --expires <period>    Expiration label value (default: 90d)
      --expires-label <key> Expiration label key (default: quay.expires-after)
      --push                Push manifest to registry
      --push-main           Also push/alias to :main
  -f, --file <path>         Containerfile path (default: Containerfile)
  -c, --context <path>      Build context path (default: .)
  -n, --name <name>         Image name (default: arcaflow-mcp)
      --push-alias <tag>    Push additional alias tag (repeatable)
  -h, --help                Show this help
EOF
}

TAG=""
EXPIRES="90d"
EXPIRES_LABEL="quay.expires-after"
PUSH=0
PUSH_MAIN=0
PUSH_ALIASES=()
CONTAINERFILE="Containerfile"
CONTEXT="."
IMAGE_NAME="arcaflow-mcp"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -t|--tag)
      TAG="${2:-}"
      shift 2 ;;
    -e|--expires)
      EXPIRES="${2:-}"
      shift 2 ;;
    --expires-label)
      EXPIRES_LABEL="${2:-quay.expires-after}"
      shift 2 ;;
    --push)
      PUSH=1
      shift 1 ;;
    --push-main)
      PUSH_MAIN=1
      shift 1 ;;
    --push-alias)
      PUSH_ALIASES+=("${2:-}")
      shift 2 ;;
    -f|--file)
      CONTAINERFILE="${2:-Containerfile}"
      shift 2 ;;
    -c|--context)
      CONTEXT="${2:-.}"
      shift 2 ;;
    -n|--name)
      IMAGE_NAME="${2:-arcaflow-mcp}"
      shift 2 ;;
    -h|--help)
      show_help
      exit 0 ;;
    *)
      echo "Unknown option: $1" >&2
      show_help
      exit 1 ;;
  esac
done

if [[ -z "${IMAGE_REPO:-}" ]]; then
  IMAGE_REPO="localhost"
  echo "INFO: Using default IMAGE_REPO: ${IMAGE_REPO}" >&2
fi

if [[ -z "$TAG" ]]; then
  if command -v git >/dev/null 2>&1; then
    TAG="$(git rev-parse --short HEAD 2>/dev/null || true)"
  fi
  TAG=${TAG:-"local-$(date +%Y%m%d%H%M%S)"}
fi

REGISTRY_HOST="${IMAGE_REPO%%/*}"
IMAGE_REF="${IMAGE_REPO}/${IMAGE_NAME}"
LOCAL_MANIFEST_REF="${IMAGE_REF}:${TAG}"

if [[ "$PUSH" -eq 1 ]]; then
  USERNAME="${REGISTRY_USERNAME:-${QUAY_USERNAME:-}}"
  PASSWORD="${REGISTRY_PASSWORD:-${QUAY_PASSWORD:-}}"
  if [[ -n "$USERNAME" && -n "$PASSWORD" ]]; then
    if command -v podman >/dev/null 2>&1; then
      podman login -u "$USERNAME" -p "$PASSWORD" "$REGISTRY_HOST"
    fi
    if command -v buildah >/dev/null 2>&1; then
      buildah login -u "$USERNAME" -p "$PASSWORD" "$REGISTRY_HOST"
    fi
  else
    echo "INFO: No registry credentials provided; proceeding without login." >&2
  fi
fi

buildah manifest rm "$LOCAL_MANIFEST_REF" >/dev/null 2>&1 || true
buildah manifest create "$LOCAL_MANIFEST_REF"

build_for_arch() {
  local arch="$1"
  local label_revision="org.opencontainers.image.revision=${OCI_REVISION:-$TAG}"
  local build_args=(--override-arch "$arch" --override-os linux -f "$CONTAINERFILE")
  build_args+=(--label "$label_revision")
  if [[ -n "$EXPIRES" ]]; then
    build_args+=(--label "${EXPIRES_LABEL}=${EXPIRES}")
  fi
  echo "Building ${arch} image..."
  buildah bud "${build_args[@]}" -t "${IMAGE_REF}:${TAG}-${arch}" "$CONTEXT"
  buildah manifest add "$LOCAL_MANIFEST_REF" \
    "containers-storage:${IMAGE_REF}:${TAG}-${arch}"
}

build_for_arch amd64
build_for_arch arm64

CURRENT_ARCH=$(uname -m)
case "$CURRENT_ARCH" in
  x86_64) LOCAL_ARCH_TAG="${IMAGE_REF}:${TAG}-amd64" ;;
  aarch64|arm64) LOCAL_ARCH_TAG="${IMAGE_REF}:${TAG}-arm64" ;;
  *)
    echo "WARNING: Unknown arch $CURRENT_ARCH, defaulting to amd64" >&2
    LOCAL_ARCH_TAG="${IMAGE_REF}:${TAG}-amd64" ;;
esac

podman tag "$LOCAL_ARCH_TAG" "${IMAGE_REF}:${TAG}-local"

if [[ "$PUSH" -eq 1 ]]; then
  echo "Pushing multi-arch manifest to ${IMAGE_REF}:${TAG}..."
  buildah manifest push --all "$LOCAL_MANIFEST_REF" "docker://${IMAGE_REF}:${TAG}"
  if [[ "$PUSH_MAIN" -eq 1 ]]; then
    PUSH_ALIASES+=("main")
  fi
  for alias in "${PUSH_ALIASES[@]}"; do
    if [[ -n "$alias" ]]; then
      echo "Also pushing manifest to :${alias}..."
      buildah manifest push --all "$LOCAL_MANIFEST_REF" \
        "docker://${IMAGE_REF}:${alias}"
    fi
  done
else
  echo "Built manifest locally: ${LOCAL_MANIFEST_REF} (not pushed)" >&2
  echo "Local runnable image: ${IMAGE_REF}:${TAG}-local" >&2
fi

echo "Done: ${IMAGE_REF}:${TAG}" >&2
