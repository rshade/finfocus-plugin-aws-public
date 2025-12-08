# Implementation Plan: Lambda Function Cost Estimation

**Branch**: `001-lambda-cost-estimation` | **Date**: 2025-12-07 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-lambda-cost-estimation/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement Lambda function cost estimation for the PulumiCost AWS Public plugin. Add Lambda pricing support using request count and GB-seconds dimensions, extending the existing pricing client interface and creating a new estimator function. Lambda is currently a stub service returning $0 and needs pricing support for accurate serverless compute cost estimates.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.25.4
**Primary Dependencies**: gRPC, zerolog, embedded JSON pricing data
**Storage**: Embedded JSON files (no runtime storage)
**Testing**: Go testing with table-driven tests
**Target Platform**: Linux (gRPC service)
**Project Type**: Single project (plugin)
**Performance Goals**: <100ms per GetProjectedCost() RPC, support 100 concurrent Lambda cost estimation requests per second
**Constraints**: <50MB memory footprint, thread-safe pricing lookups, embedded pricing data only
**Scale/Scope**: Local plugin with system-dependent limits (no fixed upper bounds), handles up to 1000 concurrent requests with <100MB memory. Actual limits depend on user system's available RAM and CPU cores.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

✅ **I. Code Quality & Simplicity**: Feature follows KISS principle with straightforward cost calculation logic. Single responsibility for Lambda pricing estimation.

✅ **II. Testing Discipline**: Design includes comprehensive unit tests for cost calculations and integration tests for gRPC handlers. No over-engineering of test infrastructure.

✅ **III. Protocol & Interface Consistency**: Uses gRPC CostSourceService protocol, zerolog structured logging, proto-defined types only. Thread-safe implementation required.

✅ **IV. Performance & Reliability**: Meets <100ms latency target, <50MB memory constraint, thread-safe concurrent access. Uses embedded pricing data with sync.Once initialization.

✅ **V. Build & Release Quality**: Code will pass `make lint` and `make test`. Uses region-specific build tags and embedded pricing data generation.

✅ **Security Requirements**: No credentials or network calls at runtime. Input validation for ResourceDescriptor fields. Loopback-only gRPC service.

✅ **Development Workflow**: Follows conventional commit format, includes tests, passes CI checks. Updates CLAUDE.md if new patterns emerge.

**Gate Status**: PASS - No constitution violations detected. Phase 1 design completed successfully and maintains architectural compliance.

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
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
# Plugin Architecture (Single Project)
cmd/pulumicost-plugin-aws-public/
└── main.go                    # gRPC service entry point

internal/
├── plugin/                    # Core plugin logic
│   ├── plugin.go             # AWSPublicPlugin struct and gRPC handlers
│   ├── projected.go          # GetProjectedCost implementation (+ NEW: estimateLambda)
│   ├── supports.go           # Supports method (+ update Lambda support)
│   ├── actual.go             # GetActualCost implementation
│   └── *_test.go             # Unit and integration tests
└── pricing/                   # Pricing data management
    ├── client.go             # PricingClient interface (+ NEW: Lambda methods)
    ├── types.go              # Pricing data structures (+ NEW: lambdaPrice)
    ├── embed_*.go            # Region-specific embedded pricing data
    └── client_test.go        # Pricing client tests

tools/generate-pricing/       # Build-time pricing data generation
└── main.go                   # Downloads and embeds AWS pricing (+ Lambda support)
```

**Structure Decision**: Single project following established plugin architecture. Lambda implementation extends existing pricing client and plugin handlers without introducing new packages or complex dependencies.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
