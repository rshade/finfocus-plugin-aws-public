#!/usr/bin/env bash
# build-router.sh - Build the multi-region router binary for local development.
# The router binary requires no region build tags since it contains no
# embedded pricing data.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Compute dev version from latest tag
LATEST_TAG=$(git -C "${REPO_ROOT}" describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
LATEST_VERSION="${LATEST_TAG#v}"
MAJOR=$(echo "$LATEST_VERSION" | cut -d. -f1)
MINOR=$(echo "$LATEST_VERSION" | cut -d. -f2)
PATCH=$(echo "$LATEST_VERSION" | cut -d. -f3 | grep -E '^[0-9]+$' || echo "0")
NEXT_PATCH=$((PATCH + 1))
DEV_VERSION="${MAJOR}.${MINOR}.${NEXT_PATCH}-dev"

OUTPUT="${REPO_ROOT}/finfocus-plugin-aws-public"

echo "Building router binary (version: ${DEV_VERSION})..."
go build -ldflags "-X main.version=${DEV_VERSION}" \
    -o "${OUTPUT}" \
    "${REPO_ROOT}/cmd/finfocus-plugin-aws-public-router"

echo "Router binary built: ${OUTPUT}"
ls -lh "${OUTPUT}"
