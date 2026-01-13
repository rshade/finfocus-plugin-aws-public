# Implementation Plan: Asia Pacific Region Support

**Branch**: `002-ap-region-support` | **Date**: 2025-11-18 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-ap-region-support/spec.md`

## Summary

Add support for four Asia Pacific AWS regions (ap-southeast-1, ap-southeast-2, ap-northeast-1, ap-south-1) by extending the existing region-specific build infrastructure. Each AP region will have its own binary with embedded pricing data, following the established pattern used for US and EU regions (us-east-1, us-west-2, eu-west-1).

**Technical Approach**: Replicate the existing build tag + embed pattern for four new AP regions. Add region build tags (region_apse1, region_apse2, region_apne1, region_aps1), create corresponding embed files, update GoReleaser configuration with four new build targets, extend generate-pricing tool to fetch AP region data, and update the fallback embed file to exclude new regions.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**:

- github.com/rshade/finfocus-core/pkg/pluginsdk
- github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1
- google.golang.org/grpc

**Storage**: Embedded JSON pricing files (go:embed) - no external storage
**Testing**: Go standard library testing + table-driven tests
**Target Platform**: Linux, macOS (darwin), Windows - cross-compiled for amd64 and arm64
**Project Type**: Single binary Go application (gRPC service)
**Performance Goals**:

- Plugin startup time: < 500ms (includes pricing data parse)
- GetProjectedCost() RPC: < 100ms per call
- Supports() RPC: < 10ms per call

**Constraints**:

- Binary size: < 20MB per region binary (relaxed from constitution's 10MB due to pricing data size)
- Memory footprint: < 50MB per region binary
- Concurrent RPC calls: Support at least 100 concurrent requests

**Scale/Scope**:

- 4 new region binaries (total 7 regions: 3 existing + 4 new)
- Each binary embeds ~2-5MB of pricing data
- Support for EC2 and EBS services only (S3, Lambda, RDS, DynamoDB remain stubbed)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Code Quality & Simplicity

✅ **PASS** - Replicating existing pattern without introducing new abstractions. Each AP region gets identical structure to existing US/EU regions.

### Testing Discipline

✅ **PASS** - Will extend existing test suites to cover AP regions using table-driven tests. No new testing infrastructure needed.

### Protocol & Interface Consistency

✅ **PASS** - No changes to gRPC protocol. AP region binaries implement identical CostSourceService interface as existing regions.

### Performance & Reliability

✅ **PASS** - AP region binaries use same embedded pricing pattern (sync.Once initialization, indexed lookups). Thread safety maintained.

### Build & Release Quality

⚠️ **ATTENTION REQUIRED** - GoReleaser configuration will grow from 3 to 7 region builds. Build matrix increases from 3×3×2 = 18 artifacts to 7×3×2 = 42 artifacts.

**Justification**: This is expected growth. GoReleaser handles multiple builds efficiently, and CI/CD will validate all builds before release.

### Security Requirements

✅ **PASS** - No security changes. AP region binaries use same loopback-only gRPC serving, embedded pricing data pattern.

### Development Workflow

✅ **PASS** - Feature branch `002-ap-region-support` follows naming convention. Will update CLAUDE.md if new patterns emerge.

## Project Structure

### Documentation (this feature)

```text
specs/002-ap-region-support/
├── plan.md              # This file (/speckit.plan command output)
├── spec.md              # Feature specification (already created)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── region-mappings.md  # Document AP region → build tag → binary name mapping
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/
└── finfocus-plugin-aws-public/
    └── main.go                    # [NO CHANGES - uses pluginsdk.Serve()]

internal/
├── plugin/
│   ├── plugin.go                  # [NO CHANGES - gRPC service implementation]
│   ├── supports.go                # [MINOR UPDATE - validate AP region identifiers]
│   ├── supports_test.go           # [UPDATE - add AP region test cases]
│   ├── projected.go               # [NO CHANGES - pricing calculation logic]
│   └── projected_test.go          # [UPDATE - add AP region test cases]
│
└── pricing/
    ├── client.go                  # [NO CHANGES - pricing lookup logic]
    ├── client_test.go             # [UPDATE - add AP region test cases]
    ├── types.go                   # [NO CHANGES - pricing data structures]
    ├── embed_use1.go              # [NO CHANGES - existing US East]
    ├── embed_usw2.go              # [NO CHANGES - existing US West]
    ├── embed_euw1.go              # [NO CHANGES - existing EU West]
    ├── embed_apse1.go             # [NEW - AP Southeast 1 Singapore]
    ├── embed_apse2.go             # [NEW - AP Southeast 2 Sydney]
    ├── embed_apne1.go             # [NEW - AP Northeast 1 Tokyo]
    ├── embed_aps1.go              # [NEW - AP South 1 Mumbai]
    ├── embed_fallback.go          # [UPDATE - exclude AP regions from build constraint]
    └── data/
        ├── aws_pricing_us-east-1.json      # [NO CHANGES]
        ├── aws_pricing_us-west-2.json      # [NO CHANGES]
        ├── aws_pricing_eu-west-1.json      # [NO CHANGES]
        ├── aws_pricing_ap-southeast-1.json # [NEW - generated by tools]
        ├── aws_pricing_ap-southeast-2.json # [NEW - generated by tools]
        ├── aws_pricing_ap-northeast-1.json # [NEW - generated by tools]
        └── aws_pricing_ap-south-1.json     # [NEW - generated by tools]

tools/
└── generate-pricing/
    └── main.go                    # [UPDATE - add AP regions to supported list]

tests/
├── integration/                   # [NO CHANGES - existing gRPC service tests]
└── unit/                          # [UPDATES - extend test coverage to AP regions]

.goreleaser.yaml                   # [UPDATE - add 4 new build configurations]
README.md                          # [UPDATE - document AP region support]
```

**Structure Decision**: Using the existing single-project structure (Option 1). The plugin is a single Go binary gRPC service with region-specific variants. No need for frontend/backend or mobile/API separation. The build tag pattern keeps the code simple while supporting multiple region binaries.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Binary size 20MB vs constitution's 10MB | AWS pricing data for AP regions expected to be 2-5MB per region | Cannot reduce pricing data completeness without sacrificing accuracy. Compressed archives will be smaller. |
| 42 build artifacts (7 regions × 3 OS × 2 arch) | Need native binaries for all supported platforms and regions | Cross-region binaries would require runtime region selection, increasing complexity and breaking constitution's "one binary per region" principle |

## Phase 0: Research & Decisions

See [research.md](./research.md) for detailed findings.

### Key Decisions

1. **Region Identifier Mapping**:
   - ap-southeast-1 (Singapore) → `region_apse1` → `finfocus-plugin-aws-public-ap-southeast-1`
   - ap-southeast-2 (Sydney) → `region_apse2` → `finfocus-plugin-aws-public-ap-southeast-2`
   - ap-northeast-1 (Tokyo) → `region_apne1` → `finfocus-plugin-aws-public-ap-northeast-1`
   - ap-south-1 (Mumbai) → `region_aps1` → `finfocus-plugin-aws-public-ap-south-1`

2. **Build Tag Pattern**: Follow existing 4-letter abbreviation pattern for consistency (use1, usw2, euw1 → apse1, apse2, apne1, aps1)

3. **GoReleaser Strategy**: Add 4 new build configurations parallel to existing 3 regions. No changes to archive format or release process.

4. **Pricing Data Source**: Extend generate-pricing tool to support AP regions with `--dummy` flag for development. Real AWS pricing fetch implementation deferred to future work.

5. **Fallback Build Constraint**: Update to `!region_use1 && !region_usw2 && !region_euw1 && !region_apse1 && !region_apse2 && !region_apne1 && !region_aps1`

## Phase 1: Design & Contracts

### Data Model

See [data-model.md](./data-model.md) for entity relationships.

**Key Entities**:

- **Region Mapping**: Maps AWS region identifier → build tag → binary name → city name
- **Pricing Data File**: JSON structure identical to existing regions (ec2 + ebs sections)
- **Build Configuration**: GoReleaser build entry with region-specific tags

No new data structures required in Go code. Existing `PricingData`, `EC2Pricing`, `EBSPricing` types handle AP regions.

### API Contracts

See [contracts/](./contracts/) for detailed contracts.

**gRPC Interface**: No changes. AP region binaries implement identical `CostSourceService` interface:

- `Name() → NameResponse{name: "aws-public"}`
- `Supports(ResourceDescriptor) → SupportsResponse{supported, reason}`
- `GetProjectedCost(ResourceDescriptor) → GetProjectedCostResponse{unit_price, currency, cost_per_month, billing_detail}`

**Region Validation Contract**:

- `Supports()` MUST return `supported=false` with reason="Region not supported by this binary" for non-matching regions
- Error details MUST include `pluginRegion` and `requiredRegion` in ErrorDetail.details map

### Quickstart Guide

See [quickstart.md](./quickstart.md) for developer onboarding.

**Quick Build Commands**:

```bash
# Build Singapore binary
go build -tags region_apse1 -o finfocus-plugin-aws-public-ap-southeast-1 ./cmd/finfocus-plugin-aws-public

# Build all AP region binaries with GoReleaser
goreleaser build --snapshot --clean --id ap-southeast-1,ap-southeast-2,ap-northeast-1,ap-south-1

# Generate AP region pricing data (dummy)
go run ./tools/generate-pricing --regions ap-southeast-1,ap-southeast-2,ap-northeast-1,ap-south-1 --out-dir ./internal/pricing/data --dummy
```

## Phase 2: Implementation Tasks

**Generated by `/speckit.tasks` command - not included in this plan.**

## Dependencies & Integration Points

### External Dependencies

- **finfocus-spec**: Proto definitions for CostSourceService (no changes needed)
- **pluginsdk**: Lifecycle management (no changes needed)
- **GoReleaser**: Build orchestration (configuration update only)

### Internal Dependencies

- **embed files** depend on pricing data JSON files existing in `internal/pricing/data/`
- **GoReleaser before hook** must generate pricing data before builds
- **fallback embed** must exclude all AP regions in build constraint

### Integration Checklist

- [ ] Generate pricing data files for all 4 AP regions
- [ ] Create 4 embed_*.go files with correct build tags
- [ ] Update fallback embed build constraint
- [ ] Update GoReleaser with 4 new build configurations
- [ ] Extend test suites to cover AP regions
- [ ] Update README.md with AP region support
- [ ] Manual testing with grpcurl for each AP region binary

## Risk Assessment

### Low Risk

- **Code changes minimal**: Only adding files, not modifying core logic
- **Pattern proven**: Replicating existing US/EU region structure
- **Test coverage**: Existing tests validate the pattern; extending to AP regions is straightforward

### Medium Risk

- **Build time increase**: 42 artifacts vs 18 may increase CI/CD time (mitigated by GoReleaser parallelization)
- **Pricing data accuracy**: Dummy data acceptable for development; real AWS data fetch not yet implemented

### High Risk

None identified.

## Success Metrics

From spec.md Success Criteria:

- **SC-001**: All 4 AP region binaries build without errors → Verify with `goreleaser build --snapshot`
- **SC-002**: Each binary < 20MB → Check with `ls -lh dist/` after build
- **SC-003**: Cost estimates differ across regions → Manual grpcurl test with t3.micro in each region
- **SC-004**: All tests pass → Verify with `make test`
- **SC-005**: Region mismatch errors < 100ms → Benchmark test in supports_test.go
- **SC-006**: 100+ concurrent RPC calls succeed → Load test with grpcurl + xargs parallel calls
- **SC-007**: Build time < 2 minutes per region → CI/CD timing validation
- **SC-008**: Region rejection 100% accurate → Test wrong region in each binary

## Next Steps

1. Run `/speckit.tasks` to generate implementation task breakdown
2. Execute tasks in dependency order (pricing data → embed files → goreleaser → tests)
3. Validate all success criteria before creating PR
4. Update CLAUDE.md if new patterns discovered during implementation
