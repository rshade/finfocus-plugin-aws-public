# Implementation Plan: Lambda Cost Estimation

**Branch**: `014-lambda-cost-estimation`
**Date**: 2025-12-18
**Spec**: [specs/014-lambda-cost-estimation/spec.md](./spec.md)
**Input**: Feature specification from `specs/014-lambda-cost-estimation/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See
`.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement Lambda function cost estimation using AWS Public Pricing data,
supporting both request count and compute duration (GB-seconds) dimensions. The
solution extends the existing gRPC-based plugin architecture by adding
Lambda-specific pricing types, lookup logic, and cost calculation based on
resource memory configuration and usage tags.

## Technical Context

- **Language/Version**: Go 1.25+
- **Primary Dependencies**: gRPC (pulumicost.v1 protocol), internal/pricing
  (embedded data), zerolog
- **Storage**: Embedded JSON pricing data (using `//go:embed`)
- **Testing**: Go testing (unit for calculation, integration for pricing
  lookup)
- **Target Platform**: Linux/macOS/Windows (cross-compiled binaries)
- **Project Type**: gRPC Plugin (CLI-invoked subprocess)
- **Performance Goals**: < 100ms per RPC call, < 50MB memory footprint
- **Constraints**: No runtime network calls, thread-safe execution, strict
  proto compliance
- **Scale/Scope**: 9 supported AWS regions, high concurrency support

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **Code Quality**: Changes limited to pricing logic and estimator; no
  complex abstractions introduced. (Principle I)
- [x] **Testing**: Unit tests for price calculation; integration tests for
  lookup. No external mocking. (Principle II)
- [x] **Protocol**: Uses standard `CostSourceService` gRPC definition; no
  custom types. (Principle III)
- [x] **Performance**: Uses embedded pricing data (fast lookup); thread-safe
  implementation. (Principle IV)
- [x] **Security**: No credentials required; input validation on
  ResourceDescriptor. (Security Requirements)
- [x] **Simplicity**: Direct implementation of Lambda pricing formula; reuses
  existing patterns. (Principle I)

*Post-Design Re-check:*

- [x] **Data Model**: `lambdaPrice` struct is simple and serializable.
- [x] **Contracts**: No API contract changes; purely internal logic update.
- [x] **Complexity**: No additional dependencies or architectural layers
  required.

## Project Structure

### Documentation (this feature)

```text
specs/014-lambda-cost-estimation/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── plugin/              # Core logic: supports.go, projected.go
├── pricing/             # Pricing data: client.go, types.go
tools/
└── generate-pricing/    # Data generation: main.go
```

**Structure Decision**: Standard Go project layout following existing
repository conventions. Single module structure.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| None      | N/A        | N/A                                  |
