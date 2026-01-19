# Implementation Plan: Windows vs Linux Pricing Differentiation Integration Test

**Branch**: `034-test-windows-pricing` | **Date**: 2026-01-18 | **Spec**: [specs/034-test-windows-pricing/spec.md](spec.md)
**Input**: Feature specification from `/specs/034-test-windows-pricing/spec.md`

## Summary
Add an end-to-end integration test that verifies Windows EC2 instances return different (higher) pricing than Linux EC2 instances for the same instance type. The test will also verify tenancy-based pricing (Shared vs Dedicated) and architecture-based pricing (x86 vs ARM).

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: gRPC (finfocus.v1), rs/zerolog, pluginsdk
**Storage**: N/A (Embedded pricing data)
**Testing**: Go testing (integration tests with `//go:build integration`)
**Target Platform**: Linux (plugin architecture)
**Project Type**: single (plugin)
**Performance Goals**: startup < 1s, PORT < 1s, GetProjectedCost < 100ms
**Constraints**: binary < 250MB, memory < 400MB
**Scale/Scope**: Integration tests for Windows vs Linux pricing differentiation, Dedicated tenancy, and architecture (x86 vs ARM).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Quality**: PASS - Plan follows simple integration test patterns.
- **II. Testing**: PASS - Focuses on integration tests for gRPC methods.
- **III. Protocol**: PASS - Uses standard gRPC protocol and zerolog.
- **IV. Performance**: PASS - Embedded data lookup is O(1) via map.
- **V. Release**: PASS - Integrated into `make test` via tags.

## Project Structure

### Documentation (this feature)

```text
specs/034-test-windows-pricing/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── contracts/
    └── finfocus.v1.md   # API contract reference
```

### Source Code (repository root)

```text
internal/plugin/
└── integration_pricing_test.go  # New integration test file
```

**Structure Decision**: Single project structure. Integration tests are placed within the package they test (`internal/plugin`) using the `_test.go` suffix and `integration` build tag.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

(No violations)