# Data Model: E2E Test Support and Validation

**Feature**: 001-e2e-test-support
**Date**: 2025-12-02

## Entities

### 1. ExpectedCostRange

Represents documented expected cost values for standard test resources used in
E2E validation.

| Field | Type | Description | Constraints |
|-------|------|-------------|-------------|
| ResourceType | string | Resource category | "ec2" or "ebs" |
| SKU | string | Resource SKU | e.g., "t3.micro", "gp2" |
| Region | string | AWS region | e.g., "us-east-1" |
| UnitPrice | float64 | Per-unit price | > 0 |
| MonthlyEstimate | float64 | Expected monthly cost | >= 0 |
| TolerancePercent | float64 | Allowed deviation | 1.0 (EC2), 5.0 (EBS) |
| ReferenceDate | string | Pricing reference date | ISO date format |

**Uniqueness**: Composite key of (ResourceType, SKU, Region)

**Validation Rules**:

- UnitPrice must be positive
- TolerancePercent must be between 0 and 100
- ReferenceDate must be valid ISO date

### 2. TestModeContext (Implicit)

Not a persisted entity but a runtime context affecting plugin behavior.

| Field | Type | Description | Source |
|-------|------|-------------|--------|
| Enabled | bool | Test mode active | FINFOCUS_TEST_MODE env var |
| WarningLogged | bool | Invalid value warning emitted | Runtime state |

**State Transitions**:

- Initialized at plugin startup
- Immutable after initialization
- No runtime state changes

## Relationships

```text
┌─────────────────────┐
│  AWSPublicPlugin    │
├─────────────────────┤
│ - logger            │
│ - testMode: bool    │◄──── Set from FINFOCUS_TEST_MODE
│ - region            │
└─────────┬───────────┘
          │
          │ references (read-only)
          ▼
┌─────────────────────┐
│ ExpectedCostRanges  │
│ (static map)        │
├─────────────────────┤
│ ec2:t3.micro:use1   │──┐
│ ebs:gp2:use1        │──┼─► ExpectedCostRange
│ ...                 │──┘
└─────────────────────┘
```

## Data Flow

### Test Mode Initialization

```text
Startup:
  1. Read FINFOCUS_TEST_MODE env var
  2. Validate value ("true" = enabled, "false" = disabled, other = warning + disabled)
  3. Store testMode bool in plugin struct
  4. Log test mode status

Runtime:
  1. Check p.testMode before enhanced logging
  2. If testMode: emit debug-level calculation details
  3. If not testMode: standard production logging only
```

### Expected Cost Range Lookup

```text
GetExpectedRange(resourceType, sku, region):
  1. Build key: "{resourceType}:{sku}:{region}"
  2. Lookup in ExpectedCostRanges map
  3. Return (range, found) tuple
  4. Caller validates actual cost against range.MonthlyEstimate ± tolerance
```

## Validation Functions

### IsWithinTolerance

```go
func IsWithinTolerance(actual, expected, tolerancePercent float64) bool {
    if expected == 0 {
        return actual == 0
    }
    deviation := math.Abs(actual-expected) / expected * 100
    return deviation <= tolerancePercent
}
```

### Example Validation

```go
range := ExpectedCostRanges["ec2:t3.micro:us-east-1"]
actual := 7.65 // from GetProjectedCost response

if !IsWithinTolerance(actual, range.MonthlyEstimate, range.TolerancePercent) {
    // Test failure: cost outside expected range
}
```

## No Persistent Storage

This feature does not introduce any persistent storage. All data is:

- **ExpectedCostRanges**: Compile-time constants in Go source
- **TestModeContext**: Runtime-only, derived from environment variable

No database, files, or external storage required.
