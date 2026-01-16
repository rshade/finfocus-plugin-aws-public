#!/bin/bash
set -eu

IMAGE_NAME=${1:-finfocus-plugin-aws-public:latest}

if ! command -v trivy >/dev/null 2>&1; then
  echo "Trivy not installed. Please install Trivy before running this script." >&2
  exit 1
fi

echo "Running Trivy scan for ${IMAGE_NAME}"
trivy image --severity HIGH,CRITICAL --exit-code 1 --no-progress --ignore-unfixed "${IMAGE_NAME}"
