# Implementation Plan: S3 Storage Cost Estimation

**Branch**: `011-s3-cost-estimation` | **Date**: 2025-12-07 | **Spec**: specs/011-s3-cost-estimation/spec.md
**Input**: Feature specification from `/specs/011-s3-cost-estimation/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement real S3 storage cost estimation using AWS Public Pricing data. S3 is currently a stub service returning $0 and needs full pricing support for storage classes (Standard, Standard-IA, One Zone-IA, Glacier, etc.). Follow the established EBS pattern in the codebase: extend pricing client interface, add S3 price type, update initialization, create estimator function, update router and supports, extend generate-pricing tool.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.25.4
**Primary Dependencies**: gRPC, zerolog, embedded JSON pricing data
**Storage**: Embedded JSON files in Go binaries (no runtime storage)
**Testing**: Go testing, table-driven tests for storage classes
**Target Platform**: Linux server (gRPC service on loopback)
**Project Type**: Single Go plugin binary
**Performance Goals**: GetProjectedCost() RPC < 100ms, pricing lookup < 50ms
**Constraints**: Thread-safe concurrent RPC calls, embedded pricing data only, no network calls at runtime
**Scale/Scope**: Support 100 concurrent RPC calls, 9 regional binaries

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

✅ **Code Quality & Simplicity**: Follows KISS principle, single responsibility (S3 cost estimation), stateless gRPC handlers, explicit behavior.

✅ **Testing Discipline**: Unit tests for estimateS3() and S3PricePerGBMonth(), table-driven tests for storage classes, integration tests for gRPC methods.

✅ **Protocol & Interface Consistency**: Uses gRPC CostSourceService, zerolog for logging, proto-defined types, ErrorCode enum, thread-safe handlers, region-specific binaries with build tags.

✅ **Performance & Reliability**: Embedded pricing data with sync.Once, indexed maps for lookups, thread-safe access, latency targets met (<100ms RPC, <50ms lookup).

✅ **Build & Release Quality**: Code passes make lint/test, GoReleaser builds for regions, generate-pricing tool for embedded data.

✅ **Security Requirements**: No credentials in logs/responses, embedded pricing (no runtime network), input validation, govulncheck in CI.

✅ **Development Workflow**: Feature branch naming, conventional commits, PR requirements, markdownlint.

**Result**: All gates pass. No violations requiring justification.

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
cmd/pulumicost-plugin-aws-public/
└── main.go                    # gRPC service entry point

internal/
├── plugin/
│   ├── plugin.go              # AWSPublicPlugin struct, gRPC handlers
│   ├── projected.go           # GetProjectedCost router, estimateS3()
│   ├── supports.go            # Supports() method
│   ├── actual.go              # GetActualCost with fallback
│   ├── expected.go            # Expected cost calculations
│   ├── plugin_test.go         # Mock pricing client, integration tests
│   ├── projected_test.go      # S3 estimation tests
│   └── [other test files]
└── pricing/
    ├── client.go              # PricingClient interface, S3PricePerGBMonth()
    ├── client_test.go         # Pricing lookup tests
    ├── types.go               # s3Price struct
    ├── embed_*.go             # Region-specific embedded pricing data
    └── [other embed files]

tools/
├── generate-pricing/
│   └── main.go                # Tool to fetch and generate pricing JSON
└── [other tools]

testdata/
└── [JSON test fixtures]
```

**Structure Decision**: Single Go project with plugin architecture. Core logic in internal/plugin/ and internal/pricing/, build tools in tools/, test data in testdata/. Follows existing codebase patterns.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
