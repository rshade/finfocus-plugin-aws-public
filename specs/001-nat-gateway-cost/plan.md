# Implementation Plan: VPC NAT Gateway Cost Estimation

**Branch**: `001-nat-gateway-cost` | **Date**: 2025-12-22 | **Spec**: [specs/001-nat-gateway-cost/spec.md](spec.md)

## Summary

Implement cost estimation for AWS VPC NAT Gateways by extending the existing gRPC-based plugin. This involves adding `AmazonVPC` pricing data support, implementing lookup logic for hourly and data processing rates, and providing a specialized estimation function that processes resource tags for data volume.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: gRPC, rs/zerolog, finfocus-spec sdk
**Storage**: N/A (Embedded JSON pricing data)
**Testing**: Go table-driven tests (unit + integration)
**Target Platform**: Linux (amd64/arm64)
**Project Type**: single (gRPC Plugin)
**Performance Goals**: < 100ms per GetProjectedCost() call
**Constraints**: < 250MB binary size, < 400MB memory footprint
**Scale/Scope**: Support VPC NAT Gateway across all 9 regions

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Condition | Status |
|-----------|-----------|--------|
| I. KISS | No premature abstraction | ✅ |
| III. Protocol | gRPC CostSourceService adherence | ✅ |
| IV. Perf | < 250MB binary, < 400MB memory | ✅ |
| IV. Data | NO pricing data filtering | ✅ |
| IV. Thread Safety | Concurrent lookup support | ✅ |

## Project Structure

### Documentation (this feature)

```text
specs/001-nat-gateway-cost/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
└── quickstart.md        # Phase 1 output
```

### Source Code (repository root)

```text
internal/
├── plugin/
│   ├── projected.go     # Add estimateNATGateway
│   ├── supports.go      # Add natgw to supported types
│   └── ...
├── pricing/
│   ├── client.go        # Add parseNATGatewayPricing, NATGatewayPrice methods
│   ├── types.go         # Add natGatewayPrice struct
│   ├── embed_*.go       # Add rawVPCJSON embed
│   └── ...
tools/
└── generate-pricing/
    └── main.go          # Add AmazonVPC service support
```

**Structure Decision**: Standard single-project structure with clear separation between plugin logic and pricing data handling.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
