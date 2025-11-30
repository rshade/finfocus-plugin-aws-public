# Research: Canada and South America Region Support

**Feature**: `003-ca-sa-region-support`
**Date**: 2025-11-29

## Key Decisions

### 1. Implementation Pattern
**Decision**: Replicate existing embed pattern using build tags.
**Rationale**: The project successfully uses `//go:build region_xyz` tags to selectively embed distinct `data/aws_pricing_*.json` files into region-specific binaries. This keeps binary size low (users only download what they need) and code simple (compile-time selection).
**Alternatives Considered**:
- *Runtime loading*: Rejected because it requires shipping all data (bloat) or fetching at runtime (network dependency/latency).
- *Single binary*: Rejected because the total pricing data for all regions would exceed the 50MB limit per binary.

### 2. Pricing Data Source
**Decision**: Update `tools/generate-pricing` to generate dummy data for `ca-central-1` and `sa-east-1`.
**Rationale**: The project currently relies on the `generate-pricing` tool. Extending it ensures consistent data formats across all regions.
**Verification**: Checked `tools/generate-pricing/main.go` and confirmed it accepts a comma-separated list of regions and has a loop structure that can easily accommodate new regions.

### 3. Naming Conventions
**Decision**:
- `ca-central-1` -> `region_cac1` (tag), `embed_cac1.go` (file)
- `sa-east-1` -> `region_sae1` (tag), `embed_sae1.go` (file)
**Rationale**: Follows the established 3-4 letter suffix convention (`use1`, `usw2`, `euw1`) for consistency.

## Open Questions Resolved
- *Q: Do we need new protobuf definitions?*
  - A: No, the `CostSourceService` is generic and handles any resource descriptor. The region is just a data field.
- *Q: Are there unique currency requirements?*
  - A: AWS mostly bills in USD globally for these standard services. The plugin returns costs in USD by default.

## References
- `internal/pricing/embed_use1.go` (Template for new files)
- `.goreleaser.yaml` (Template for build config)