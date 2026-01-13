# Implementation Plan: E2E Test Support and Validation

**Branch**: `001-e2e-test-support` | **Date**: 2025-12-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-e2e-test-support/spec.md`

## Summary

Add E2E test support features to the finfocus-plugin-aws-public plugin,
including test mode detection via `FINFOCUS_TEST_MODE` environment variable,
enhanced diagnostic logging when enabled, and documented expected cost ranges
for standard test resources (t3.micro EC2, gp2 EBS 8GB). This enables reliable
integration testing with finfocus-core E2E tests.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: gRPC (pluginsdk), finfocus-spec protos, zerolog
**Storage**: Embedded JSON files (go:embed) - no external storage
**Testing**: Go testing via `make test`
**Target Platform**: Linux server (gRPC plugin process)
**Project Type**: Single project (plugin binary)
**Performance Goals**: <100ms per RPC call (existing requirement)
**Constraints**: <50MB memory, no external network calls, loopback-only gRPC
**Scale/Scope**: Single plugin binary per region, concurrent RPC handling

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Code Quality | PASS | Simple env var + logging |
| II. Testing | PASS | Unit tests for test mode |
| III. Protocol | PASS | No new gRPC endpoints |
| IV. Performance | PASS | <100ms, <50MB align |
| V. Build Quality | PASS | Lint/test before commit |
| Security | PASS | No creds, loopback-only |

**Gate Result**: PASS - No violations requiring justification.

## Project Structure

### Documentation (this feature)

```text
specs/001-e2e-test-support/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A - no new endpoints)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── plugin/
│   ├── plugin.go        # Add test mode detection
│   ├── testmode.go      # NEW: Test mode utilities
│   ├── testmode_test.go # NEW: Test mode unit tests
│   ├── expected.go      # NEW: Expected cost range constants
│   └── expected_test.go # NEW: Expected range tests
├── pricing/
│   └── client.go        # Existing - no changes needed
└── config/
    └── config.go        # Add test mode config

cmd/
└── finfocus-plugin-aws-public/
    └── main.go          # Check FINFOCUS_TEST_MODE at startup
```

**Structure Decision**: Existing single-project structure. New files added to
`internal/plugin/` for test mode utilities and expected cost range constants.
No new packages needed - follows KISS principle.

## Complexity Tracking

> No violations to justify - Constitution Check passed.
