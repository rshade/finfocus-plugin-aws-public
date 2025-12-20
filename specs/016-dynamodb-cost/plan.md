# Implementation Plan: DynamoDB Cost Estimation

**Branch**: `016-dynamodb-cost` | **Date**: 2025-12-19 | **Spec**: [specs/016-dynamodb-cost/spec.md](spec.md)
**Input**: Feature specification from `/specs/016-dynamodb-cost/spec.md`

## Summary

Implement DynamoDB cost estimation for both On-Demand and Provisioned capacity modes. This involves extending the `internal/pricing` package to fetch, embed, and lookup DynamoDB rates, and implementing the `estimateDynamoDB` logic in the `internal/plugin` package.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: gRPC, RS/Zerolog, Pluginsdk
**Storage**: N/A (Embedded pricing data)
**Testing**: Go testing (Unit & Integration)
**Target Platform**: Linux/Universal (Go)
**Project Type**: Single gRPC Plugin
**Performance Goals**: < 100ms per `GetProjectedCost` RPC, < 10ms per `Supports` RPC
**Constraints**: < 50MB memory footprint, thread-safe pricing lookups
**Scale/Scope**: 12 regions, 2 capacity modes, storage estimation

## Constitution Check

*GATE: Passed. Re-checked after Phase 1 design.*

- [x] **KISS**: Simple estimator function and pricing struct, no complex abstractions.
- [x] **SRP**: `PricingClient` handles data, `AWSPublicPlugin` handles RPC logic.
- [x] **Protocol**: Adheres to `CostSourceService` v1 and uses `zerolog`.
- [x] **Performance**: Uses indexed maps for pricing lookups; data embedded at build time.
- [x] **Testing**: Table-driven unit tests for calculations; integration tests for the plugin.

## Project Structure

### Documentation (this feature)

```text
specs/016-dynamodb-cost/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── checklists/          # Validation checklists
└── spec.md              # Feature specification
```

### Source Code (repository root)

```text
cmd/pulumicost-plugin-aws-public/
  └── main.go            # Entry point (no changes expected)

internal/
├── plugin/
│   ├── plugin.go        # Supports() update
│   ├── projected.go     # estimateDynamoDB() implementation
│   ├── projected_test.go # DynamoDB test cases
│   └── supports.go      # Move dynamodb to supported list
├── pricing/
│   ├── client.go        # Extend interface and Client struct
│   ├── types.go         # Add dynamoDBPrice struct
│   └── ...              # Embedded JSON data (generated)
└── ...

tools/
└── generate-pricing/
    └── main.go          # Add DynamoDB fetch logic
```

**Structure Decision**: Single project structure follows the established pattern of regional binaries with embedded data.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None      | -          | -                                   |