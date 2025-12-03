#!/bin/bash
# Verify region configuration and generated files

set -e

# Default settings
QUIET=false
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -q|--quiet)
            QUIET=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            cat << EOF
Usage: $0 [OPTIONS]

Verify region configuration and generated files for the automated build matrix.

OPTIONS:
    -q, --quiet     Suppress success messages, only show errors
    -v, --verbose   Show detailed output
    -h, --help      Show this help message

EXAMPLES:
    $0                    # Normal verification with all output
    $0 --quiet           # Only show errors
    $0 --verbose         # Show detailed information

EXIT CODES:
    0  Success - all checks passed
    1  Error - one or more checks failed
EOF
            exit 0
            ;;
        *)
            echo "ERROR: Unknown option '$1'" >&2
            echo "Use --help for usage information" >&2
            exit 1
            ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REGIONS_CONFIG="$REPO_ROOT/internal/pricing/regions.yaml"
PRICING_DIR="$REPO_ROOT/internal/pricing"
EMBED_DIR="$PRICING_DIR"
GORELEASER_CONFIG="$REPO_ROOT/.goreleaser.yaml"
PARSE_REGIONS_TOOL="$REPO_ROOT/tools/parse-regions"

if ! $QUIET; then
    echo "Verifying region configuration..."
fi

# Check if regions.yaml exists
if [[ ! -f "$REGIONS_CONFIG" ]]; then
    echo "ERROR: regions.yaml not found at $REGIONS_CONFIG" >&2
    exit 1
fi

if ! $QUIET; then
    echo "✓ regions.yaml exists"
fi

# Parse regions from YAML using proper YAML parser (Go tool)
# This replaces fragile sed-based parsing with robust YAML handling
if ! command -v go &> /dev/null; then
    echo "ERROR: go is required but not found in PATH" >&2
    exit 1
fi

# Parse regions using Go tool - outputs id,name,tag per line as CSV
# Tool has its own go.mod so must be run from its directory
if ! region_data=$(cd "$PARSE_REGIONS_TOOL" && go run . -config "$REGIONS_CONFIG" -format csv 2>&1); then
    echo "ERROR: Failed to parse regions.yaml: $region_data" >&2
    exit 1
fi

# Build arrays from CSV output
id_array=()
region_array=()
tag_array=()
while IFS=',' read -r id name tag; do
    id_array+=("$id")
    region_array+=("$name")
    tag_array+=("$tag")
done <<< "$region_data"

if [[ ${#region_array[@]} -ne ${#id_array[@]} ]] || [[ ${#region_array[@]} -ne ${#tag_array[@]} ]]; then
    echo "ERROR: regions.yaml is malformed (mismatched counts for name/id/tag)" >&2
    exit 1
fi

if $VERBOSE; then
    echo "Found regions: ${region_array[*]@Q}"
    echo "Found region IDs: ${id_array[*]@Q}"
    echo "Found region tags: ${tag_array[*]@Q}"
fi

# Check embed files exist and have correct build tags
for i in "${!region_array[@]}"; do
    region="${region_array[$i]}"
    region_id="${id_array[$i]}"
    expected_tag="${tag_array[$i]}"
    embed_file="$EMBED_DIR/embed_${region_id}.go"

    if [[ ! -f "$embed_file" ]]; then
        echo "ERROR: Embed file missing: $embed_file" >&2
        exit 1
    fi
    if ! $QUIET; then
        echo "✓ Embed file exists: $embed_file"
    fi

    # Check build tag
    if ! grep -q "//go:build $expected_tag" "$embed_file"; then
        echo "ERROR: Build tag mismatch in $embed_file: expected '$expected_tag'" >&2
        exit 1
    fi
    if ! $QUIET; then
        echo "✓ Build tag correct: $expected_tag"
    fi
done

# Check pricing data files exist
for region in "${region_array[@]}"; do
    pricing_file="$PRICING_DIR/data/aws_pricing_$region.json"
    if [[ ! -f "$pricing_file" ]]; then
        echo "ERROR: Pricing data missing: $pricing_file" >&2
        exit 1
    fi
    if ! $QUIET; then
        echo "✓ Pricing data exists: $pricing_file"
    fi
done

# Check GoReleaser config exists and has correct build blocks
if [[ ! -f "$GORELEASER_CONFIG" ]]; then
    echo "ERROR: GoReleaser config missing: $GORELEASER_CONFIG" >&2
    exit 1
fi
if ! $QUIET; then
    echo "✓ GoReleaser config exists: $GORELEASER_CONFIG"
fi

# Check each region has a build block in GoReleaser config
for region in "${region_array[@]}"; do
    if ! grep -q "id: $region" "$GORELEASER_CONFIG"; then
        echo "ERROR: Build block missing for region $region in GoReleaser config" >&2
        exit 1
    fi
    if ! $QUIET; then
        echo "✓ Build block exists for region: $region"
    fi
done

if ! $QUIET; then
    echo "All region configurations verified successfully!"
fi
