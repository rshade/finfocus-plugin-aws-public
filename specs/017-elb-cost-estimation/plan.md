# Implementation Plan: Implement Elastic Load Balancing (ALB/NLB) cost estimation

**Branch**: `017-elb-cost-estimation` | **Date**: 2025-12-19 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/017-elb-cost-estimation/spec.md`

## Summary

Implement cost estimation for AWS Application Load Balancers (ALB) and Network Load Balancers (NLB). This involves updating the pricing client to fetch `AWSELB` data, adding support for the "elb" resource type, and implementing a calculation logic that handles fixed hourly rates plus variable capacity unit charges (LCU/NLCU) based on resource tags.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: gRPC, pulumicost.v1 protocol, rs/zerolog, pluginsdk
**Storage**: Embedded JSON pricing data (Go 1.16+ `embed`), parsed into indexed maps
**Testing**: Go standard `testing` library, table-driven unit and integration tests
**Target Platform**: Linux (multiple regional binaries)
**Project Type**: single (Go gRPC plugin)
**Performance Goals**: < 100ms per `GetProjectedCost` RPC, < 500ms startup
**Constraints**: < 50MB memory footprint, < 10MB binary size
**Scale/Scope**: Supports ALB and NLB across all configured regions

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **[x] KISS**: Logic follows existing estimator patterns in `projected.go`.
- **[x] SRP**: `estimateELB` handles calculation; `PricingClient` handles data retrieval.
- **[x] Protocol Consistency**: Uses `pulumicost.v1` types and `pluginsdk.Serve()`.
- **[x] Thread Safety**: Pricing data is cached and read-only after initialization.
- **[x] ZeroLog**: Structured JSON logging to stderr.

## Project Structure

### Documentation (this feature)

```text
specs/017-elb-cost-estimation/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── checklists/
    └── requirements.md  # Spec quality checklist
```

### Source Code (repository root)

```text
internal/
├── plugin/
│   ├── projected.go    # Add estimateELB(), update switch
│   ├── supports.go     # Add "elb", "alb", "nlb"
│   └── elb_test.go     # New test file for ELB
├── pricing/
│   ├── client.go       # Add ALB/NLB lookup methods, update parser
│   └── types.go        # Add elbPrice struct
tools/
└── generate-pricing/
    └── main.go         # Add AWSELB service code
```

**Structure Decision**: Standard Go package structure as established in the project.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None      | N/A        | N/A                                 |