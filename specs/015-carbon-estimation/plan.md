# Implementation Plan: Carbon Emission Estimation

**Branch**: `015-carbon-estimation` | **Date**: 2025-12-19 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/015-carbon-estimation/spec.md`

## Summary

Implement carbon emission estimation for EC2 instances using Cloud Carbon Footprint (CCF) methodology. The feature adds `impact_metrics` containing `METRIC_KIND_CARBON_FOOTPRINT` (gCO2e) to `GetProjectedCostResponse` by embedding CCF instance specification data (vCPU, min/max watts) and grid emission factors, then applying the CCF formula during EC2 cost estimation.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: finfocus-spec v0.4.10+ (MetricKind, ImpactMetric), zerolog, gRPC
**Storage**: Embedded data via `//go:embed` (CSV for instance specs, constants for grid factors)
**Testing**: Go testing with table-driven tests, integration tests for gRPC
**Target Platform**: Linux server (gRPC plugin binary)
**Project Type**: Single project (existing Go gRPC plugin)
**Performance Goals**: Carbon calculation < 1ms per call (lookup-based)
**Constraints**: No external API calls at runtime, thread-safe, <50MB memory
**Scale/Scope**: 500+ instance types, 12 AWS regions

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Code Quality & Simplicity | ✅ PASS | Single-purpose carbon module, no over-engineering |
| II. Testing Discipline | ✅ PASS | Unit tests for formula, table-driven for instance lookups |
| III. Protocol & Interface | ✅ PASS | Uses proto-defined ImpactMetric, MetricKind from v0.4.10 |
| IV. Performance & Reliability | ✅ PASS | Embedded data parsed once via sync.Once, indexed maps |
| V. Build & Release Quality | ✅ PASS | Existing GoReleaser workflow unchanged |
| Security | ✅ PASS | No credentials, embedded data only, loopback gRPC |

## Project Structure

### Documentation (this feature)

```text
specs/015-carbon-estimation/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (gRPC changes documented)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── carbon/                      # NEW: Carbon estimation module
│   ├── constants.go             # PUE, default utilization, hours per month
│   ├── estimator.go             # CarbonEstimator interface + implementation
│   ├── estimator_test.go        # Unit tests for carbon formula
│   ├── instance_specs.go        # Instance type to vCPU/watts mapping (includes go:embed)
│   ├── instance_specs_test.go   # Lookup tests
│   ├── grid_factors.go          # Region to grid emission factor mapping
│   ├── utilization.go           # getUtilization helper with priority logic
│   └── utilization_test.go      # Utilization clamping tests
├── plugin/
│   ├── projected.go             # MODIFY: Add carbon calculation to estimateEC2
│   ├── projected_test.go        # MODIFY: Add carbon metric assertions
│   └── supports.go              # MODIFY: Return supported_metrics for EC2
└── pricing/
    └── client.go                # UNCHANGED (financial pricing only)

data/
└── ccf_instance_specs.csv       # NEW: CCF instance data (downloaded, embedded at build)
```

**Structure Decision**: New `internal/carbon/` package keeps carbon logic separate from financial pricing, following Single Responsibility Principle. Modifications to `plugin/` are minimal - just wiring up the carbon estimator.

## Complexity Tracking

> No violations detected. Feature uses existing patterns (embedded data, sync.Once, indexed maps).

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
