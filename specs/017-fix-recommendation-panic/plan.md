# Implementation Plan: Bug Fix and Documentation Sprint - Dec 2025

**Branch**: `017-fix-recommendation-panic` | **Date**: 2025-12-19 | **Spec**: [specs/017-fix-recommendation-panic/spec.md](spec.md)
**Input**: Feature specification from `/specs/017-fix-recommendation-panic/spec.md`

## Summary

This sprint focuses on improving the stability, reliability, and documentation of the FinFocus AWS Public Plugin. The primary goal is to address several reported bugs, including a high-priority panic in recommendation processing and validation errors in S3 region fallback. Additionally, the sprint will consolidate documentation, add a dedicated troubleshooting guide, and formalize the deprecation of the `PORT` environment variable fallback.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: gRPC, zerolog, pluginsdk
**Storage**: N/A (embedded pricing data)
**Testing**: Go testing (unit and integration)
**Target Platform**: Linux, macOS, Windows (cross-compiled)
**Project Type**: gRPC Plugin (Single project)
**Performance Goals**: <100ms per GetProjectedCost call, <500ms startup time
**Constraints**: <50MB memory footprint, <10MB binary size per region
**Scale/Scope**: ~10 bug fixes and documentation tasks

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. Code Quality & Simplicity**: Adheres to KISS and SRP. No complex abstractions planned for bug fixes.
- [x] **II. Testing Discipline**: All bug fixes will include corresponding unit or integration tests.
- [x] **III. Protocol & Interface Consistency**: All logging will use `zerolog` for structured JSON. No breaking gRPC changes.
- [x] **IV. Performance & Reliability**: No impact on performance; validation improvements will enhance reliability.
- [x] **V. Build & Release Quality**: All changes will pass `make lint` and `make test`.

## Project Structure

### Documentation (this feature)

```text
specs/017-fix-recommendation-panic/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
cmd/
└── finfocus-plugin-aws-public/
    └── main.go          # PORT deprecation warning

internal/
├── carbon/
│   ├── instance_specs.go # CSV parsing error logging, trailing newlines
│   └── data/            # CCF data integrity
├── plugin/
│   ├── arn.go           # S3 region fallback fix
│   ├── projected.go     # Cost validation restoration
│   ├── recommendations.go # Panic fix, correlation ID docs
│   └── supports.go      # Utilization doc cleanup
└── pricing/
    └── client.go        # EC2 OS mapping docs

TROUBLESHOOTING.md       # New troubleshooting guide
```

**Structure Decision**: Single Go project following established finfocus-plugin conventions.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |