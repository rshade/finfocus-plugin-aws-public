#!/bin/bash
# Build binaries for a single AWS region
# Usage: ./scripts/build-region.sh <region> [--snapshot]
#
# This script:
# 1. Generates pricing data for the specified region only
# 2. Builds 6 binaries (3 OS × 2 arch) using goreleaser
# 3. Cleans up Go build cache to free disk space
#
# Designed to be run sequentially for each region to minimize disk usage.

set -euo pipefail

REGION="${1:-}"
SNAPSHOT_FLAG=""

if [[ "$REGION" == "" ]]; then
    echo "Usage: $0 <region> [--snapshot]"
    echo "Example: $0 us-east-1 --snapshot"
    exit 1
fi

# Check for --snapshot flag
if [[ "${2:-}" == "--snapshot" ]]; then
    SNAPSHOT_FLAG="--snapshot"
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
  - format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
    format_overrides:
      - goos: windows
        format: zip

checksum:
  disable: true

snapshot:
  version_template: "{{ incpatch .Version }}-next"

changelog:
  disable: true

source:
  enabled: false
EOF

goreleaser build --config .goreleaser.region.yaml $SNAPSHOT_FLAG

# Step 3: Clean up to free disk space for next region
echo "Cleaning up build cache..."
rm -f .goreleaser.region.yaml
go clean -cache

echo "=== Completed: $REGION ==="
