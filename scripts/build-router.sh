#!/usr/bin/env bash
# build-router.sh - Build the multi-region router binary.
# The router binary requires no region build tags since it contains no embedded pricing data.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

OUTPUT_DIR="${REPO_ROOT}/dist"
mkdir -p "${OUTPUT_DIR}"

echo "Building router binary..."
go build -o "${OUTPUT_DIR}/finfocus-plugin-aws-public" ./cmd/finfocus-plugin-aws-public-router

echo "Router binary built: ${OUTPUT_DIR}/finfocus-plugin-aws-public"
ls -lh "${OUTPUT_DIR}/finfocus-plugin-aws-public"
