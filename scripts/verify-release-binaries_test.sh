#!/bin/bash
# Test suite for verify-release-binaries.sh

set -e

TEST_DIR=$(mktemp -d)
SCRIPT_TO_TEST="$(pwd)/scripts/verify-release-binaries.sh"
trap 'rm -rf "$TEST_DIR"' EXIT

echo "Running tests in $TEST_DIR"

# make_archive <dist-dir> <region> <size-in-bytes>
make_archive() {
    local dist="$1" region="$2" size="$3"
    local stage="$TEST_DIR/stage-$region"
    mkdir -p "$dist" "$stage"
    truncate -s "$size" "$stage/finfocus-plugin-aws-public-$region"
    tar -czf "$dist/finfocus-plugin-aws-public_0.0.0_Linux_x86_64_$region.tar.gz" \
        -C "$stage" "finfocus-plugin-aws-public-$region"
    rm -rf "$stage"
}

run_verify() {
    MIN_BINARY_SIZE=1000 WARN_BINARY_SIZE=5000 MAX_BINARY_SIZE=10000 \
        bash "$SCRIPT_TO_TEST" "$1"
}

# An oversized binary fails no matter which archive find returns first.
for order in "aaa-small zzz-big" "zzz-small aaa-big"; do
    set -- $order
    dist="$TEST_DIR/oversized-$1"
    make_archive "$dist" "$1" 2000
    make_archive "$dist" "$2" 20000
    if run_verify "$dist" >"$TEST_DIR/out.txt" 2>&1; then
        echo "FAIL: oversized binary passed ($order)"
        cat "$TEST_DIR/out.txt"
        exit 1
    fi
    grep -q "too large" "$TEST_DIR/out.txt" || { echo "FAIL: no 'too large' message ($order)"; exit 1; }
done
echo "PASS: oversized binary is caught regardless of archive order"

# Every archive is verified, not just one.
dist="$TEST_DIR/all"
make_archive "$dist" "r1" 2000
make_archive "$dist" "r2" 2000
make_archive "$dist" "r3" 2000
run_verify "$dist" >"$TEST_DIR/out.txt" 2>&1
[ "$(grep -c 'OK' "$TEST_DIR/out.txt")" -eq 3 ] || { echo "FAIL: expected 3 verified archives"; cat "$TEST_DIR/out.txt"; exit 1; }
echo "PASS: every archive is verified"

# An undersized binary (fallback pricing) fails.
dist="$TEST_DIR/small"
make_archive "$dist" "r1" 100
if run_verify "$dist" >"$TEST_DIR/out.txt" 2>&1; then
    echo "FAIL: undersized binary passed"
    exit 1
fi
echo "PASS: undersized binary is caught"

# A binary between WARN and MAX passes with a warning.
dist="$TEST_DIR/warn"
make_archive "$dist" "r1" 7000
run_verify "$dist" >"$TEST_DIR/out.txt" 2>&1
grep -q "WARN" "$TEST_DIR/out.txt" || { echo "FAIL: expected warning"; exit 1; }
echo "PASS: large binary warns but passes"

echo "All tests passed"
