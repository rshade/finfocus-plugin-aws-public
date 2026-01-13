# Research: Asia Pacific Region Support

**Feature**: 002-ap-region-support
**Date**: 2025-11-18
**Phase**: Phase 0 - Research & Decisions

## Overview

This document consolidates research findings for adding Asia Pacific AWS region support to the finfocus-plugin-aws-public. All decisions are based on existing codebase patterns and AWS region naming conventions.

## Research Questions & Findings

### Q1: What build tag naming convention should we use for AP regions?

**Decision**: Use 4-letter abbreviations following the established pattern

**Rationale**:

- Existing tags: `region_use1` (us-east-1), `region_usw2` (us-west-2), `region_euw1` (eu-west-1)
- Pattern: `region_` + 2-letter area + 1-letter direction + 1-digit number
- AP regions follow same structure:
  - `region_apse1` for ap-southeast-1 (Singapore)
  - `region_apse2` for ap-southeast-2 (Sydney)
  - `region_apne1` for ap-northeast-1 (Tokyo)
  - `region_aps1` for ap-south-1 (Mumbai)

**Alternatives Considered**:

- Full region name in tag (e.g., `region_ap_southeast_1`) - **Rejected**: Inconsistent with existing pattern, more verbose
- Numeric-only suffix (e.g., `region_ap1`) - **Rejected**: Loses geographic clarity

**References**:

- `internal/pricing/embed_use1.go:1` - existing build tag pattern
- `internal/pricing/embed_fallback.go:1` - build constraint negation pattern

### Q2: What binary naming convention should we follow?

**Decision**: `finfocus-plugin-aws-public-<region>` where `<region>` is the AWS region identifier

**Rationale**:

- Existing binaries: `finfocus-plugin-aws-public-us-east-1`, etc.
- Users/operators understand AWS region names (ap-southeast-1) more than abbreviations (apse1)
- GoReleaser `binary:` field in `.goreleaser.yaml` uses this pattern

**Examples**:

- `finfocus-plugin-aws-public-ap-southeast-1` (Singapore)
- `finfocus-plugin-aws-public-ap-southeast-2` (Sydney)
- `finfocus-plugin-aws-public-ap-northeast-1` (Tokyo)
- `finfocus-plugin-aws-public-ap-south-1` (Mumbai)

**Alternatives Considered**:

- Use build tag in binary name (e.g., `finfocus-plugin-aws-public-apse1`) - **Rejected**: Less clear to operators
- City names (e.g., `finfocus-plugin-aws-public-singapore`) - **Rejected**: Inconsistent with existing pattern, AWS uses region IDs

**References**:

- `.goreleaser.yaml:12` - existing binary naming in build configurations

### Q3: How should we structure the embed files for AP regions?

**Decision**: Replicate existing embed file pattern with region-specific build tags

**Structure** (for each AP region):

```go
//go:build region_<tag>

package pricing

import _ "embed"

//go:embed data/aws_pricing_<region>.json
var rawPricingJSON []byte
```

**File Matrix**:

| Region | Build Tag | Embed File | Pricing File |
|--------|-----------|------------|--------------|
| ap-southeast-1 | region_apse1 | embed_apse1.go | aws_pricing_ap-southeast-1.json |
| ap-southeast-2 | region_apse2 | embed_apse2.go | aws_pricing_ap-southeast-2.json |
| ap-northeast-1 | region_apne1 | embed_apne1.go | aws_pricing_ap-northeast-1.json |
| ap-south-1 | region_aps1 | embed_aps1.go | aws_pricing_ap-south-1.json |

**Rationale**:

- Maintains consistency with existing embed files (embed_use1.go, embed_usw2.go, embed_euw1.go)
- Build tags ensure exactly one embed file is selected per binary
- `go:embed` directive uses relative path from package to data directory

**Critical Detail**: Fallback embed file must be updated to exclude new AP regions:

```go
//go:build !region_use1 && !region_usw2 && !region_euw1 && !region_apse1 && !region_apse2 && !region_apne1 && !region_aps1
```

**References**:

- `internal/pricing/embed_use1.go` - template for new embed files
- `internal/pricing/embed_fallback.go` - negation pattern for development fallback

### Q4: How should GoReleaser configuration be extended?

**Decision**: Add 4 parallel build configurations, one per AP region

**Configuration Pattern** (example for ap-southeast-1):

```yaml
- id: ap-southeast-1
  main: ./cmd/finfocus-plugin-aws-public
  binary: finfocus-plugin-aws-public-ap-southeast-1
  env:
    - CGO_ENABLED=0
  goos:
    - linux
    - darwin
    - windows
  goarch:
    - amd64
    - arm64
  tags:
    - region_apse1
  ldflags:
    - -s -w -X main.version={{.Version}}
```

**Build Matrix Expansion**:

- **Before**: 3 regions × 3 OS × 2 arch = 18 artifacts
- **After**: 7 regions × 3 OS × 2 arch = 42 artifacts

**Rationale**:

- Each build configuration is independent and can be parallelized by GoReleaser
- Existing archive, checksum, and changelog configurations work for all builds
- No changes needed to archive format or release process

**Performance Considerations**:

- GoReleaser builds in parallel by default
- CI/CD may need increased timeout for 42 artifacts (estimate: +5-10 minutes build time)
- Consider using build matrix caching in GitHub Actions

**References**:

- `.goreleaser.yaml:9-25` - existing us-east-1 build configuration template
- `.goreleaser.yaml:3-6` - before hooks that generate pricing data

### Q5: How should generate-pricing tool be extended?

**Decision**: Add AP regions to supported region list with existing `--regions` flag

**Required Changes**:

1. Update supported regions list to include: `ap-southeast-1`, `ap-southeast-2`, `ap-northeast-1`, `ap-south-1`
2. Ensure `--dummy` flag generates valid pricing data for AP regions
3. Verify output file naming matches expected pattern: `aws_pricing_<region>.json`

**Example Command**:

```bash
go run ./tools/generate-pricing \
  --regions ap-southeast-1,ap-southeast-2,ap-northeast-1,ap-south-1 \
  --out-dir ./internal/pricing/data \
  --dummy
```

**Dummy Pricing Data Structure** (consistent with existing regions):

```json
{
  "region": "ap-southeast-1",
  "currency": "USD",
  "ec2": {
    "t3.micro": {
      "instance_type": "t3.micro",
      "operating_system": "Linux",
      "tenancy": "Shared",
      "hourly_rate": 0.0116
    }
  },
  "ebs": {
    "gp3": {
      "volume_type": "gp3",
      "rate_per_gb_month": 0.088
    }
  }
}
```

**Pricing Variations** (for realistic dummy data):

- Singapore (ap-southeast-1): ~12% higher than US pricing
- Sydney (ap-southeast-2): ~15% higher than US pricing
- Tokyo (ap-northeast-1): ~18% higher than US pricing
- Mumbai (ap-south-1): ~8% higher than US pricing

**Future Work** (out of scope for this feature):

- Implement real AWS Pricing API fetch for AP regions
- Add pricing data validation against AWS public pricing pages
- Implement pricing data refresh mechanism

**References**:

- `tools/generate-pricing/main.go` - existing pricing generation tool
- `.goreleaser.yaml:5` - before hook that calls generate-pricing

### Q6: Do we need changes to plugin logic for AP regions?

**Decision**: No changes needed to core plugin logic (plugin.go, projected.go, client.go)

**Rationale**:

- Region validation in `Supports()` is data-driven (checks ResourceDescriptor.region against pricing client's embedded region)
- Pricing calculations in `GetProjectedCost()` are region-agnostic (lookup by instance type/volume type)
- Client initialization with `sync.Once` works for any region's pricing data

**Test Coverage Updates**:

- **supports_test.go**: Add test cases for AP region identifiers
- **projected_test.go**: Add test cases with AP region ResourceDescriptors
- **client_test.go**: Verify pricing lookups work with AP region pricing data

**Example Test Addition** (supports_test.go):

```go
{
    name: "supports ap-southeast-1",
    region: "ap-southeast-1",
    resourceType: "ec2",
    want: true,
},
{
    name: "rejects ap-southeast-2 when us-east-1 binary",
    region: "ap-southeast-2",
    resourceType: "ec2",
    want: false,
    wantReason: "Region not supported by this binary",
},
```

**References**:

- `internal/plugin/supports.go` - region validation logic
- `internal/plugin/projected.go` - pricing calculation logic
- `internal/pricing/client.go` - pricing data lookup

## Technology Stack

### Build System

- **GoReleaser 1.x**: Multi-platform binary builds with build tags
- **Go 1.25+**: Build tags (`//go:build`), embed directive (`//go:embed`)
- **Make**: Task automation for lint, test, build

### Data Format

- **JSON**: Pricing data interchange format (embedded at build time)
- **go:embed**: Compile-time embedding of pricing JSON files
- **sync.Once**: Thread-safe initialization of pricing data

### Testing

- **Go testing package**: Unit and integration tests
- **Table-driven tests**: Extend existing test tables with AP region cases
- **grpcurl** (manual): End-to-end gRPC testing for validation

## Best Practices Applied

### 1. Build Tag Consistency

- Follow existing 4-letter abbreviation pattern
- Use positive tags in embed files (`//go:build region_apse1`)
- Use negative tags in fallback (`//go:build !region_use1 && !region_apse1 && ...`)

### 2. File Organization

- One embed file per region in `internal/pricing/`
- Pricing data files in `internal/pricing/data/`
- Naming convention: `embed_<tag>.go` and `aws_pricing_<region>.json`

### 3. Testing Strategy

- Extend existing table-driven tests (don't create new test infrastructure)
- Test region validation for all AP regions
- Test pricing lookups with AP region data
- Manual gRPC testing with grpcurl before release

### 4. Documentation

- Update README.md with AP region support section
- Document build tag → region → binary name mapping
- Include build examples for each AP region

## Integration Patterns

### Build-Time Data Embedding

```
generate-pricing (--dummy)
  → creates aws_pricing_ap-southeast-1.json
  → go:embed in embed_apse1.go
  → compiled into binary
  → parsed once with sync.Once
  → indexed for fast lookup
```

### Region Selection Flow

```
User runs: finfocus-plugin-aws-public-ap-southeast-1
  → Binary has region_apse1 build tag
  → embed_apse1.go selected at compile time
  → rawPricingJSON contains Singapore pricing
  → plugin serves gRPC on PORT
  → Supports() validates region matches "ap-southeast-1"
  → GetProjectedCost() uses Singapore pricing
```

### Build Tag Exclusion

```
Build with -tags region_apse1:
  ✅ embed_apse1.go (has region_apse1)
  ❌ embed_use1.go (has region_use1)
  ❌ embed_apse2.go (has region_apse2)
  ❌ embed_fallback.go (!region_use1 && !region_apse1 && ... fails)

Result: Exactly one pricing data source
```

## Potential Issues & Mitigations

### Issue 1: Build Time Increase

**Problem**: 42 artifacts vs 18 may increase CI/CD time significantly

**Mitigation**:

- GoReleaser builds in parallel (utilize all CPU cores)
- Consider GitHub Actions build matrix parallelization
- Monitor build times and adjust timeouts if needed
- Cache Go modules and build artifacts

### Issue 2: Pricing Data Accuracy

**Problem**: Dummy pricing data is not accurate for real cost estimation

**Mitigation**:

- Document that current implementation uses dummy data
- Mark as "estimated" in billing_detail field
- Implement real AWS Pricing API fetch in future work
- Validate dummy data is within realistic ranges

### Issue 3: Binary Size Growth

**Problem**: 7 regions × 3-5MB pricing data = 21-35MB total artifacts

**Mitigation**:

- Each binary only embeds its region (still <20MB per binary)
- GoReleaser archives compress binaries (typically 30-50% reduction)
- Binary size <20MB is acceptable per constitution complexity tracking
- Consider pruning unnecessary instance types in future

## Summary of Decisions

| Decision Point | Choice | Rationale |
|----------------|--------|-----------|
| Build Tags | region_apse1, region_apse2, region_apne1, region_aps1 | Consistent with existing 4-letter pattern |
| Binary Names | finfocus-plugin-aws-public-<region> | User-friendly AWS region identifiers |
| Embed Files | One per region (embed_apse1.go, etc.) | Matches existing pattern |
| GoReleaser | Add 4 parallel build configs | No changes to archive/release process |
| Pricing Tool | Extend --regions flag, keep --dummy | Minimal changes to existing tool |
| Plugin Logic | No changes | Region-agnostic design already supports AP regions |
| Testing | Extend table-driven tests | Reuse existing test infrastructure |

## Next Phase

Proceed to **Phase 1: Design & Contracts** to document:

- Data model (region mappings)
- API contracts (no changes, document consistency)
- Quickstart guide (build commands for AP regions)
