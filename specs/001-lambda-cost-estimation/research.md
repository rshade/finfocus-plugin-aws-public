# Research Findings: Lambda Function Cost Estimation

**Date**: 2025-12-07
**Feature**: 001-lambda-cost-estimation

## Executive Summary

Research confirms the Lambda cost estimation approach outlined in the specification is technically sound and follows AWS pricing model accurately. No major unknowns or blockers identified. Implementation can proceed with the defined technical approach.

## Research Tasks Completed

### AWS Lambda Pricing Model Validation

**Decision**: Use request count + GB-seconds pricing model
**Rationale**: Matches AWS Lambda's actual billing dimensions. Request pricing is per million requests, duration pricing is per GB-second of compute time.
**Alternatives Considered**:
- Provisioned concurrency pricing: Rejected as out of scope per specification
- ARM vs x86 pricing differences: Rejected as out of scope per specification
- Free tier automatic deduction: Rejected as complex and out of scope

### Pricing Data Source Analysis

**Decision**: Use AWS Public Pricing API with embedded JSON approach
**Rationale**: Consistent with existing plugin architecture. Eliminates runtime dependencies and ensures predictable performance.
**Alternatives Considered**:
- Runtime API calls: Rejected due to network dependency and performance impact
- Hardcoded pricing constants: Rejected as they become stale and region-specific

### Implementation Pattern Consistency

**Decision**: Follow existing EC2 pricing pattern for Lambda implementation
**Rationale**: Maintains code consistency and leverages proven architecture. The spec explicitly references following the EC2 pattern.
**Alternatives Considered**:
- Custom Lambda-specific architecture: Rejected as unnecessary complexity
- Separate Lambda pricing service: Rejected as over-engineering for single resource type

### Input Validation Strategy

**Decision**: Return errors for invalid inputs rather than silent defaults
**Rationale**: Provides clear feedback to users about missing or malformed data. Aligns with defensive programming principles.
**Alternatives Considered**:
- Silent defaults (treat invalid as 0): Rejected as could hide configuration errors
- Use absolute values for negatives: Rejected as mathematically incorrect

### Error Handling Patterns

**Decision**: Use existing pricing client error patterns (return (0, false) for unavailable data)
**Rationale**: Maintains consistency with existing codebase. Pricing lookups already handle missing data gracefully.
**Alternatives Considered**:
- Custom Lambda error handling: Rejected as unnecessary divergence
- Fail entire request on pricing unavailability: Rejected as too brittle for local plugin

## Technical Specifications Confirmed

### Lambda Pricing Dimensions
- **Requests**: $0.20 per million requests (varies by region)
- **Duration**: $0.0000166667 per GB-second (varies by region)
- **Free Tier**: 1M requests + 400K GB-seconds per month

### Cost Calculation Formula
```
GB-seconds = (memoryMB / 1024) × (durationMs / 1000) × requestCount
RequestCost = requestCount × requestPricePerMillion / 1,000,000
DurationCost = gbSeconds × pricePerGBSecond
TotalCost = RequestCost + DurationCost
```

### Input Requirements
- Memory size from `resource.Sku` (default 128MB)
- Request count from `tags["requests_per_month"]` (required)
- Duration from `tags["avg_duration_ms"]` (required)

## Risk Assessment

**Low Risk**: Implementation follows established patterns and well-understood AWS pricing model.
**Medium Risk**: Input validation strategy may need refinement based on user feedback.
**No High Risks**: All major technical decisions are validated and consistent with architecture.

## Recommendations

1. Proceed with implementation using the specified technical approach
2. Monitor input validation behavior in testing - may need adjustment based on real usage patterns
3. Consider adding more detailed logging for cost calculation inputs/outputs for debugging
4. Validate pricing accuracy against AWS calculator for multiple scenarios

## Next Steps

Implementation can begin immediately. No additional research required before proceeding to Phase 1 design.</content>
<parameter name="filePath">/mnt/c/GitHub/go/src/github.com/rshade/pulumicost-plugin-aws-public/specs/001-lambda-cost-estimation/research.md