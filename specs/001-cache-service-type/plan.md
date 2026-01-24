# Implementation Plan: Cache Normalized Service Type

**Branch**: `001-cache-service-type` | **Date**: 2026-01-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-cache-service-type/spec.md`

## Summary

Performance optimization to eliminate duplicate `detectService()` and `normalizeResourceType()` calls within a single request lifecycle. Currently, these pure string functions are called 2-3 times per resource across validation, support checks, and cost routing. The optimization introduces a lightweight memoized wrapper that computes the normalized service type exactly once per resource.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: gRPC (finfocus-spec/sdk/go/pluginsdk), zerolog
**Storage**: N/A (pure in-memory optimization, no persistence)
**Testing**: Go testing with benchmarks (go test -bench, go test -race)
**Target Platform**: Linux server (gRPC plugin binary)
**Project Type**: Single project (gRPC service plugin)
**Performance Goals**: Reduce detectService calls from 2-3 to 1 per resource; <100ms GetProjectedCost RPC
**Constraints**: <100 bytes memory overhead per resource; thread-safe for concurrent gRPC calls; zero behavior change
**Scale/Scope**: Single package refactoring (internal/plugin); ~15 call sites across 7 files

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Code Quality & Simplicity | ✅ PASS | Memoized wrapper is simple, single-purpose struct |
| II. Testing Discipline | ✅ PASS | Benchmark tests + race detection required; no new mocks needed |
| III. Protocol & Interface Consistency | ✅ PASS | No gRPC protocol changes; internal refactoring only |
| IV. Performance & Reliability | ✅ PASS | Optimization must maintain <100ms RPC, thread-safe |
| V. Build & Release Quality | ✅ PASS | Must pass make lint, make test |
| Security Requirements | ✅ PASS | No new inputs/outputs; pure internal optimization |

**Gate Result**: PASS - No violations. Proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/001-cache-service-type/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
internal/plugin/
├── projected.go         # Contains detectService(), normalizeResourceType()
├── supports.go          # Calls detectService() for resource support checks
├── validation.go        # Calls detectService() for region/service validation
├── actual.go            # Calls detectService() for actual cost routing
├── pricingspec.go       # Calls detectService() for pricing spec routing
├── recommendations.go   # Calls detectService() in batch processing
├── plugin.go            # Calls detectService() for metadata extraction
└── service_cache.go     # NEW: Memoized service type wrapper (proposed)
```

**Structure Decision**: Single package modification within `internal/plugin/`. No new packages or directories needed. The optimization is contained within the existing plugin package.

## Complexity Tracking

> No violations to justify. This is a pure refactoring with no new abstractions beyond a simple memoized wrapper struct.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
