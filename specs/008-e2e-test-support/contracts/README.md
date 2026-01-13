# Contracts: E2E Test Support and Validation

**Feature**: 001-e2e-test-support
**Date**: 2025-12-02

## No New API Contracts

This feature does not introduce new gRPC endpoints or API contracts.

**Rationale**:

- Reuses existing `GetProjectedCost`, `GetActualCost`, and `Supports` methods
- Expected cost ranges are compile-time constants (no query endpoint)
- Test mode is environment-variable driven (no runtime toggle API)
- Follows KISS principle from constitution

## Existing Contracts Used

The feature relies on existing proto-defined contracts from `finfocus-spec`:

### GetProjectedCost

```protobuf
rpc GetProjectedCost(GetProjectedCostRequest) returns (GetProjectedCostResponse);
```

Used for: Validating projected cost calculations

### GetActualCost

```protobuf
rpc GetActualCost(GetActualCostRequest) returns (GetActualCostResponse);
```

Used for: Validating fallback actual cost calculations

### Supports

```protobuf
rpc Supports(SupportsRequest) returns (SupportsResponse);
```

Used for: Verifying test resource compatibility

## Behavioral Changes (Non-Breaking)

When `FINFOCUS_TEST_MODE=true`:

1. **Enhanced logging**: Additional debug-level logs with calculation details
2. **Startup validation**: Warning logged for invalid env var values

These are additive behaviors with no changes to request/response formats.
