#!/bin/bash
# Build and release binaries for a single AWS region
# Usage: ./scripts/release-region.sh <region>
#
# This script:
# 1. Generates pricing data for the specified region only
# 2. Builds and archives 6 binaries (3 OS × 2 arch) using goreleaser
# 3. Cleans up Go build cache to free disk space
#
# Designed to be run sequentially for each region to minimize disk usage.
# Archives are accumulated in dist/ for final upload.

set -euo pipefail

REGION="${1:-}"

if [[ "$REGION" == "" ]]; then
    echo "Usage: $0 <region>"
    echo "Example: $0 us-east-1"
    exit 1
fi

# Map region to build tag
declare -A REGION_TAGS=(
    ["us-east-1"]="region_use1"
    ["us-west-2"]="region_usw2"
    ["eu-west-1"]="region_euw1"
    ["ap-southeast-1"]="region_apse1"
    ["ap-southeast-2"]="region_apse2"
    ["ap-northeast-1"]="region_apne1"
    ["ap-south-1"]="region_aps1"
    ["ca-central-1"]="region_cac1"
    ["sa-east-1"]="region_sae1"
)

TAG="${REGION_TAGS[$REGION]:-}"
if [[ "$TAG" == "" ]]; then
    echo "ERROR: Unknown region '$REGION'"
    echo "Valid regions: ${!REGION_TAGS[*]}"
    exit 1
fi

echo "=== Building region: $REGION (tag: $TAG) ==="

# Step 1: Generate pricing data for this region only
echo "Generating pricing data for $REGION..."
go run ./tools/generate-pricing --regions "$REGION" --out-dir ./internal/pricing/data

# Step 2: Build using goreleaser with region-specific config
# Note: We use --skip=validate,announce,publish to just build archives
# The main workflow handles the final release upload
echo "Building binaries for $REGION..."
cat > ".goreleaser.region.yaml" << EOF
version: 2

builds:
  - id: $REGION
    main: ./cmd/pulumicost-plugin-aws-public
    binary: pulumicost-plugin-aws-public-$REGION
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    tags:
      - $TAG
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - id: $REGION
    format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}_$REGION
    format_overrides:
      - goos: windows
        format: zip

# Disable checksum per-region - we'll generate combined checksums at the end
checksum:
  disable: true

changelog:
  disable: true

source:
  enabled: false

# Skip everything except build+archive - we just want the artifacts
release:
  disable: true
EOF

goreleaser release --config .goreleaser.region.yaml --skip=validate,announce,publish

# Step 3: Clean up to free disk space for next region
echo "Cleaning up build cache..."
rm -f .goreleaser.region.yaml
# Remove the raw binaries but keep archives
rm -rf dist/${REGION}_*
go clean -cache

echo "=== Completed: $REGION ==="
