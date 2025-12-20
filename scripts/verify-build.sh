#!/bin/bash
# scripts/verify-build.sh
# Verify that a binary has real AWS pricing data embedded
#
# Usage:
#   ./scripts/verify-build.sh <binary-path>
#   ./scripts/verify-build.sh pulumicost-plugin-aws-public-us-east-1
#
# This script checks:
# 1. Binary exists
# 2. Binary size is > 10MB (indicates embedded pricing data)
# 3. Binary is executable
#
# Returns 0 if verified, 1 if failed.

set -e

BINARY="${1:-pulumicost-plugin-aws-public-us-east-1}"

if [ ! -f "$BINARY" ]; then
    echo "❌ ERROR: Binary not found: $BINARY"
    exit 1
fi

if [ ! -x "$BINARY" ]; then
    echo "⚠️  WARNING: Binary not executable: $BINARY"
    chmod +x "$BINARY"
    echo "✓ Made executable"
fi

# Get binary size (cross-platform)
SIZE=$(stat -c%s "$BINARY" 2>/dev/null || stat -f%z "$BINARY" 2>/dev/null)
MIN_SIZE=10000000  # 10MB minimum with embedded JSON

echo "Verifying binary: $BINARY"
echo "Binary size: $SIZE bytes"

if [ "$SIZE" -lt "$MIN_SIZE" ]; then
    echo ""
    echo "❌ FAIL: Binary too small - pricing data missing!"
    echo "   Expected: > $MIN_SIZE bytes (with embedded pricing JSON)"
    echo "   Got:      $SIZE bytes (likely fallback/dummy pricing only)"
    echo ""
    echo "This indicates the binary was built without region tags."
    echo "The v0.0.10 release had this issue, resulting in all costs being $0."
    echo ""
    echo "To rebuild with correct tags:"
    echo "  go build -tags=region_use1 -o $BINARY ./cmd/pulumicost-plugin-aws-public/"
    exit 1
fi

echo ""
echo "✓ PASS: Binary size indicates embedded pricing data is present"
echo "        ($SIZE bytes is adequate for ~7MB pricing JSON)"
exit 0
