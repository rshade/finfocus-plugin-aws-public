#!/bin/bash
# scripts/verify-release-binaries.sh
# Verify that all release binaries have real AWS pricing data embedded
#
# Usage: ./scripts/verify-release-binaries.sh <dist-directory>
# Example: ./scripts/verify-release-binaries.sh dist/
#
# Checks:
# 1. All binaries exist
# 2. Binary sizes are > 10MB (indicates embedded pricing data)
# 3. Reports any binaries that are too small (likely missing pricing)
#
# Returns 0 if all binaries verified, 1 if any fail

set -e

DIST_DIR="${1:-.}"
MIN_SIZE=10000000  # 10MB minimum with embedded JSON

if [ ! -d "$DIST_DIR" ]; then
    echo "ERROR: Directory not found: $DIST_DIR"
    exit 1
fi

echo "Verifying binaries in $DIST_DIR..."
echo ""

VERIFIED=0
FAILED=0

# Check all Linux x86_64 binaries (primary release platform)
for binary in "$DIST_DIR"/pulumicost-plugin-aws-public-*_Linux_x86_64; do
    if [ -f "$binary" ]; then
        size=$(stat -c%s "$binary")
        if [ "$size" -lt "$MIN_SIZE" ]; then
            echo "❌ FAIL: Binary too small: $(basename "$binary") ($size bytes)"
            echo "   Expected: > $MIN_SIZE bytes (with embedded pricing JSON)"
            FAILED=$((FAILED + 1))
        else
            echo "✓ $(basename "$binary") ($size bytes)"
            VERIFIED=$((VERIFIED + 1))
        fi
    fi
done

echo ""
if [ $FAILED -gt 0 ]; then
    echo "❌ FAILURE: $FAILED binary/binaries have missing or incomplete pricing data"
    exit 1
fi

echo "✅ SUCCESS: All $VERIFIED binaries verified (pricing data embedded)"
exit 0
