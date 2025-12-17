# Implementation Plan: GetRecommendations RPC

**Branch**: `012-recommendations` | **Date**: 2025-12-15 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/012-recommendations/spec.md`

## Summary

Implement the `GetRecommendations` RPC by implementing the `pluginsdk.RecommendationsProvider` interface. The plugin will provide EC2 generation upgrade suggestions, Graviton/ARM migration recommendations, and EBS gp2→gp3 volume type upgrades based on embedded public AWS pricing data comparisons.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: pulumicost-spec v0.4.7 (gRPC + pluginsdk), zerolog
**Storage**: N/A (embedded pricing data via `//go:embed`)
**Testing**: Go testing (`make test`), table-driven tests
**Target Platform**: Linux server (gRPC plugin process)
**Project Type**: Single Go module (gRPC plugin)
**Performance Goals**: < 100ms per GetRecommendations call (SC-001)
**Constraints**: < 50MB memory per region binary, thread-safe for concurrent calls
**Scale/Scope**: Max 2 EC2 recommendations + 1 EBS recommendation per resource

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Code Quality & Simplicity | PASS | Single file `recommendations.go`, simple mappings |
| II. Testing Discipline | PASS | Table-driven tests for mappings, unit tests for helpers |
| III. Protocol & Interface Consistency | PASS | Implements `RecommendationsProvider` interface from pluginsdk |
| IV. Performance & Reliability | PASS | Map lookups O(1), < 100ms target |
| V. Build & Release Quality | PASS | No new build tags needed, same binary |

## Project Structure

### Documentation (this feature)

```text
specs/012-recommendations/
├── plan.md              # This file
├── research.md          # Phase 0: Proto mapping research
├── data-model.md        # Phase 1: Instance family mappings
├── quickstart.md        # Phase 1: Quick implementation guide
├── contracts/           # N/A (uses existing proto)
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
internal/
├── plugin/
│   ├── plugin.go           # Existing - no changes
│   ├── recommendations.go  # NEW: GetRecommendations implementation
│   ├── recommendations_test.go  # NEW: Unit tests
│   └── instance_type.go    # NEW: parseInstanceType + mappings
└── pricing/
    └── client.go           # Existing - no changes (uses EC2/EBS lookups)
```

**Structure Decision**: Follows existing plugin package structure. New files added to `internal/plugin/` for recommendation logic.

## Complexity Tracking

> No violations - implementation follows KISS principle with simple map lookups.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
