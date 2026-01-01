# Implementation Plan: Comprehensive Carbon Estimation Expansion

**Branch**: `001-carbon-estimation` | **Date**: 2025-12-31 | **Spec**: /specs/001-carbon-estimation/spec.md
**Input**: Feature specification from `/specs/001-carbon-estimation/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Expand carbon footprint estimation from EC2-only to all supported AWS services (EC2, EBS, S3, Lambda, RDS, DynamoDB, EKS), adding GPU power consumption, embodied carbon, and establishing grid factor update process. Use CCF methodology with embedded power coefficients and regional grid factors.

## Technical Context

**Language/Version**: Go 1.25.5
**Primary Dependencies**: gRPC, zerolog, embedded JSON pricing data
**Storage**: Embedded JSON files (no runtime storage)
**Testing**: Go testing with table-driven tests for carbon calculations
**Target Platform**: Linux server (gRPC plugin)
**Project Type**: Single project (gRPC plugin)
**Performance Goals**: <100ms per GetProjectedCost RPC, <500ms plugin startup
**Constraints**: <250MB binary size, <400MB memory footprint, thread-safe concurrent calls
**Scale/Scope**: 7 AWS services, embedded pricing data ~150MB per region, support 100+ concurrent RPC calls

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**I. Code Quality & Simplicity**: ✅ Satisfied - Extends existing carbon package with service-specific estimators, maintaining KISS and single responsibility.

**II. Testing Discipline**: ✅ Satisfied - Will add unit tests for each carbon estimator with table-driven tests, no mocking of owned dependencies.

**III. Protocol & Interface Consistency**: ✅ Satisfied - Uses existing GetProjectedCost gRPC method with METRIC_KIND_CARBON_FOOTPRINT, proto-defined types, and thread-safe operation.

**IV. Performance & Reliability**: ✅ Satisfied - Carbon coefficients embedded at build time, lookups use indexed data structures, within latency and resource limits.

**V. Build & Release Quality**: ✅ Satisfied - All code will pass make lint/test, GoReleaser builds, region-specific binaries.

**Security Requirements**: ✅ Satisfied - No credentials/secrets, embedded data only, no runtime network calls.

**Development Workflow**: ✅ Satisfied - Conventional commits, PR reviews, markdownlint on docs.

**Governance**: ✅ Satisfied - No constitution amendments needed.

**Gates Evaluation**: PASS - No constitution violations. Feature implementation can proceed.

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
# [REMOVE IF UNUSED] Option 1: Single project (DEFAULT)
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# [REMOVE IF UNUSED] Option 2: Web application (when "frontend" + "backend" detected)
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/

# [REMOVE IF UNUSED] Option 3: Mobile + API (when "iOS/Android" detected)
api/
└── [same as backend above]

ios/ or android/
└── [platform-specific structure: feature modules, UI flows, platform tests]
```

**Structure Decision**: [Document the selected structure and reference the real
directories captured above]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
