# Data Model: Lambda Function Cost Estimation

**Date**: 2025-12-07
**Feature**: 001-lambda-cost-estimation

## Overview

The Lambda cost estimation feature extends the existing plugin data model with Lambda-specific pricing structures and cost calculation logic. The implementation follows the established pattern of embedded pricing data and stateless cost estimation.

## Core Entities

### Lambda Function

**Purpose**: Represents a serverless compute resource with memory allocation, execution patterns, and regional pricing.

**Attributes**:
- `memorySize`: Integer (MB) - Memory allocation for the function (default: 128MB)
- `requestsPerMonth`: Integer - Expected monthly request volume
- `avgDurationMs`: Integer - Average execution time in milliseconds
- `region`: String - AWS region code (e.g., "us-east-1")

**Relationships**:
- Uses → Pricing Data (for cost calculation)
- Belongs to → ResourceDescriptor (input container)

**Validation Rules**:
- `memorySize`: Must be positive integer, defaults to 128MB if invalid
- `requestsPerMonth`: Must be non-negative integer, required for cost calculation
- `avgDurationMs`: Must be non-negative integer, required for cost calculation
- `region`: Must match supported pricing regions

**State Transitions**: N/A (stateless cost estimation)

### Pricing Data

**Purpose**: Contains AWS Lambda pricing information for cost calculations.

**Attributes**:
- `requestPrice`: Float64 - Price per request ($/request, typically $0.20/million = $0.0000002)
- `gbSecondPrice`: Float64 - Price per GB-second of compute time
- `currency`: String - Always "USD" for AWS pricing
- `region`: String - AWS region this pricing applies to

**Relationships**:
- Used by → Lambda Function (cost calculation)
- Managed by → Pricing Client (loading and indexing)

**Validation Rules**:
- All price fields must be positive non-zero values
- Currency must be "USD"
- Region must be valid AWS region code

## Data Flow

### Input Processing
```
ResourceDescriptor (gRPC input)
├── Provider: "aws"
├── ResourceType: "lambda"
├── Sku: "512" (memory in MB)
├── Region: "us-east-1"
└── Tags:
    ├── "requests_per_month": "1000000"
    └── "avg_duration_ms": "200"
```

### Cost Calculation
```
Lambda Function Entity
├── Extract memorySize from Sku (default 128MB)
├── Extract requestsPerMonth from tags (required)
├── Extract avgDurationMs from tags (required)
└── Validate region matches pricing data

Pricing Data Lookup
├── Get requestPrice for region
├── Get gbSecondPrice for region
└── Validate pricing data available

Cost Computation
├── gbSeconds = (memorySize/1024) × (avgDurationMs/1000) × requestsPerMonth
├── requestCost = requestsPerMonth × requestPrice
├── durationCost = gbSeconds × gbSecondPrice
└── totalCost = requestCost + durationCost
```

### Output Generation
```
GetProjectedCostResponse
├── UnitPrice: gbSecondPrice (primary pricing dimension)
├── Currency: "USD"
├── CostPerMonth: totalCost
└── BillingDetail: "Lambda {memorySize}MB, {requestsPerMonth} requests/month, {avgDurationMs}ms avg duration, {gbSeconds} GB-seconds"
```

## Error Handling

### Input Validation Errors
- **Invalid memory size**: Default to 128MB, log warning
- **Missing request count**: Return $0 cost with explanatory billing detail
- **Missing duration**: Return $0 cost with explanatory billing detail
- **Invalid numeric values**: Return error (per clarification requirements)

### Pricing Data Errors
- **Region not supported**: Return error via gRPC status
- **Pricing data unavailable**: Return (0, false) following existing patterns
- **Pricing data corrupted**: Return error via gRPC status

## Performance Considerations

### Memory Usage
- Pricing data: <50MB per region binary (embedded JSON)
- Per-request overhead: Minimal (stateless calculations)
- Concurrent access: Thread-safe via existing pricing client patterns

### Latency Targets
- Pricing lookup: <50ms (with performance monitoring)
- Cost calculation: <1ms
- Total RPC: <100ms

## Testing Data Model

### Unit Test Scenarios
- Valid Lambda function with all required inputs
- Missing request count tag
- Missing duration tag
- Invalid memory size (non-numeric, negative)
- Invalid request count (negative)
- Invalid duration (negative)
- Unsupported region

### Integration Test Scenarios
- End-to-end cost calculation with mock pricing client
- Concurrent access to pricing data
- Memory usage validation
- Performance benchmark against latency targets

## Future Extensions

### Potential Enhancements (Out of Scope)
- Provisioned concurrency pricing
- ARM/x86 architecture differentiation
- Ephemeral storage pricing
- Lambda@Edge pricing

### Data Model Impact
- Additional pricing dimensions would require new attributes
- Architecture-specific pricing would require architecture field
- Enhanced storage would require storage size field</content>
<parameter name="filePath">/mnt/c/GitHub/go/src/github.com/rshade/pulumicost-plugin-aws-public/specs/001-lambda-cost-estimation/data-model.md