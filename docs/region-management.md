# AWS Region Management

This document provides information about managing AWS regions in the pulumicost-plugin-aws-public project.

## Overview

The plugin supports multiple AWS regions through a centralized configuration system. Each region has its own binary with embedded pricing data.

## Architecture

### Region Configuration

All region information is centralized in `internal/pricing/regions.yaml`:

```yaml
regions:
  - id: use1
    name: us-east-1
    tag: region_use1
  - id: usw2
    name: us-west-2
    tag: region_usw2
  # ... more regions
```

### Generated Files

For each region, the following files are automatically generated:

#### Embed Files (`internal/pricing/embed_{id}.go`)

```go
//go:build region_{id}

package pricing

import _ "embed"

//go:embed data/aws_pricing_{name}.json
var rawPricingJSON []byte
```

#### GoReleaser Configuration (`.goreleaser.yaml`)

Contains build blocks for all regions with appropriate build tags and settings.

## Adding New Regions

### Step-by-Step Process

1. **Update regions.yaml**

   ```bash
   # Edit internal/pricing/regions.yaml
   vim internal/pricing/regions.yaml
   ```

   Add the new region:

   ```yaml
   regions:
     - id: euw3      # Short identifier (must be unique)
       name: eu-west-3  # Full AWS region name
       tag: region_euw3 # Build tag (must follow pattern)
   ```

2. **Generate Configuration Files**

   ```bash
   # Generate embed files
   make generate-embeds

   # Generate GoReleaser configuration
   make generate-goreleaser

   # Verify all configurations
   make verify-regions
   ```

3. **Test the New Region**

   ```bash
   # Build the specific region
   make build-region REGION=eu-west-3

   # Test with sample requests
   ./pulumicost-plugin-aws-public-eu-west-3
   ```

### Validation Rules

- **ID**: Must be unique, lowercase alphanumeric
- **Name**: Must be valid AWS region identifier
- **Tag**: Must follow `region_{id}` pattern
- **Pricing Data**: Must exist for the region

## Build System

### Makefile Targets

- `make generate-embeds` - Generate embed files from regions.yaml
- `make generate-goreleaser` - Generate .goreleaser.yaml
- `make verify-regions` - Validate all region configurations
- `make build-region REGION=<name>` - Build specific region binary

### CI/CD Integration

The GitHub Actions workflows automatically run generation scripts:

- **test.yml**: Generates configs before running tests and builds
- **release.yml**: Generates configs before creating releases

### Sequential Building

Due to build image disk constraints, regions are built sequentially:

1. Generate shared configurations
2. Build one region at a time
3. Clean cache between builds
4. Verify each binary immediately

## Verification

### Automated Checks

The `verify-regions` script performs comprehensive validation:

- ✅ regions.yaml exists and is valid
- ✅ All embed files exist with correct build tags
- ✅ All pricing data files exist
- ✅ GoReleaser config includes all regions

### Manual Testing

Test each region binary:

```bash
# Start the plugin
./pulumicost-plugin-aws-public-us-east-1

# Test with gRPC calls (region should match)
# Requests for wrong regions return UNSUPPORTED_REGION errors
```

## Troubleshooting

### Common Issues

#### Missing embed file

```text
ERROR: Embed file missing: internal/pricing/embed_xyz.go
```

→ Run `make generate-embeds`

#### Build tag mismatch

```text
ERROR: Build tag mismatch in embed_xyz.go: expected 'region_xyz'
```

→ Check regions.yaml tag format

#### Pricing data missing

```text
ERROR: Pricing data missing: internal/pricing/data/aws_pricing_xyz.json
```

→ Run pricing data generation for that region

### Debug Commands

```bash
# Check current regions
cat internal/pricing/regions.yaml

# List generated embed files
ls internal/pricing/embed_*.go

# Check GoReleaser config
head -20 .goreleaser.yaml

# Validate configurations
make verify-regions
```

## File Structure

```text
internal/pricing/
├── regions.yaml           # Central region configuration
├── embed_*.go            # Generated embed files (one per region)
├── data/
│   └── aws_pricing_*.json # Pricing data files
└── types.go              # Shared types

tools/
├── generate-embeds/      # Embed file generator
└── generate-goreleaser/  # GoReleaser config generator

scripts/
├── verify-regions.sh     # Configuration validator
├── build-region.sh       # Single region builder
└── release-region.sh     # Single region releaser
```

## Performance Considerations

- **Binary Size**: Each region binary contains only its pricing data
- **Build Time**: Sequential building prevents disk exhaustion
- **Cache Management**: Go build cache cleared between regions
- **CI/CD**: Generation runs once, then regions build in parallel where possible

## Future Enhancements

- Parallel region building (if disk constraints allow)
- Automatic region detection from AWS APIs
- Region-specific feature flags
- Enhanced validation with AWS region metadata
