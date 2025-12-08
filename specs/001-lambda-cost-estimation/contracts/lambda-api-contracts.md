# Lambda Cost Estimation API Contracts

**Date**: 2025-12-07
**Feature**: 001-lambda-cost-estimation

## Overview

This document specifies the API contracts for Lambda function cost estimation within the PulumiCost AWS Public plugin. The implementation extends the existing gRPC CostSourceService protocol with Lambda-specific cost calculation logic.

## Service Interface

### CostSourceService (Extended)

The Lambda feature extends the existing `CostSourceService` with enhanced support for Lambda resource type.

#### Supports Method (Enhanced)

**Endpoint**: `Supports(ResourceDescriptor) -> SupportsResponse`

**Purpose**: Determine if the plugin can provide cost estimates for the given resource.

**Lambda-Specific Behavior**:
- **Input Validation**: Accepts `ResourceDescriptor` with `resource_type: "lambda"`
- **Response**: Returns `supported: true` without "Limited support" reason
- **Error Conditions**:
  - Invalid resource type: Returns `supported: false, reason: "Unsupported resource type"`
  - Region mismatch: Returns `supported: false, reason: "Region not supported"`

**Contract**:
```protobuf
message ResourceDescriptor {
  string provider = 1;        // Must be "aws"
  string resource_type = 2;   // Must be "lambda"
  string sku = 3;             // Memory size in MB (e.g., "128", "512")
  string region = 4;          // AWS region (e.g., "us-east-1")
  map<string, string> tags = 5; // Contains Lambda configuration
}

message SupportsResponse {
  bool supported = 1;
  string reason = 2;  // Empty for Lambda (fully supported)
}
```

#### GetProjectedCost Method (Enhanced)

**Endpoint**: `GetProjectedCost(ResourceDescriptor) -> GetProjectedCostResponse`

**Purpose**: Calculate projected monthly cost for Lambda functions based on usage patterns.

**Lambda-Specific Behavior**:
- **Input Processing**:
  - Memory size from `resource.sku` (default 128MB if invalid)
  - Request count from `resource.tags["requests_per_month"]` (required)
  - Duration from `resource.tags["avg_duration_ms"]` (required)
- **Cost Calculation**:
  - GB-seconds = (memoryMB/1024) × (durationMs/1000) × requestCount
  - Total cost = (requestCount × requestPrice) + (gbSeconds × gbSecondPrice)
- **Error Handling**:
  - Missing required tags: Return $0 cost with explanatory billing detail
  - Invalid inputs: Return error (per clarification requirements)
  - Pricing unavailable: Return (0, false) following existing patterns

**Contract**:
```protobuf
message GetProjectedCostResponse {
  double unit_price = 1;      // $/GB-second (primary pricing unit)
  string currency = 2;        // Always "USD"
  double cost_per_month = 3;  // Total monthly cost
  string billing_detail = 4;  // Human-readable cost breakdown
}
```

**Example Request/Response**:

```json
// Input
{
  "provider": "aws",
  "resource_type": "lambda",
  "sku": "512",
  "region": "us-east-1",
  "tags": {
    "requests_per_month": "1000000",
    "avg_duration_ms": "200"
  }
}

// Output
{
  "unit_price": 0.0000166667,
  "currency": "USD",
  "cost_per_month": 1.87,
  "billing_detail": "Lambda 512MB, 1M requests/month, 200ms avg duration, 100K GB-seconds"
}
```

## Functional Requirements Mapping

### FR-001: Memory Size Extraction
- **Contract**: `resource.sku` contains memory size in MB
- **Validation**: Default to 128MB if missing or invalid
- **Error Response**: Continue with default, no error returned

### FR-002: Request Count Extraction
- **Contract**: `resource.tags["requests_per_month"]` contains monthly request count
- **Validation**: Must be non-negative integer
- **Error Response**: Return $0 cost with explanatory billing detail

### FR-003: Duration Extraction
- **Contract**: `resource.tags["avg_duration_ms"]` contains average execution time
- **Validation**: Must be non-negative integer
- **Error Response**: Return $0 cost with explanatory billing detail

### FR-004: GB-Seconds Calculation
- **Contract**: Formula: `(memoryGB × durationSeconds × requestCount)`
- **Precision**: Float64 arithmetic with standard IEEE 754 behavior
- **Range**: No enforced limits (system-dependent)

### FR-005: Total Cost Calculation
- **Contract**: Formula: `(requestCost + durationCost)`
- **Components**: Separate request and duration pricing
- **Currency**: Always USD

### FR-006: Missing Tags Handling
- **Contract**: Return $0 cost when required inputs missing
- **Billing Detail**: Must explain why cost is $0
- **Example**: `"Missing required tags: requests_per_month, avg_duration_ms"`

### FR-007: Free Tier Information
- **Contract**: Include free tier details in billing detail
- **Format**: `"Includes 1M requests and 400K GB-seconds free tier"`
- **Calculation**: Does not affect actual cost (display only)

### FR-008: Supports Method
- **Contract**: Return `supported: true` for Lambda resource type
- **Reason Field**: Must be empty (no "Limited support" message)

### FR-009: Regional Pricing Data
- **Contract**: All 9 regional binaries include Lambda pricing
- **Validation**: Build process ensures pricing data availability
- **Fallback**: Use existing error patterns if pricing unavailable

### FR-010: Thread Safety
- **Contract**: Concurrent gRPC calls supported
- **Implementation**: Pricing client uses thread-safe patterns
- **Testing**: Concurrent access validation required

## Error Conditions

### gRPC Status Codes
- `INVALID_ARGUMENT` (3): Malformed ResourceDescriptor or invalid input values
- `NOT_FOUND` (5): Unsupported region or resource type
- `INTERNAL` (13): Pricing data corruption or calculation errors
- `UNAVAILABLE` (14): Temporary pricing data unavailability

### Error Detail Messages
- **Invalid memory size**: `"Invalid memory size in sku: {value}, using default 128MB"`
- **Missing request count**: `"Missing required tag: requests_per_month"`
- **Missing duration**: `"Missing required tag: avg_duration_ms"`
- **Invalid numeric value**: `"Invalid numeric value in tag {key}: {value}"`

## Performance Contracts

### Latency Guarantees
- **Supports()**: <10ms per call
- **GetProjectedCost()**: <100ms per call (including pricing lookup)
- **Concurrent Load**: Support 100 simultaneous requests

### Resource Limits
- **Memory Usage**: <50MB per region binary
- **Binary Size**: <10MB per region binary
- **CPU Overhead**: Minimal (stateless calculations)

## Testing Contracts

### Unit Test Contracts
- **Cost Calculation Accuracy**: Within 1% of manual AWS calculations
- **Input Validation**: All error conditions properly handled
- **Memory Size Defaults**: Invalid values default to 128MB
- **Tag Parsing**: Correct extraction from tags map

### Integration Test Contracts
- **gRPC Communication**: Proper request/response serialization
- **Concurrent Access**: No race conditions or data corruption
- **Error Propagation**: gRPC status codes correctly returned
- **Performance**: Meet latency targets under load

## Backward Compatibility

### Existing Behavior Preservation
- **Non-Lambda resources**: Unchanged behavior
- **Existing pricing methods**: No modifications to EC2, EBS, S3, RDS pricing
- **gRPC protocol**: No changes to message formats or service interface

### Migration Path
- **Gradual rollout**: Lambda support added without breaking changes
- **Feature flag**: Could be controlled via build tags if needed
- **Monitoring**: New Lambda-specific metrics added to existing logging</content>
<parameter name="filePath">/mnt/c/GitHub/go/src/github.com/rshade/pulumicost-plugin-aws-public/specs/001-lambda-cost-estimation/contracts/lambda-api-contracts.md