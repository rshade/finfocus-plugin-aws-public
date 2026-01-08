# Implementation Plan: Core Protocol Intelligence

**Branch**: `030-core-protocol-intelligence` | **Date**: 2026-01-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/030-core-protocol-intelligence/spec.md`

## Summary

Protocol enhancement to enrich cost estimation responses with metadata enabling PulumiCost Core's advanced features: Cost Time Machine forecasting (GrowthType hints), Blast Radius topology visualization (parent_resource_id lineage), and Dev Mode realistic estimates (UsageProfile-based hour adjustments). This is a metadata-only enhancement that does not change pricing calculations, adding three protocol extensions: UsageProfile enum with DEVELOPMENT profile (160 hrs/month), GrowthType classification (STATIC/LINEAR), and CostAllocationLineage message with parent-child relationships extracted from tags.

## Technical Context

**Language/Version**: Go 1.25.5+
**Primary Dependencies**: gRPC (pluginsdk), pulumicost-spec (version TBD with protocol extensions), zerolog
**Storage**: Embedded JSON pricing data (memory-mapped, no runtime storage)
**Testing**: Go testing (`make test`, `go test -v ./...`)
**Target Platform**: Linux server (region-specific binaries with build tags: region_use1, region_usw2, region_euw1)
**Project Type**: single (gRPC plugin with embedded pricing data)
**Performance Goals**: < 100ms per GetProjectedCost() RPC, < 500ms plugin startup, < 10ms Supports() RPC
**Constraints**: < 400MB memory footprint, < 250MB binary size per region, thread-safe for concurrent gRPC calls (100+ concurrent), feature detection for protocol field availability (no version checks)
**Scale/Scope**: 10 AWS services (EC2, EBS, EKS, S3, Lambda, DynamoDB, ELB, NAT Gateway, CloudWatch, ElastiCache, RDS), 3 protocol extensions

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Initial Gate (Pre-Design)

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Code Quality & Simplicity | ✅ PASS | Pure transformation functions for metadata extraction, single responsibility (dev mode, growth hints, lineage), no premature abstraction |
| II. Testing Discipline | ✅ PASS | Unit tests for metadata transformations, integration tests for gRPC methods, table-driven tests for service classifications |
| III. Protocol & Interface Consistency | ✅ PASS | Uses proto-defined types (ResourceDescriptor, GetProjectedCostResponse), zerolog structured logging, thread-safe gRPC handlers, feature detection (no version checks) |
| IV. Performance & Reliability | ✅ PASS | Embedded pricing data (cached with sync.Once), indexed maps for lookups, < 100ms GetProjectedCost() target, < 400MB memory, < 250MB binary, 100+ concurrent RPC support |
| V. Build & Release Quality | ✅ PASS | make lint, make test, GoReleaser for region builds, build tags for region selection |

**Overall Gate Status (Pre-Design)**: ✅ PASS - Proceeded to Phase 0 research

---

### Post-Design Re-Evaluation (After Phase 1)

| Principle | Status | Notes | Design Validation |
|-----------|--------|-------|------------------|
| I. Code Quality & Simplicity | ✅ PASS | Pure transformation functions confirmed (data-model.md), single responsibility per enrichment type, no premature abstraction (research.md RQ-7) |
| II. Testing Discipline | ✅ PASS | Unit test strategy defined (table-driven, service classification tests, feature detection tests), integration tests for gRPC, no external dependency mocking |
| III. Protocol & Interface Consistency | ✅ PASS | Feature detection via type assertion (research.md RQ-6), proto-defined enums (UsageProfile, GrowthType), CostAllocationLineage message, optional fields for backward compatibility |
| IV. Performance & Reliability | ✅ PASS | Read-only service classification map (thread-safe, no locks), < 10ms enrichment overhead (research.md RQ-10), < 100ms total GetProjectedCost() target, indexed map O(1) lookups |
| V. Build & Release Quality | ✅ PASS | No breaking changes to existing cost calculations (SC-004), backward compatible via optional proto fields, make lint/test commands documented |

**Overall Gate Status (Post-Design)**: ✅ PASS - Proceed to Phase 2 task breakdown

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
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
│   ├── constants.go        # Add hoursPerMonthDev = 160
│   ├── projected.go        # Add UsageProfile handling, GrowthType, lineage extraction
│   └── projected_test.go   # Tests for all three features
├── pricing/
│   └── [existing pricing lookup code - no changes]
└── [existing internal packages - no changes]

test/
├── integration/
│   └── [existing integration tests]
└── fixtures/
    └── [existing test fixtures]

cmd/
└── pulumicost-plugin-aws-public/
    └── main.go            # No changes (gRPC lifecycle unchanged)

tools/
└── [existing generation tools - no changes]

CLAUDE.md                  # Document growth classification, dev mode, topology relationships
go.mod                     # Update pulumicost-spec dependency
```

**Structure Decision**: Single project (existing gRPC plugin structure) - modifying `internal/plugin/projected.go` to add metadata enrichment without changing pricing lookup logic, maintaining clean separation of concerns.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
