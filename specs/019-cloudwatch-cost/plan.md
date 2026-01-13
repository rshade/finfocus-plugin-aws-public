# Implementation Plan: CloudWatch Cost Estimation

**Branch**: `019-cloudwatch-cost` | **Date**: 2025-12-30 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/019-cloudwatch-cost/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement CloudWatch cost estimation for logs (ingestion/storage) and custom metrics using AWS public pricing data. Following the established pattern (EC2, EBS, EKS, ELB, NAT Gateway), pricing data will be fetched via AWS Price List API (service code: `AmazonCloudWatch`) and embedded as JSON at build time. Cost calculation uses tiered pricing for metrics (first 10k @ $0.30, next 240k @ $0.10, next 750k @ $0.05, over 1M @ $0.02) and tiered pricing for log ingestion ($0.50 → $0.25 → $0.10 → $0.05 per GB by volume tier).

**Foundational Work**: This feature includes normalizing the "pricing not found" error messages across ALL services. A codebase audit revealed inconsistent messaging (some services silent, some verbose, some different formats). New constants will standardize the soft failure pattern: return $0.00 with `BillingDetail` using `PricingNotFoundTemplate` or `PricingUnavailableTemplate`.

## Technical Context

**Language/Version**: Go 1.25+ (same as existing codebase)
**Primary Dependencies**: gRPC via finfocus-spec/sdk/go/pluginsdk, zerolog for logging
**Storage**: Embedded JSON via `//go:embed` (no external storage)
**Testing**: Go testing with table-driven tests, integration tests with -tags=integration
**Target Platform**: Linux/Darwin/Windows (cross-compiled region-specific binaries)
**Project Type**: Single Go module with internal packages
**Performance Goals**: <100ms per GetProjectedCost RPC, <10ms for Supports RPC
**Constraints**: Binary size < 250MB per region (constitution), memory < 400MB runtime
**Scale/Scope**: 12 AWS regions (existing), ~98K products per region (CloudWatch adds ~2-5K)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Code Quality & Simplicity ✅

| Principle | Compliance | Notes |
|-----------|------------|-------|
| KISS | ✅ | Follows existing service pattern (NAT Gateway has multi-component pricing, similar to CloudWatch) |
| Single Responsibility | ✅ | `estimateCloudWatch()` does ONE thing - calculates CloudWatch costs |
| Explicit is better than implicit | ✅ | Tags explicitly specify usage: `log_ingestion_gb`, `log_storage_gb`, `custom_metrics` |
| Stateless components | ✅ | Each gRPC call is independent; pricing data is immutable after init |
| File size guidance (<300 lines) | ✅ | CloudWatch estimation logic ~100-150 lines, fits in projected.go |

### II. Testing Discipline ✅

| Requirement | Compliance | Notes |
|-------------|------------|-------|
| Unit tests for transformations | ✅ | tiered pricing calculation, tag extraction testable as pure functions |
| Integration tests for gRPC | ✅ | Supports() and GetProjectedCost() integration tests |
| No mocking of dependencies we don't own | ✅ | Use mock pricing client interface |
| Tests run via `make test` | ✅ | Standard test harness |
| Critical path focus | ✅ | Logs + metrics pricing lookups are critical |

### III. Protocol & Interface Consistency ✅

| Requirement | Compliance | Notes |
|-------------|------------|-------|
| gRPC protocol | ✅ | Uses existing CostSourceService methods |
| No stdout except PORT | ✅ | Follows existing pattern |
| Proto-defined types | ✅ | ResourceDescriptor, GetProjectedCostResponse |
| Error codes from enum | ✅ | ERROR_CODE_INVALID_RESOURCE for bad tags |
| Thread safety | ✅ | Read-only after init, same as other services |
| Region-specific binaries | ✅ | CloudWatch JSON per region |

### IV. Performance & Reliability ✅

| Requirement | Compliance | Notes |
|-------------|------------|-------|
| sync.Once parsing | ✅ | CloudWatch pricing parsed once |
| Indexed lookups | ✅ | Map-based price lookups O(1) |
| GetProjectedCost < 100ms | ✅ | Simple map lookups + arithmetic |
| Binary size < 250MB | ⚠️ Monitor | CloudWatch JSON estimated 2-5MB, well under limit |
| Memory < 400MB | ✅ | No significant memory increase |

### V. Build & Release Quality ✅

| Requirement | Compliance | Notes |
|-------------|------------|-------|
| `make lint` passes | ✅ | Standard Go code |
| `make test` passes | ✅ | Unit + integration tests |
| GoReleaser builds | ✅ | Add cloudwatch_*.json embed |
| Build tags compile | ✅ | Same pattern as other services |

**GATE STATUS: ✅ PASSED** - All constitution principles satisfied.

## Project Structure

### Documentation (this feature)

```text
specs/019-cloudwatch-cost/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/
├── plugin/
│   ├── constants.go          # NEW: PricingNotFoundTemplate, PricingUnavailableTemplate
│   ├── constants_test.go     # NEW: Tests for constant usage
│   ├── projected.go          # MODIFY: Add estimateCloudWatch(), update all services to use constants
│   ├── projected_test.go     # ADD: CloudWatch estimation tests
│   ├── supports.go           # ADD: cloudwatch case in Supports() switch
│   └── supports_test.go      # ADD: CloudWatch support tests
├── pricing/
│   ├── client.go             # ADD: parseCloudWatchPricing(), CloudWatch lookup methods
│   ├── types.go              # ADD: cloudWatchPrice type definition
│   ├── data/                 # ADD: cloudwatch_{region}.json files (generated)
│   ├── embed_use1.go         # ADD: //go:embed data/cloudwatch_us-east-1.json
│   ├── embed_usw2.go         # ADD: //go:embed data/cloudwatch_us-west-2.json
│   ├── embed_euw1.go         # ADD: //go:embed data/cloudwatch_eu-west-1.json
│   └── embed_*.go            # ADD: embed directive for all 12 regions
└── ...

tools/
└── generate-pricing/
    └── main.go               # ADD: "AmazonCloudWatch" to serviceConfig map

cmd/
└── finfocus-plugin-aws-public/
    └── main.go               # NO CHANGES (pluginsdk.Serve handles everything)

CLAUDE.md                     # UPDATE: Document soft failure pattern and constants
```

**Structure Decision**: Single Go module following existing service implementation pattern.
All CloudWatch code integrates into existing files (projected.go, supports.go, client.go, types.go)
with new embed files for CloudWatch pricing data per region.

**Foundational Refactor**: New `constants.go` file centralizes pricing error messages.
All existing services (EC2, EBS, S3, RDS, EKS, Lambda, ELB, NAT Gateway, DynamoDB)
will be updated to use these constants for consistent soft failure behavior.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Tiered pricing logic | FR-007 requires AWS tiered pricing for metrics | First-tier-only would overestimate 10x for high-volume users |
| Multi-component cost | Logs have ingestion + storage; metrics separate | Single rate would not match AWS billing model |
| Foundational refactor | Normalize pricing error messages across 9 services | Per-service inconsistency creates poor UX |

**Note**: The tiered pricing and multi-component costs are **requirements** from the spec.
The foundational refactor (constants.go) is technical debt paydown that improves consistency.
