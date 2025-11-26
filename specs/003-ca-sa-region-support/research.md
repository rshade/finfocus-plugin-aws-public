# Research: Canada and South America Region Support

**Branch**: `003-ca-sa-region-support` | **Date**: 2025-11-20

## Overview

Research findings for adding ca-central-1 and sa-east-1 region support. This feature follows established patterns from 002-ap-region-support.

## Build Tag Naming Convention

**Decision**: Use `region_cac1` for ca-central-1 and `region_sae1` for sa-east-1

**Rationale**: Follows existing convention of abbreviated region codes:
- us-east-1 → use1
- us-west-2 → usw2
- eu-west-1 → euw1
- ap-southeast-1 → apse1
- ap-south-1 → aps1

**Alternatives considered**:
- `region_ca_c1` - Rejected: inconsistent with single underscore pattern
- `region_canada` - Rejected: too verbose, breaks pattern

## Embed File Pattern

**Decision**: Follow exact pattern from existing embed files

**Rationale**: Existing embed files use:
```go
//go:build region_<tag>

package pricing

import _ "embed"

//go:embed data/aws_pricing_<region>.json
var rawPricingJSON []byte
```

This pattern:
- Uses build tags to select exactly one embed file
- Embeds region-specific JSON at compile time
- Exports rawPricingJSON for the pricing client

**Alternatives considered**:
- Dynamic loading from filesystem - Rejected: violates embedded data principle
- Multiple regions per binary - Rejected: violates region-specific binary pattern

## GoReleaser Configuration

**Decision**: Add two new build targets following existing pattern

**Rationale**: Each region requires:
- Unique build id
- Binary name: pulumicost-plugin-aws-public-<region>
- CGO_ENABLED=0 for static linking
- Multi-platform (linux, darwin, windows) + multi-arch (amd64, arm64)
- Region-specific build tag

**Alternatives considered**:
- Single multi-region binary - Rejected: violates architecture
- Separate GoReleaser configs - Rejected: unnecessary complexity

## Pricing Generator Extension

**Decision**: Add ca-central-1 and sa-east-1 to --regions flag default list

**Rationale**: The generate-pricing tool in `tools/generate-pricing/main.go` accepts a --regions flag. Adding these regions enables:
- Dummy data generation for development
- Real pricing fetch when implemented

**Alternatives considered**:
- Separate generator for each region - Rejected: duplication
- Hard-coded region list - Rejected: less flexible

## Fallback Embed Update

**Decision**: Add `!region_cac1 && !region_sae1` to fallback build constraint

**Rationale**: The fallback embed file must exclude all region tags to ensure exactly one embed file is selected. Current constraint:
```go
//go:build !region_use1 && !region_usw2 && !region_euw1 && !region_apse1 && !region_apse2 && !region_apne1 && !region_aps1
```

Must become:
```go
//go:build !region_use1 && !region_usw2 && !region_euw1 && !region_apse1 && !region_apse2 && !region_apne1 && !region_aps1 && !region_cac1 && !region_sae1
```

**Alternatives considered**: None - this is required for correct build tag selection

## Test Coverage

**Decision**: Add region-specific test cases to existing test suites

**Rationale**: Follow patterns from 002-ap-region-support:
- Add ca-central-1 and sa-east-1 test cases to pricing client tests
- Add region mismatch tests
- Add success criteria validation tests (concurrency, latency, cross-region)

**Alternatives considered**:
- Separate test files per region - Rejected: leads to duplication
- Skip testing for new regions - Rejected: violates testing discipline

## Performance Validation

**Decision**: Validate binary size <20MB and region mismatch <100ms

**Rationale**: Based on 002-ap-region-support success criteria:
- All AP region binaries are 16MB (passes <20MB requirement)
- Region mismatch latency: 0.01ms (passes <100ms requirement)

New regions should achieve same metrics.

**Alternatives considered**: None - these are constitution requirements

## Summary of Implementation Tasks

1. Create `embed_cac1.go` and `embed_sae1.go`
2. Update `embed_fallback.go` build constraint
3. Add ca-central-1 and sa-east-1 to GoReleaser builds
4. Update generate-pricing --regions in GoReleaser before hook
5. Add test cases for both regions
6. Validate binary size and latency metrics
