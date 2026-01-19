#!/bin/bash
# Test suite for verify-regions.sh

set -e

# Setup
TEST_DIR=$(mktemp -d)
SCRIPT_TO_TEST="$(pwd)/scripts/verify-regions.sh"
trap 'rm -rf "$TEST_DIR"' EXIT

echo "Running tests in $TEST_DIR"

# Helper to create mock repo structure
setup_repo() {
    local root="$1"
    mkdir -p "$root/internal/pricing/data"
    mkdir -p "$root/tools/parse-regions"
    mkdir -p "$root/internal/carbon/data"
    
    # Create regions.yaml
    cat > "$root/internal/pricing/regions.yaml" <<EOF
regions:
  - id: use1
    name: us-east-1
    tag: region_use1
EOF

    # Create mock parse-regions tool
    cat > "$root/tools/parse-regions/main.go" <<EOF
package main
import (
    "flag"
    "fmt"
)
func main() {
    // Mimic the behavior: parse flags and output CSV
    flag.String("config", "", "")
    flag.String("format", "", "")
    flag.Parse()
    fmt.Println("use1,us-east-1,region_use1")
}
EOF
    cat > "$root/tools/parse-regions/go.mod" <<EOF
module tools/parse-regions
go 1.22
EOF

    # Create embed file
    cat > "$root/internal/pricing/embed_use1.go" <<EOF
//go:build region_use1
package pricing
EOF

    # Create pricing data
    for svc in ec2 s3 rds eks lambda dynamodb elb; do
        touch "$root/internal/pricing/data/${svc}_us-east-1.json"
    done

    # Create carbon data
    touch "$root/internal/carbon/data/ccf_instance_specs.csv"

    # Create goreleaser config
    cat > "$root/.goreleaser.yaml" <<EOF
builds:
  - id: us-east-1
EOF
}

# Copy the script to test dir so it resolves REPO_ROOT correctly relative to itself
mkdir -p "$TEST_DIR/scripts"
cp "$SCRIPT_TO_TEST" "$TEST_DIR/scripts/verify-regions.sh"
chmod +x "$TEST_DIR/scripts/verify-regions.sh"

# Run Test 1: Success case (line $LINENO)
echo "Test 1 (line $LINENO): Success case"
setup_repo "$TEST_DIR"
if "$TEST_DIR/scripts/verify-regions.sh" --quiet; then
    echo "PASS: verify-regions.sh succeeded as expected"
else
    echo "FAIL: verify-regions.sh failed unexpectedly"
    exit 1
fi

# Run Test 2: Missing regions.yaml (line $LINENO)
echo "Test 2 (line $LINENO): Missing regions.yaml"
rm "$TEST_DIR/internal/pricing/regions.yaml"
# Suppress expected error output for negative test cases
if "$TEST_DIR/scripts/verify-regions.sh" --quiet 2>/dev/null; then
    echo "FAIL: verify-regions.sh should have failed"
    exit 1
else
    echo "PASS: verify-regions.sh failed as expected"
fi

# Reset
setup_repo "$TEST_DIR"

# Run Test 3: Missing embed file (line $LINENO)
echo "Test 3 (line $LINENO): Missing embed file"
rm "$TEST_DIR/internal/pricing/embed_use1.go"
# Suppress expected error output for negative test cases
if "$TEST_DIR/scripts/verify-regions.sh" --quiet 2>/dev/null; then
    echo "FAIL: verify-regions.sh should have failed"
    exit 1
else
    echo "PASS: verify-regions.sh failed as expected"
fi

# Reset
setup_repo "$TEST_DIR"

# Run Test 4: Missing pricing data (line $LINENO)
echo "Test 4 (line $LINENO): Missing pricing data"
rm "$TEST_DIR/internal/pricing/data/ec2_us-east-1.json"
# Suppress expected error output for negative test cases
if "$TEST_DIR/scripts/verify-regions.sh" --quiet 2>/dev/null; then
    echo "FAIL: verify-regions.sh should have failed"
    exit 1
else
    echo "PASS: verify-regions.sh failed as expected"
fi

echo "All tests passed!"
