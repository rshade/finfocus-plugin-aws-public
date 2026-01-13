# Implementation Plan: DynamoDB Hardening Bundle

**Branch**: `020-dynamodb-hardening` | **Date**: 2025-12-31 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/020-dynamodb-hardening/spec.md`

## Summary

Harden DynamoDB cost estimation with comprehensive input validation, proper error handling for missing pricing data, and integration tests. This addresses four related issues (#147, #149, #151, #152) in a single release to improve observability and prevent silent $0 estimates.

**Technical Approach**: Modify `estimateDynamoDB()` in `internal/plugin/projected.go` to validate tag inputs and check pricing lookup return values. Add warning logs via zerolog when validation fails or pricing is unavailable. Create new integration test file for DynamoDB end-to-end validation.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: zerolog (logging), pluginsdk (gRPC), finfocus-spec proto
**Storage**: N/A (embedded pricing data)
**Testing**: Go testing with table-driven tests, integration tests with build tags
**Target Platform**: Linux server (gRPC plugin binary)
**Project Type**: Single project (gRPC plugin)
**Performance Goals**: < 100ms per GetProjectedCost() RPC
**Constraints**: No new dependencies, maintain thread safety
**Scale/Scope**: Modifying ~150 lines in projected.go, adding ~200 lines of tests

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Code Quality & Simplicity | ✅ PASS | Adding validation/logging, no new abstractions |
| II. Testing Discipline | ✅ PASS | Adding unit + integration tests for new behavior |
| III. Protocol Consistency | ✅ PASS | Uses zerolog, proto types unchanged |
| IV. Performance & Reliability | ✅ PASS | Validation adds negligible overhead |
| V. Build & Release Quality | ✅ PASS | Tests run via `make test` |
| Security | ✅ PASS | Input validation improves security posture |

**No violations - proceeding with Phase 0.**

## Project Structure

### Documentation (this feature)

```text
specs/020-dynamodb-hardening/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (minimal - no new entities)
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A - no API changes)
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── plugin/
│   ├── projected.go                    # MODIFY: estimateDynamoDB() validation + error handling
│   ├── projected_test.go               # MODIFY: Add unit tests for validation/warnings
│   └── integration_dynamodb_test.go    # NEW: Integration tests for DynamoDB
└── pricing/
    ├── client.go                       # REVIEW: Verify product family matching
    └── client_test.go                  # MODIFY: Add pricing extraction tests
```

**Structure Decision**: Single project structure. Changes are localized to existing files in `internal/plugin/` and `internal/pricing/`.

## Complexity Tracking

> No constitution violations - this section is empty.

## Phase 0: Research Complete

No external research required. The feature is well-defined and uses existing patterns:

1. **Validation Pattern**: Follow existing tag parsing pattern in `estimateDynamoDB()` but add bounds checking
2. **Warning Logging**: Use zerolog `Warn()` level consistent with constitution requirements
3. **Integration Test Pattern**: Follow existing `integration_test.go` pattern with build tags

### Key Findings

- **Current Implementation**: `estimateDynamoDB()` at `projected.go:474-593` ignores `ok` return values from pricing lookups (lines 522-523, 567-568)
- **Storage Parsing**: Currently at line 488, only accepts `v > 0`, should also reject negative values explicitly
- **Pricing Client**: All 5 DynamoDB methods properly return `(float64, bool)` - just need to check the bool
- **Integration Test Pattern**: Use `//go:build integration` tag, build binary with region tag, connect via gRPC

## Phase 1: Design & Contracts

### Data Model

No new entities. Existing entities unchanged:

- **ResourceDescriptor**: Input (unchanged)
- **GetProjectedCostResponse**: Output (unchanged)
- **PricingClient**: Pricing lookup interface (unchanged)

### API Contracts

No API changes. The gRPC interface remains unchanged:

- `GetProjectedCost(ResourceDescriptor) -> GetProjectedCostResponse`

The changes are internal:
- Input validation happens before pricing calculation
- Warning logs emitted via zerolog to stderr
- `billing_detail` field may include "(pricing unavailable)" notation

### Implementation Design

#### 1. Input Validation Helper (New)

```go
// validateNonNegativeInt64 validates and parses an int64 tag value.
// Returns the parsed value (defaulting to 0 if negative) and logs a warning if invalid.
func (p *AWSPublicPlugin) validateNonNegativeInt64(traceID, tagName, value string) int64 {
    v, err := strconv.ParseInt(value, 10, 64)
    if err != nil {
        p.logger.Warn().
            Str(pluginsdk.FieldTraceID, traceID).
            Str("tag", tagName).
            Str("value", value).
            Msg("invalid integer value, defaulting to 0")
        return 0
    }
    if v < 0 {
        p.logger.Warn().
            Str(pluginsdk.FieldTraceID, traceID).
            Str("tag", tagName).
            Int64("value", v).
            Msg("negative value, defaulting to 0")
        return 0
    }
    return v
}

// validateNonNegativeFloat64 validates and parses a float64 tag value.
func (p *AWSPublicPlugin) validateNonNegativeFloat64(traceID, tagName, value string) float64 {
    v, err := strconv.ParseFloat(value, 64)
    if err != nil {
        p.logger.Warn().
            Str(pluginsdk.FieldTraceID, traceID).
            Str("tag", tagName).
            Str("value", value).
            Msg("invalid float value, defaulting to 0")
        return 0
    }
    if v < 0 {
        p.logger.Warn().
            Str(pluginsdk.FieldTraceID, traceID).
            Str("tag", tagName).
            Float64("value", v).
            Msg("negative value, defaulting to 0")
        return 0
    }
    return v
}
```

#### 2. Modified Pricing Lookup Pattern

```go
// Before (ignores ok):
rcuPrice, _ := p.pricing.DynamoDBProvisionedRCUPrice()

// After (checks ok, logs warning):
rcuPrice, rcuFound := p.pricing.DynamoDBProvisionedRCUPrice()
if !rcuFound {
    p.logger.Warn().
        Str(pluginsdk.FieldTraceID, traceID).
        Str("component", "RCU").
        Msg("DynamoDB provisioned RCU pricing unavailable")
}
```

#### 3. Billing Detail Pattern

```go
var unavailable []string
if !rcuFound {
    unavailable = append(unavailable, "RCU")
}
// ... other checks ...

billingDetail := fmt.Sprintf("DynamoDB provisioned, %d RCUs, %d WCUs, 730 hrs/month, %.0fGB storage",
    readUnits, writeUnits, storageGB)

if len(unavailable) > 0 {
    billingDetail += fmt.Sprintf(" (pricing unavailable: %s)", strings.Join(unavailable, ", "))
}
```

### Quickstart

See [quickstart.md](./quickstart.md) for implementation steps.

## Files to Modify

| File | Type | Changes |
|------|------|---------|
| `internal/plugin/projected.go` | MODIFY | Add validation helpers, modify `estimateDynamoDB()` |
| `internal/plugin/projected_test.go` | MODIFY | Add table-driven tests for validation scenarios |
| `internal/plugin/integration_dynamodb_test.go` | NEW | Integration tests for provisioned/on-demand modes |
| `internal/pricing/client_test.go` | MODIFY | Add pricing extraction tests for DynamoDB |

## Next Steps

Run `/speckit.tasks` to generate the task breakdown for implementation.
