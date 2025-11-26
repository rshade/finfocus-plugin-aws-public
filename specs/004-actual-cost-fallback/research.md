# Research: Fallback GetActualCost Implementation

**Date**: 2025-11-25
**Feature**: 004-actual-cost-fallback

## Overview

This research document captures technical decisions for implementing
GetActualCost using existing GetProjectedCost logic with time-based
pro-rating.

## Research Items

### 1. Protobuf Timestamp Handling in Go (v0.3.0)

**Decision**: Use `timestamppb.Timestamp.AsTime()` for conversion to Go
`time.Time`.

**Rationale**: The proto timestamps in `GetActualCostRequest.Start` and
`GetActualCostRequest.End` are `google.protobuf.Timestamp` types. The Go
protobuf library provides `AsTime()` method that converts to native
`time.Time` with nanosecond precision.

**Alternatives considered**:

- Manual conversion via Seconds/Nanos fields: More error-prone, no benefit
- Third-party time libraries: Unnecessary complexity

**Implementation**:

```go
import "google.golang.org/protobuf/types/known/timestamppb"

startTime := req.Start.AsTime()
endTime := req.End.AsTime()
duration := endTime.Sub(startTime)
runtimeHours := duration.Hours()
```

### 2. Runtime Hours Calculation

**Decision**: Use `time.Duration.Hours()` for floating-point hour calculation.

**Rationale**: Go's `time.Duration` provides precise nanosecond-level
arithmetic. The `Hours()` method returns `float64` which preserves
fractional hours for accurate pro-rating.

**Alternatives considered**:

- Integer hour truncation: Loses precision for sub-hour granularity
- Manual division: Redundant when stdlib provides this

**Edge cases handled**:

- Same-day ranges: `duration.Hours()` handles fractional hours correctly
- Zero duration: Returns 0.0, which yields $0.00 cost
- Negative duration (start > end): Detected and rejected with error

### 3. Reusing GetProjectedCost Logic (v0.3.0)

**Decision**: Parse `ResourceId` as JSON-encoded ResourceDescriptor, then call
internal helper methods (`estimateEC2`, `estimateEBS`, `estimateStub`) to get
monthly cost, and apply pro-rating formula.

**Rationale**: The v0.3.0 proto uses `resource_id` (string) instead of a
direct `ResourceDescriptor`. We JSON-encode the ResourceDescriptor in the
`resource_id` field for the fallback use case. The existing methods handle:

- Region validation
- Resource type routing
- Pricing lookups
- Billing detail generation

**Alternatives considered**:

- Calling `GetProjectedCost` method directly: Works but creates unnecessary
  proto marshaling overhead
- Duplicating pricing lookup logic: Violates DRY principle
- Using Tags map for resource info: Less structured, harder to validate

**Implementation approach**:

```go
// Parse ResourceId as JSON-encoded ResourceDescriptor
resource, err := p.parseResourceFromRequest(req)
if err != nil {
    return nil, err
}

// Get projected response using existing helper
projectedResp, err := p.getProjectedForResource(ctx, resource)
if err != nil {
    return nil, err
}

// Apply fallback formula
actualCost := projectedResp.CostPerMonth * (runtimeHours / hoursPerMonth)
```

### 4. Error Code Usage (v0.3.0)

**Decision**: Use existing proto ErrorCode enum values via gRPC status.

**Rationale**: Constitution III mandates proto-defined error codes only.

**Error mappings**:

| Condition            | ErrorCode             | gRPC Code       |
| -------------------- | --------------------- | --------------- |
| Invalid ResourceId   | INVALID_RESOURCE      | InvalidArgument |
| Region mismatch      | UNSUPPORTED_REGION    | FailedPrecond.  |
| Invalid range (s>e)  | INVALID_RESOURCE      | InvalidArgument |
| Nil Start/End        | INVALID_RESOURCE      | InvalidArgument |

### 5. Billing Detail Format (v0.3.0)

**Decision**: Include fallback calculation explanation in `ActualCostResult.Source`.

**Format**: `Fallback estimate: $X.XX/month × Y.YY hours / 730 = $Z.ZZ`

**Rationale**: The v0.3.0 proto uses `ActualCostResult.Source` field for
metadata. Users need to understand this is a pro-rated estimate, not actual
AWS billing data. Including the formula makes the calculation transparent
and debuggable.

**Example outputs** (in `source` field):

- EC2: `Fallback estimate: On-demand Linux, Shared × 24.00 hours / 730`
- EBS: `Fallback estimate: gp3 volume, 100 GB × 168.00 hours / 730`
- Stub: `S3 cost estimation not implemented - $0.00 for any duration`
- Zero duration: `aws-public-fallback`

### 6. Thread Safety

**Decision**: No additional synchronization needed.

**Rationale**: The existing pricing client is thread-safe (uses `sync.Once`
for initialization). The new GetActualCost method:

- Only reads from pricing client (no writes)
- Uses local variables for calculation
- Creates new response objects per call

No shared mutable state is introduced.

## Summary

All research items resolved. No NEEDS CLARIFICATION markers remain.
Implementation can proceed with:

1. Use `timestamppb.Timestamp.AsTime()` for time conversion
2. Use `time.Duration.Hours()` for runtime calculation
3. Reuse internal helper methods for pricing lookups
4. Apply formula: `cost = monthly_cost × (runtime_hours / 730)`
5. Return descriptive billing_detail with calculation basis
