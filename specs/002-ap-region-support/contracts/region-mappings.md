# Contract: Region Mappings

**Feature**: 002-ap-region-support
**Version**: 1.0
**Status**: Proposed

## Overview

This contract defines the authoritative mapping between AWS region identifiers, Go build tags, binary names, and geographic locations for all supported regions in finfocus-plugin-aws-public.

## Complete Region Mapping Table

| AWS Region ID | Build Tag | Binary Name | City | Geographic Area | Status |
|---------------|-----------|-------------|------|-----------------|--------|
| us-east-1 | region_use1 | finfocus-plugin-aws-public-us-east-1 | N. Virginia | North America | Existing |
| us-west-2 | region_usw2 | finfocus-plugin-aws-public-us-west-2 | Oregon | North America | Existing |
| eu-west-1 | region_euw1 | finfocus-plugin-aws-public-eu-west-1 | Ireland | Europe | Existing |
| ap-southeast-1 | region_apse1 | finfocus-plugin-aws-public-ap-southeast-1 | Singapore | Asia Pacific | **New** |
| ap-southeast-2 | region_apse2 | finfocus-plugin-aws-public-ap-southeast-2 | Sydney | Asia Pacific | **New** |
| ap-northeast-1 | region_apne1 | finfocus-plugin-aws-public-ap-northeast-1 | Tokyo | Asia Pacific | **New** |
| ap-south-1 | region_aps1 | finfocus-plugin-aws-public-ap-south-1 | Mumbai | Asia Pacific | **New** |

## Build Tag Naming Convention

### Pattern

```
region_<area><direction><number>
```

Where:
- `<area>` = 2-letter geographic area code (us, eu, ap)
- `<direction>` = 1-letter compass direction or geographic qualifier (e, w, se, ne, s)
- `<number>` = 1-digit region sequence number

### Examples

- `region_use1` = United States East region 1
- `region_apse1` = Asia Pacific Southeast region 1
- `region_apne1` = Asia Pacific Northeast region 1
- `region_aps1` = Asia Pacific South region 1

### Special Cases

When direction is two letters (e.g., "southeast", "northeast"), both letters are included:
- Southeast: `se` (not `s` or `e` alone)
- Northeast: `ne` (not `n` or `e` alone)
- South: `s` (single direction, no east/west component)

## Binary Naming Convention

### Pattern

```
finfocus-plugin-aws-public-<aws-region-id>
```

Where `<aws-region-id>` is the official AWS region identifier with hyphens.

### Rationale

- Users and operators familiar with AWS region IDs
- Consistent with AWS CLI and console naming
- No ambiguity (use official AWS names, not abbreviations)

## File Naming Conventions

### Embed Files

**Pattern**: `embed_<build_tag_suffix>.go`

Where `<build_tag_suffix>` is the 4-letter abbreviation from the build tag (without "region_" prefix).

**Examples**:
- `embed_use1.go` (for region_use1)
- `embed_apse1.go` (for region_apse1)
- `embed_apne1.go` (for region_apne1)

**Location**: `internal/pricing/`

### Pricing Data Files

**Pattern**: `aws_pricing_<aws-region-id>.json`

Where `<aws-region-id>` matches the AWS Region ID column exactly.

**Examples**:
- `aws_pricing_us-east-1.json`
- `aws_pricing_ap-southeast-1.json`
- `aws_pricing_ap-northeast-1.json`

**Location**: `internal/pricing/data/`

## GoReleaser Build IDs

**Pattern**: Use AWS region ID as build ID (for human readability in GoReleaser output)

**Examples**:
```yaml
- id: us-east-1
  binary: finfocus-plugin-aws-public-us-east-1
  tags: [region_use1]

- id: ap-southeast-1
  binary: finfocus-plugin-aws-public-ap-southeast-1
  tags: [region_apse1]
```

**Rationale**: Build ID appears in GoReleaser logs and artifact metadata; AWS region ID is more recognizable than abbreviated tag.

## Region Validation Contract

### Supports() Method Behavior

When a binary is queried with a `ResourceDescriptor` for a different region:

**Request**:
```protobuf
SupportsRequest {
  resource: ResourceDescriptor {
    provider: "aws"
    resource_type: "ec2"
    sku: "t3.micro"
    region: "us-west-2"  // Different from binary's region
  }
}
```

**Response** (from ap-southeast-1 binary):
```protobuf
SupportsResponse {
  supported: false
  reason: "Region not supported by this binary"
}
```

### GetProjectedCost() Error Behavior

When a binary receives a `GetProjectedCostRequest` for a different region:

**Error Details**:
```
gRPC Status: Code = FailedPrecondition (9)
Message: "Region ap-southeast-1 not supported by this binary"
ErrorDetail: {
  error_code: ERROR_CODE_UNSUPPORTED_REGION
  details: {
    "pluginRegion": "ap-southeast-1",
    "requiredRegion": "us-west-2"
  }
}
```

**Client Interpretation**:
- `pluginRegion`: The region this binary supports (embedded in pricing data)
- `requiredRegion`: The region the resource needs (from ResourceDescriptor)

## Build Tag Selection Matrix

### Single Build Tag Selection

When building with `-tags <tag>`, exactly one embed file is selected:

| Build Tags | Selected Embed | Pricing Region |
|------------|----------------|----------------|
| region_use1 | embed_use1.go | us-east-1 |
| region_usw2 | embed_usw2.go | us-west-2 |
| region_euw1 | embed_euw1.go | eu-west-1 |
| region_apse1 | embed_apse1.go | ap-southeast-1 |
| region_apse2 | embed_apse2.go | ap-southeast-2 |
| region_apne1 | embed_apne1.go | ap-northeast-1 |
| region_aps1 | embed_aps1.go | ap-south-1 |
| (none) | embed_fallback.go | unknown (dummy data) |

### Fallback Build Constraint

The fallback embed file MUST exclude ALL region tags:

```go
//go:build !region_use1 && !region_usw2 && !region_euw1 && !region_apse1 && !region_apse2 && !region_apne1 && !region_aps1
```

**Critical**: When adding a new region, the fallback constraint MUST be updated to exclude the new tag.

## Region Addition Checklist

When adding a new region to this contract:

1. [ ] Add row to Complete Region Mapping Table
2. [ ] Verify build tag follows naming convention
3. [ ] Verify binary name follows `finfocus-plugin-aws-public-<region>` pattern
4. [ ] Create embed file with pattern `embed_<suffix>.go`
5. [ ] Create pricing data file with pattern `aws_pricing_<region>.json`
6. [ ] Add build configuration to `.goreleaser.yaml` with matching ID
7. [ ] Update fallback embed build constraint to exclude new tag
8. [ ] Update this contract document

## Validation Rules

### Build Tag Constraints

- MUST start with `region_`
- MUST be 4 letters after `region_` prefix (total 11 characters including `region_`)
- MUST be lowercase
- MUST be unique across all regions
- MUST NOT contain underscores except after `region` prefix

### AWS Region ID Constraints

- MUST match official AWS region naming
- MUST use hyphens (not underscores)
- MUST be lowercase
- MUST be in format: `<area>-<direction>-<number>` (e.g., "ap-southeast-1")

### Binary Name Constraints

- MUST start with `finfocus-plugin-aws-public-`
- MUST end with AWS region ID
- MUST use hyphens throughout (no underscores)
- MUST be lowercase

## Testing Contract

### Region Identifier Tests

For each region in the mapping table, test suite MUST verify:

1. Binary built with correct build tag contains correct region in pricing data
2. `Supports()` returns `true` for matching region, `false` for non-matching
3. `GetProjectedCost()` succeeds for matching region
4. `GetProjectedCost()` returns `ERROR_CODE_UNSUPPORTED_REGION` for non-matching region

### Build Tag Exclusivity Tests

Verify that building with any single region tag results in:
- Exactly one embed file compiled
- Pricing data region matches build tag
- Fallback embed excluded

## Version History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | 2025-11-18 | Initial contract with 7 regions (3 existing + 4 AP) | Spec 002 |

## References

- AWS Regions Documentation: https://docs.aws.amazon.com/general/latest/gr/rande.html
- Go Build Tags: https://pkg.go.dev/cmd/go#hdr-Build_constraints
- GoReleaser Build Configuration: https://goreleaser.com/customization/build/
