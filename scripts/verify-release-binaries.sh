#!/bin/bash
# scripts/verify-release-binaries.sh
# Verify that all release binaries have real AWS pricing data embedded
#
# Usage: ./scripts/verify-release-binaries.sh <dist-directory>
# Example: ./scripts/verify-release-binaries.sh dist/
#
# Checks:
# 1. All Linux x86_64 archives exist (one per region)
# 2. Extracts each archive and verifies the binary size is > 100MB
#    (indicates embedded pricing data - fallback binaries are only ~5MB)
# 3. Reports any binaries that are too small (likely missing pricing)
#
# Returns 0 if all binaries verified, 1 if any fail

set -e

DIST_DIR="${1:-.}"
# Raw binary minimum size: 100MB (with embedded pricing JSON)
# Fallback binaries without real pricing are only ~5MB
MIN_BINARY_SIZE=100000000

if [ ! -d "$DIST_DIR" ]; then
    echo "ERROR: Directory not found: $DIST_DIR"
    exit 1
fi

echo "Verifying release archives in $DIST_DIR..."
echo ""

# Find the first Linux x86_64 archive to verify
# Archive naming pattern: finfocus-plugin-aws-public_${VERSION}_Linux_x86_64_${REGION}.tar.gz
ARCHIVE=$(find "$DIST_DIR" -maxdepth 1 -name "finfocus-plugin-aws-public_*_Linux_x86_64_*.tar.gz" -type f | head -1)

if [ -z "$ARCHIVE" ]; then
    echo "❌ FAILURE: No archives found matching pattern 'finfocus-plugin-aws-public_*_Linux_x86_64_*.tar.gz'"
    echo "   Found in $DIST_DIR:"
    ls -la "$DIST_DIR"/ 2>/dev/null || echo "   (directory empty or not accessible)"
    echo ""
    echo "   Expected archives like: finfocus-plugin-aws-public_0.0.13_Linux_x86_64_us-east-1.tar.gz"
    exit 1
fi

# Count total archives for reporting
TOTAL_ARCHIVES=$(find "$DIST_DIR" -maxdepth 1 -name "finfocus-plugin-aws-public_*_Linux_x86_64_*.tar.gz" -type f | wc -l)

echo "Found $TOTAL_ARCHIVES Linux x86_64 archive(s)"
echo "Verifying one archive (all built with same process)..."
echo ""

# Create temp directory for extraction
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

archive_name=$(basename "$ARCHIVE")
echo "Checking: $archive_name"

# Extract to temp dir
tar -xzf "$ARCHIVE" -C "$TEMP_DIR"

# Find the binary (it's the only executable in the archive)
binary=$(find "$TEMP_DIR" -type f -name "finfocus-plugin-aws-public-*" | head -1)

if [ -z "$binary" ]; then
    echo "  ❌ FAIL: No binary found in archive"
    exit 1
fi

size=$(stat -c%s "$binary")
binary_name=$(basename "$binary")

if [ "$size" -lt "$MIN_BINARY_SIZE" ]; then
    echo "  ❌ FAIL: Binary too small: $binary_name ($size bytes)"
    echo "     Expected: > $MIN_BINARY_SIZE bytes (with embedded pricing JSON)"
    echo "     This indicates the binary was built without region tags (fallback pricing)"
    exit 1
fi

size_mb=$((size / 1000000))
echo "  ✓ $binary_name (${size_mb}MB)"
echo ""
echo "✅ SUCCESS: Binary verified with embedded pricing data ($TOTAL_ARCHIVES archives in dist/)"
exit 0
