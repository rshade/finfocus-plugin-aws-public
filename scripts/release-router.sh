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
mv _build/router/*.tar.gz _build/router/*.zip dist/ 2>/dev/null || true
rm -rf _build

echo "=== Router build complete ==="
found=false
for f in dist/finfocus-plugin-aws-public_*; do
  [ -e "$f" ] || continue
  case "$(basename "$f")" in *-[a-z]*) continue;; esac
  ls -lh "$f"
  found=true
done
[ "$found" = true ] || { echo "ERROR: No router archives found" >&2; exit 1; }
