# Implementation Plan: SDK Migration and Code Consolidation

**Branch**: `013-sdk-migration` | **Date**: 2025-12-16 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/013-sdk-migration/spec.md`

## Summary

Consolidate ~210 lines of duplicate code patterns and migrate to finfocus-spec
v0.4.8 SDK helpers for environment variables, request validation, property
mapping, and ARN parsing. This reduces maintenance burden, ensures ecosystem
consistency, and enables ARN-based resource identification for AWS integrations.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: finfocus-spec v0.4.8 (pluginsdk, mapping packages),
gRPC, zerolog
**Storage**: N/A (embedded pricing data via go:embed)
**Testing**: Go testing with table-driven tests, `make test`
**Target Platform**: Linux server (gRPC plugin binary)
**Project Type**: Single Go module with internal packages
**Performance Goals**: <100ms GetProjectedCost RPC, <10ms Supports RPC
**Constraints**: <50MB memory per region binary, thread-safe for concurrent RPCs
**Scale/Scope**: 7 AWS services, 9 regions, ~19 functional requirements

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. Code Quality & Simplicity | PASS | Consolidating duplicates reduces complexity |
| II. Testing Discipline | PASS | Unit tests for new helpers, existing tests unchanged |
| III. Protocol & Interface Consistency | PASS | SDK uses same ErrorCode enum, zerolog logging |
| IV. Performance & Reliability | PASS | sync.Once for initialization, thread-safe helpers |
| V. Build & Release Quality | PASS | make lint, make test required |

**Gate Status**: PASSED - No violations requiring justification

## Project Structure

### Documentation (this feature)

```text
specs/013-sdk-migration/
├── plan.md              # This file
├── research.md          # Phase 0 output - SDK API research
├── data-model.md        # Phase 1 output - entity definitions
├── quickstart.md        # Phase 1 output - implementation guide
├── contracts/           # Phase 1 output - N/A (no new APIs)
└── tasks.md             # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── plugin/
│   ├── ec2_attrs.go         # NEW: EC2Attributes type + extractors (FR-001-003)
│   ├── ec2_attrs_test.go    # NEW: Tests for EC2 attribute extraction
│   ├── validation.go        # NEW: Shared validation helpers (FR-009-011)
│   ├── validation_test.go   # NEW: Tests for validation helpers
│   ├── arn.go               # NEW: ARN parsing (FR-015-019)
│   ├── arn_test.go          # NEW: Tests for ARN parsing
│   ├── estimate.go          # MODIFY: Use ec2_attrs.go helpers
│   ├── projected.go         # MODIFY: Use ec2_attrs.go, validation.go
│   ├── actual.go            # MODIFY: Use validation.go, ARN parsing
│   ├── pricingspec.go       # MODIFY: Use validation.go
│   └── plugin.go            # MODIFY: Use pluginsdk.GetLogLevel() (FR-007)
├── regionsconfig/           # NEW: Shared RegionConfig package (FR-004-005)
│   ├── config.go            # RegionConfig type, Load(), Validate()
│   └── config_test.go       # Tests for config loading and validation
└── pricing/
    └── client.go            # UNCHANGED (thread-safe already)

tools/
├── generate-embeds/
│   └── main.go              # MODIFY: Import internal/regionsconfig
└── generate-goreleaser/
    └── main.go              # MODIFY: Import internal/regionsconfig

cmd/
└── finfocus-plugin-aws-public/
    └── main.go              # MODIFY: Use pluginsdk.GetPort() with PORT fallback (FR-006, FR-008)
```

**Structure Decision**: Single Go module with internal packages. New shared code
goes in `internal/plugin/` (plugin-specific) and `internal/regionsconfig/`
(tool-shared). No new top-level directories.

## Complexity Tracking

> No Constitution violations requiring justification.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
