# Implementation Plan: RDS Instance Cost Estimation

**Branch**: `009-rds-cost-estimation` | **Date**: 2025-12-02 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/009-rds-cost-estimation/spec.md`

## Summary

Implement RDS database instance cost estimation using AWS Public Pricing data. This extends
the existing EC2/EBS pricing pattern to support RDS instance hours (hourly rate × 730) and
storage costs (per GB-month). Supports MySQL, PostgreSQL, MariaDB, Oracle SE2, and SQL
Server Express engines with Single-AZ deployment pricing only.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: gRPC via pulumicost-spec/sdk/go/pluginsdk, zerolog for logging
**Storage**: Embedded JSON pricing data via `//go:embed`
**Testing**: Go testing with table-driven tests, mock pricing client
**Target Platform**: Linux server (9 regional binaries)
**Project Type**: Single project (gRPC plugin)
**Performance Goals**: < 100ms per GetProjectedCost() call, < 10ms for Supports()
**Constraints**: < 50MB memory per binary, thread-safe concurrent access
**Scale/Scope**: 100 concurrent RPC calls without errors

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. Code Quality & Simplicity | PASS | Follows existing EC2/EBS pattern; no new abstractions |
| II. Testing Discipline | PASS | Table-driven unit tests planned; mock pricing client exists |
| III. Protocol & Interface Consistency | PASS | Uses proto-defined types; no stdout except PORT |
| IV. Performance & Reliability | PASS | sync.Once + indexed maps; thread-safe lookups |
| V. Build & Release Quality | PASS | GoReleaser builds for 9 regions; make lint/test |

## Project Structure

### Documentation (this feature)

```text
specs/009-rds-cost-estimation/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A - gRPC proto already defined)
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── plugin/
│   ├── plugin.go           # Plugin struct (no changes needed)
│   ├── projected.go        # Add estimateRDS() function, update router
│   ├── projected_test.go   # Add RDS test cases
│   └── supports.go         # Move RDS from stub to fully-supported
├── pricing/
│   ├── client.go           # Add RDS interface methods, indexes, init logic
│   ├── types.go            # Add rdsInstancePrice, rdsStoragePrice structs
│   ├── embed_*.go          # No changes (RDS data embedded in same files)
│   └── client_test.go      # Add RDS lookup tests
tools/
└── generate-pricing/
    └── main.go             # Add AmazonRDS service code support
```

**Structure Decision**: Single project structure, extending existing internal packages.
No new directories needed; RDS support integrates into existing pricing and plugin packages.

## Complexity Tracking

> No constitution violations. All changes follow established patterns.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
