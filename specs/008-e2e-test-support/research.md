# Research: E2E Test Support and Validation

**Feature**: 001-e2e-test-support
**Date**: 2025-12-02

## Research Topics

### 1. Test Mode Detection via Environment Variable

**Decision**: Use `os.Getenv("FINFOCUS_TEST_MODE")` with strict "true" check

**Rationale**:

- Simple, standard Go pattern for env var detection
- Strict "true" check prevents accidental enabling (e.g., "1", "yes", empty)
- Invalid values treated as disabled with startup warning (per clarification)
- Zero runtime overhead when disabled (single check at startup)

**Alternatives Considered**:

- Boolean parsing with `strconv.ParseBool`: Rejected - too permissive
  ("1", "t", "T" would enable test mode)
- Multiple valid values ("true", "1", "yes"): Rejected - increases risk of
  accidental enabling in production

**Implementation Pattern**:

```go
func IsTestMode() bool {
    return os.Getenv("FINFOCUS_TEST_MODE") == "true"
}

func ValidateTestModeEnv(logger zerolog.Logger) {
    val := os.Getenv("FINFOCUS_TEST_MODE")
    if val != "" && val != "true" && val != "false" {
        logger.Warn().
            Str("value", val).
            Msg("Invalid FINFOCUS_TEST_MODE value; treating as disabled")
    }
}
```

### 2. Enhanced Logging Strategy

**Decision**: Use existing zerolog logger with conditional debug-level logging

**Rationale**:

- Zerolog already integrated (feature 005-zerolog-logging completed)
- Debug level logs suppressed in production unless LOG_LEVEL=debug
- Test mode enables additional context without changing log levels
- Zero overhead when test mode disabled (log calls short-circuit)

**Alternatives Considered**:

- Separate test logger: Rejected - adds complexity, violates KISS
- Always-on verbose logging: Rejected - impacts production performance
- Custom log sink for tests: Rejected - over-engineering

**Implementation Pattern**:

```go
func (p *AWSPublicPlugin) GetProjectedCost(
    ctx context.Context,
    req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
    // Standard logging (always)
    p.logger.Info().Str("trace_id", traceID).Msg("GetProjectedCost called")

    // Enhanced logging (test mode only)
    if p.testMode {
        p.logger.Debug().
            Str("resource_type", req.Resource.ResourceType).
            Str("sku", req.Resource.Sku).
            Str("region", req.Resource.Region).
            Msg("Test mode: request details")
    }

    // ... calculation ...

    if p.testMode {
        p.logger.Debug().
            Float64("unit_price", resp.UnitPrice).
            Float64("cost_per_month", resp.CostPerMonth).
            Msg("Test mode: calculation result")
    }

    return resp, nil
}
```

### 3. Expected Cost Range Data Structure

**Decision**: Static Go constants with struct for range definition

**Rationale**:

- Expected costs are static reference values, not dynamic data
- Constants enable compile-time validation
- No external file loading or parsing needed
- Easy to update during pricing data refresh

**Alternatives Considered**:

- JSON file with expected ranges: Rejected - adds file I/O, parsing overhead
- New gRPC endpoint (GetExpectedCostRange): Rejected - violates KISS, adds
  protocol surface, User Story 4 is P4 priority
- Proto-embedded metadata: Rejected - requires finfocus-spec changes

**Data Structure**:

```go
// ExpectedCostRange defines expected cost values with tolerance for validation
type ExpectedCostRange struct {
    ResourceType    string  // "ec2" or "ebs"
    SKU             string  // "t3.micro" or "gp2"
    Region          string  // "us-east-1"
    UnitPrice       float64 // Hourly rate (EC2) or GB-month rate (EBS)
    MonthlyEstimate float64 // Expected monthly cost
    TolerancePercent float64 // Allowed deviation (1% for EC2, 5% for EBS)
    ReferenceDate   string  // When pricing was captured (e.g., "2025-12-01")
}

// ExpectedCostRanges contains all documented test resource expectations
var ExpectedCostRanges = map[string]ExpectedCostRange{
    "ec2:t3.micro:us-east-1": {
        ResourceType:     "ec2",
        SKU:              "t3.micro",
        Region:           "us-east-1",
        UnitPrice:        0.0104,
        MonthlyEstimate:  7.592, // 0.0104 * 730
        TolerancePercent: 1.0,
        ReferenceDate:    "2025-12-01",
    },
    "ebs:gp2:us-east-1": {
        ResourceType:     "ebs",
        SKU:              "gp2",
        Region:           "us-east-1",
        UnitPrice:        0.10,    // per GB-month
        MonthlyEstimate:  0.80,    // 8 GB default
        TolerancePercent: 5.0,
        ReferenceDate:    "2025-12-01",
    },
}
```

### 4. Integration with Existing Plugin Architecture

**Decision**: Add testMode field to AWSPublicPlugin struct, set at construction

**Rationale**:

- Plugin struct already exists with logger field
- Single initialization point in NewAWSPublicPlugin()
- Consistent with existing pattern for plugin configuration
- Thread-safe (field is read-only after construction)

**Alternatives Considered**:

- Global variable for test mode: Rejected - makes testing harder, not thread-safe
- Check env var on every call: Rejected - unnecessary overhead
- Separate TestAWSPublicPlugin type: Rejected - over-engineering

**Implementation Pattern**:

```go
type AWSPublicPlugin struct {
    logger   zerolog.Logger
    testMode bool  // NEW: Set from FINFOCUS_TEST_MODE at construction
    // ... existing fields
}

func NewAWSPublicPlugin(logger zerolog.Logger) *AWSPublicPlugin {
    testMode := os.Getenv("FINFOCUS_TEST_MODE") == "true"

    // Log test mode status at startup
    if testMode {
        logger.Info().Msg("Test mode enabled")
    }

    return &AWSPublicPlugin{
        logger:   logger,
        testMode: testMode,
    }
}
```

### 5. Backward Compatibility Verification

**Decision**: All changes are additive; no breaking changes to existing behavior

**Verification Checklist**:

- [ ] GetProjectedCost returns same values when test mode disabled
- [ ] GetActualCost returns same values when test mode disabled
- [ ] Supports returns same values when test mode disabled
- [ ] No new required environment variables
- [ ] No new gRPC methods required by callers
- [ ] Existing tests pass without modification
- [ ] Performance unchanged when test mode disabled

**Implementation Note**: Add regression test that runs same cost calculations
with and without test mode, asserting identical responses.

## Summary

All research topics resolved. No external dependencies or new technologies
required. Implementation follows existing patterns established in the codebase.
Ready to proceed to Phase 1 design.
