#!/usr/bin/env bash
# Scan a container image with Trivy.
set -euo pipefail

IMAGE_REF="${1:-}"
if [[ -z "$IMAGE_REF" ]]; then
  echo "Usage: $(basename "$0") <image-ref>" >&2
  exit 2
fi

if ! command -v trivy >/dev/null 2>&1; then
  echo "ERROR: trivy not found. Install from https://aquasecurity.github.io/trivy/" >&2
  exit 3
fi

trivy image \
  --severity HIGH,CRITICAL \
  --exit-code 1 \
  --ignore-unfixed \
  --format table \
  "$IMAGE_REF"

echo "Trivy scan passed for $IMAGE_REF"
