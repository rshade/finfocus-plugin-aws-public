#!/bin/bash
# Build and archive the multi-region router binary.
# Usage: ./scripts/release-router.sh
#
# This script:
# 1. Creates a per-router GoReleaser config (no build tags, no pricing)
# 2. Builds archives for 6 platforms (3 OS × 2 arch)
# 3. Moves archives to dist/ for final upload
#
# The router binary is named finfocus-plugin-aws-public (no region suffix).

set -euo pipefail

echo "=== Building router binary ==="

# GoReleaser builds: linux/{amd64,arm64}, darwin/{amd64,arm64}, windows/{amd64,arm64}
# = 4 tar.gz + 2 zip = 6 archives. Update if platform matrix changes.
expected_count=6

# Ensure temp config is cleaned up on any exit (including set -e failures)
trap 'rm -f .goreleaser.router.yaml' EXIT

cat > ".goreleaser.router.yaml" << 'EOF'
version: 2

dist: _build/router

builds:
  - id: router
    main: ./cmd/finfocus-plugin-aws-public-router
    binary: finfocus-plugin-aws-public
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - id: router
    format: tar.gz
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

changelog:
  disable: true

source:
  enabled: false

release:
  disable: true
EOF

goreleaser release --config .goreleaser.router.yaml --skip=validate,announce,publish --clean

# Move archives to main dist folder
mkdir -p dist
shopt -s nullglob
archives=(_build/router/*.tar.gz _build/router/*.zip)
if [ "${#archives[@]}" -eq 0 ]; then
  echo "ERROR: No archives found matching _build/router/*.tar.gz or _build/router/*.zip" >&2
  exit 1
fi
if [ "${#archives[@]}" -ne "$expected_count" ]; then
  echo "ERROR: Expected $expected_count archives but found ${#archives[@]} in _build/router/" >&2
  echo "  Expected: 4 tar.gz (linux/darwin × amd64/arm64) + 2 zip (windows × amd64/arm64)" >&2
  printf "  Found: %s\n" "${archives[@]}" >&2
  exit 1
fi
mv "${archives[@]}" dist/
shopt -u nullglob
rm -rf _build

echo "=== Router build complete ==="
shopt -s nullglob
dist_archives=(dist/finfocus-plugin-aws-public_*.tar.gz dist/finfocus-plugin-aws-public_*.zip)
shopt -u nullglob
if [ "${#dist_archives[@]}" -ne "$expected_count" ]; then
  echo "ERROR: Expected $expected_count archives in dist/ but found ${#dist_archives[@]}" >&2
  exit 1
fi
for f in "${dist_archives[@]}"; do
  ls -lh "$f"
done
