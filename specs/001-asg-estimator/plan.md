# Implementation Plan: ASG Cost Estimator

**Branch**: `001-asg-estimator` | **Date**: 2026-03-26 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-asg-estimator/spec.md`

## Summary

Add a cost estimator for `aws:autoscaling/group:Group` resources that
delegates to existing EC2 pricing data. The ASG estimator multiplies the
per-instance on-demand EC2 hourly rate by desired capacity × 730 hours/month.
No new pricing data is needed — ASGs have no direct AWS charge; cost is
the aggregate of managed instances. Carbon estimation delegates to the
existing EC2 carbon estimator, scaled by instance count.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: finfocus-spec v0.5.6 (pluginsdk, pbc, pbcconnect),
connectrpc.com/connect v1.19.1, zerolog
**Storage**: N/A (in-memory pricing data via `//go:embed`)
**Testing**: Go testing with table-driven tests, integration tests with
`-tags=integration`
**Target Platform**: Linux server (gRPC service on 127.0.0.1)
**Project Type**: Single Go module
**Performance Goals**: GetProjectedCost RPC < 100ms, Supports < 10ms, startup
< 500ms
**Constraints**: No runtime network calls (air-gapped), binary < 250MB,
memory < 400MB
**Scale/Scope**: ~200 lines of new estimator code, ~400 lines of tests,
updates to ~10 existing files, 2 new files

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --------- | ------ | ----- |
| I. Code Quality & Simplicity | PASS | Single-purpose estimator function, no new abstractions. Reuses existing EC2 pricing lookup — no new data structures. |
| II. Testing Discipline | PASS | Table-driven unit tests for cost calculation, tag extraction, normalization. Integration test for full RPC path. |
| III. Protocol & Interface Consistency | PASS | Uses proto-defined types only. No stdout logging. gRPC error codes from proto enum. Thread-safe (read-only pricing data). |
| IV. Performance & Reliability | PASS | No new pricing data parsing. ASG estimate is a map lookup + multiplication — well under 100ms. No binary size increase. |
| V. Build & Release Quality | PASS | No new build tags. No new embed files. Existing `make lint` and `make test` cover new code. |
| Security | PASS | No new inputs beyond existing ResourceDescriptor. No network calls. Input validation via existing helpers. |

No violations. No complexity tracking needed.

## Project Structure

### Documentation (this feature)

```text
specs/001-asg-estimator/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research findings
├── data-model.md        # Phase 1 data model
├── quickstart.md        # Phase 1 quickstart guide
├── contracts/           # Phase 1 API contracts
│   └── asg-rpc.md       # gRPC contract for ASG RPCs
└── checklists/
    └── requirements.md  # Spec quality checklist
```

### Source Code (repository root)

```text
internal/
  plugin/
    constants.go          # ADD: serviceASG constant
    projected.go          # ADD: case serviceASG in switch, estimateASG() method
    supports.go           # ADD: case serviceASG in switch, getSupportedMetrics update
    pricingspec.go        # ADD: case serviceASG in switch, asgPricingSpec() method
    actual.go             # ADD: case serviceASG in getProjectedForResource switch
    classification.go     # ADD: ASG entry in serviceClassifications map
    arn.go                # UPDATE: autoscaling case to detect ASG vs LaunchConfig
    asg_attrs.go          # NEW: ASG-specific tag extraction helpers
  carbon/
    asg_estimator.go      # NEW: ASG carbon estimator (delegates to EC2)
    types.go              # ADD: ASGConfig struct
```

**Structure Decision**: All new code fits within existing package structure.
One new file in `internal/plugin/` for ASG tag extraction (follows pattern of
`ec2_attrs.go`). One new file in `internal/carbon/` for ASG carbon estimator
(follows pattern of `elasticache_estimator.go`). No new packages needed.
