# Implementation Plan: Pre-allocate Map Capacity for Pricing Indexes

**Branch**: `021-map-prealloc` | **Date**: 2026-01-04 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/021-map-prealloc/spec.md`

## Summary

Add pre-allocation capacity hints to six pricing index maps in `internal/pricing/client.go` to reduce GC pressure during initialization. The change modifies the `init()` function's `make()` calls to include capacity parameters based on observed AWS pricing data volumes.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: zerolog (logging), sync (thread safety), pulumicost-spec SDK (gRPC)
**Storage**: N/A (embedded pricing data via `//go:embed`)
**Testing**: Go testing with existing `BenchmarkNewClient` and `BenchmarkNewClient_Parallel`
**Target Platform**: Linux server (gRPC plugin binary)
**Project Type**: Single project (internal library modification)
**Performance Goals**: 10% reduction in allocations (allocs/op), no timing regression
**Constraints**: Memory usage (B/op) may increase up to 10% due to upfront allocation
**Scale/Scope**: ~90,000 EC2 products, 6 map indexes, single file change

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. Code Quality & Simplicity | PASS | Single-line changes to existing `make()` calls; no new abstractions |
| II. Testing Discipline | PASS | Existing benchmarks validate optimization; no new mocking required |
| III. Protocol & Interface Consistency | PASS | No gRPC protocol changes; internal optimization only |
| IV. Performance & Reliability | PASS | Improves initialization performance; aligns with <500ms startup goal |
| V. Build & Release Quality | PASS | No build tag or GoReleaser changes required |

**Gate Status**: PASS - No violations. Proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/021-map-prealloc/
├── plan.md              # This file
├── research.md          # Phase 0 output (minimal - no unknowns)
├── spec.md              # Feature specification
└── checklists/
    └── requirements.md  # Validation checklist
```

### Source Code (repository root)

```text
internal/
└── pricing/
    ├── client.go           # MODIFY: Add capacity to make() calls (lines 192-197)
    └── client_test.go      # VERIFY: Existing benchmarks validate optimization
```

**Structure Decision**: No new files or directories. Single-file modification to existing `client.go`.

## Complexity Tracking

> No violations requiring justification. This feature adds zero new abstractions.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |

## Phase 0: Research Summary

**Status**: No NEEDS CLARIFICATION items in Technical Context.

This is a well-understood Go optimization pattern. The research phase is minimal:

### Decision: Capacity Values

| Map | Capacity | Rationale |
|-----|----------|-----------|
| `ec2Index` | 100,000 | ~90k products in us-east-1; 10% buffer for growth |
| `ebsIndex` | 50 | ~20-30 volume types; 2x buffer |
| `s3Index` | 100 | ~50-100 storage classes; comfortable margin |
| `rdsInstanceIndex` | 5,000 | Instance type × engine combinations |
| `rdsStorageIndex` | 100 | Storage types across engines |
| `elasticacheIndex` | 1,000 | Node type × engine (Redis/Memcached/Valkey) |

**Alternatives Considered**:
- Dynamic sizing based on data inspection: Rejected (adds complexity, minimal benefit)
- Using `len(products)` from JSON: Rejected (requires two-pass parsing)

### Decision: Benchmark Validation

Existing infrastructure is sufficient:
- `BenchmarkNewClient` measures initialization time and allocations
- `BenchmarkNewClient_Parallel` validates thread safety under load
- CI comparison (from clarification) will use `benchstat` for statistical comparison

## Phase 1: Design

### Data Model

No new data structures. Existing types unchanged:

```go
// Unchanged - only capacity hints added to make() calls
type Client struct {
    ec2Index           map[string]ec2Price           // capacity: 100,000
    ebsIndex           map[string]ebsPrice           // capacity: 50
    s3Index            map[string]s3Price            // capacity: 100
    rdsInstanceIndex   map[string]rdsInstancePrice   // capacity: 5,000
    rdsStorageIndex    map[string]rdsStoragePrice    // capacity: 100
    elasticacheIndex   map[string]elasticacheInstancePrice // capacity: 1,000
    // ... other fields unchanged
}
```

### Code Changes

**File**: `internal/pricing/client.go`
**Location**: `init()` function, lines 192-197

**Before**:

```go
c.ec2Index = make(map[string]ec2Price)
c.ebsIndex = make(map[string]ebsPrice)
c.s3Index = make(map[string]s3Price)
c.rdsInstanceIndex = make(map[string]rdsInstancePrice)
c.rdsStorageIndex = make(map[string]rdsStoragePrice)
c.elasticacheIndex = make(map[string]elasticacheInstancePrice)
```

**After**:

```go
// Pre-allocate map capacities based on typical AWS pricing data volumes.
// Capacity estimates derived from us-east-1 (largest region) with buffer for growth.
// See GitHub issue #176 for sizing rationale.
c.ec2Index = make(map[string]ec2Price, 100000)              // ~90k EC2 products
c.ebsIndex = make(map[string]ebsPrice, 50)                  // ~20-30 volume types
c.s3Index = make(map[string]s3Price, 100)                   // ~50-100 storage classes
c.rdsInstanceIndex = make(map[string]rdsInstancePrice, 5000) // instance×engine combos
c.rdsStorageIndex = make(map[string]rdsStoragePrice, 100)   // storage types
c.elasticacheIndex = make(map[string]elasticacheInstancePrice, 1000) // node×engine
```

### Contracts

N/A - No API or interface changes. Internal optimization only.

### Validation Strategy

1. **Before implementing**: Run baseline benchmarks on `main` branch

   ```bash
   go test -tags=region_use1 -bench=BenchmarkNewClient -benchmem -count=10 ./internal/pricing/... > baseline.txt
   ```

2. **After implementing**: Run benchmarks on feature branch

   ```bash
   go test -tags=region_use1 -bench=BenchmarkNewClient -benchmem -count=10 ./internal/pricing/... > after.txt
   ```

3. **Compare results**: Use benchstat for statistical comparison

   ```bash
   benchstat baseline.txt after.txt
   ```

4. **Success criteria validation**:
   - SC-001: allocs/op decreased by ≥10%
   - SC-002: ns/op within 5% of baseline (or improved)
   - SC-003/SC-004: `make test` passes
   - SC-005: B/op increase ≤10%

## Implementation Tasks

> Note: Detailed task breakdown will be generated by `/speckit.tasks`

1. Capture baseline benchmark results
2. Modify `client.go` with pre-allocated capacities
3. Add explanatory comments documenting capacity rationale
4. Run benchmarks and compare against baseline
5. Verify all existing tests pass
6. Update PR with benchmark comparison results

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Capacities too small (still rehashing) | Low | Low | Conservative estimates with buffers |
| Memory increase exceeds 10% | Low | Low | Monitor B/op in benchmark; acceptable tradeoff |
| No measurable improvement | Medium | Low | Feature still provides code documentation value |

## Post-Phase 1 Constitution Re-Check

| Principle | Status | Post-Design Notes |
|-----------|--------|-------------------|
| I. Code Quality & Simplicity | PASS | 6 lines changed + comments; no new abstractions |
| II. Testing Discipline | PASS | Existing benchmarks sufficient |
| III. Protocol & Interface Consistency | PASS | No protocol impact |
| IV. Performance & Reliability | PASS | Explicit improvement to initialization |
| V. Build & Release Quality | PASS | No build process changes |

**Final Gate Status**: PASS - Ready for task generation.
