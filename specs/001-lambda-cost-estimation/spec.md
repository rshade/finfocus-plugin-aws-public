# Feature Specification: Lambda Function Cost Estimation

**Feature Branch**: `001-lambda-cost-estimation`  
**Created**: Sun Dec 07 2025  
**Status**: Draft  
**Input**: User description: "title:	feat(lambda): implement Lambda function cost estimation
state:	OPEN
author:	rshade
labels:	aws-service, enhancement, priority: high
comments:	0
assignees:	
projects:	
milestone:	
number:	53
--
## Overview

Implement Lambda function cost estimation using AWS Public Pricing data. Lambda is currently a stub service returning $0 and needs pricing support for request count and GB-seconds (compute duration).

## Clarifications

### Session 2025-12-07
- Q: What are the expected maximum request volumes and memory sizes to support? → A: This is a local plugin
- Q: How should the system handle AWS Pricing API unavailability? → A: Use existing techniques
- Q: What is the target throughput for concurrent Lambda cost estimation requests? → A: B
- Q: How should invalid or negative values in tags be handled? → A: B
- Q: What are the concurrency and memory limits for the pricing client? → A: B

## User Story

As a cloud cost analyst,
I want accurate Lambda function cost estimates based on request volume and compute duration,
So that I can understand the serverless compute component of my AWS spending.

## Problem Statement

Lambda is a core serverless offering used extensively across AWS environments. The current stub implementation returns $0. Lambda has a straightforward pricing model (requests + GB-seconds) that can be estimated from expected usage patterns provided via resource tags.

## Proposed Solution

Implement Lambda pricing using request count and GB-seconds dimensions.

### Pricing Dimensions

1. **Request count** - price per million requests
2. **GB-seconds** - price per GB-second of compute time

### Technical Approach

1. **Extend Pricing Client Interface** (`internal/pricing/client.go:13-27`)
   \`\`\`go
   type PricingClient interface {
       // ... existing methods ...
       LambdaPricePerRequest() (float64, bool)           // $/request
       LambdaPricePerGBSecond() (float64, bool)          // $/GB-second
   }
   \`\`\`

2. **Add Lambda Price Types** (`internal/pricing/types.go`)
   \`\`\`go
   type lambdaPrice struct {
       RequestPrice   float64  // $/request (typically $0.20/million = $0.0000002)
       GBSecondPrice  float64  // $/GB-second (typically $0.0000166667)
       Currency       string
   }
   \`\`\`

3. **Update Client Struct** (`internal/pricing/client.go:29-41`)
   \`\`\`go
   type Client struct {
       // ... existing fields ...
       lambdaPricing *lambdaPrice  // Single pricing struct, no index needed
   }
   \`\`\`

4. **Update Pricing Initialization** (`internal/pricing/client.go:52-166`)
   - Filter by \`ProductFamily == "Serverless"\` or \`servicecode == "AWSLambda"\`
   - Extract request pricing (\`unit == "Requests"\`)
   - Extract duration pricing (\`unit == "Second"\` or \`unit == "Lambda-GB-Second"\`)

5. **Create Estimator Function** (`internal/plugin/projected.go`)
   \`\`\`go
   // estimateLambda calculates projected monthly cost for Lambda functions.
   // Uses request count and GB-seconds from resource tags.
   func (p *AWSPublicPlugin) estimateLambda(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
       // resource.Sku = memory size in MB (e.g., "128", "512", "1024")
       // resource.Tags["requests_per_month"] = expected monthly requests
       // resource.Tags["avg_duration_ms"] = average execution time in ms
   }
   \`\`\`

6. **Update Router** (`internal/plugin/projected.go:73`)
   - Change \`case "lambda":\` to call \`p.estimateLambda(traceID, resource)\`

7. **Update Supports()** (`internal/plugin/supports.go:85-99`)
   - Move "lambda" from stub case to fully-supported case

8. **Extend Generate-Pricing Tool** (`tools/generate-pricing/main.go`)
   - Add \`AWSLambda\` service code support

### Files Likely Affected

- \`internal/pricing/client.go\` - Add Lambda interface methods, pricing struct, init logic
- \`internal/pricing/types.go\` - Add \`lambdaPrice\` struct
- \`internal/plugin/projected.go\` - Add \`estimateLambda()\` function, update router
- \`internal/plugin/supports.go\` - Move Lambda to fully-supported
- \`internal/plugin/plugin_test.go\` - Update mock pricing client
- \`internal/plugin/projected_test.go\` - Add Lambda test cases
- \`tools/generate-pricing/main.go\` - Add Lambda pricing fetch support

### ResourceDescriptor Mapping

\`\`\`go
// Input
ResourceDescriptor{
    Provider:     "aws",
    ResourceType: "lambda",
    Sku:          "512",               // Memory in MB
    Region:       "us-east-1",
    Tags: map[string]string{
        "requests_per_month": "1000000",    // 1M requests
        "avg_duration_ms":    "200",        // 200ms average
    },
}

// Output (calculation example)
// GB-seconds = (512MB / 1024) * (200ms / 1000) * 1,000,000 requests = 100,000 GB-seconds
// Request cost = 1,000,000 * $0.0000002 = $0.20
// Duration cost = 100,000 * $0.0000166667 = $1.67
// Total = $1.87

GetProjectedCostResponse{
    UnitPrice:     0.0000166667,       // $/GB-second (primary unit)
    Currency:      "USD",
    CostPerMonth:  1.87,
    BillingDetail: "Lambda 512MB, 1M requests/month, 200ms avg duration, 100K GB-seconds",
}
\`\`\`

## Acceptance Criteria

- [ ] \`Supports()\` returns \`supported=true\` without "Limited support" reason for Lambda
- [ ] \`GetProjectedCost()\` calculates cost from requests and GB-seconds
- [ ] Memory size extracted from \`resource.Sku\` (default 128MB if missing)
- [ ] Requests extracted from \`tags["requests_per_month"]\` (default 0 if missing)
- [ ] Duration extracted from \`tags["avg_duration_ms"]\` (default 100ms if missing)
- [ ] Missing request/duration tags return $0 with explanatory \`BillingDetail\`
- [ ] Free tier consideration noted in billing detail (1M requests, 400K GB-seconds)
- [ ] All 9 regional binaries include Lambda pricing data
- [ ] Thread-safe pricing lookups
- [ ] Unit tests cover various memory/request/duration combinations

## Out of Scope

- Provisioned concurrency pricing
- Lambda@Edge pricing (different rates)
- Lambda extensions pricing
- Free tier automatic deduction
- ARM vs x86 architecture pricing differences
- Ephemeral storage beyond 512MB

## Technical Notes

### AWS Pricing API

Lambda pricing is available at:
\`\`\`
https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AWSLambda/current/<region>/index.json
\`\`\`

### Product Family Filters

Lambda uses multiple product families:
- \`"Serverless"\` - For compute duration
- \`"AWS Lambda"\` - For requests

Key attributes:
- \`group\`: "AWS-Lambda-Requests" vs "AWS-Lambda-Duration"
- \`usagetype\`: Contains region prefix and usage type

### Cost Calculation

\`\`\`go
// Convert memory to GB
memoryGB := float64(memoryMB) / 1024.0

// Convert duration to seconds
durationSeconds := float64(durationMs) / 1000.0

// Calculate GB-seconds
gbSeconds := memoryGB * durationSeconds * requestCount

// Calculate costs
requestCost := requestCount * requestPricePerRequest
durationCost := gbSeconds * pricePerGBSecond
totalMonthlyCost := requestCost + durationCost
\`\`\`

### Lambda Pricing Constants (us-east-1)

\`\`\`go
// These should come from pricing data, not hardcoded
requestPrice := 0.20 / 1_000_000  // $0.20 per million = $0.0000002
gbSecondPrice := 0.0000166667     // Per GB-second
\`\`\`

### Logging Requirements

\`\`\`go
p.logger.Debug().
    Str(pluginsdk.FieldTraceID, traceID).
    Str(pluginsdk.FieldOperation, "GetProjectedCost").
    Int("memory_mb", memoryMB).
    Int64("requests", requestCount).
    Int("duration_ms", durationMs).
    Float64("gb_seconds", gbSeconds).
    Str("aws_region", p.region).
    Msg("Lambda pricing lookup successful")
\`\`\`

## Testing Strategy

- [ ] Unit tests for \`estimateLambda()\` with mock pricing client
- [ ] Unit tests for \`LambdaPricePerRequest()\` and \`LambdaPricePerGBSecond()\`
- [ ] Table-driven tests for memory sizes (128, 256, 512, 1024, 2048, 3008, 10240)
- [ ] Test with various request volumes (1K, 1M, 100M)
- [ ] Test with various durations (100ms, 500ms, 1s, 5s, 15min max)
- [ ] Test default values when tags missing
- [ ] Concurrent access test

## Related

- Current stub implementation: \`internal/plugin/projected.go:73-74\`, \`internal/plugin/projected.go:255-264\`
- EC2 pattern to follow: \`internal/plugin/projected.go:105-180\`
- AWS Lambda Pricing: https://aws.amazon.com/lambda/pricing/"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Accurate Lambda Cost Estimates (Priority: P1)

As a cloud cost analyst, I want accurate Lambda function cost estimates based on request volume and compute duration, so that I can understand the serverless compute component of my AWS spending.

**Why this priority**: This is the core functionality required to provide value to cloud cost analysts using the plugin.

**Independent Test**: Can be fully tested by providing Lambda resource descriptors with memory, request count, and duration tags, and verifying that the returned cost matches expected calculations based on AWS pricing data.

**Acceptance Scenarios**:

1. **Given** a Lambda resource with memory size, request count, and duration tags, **When** GetProjectedCost is called, **Then** it returns accurate monthly cost based on GB-seconds and request pricing
2. **Given** a Lambda resource missing request tags, **When** GetProjectedCost is called, **Then** it returns $0 cost with explanatory billing detail
3. **Given** a Lambda resource missing duration tags, **When** GetProjectedCost is called, **Then** it returns $0 cost with explanatory billing detail
4. **Given** a Lambda resource with invalid memory size, **When** GetProjectedCost is called, **Then** it defaults to 128MB and calculates accordingly
5. **Given** Supports is called for Lambda resource type, **When** the call completes, **Then** it returns supported=true without "Limited support" reason

---

### Edge Cases

- What happens when memory size is invalid or zero? (Should default to 128MB)
- How does system handle negative request counts or durations? (Return error for invalid input)
- What if pricing data is unavailable for the region? (Use existing techniques - return (0, false) for missing pricing)
- How does system handle very large request volumes (100M+)? (Should calculate accurately without overflow)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST extract memory size from resource.Sku (default 128MB if missing or invalid)
- **FR-002**: System MUST extract request count from tags["requests_per_month"] (default 0 if missing)
- **FR-003**: System MUST extract duration from tags["avg_duration_ms"] (default 100ms if missing)
- **FR-004**: System MUST calculate GB-seconds as (memoryGB * durationSeconds * requestCount)
- **FR-005**: System MUST calculate total cost as (requestCost + durationCost)
- **FR-006**: System MUST return $0 cost with explanatory BillingDetail when required tags are missing
- **FR-007**: System MUST include free tier information in billing detail (1M requests, 400K GB-seconds)
- **FR-008**: System MUST support Lambda resource type in Supports() method returning fully supported
- **FR-009**: System MUST include Lambda pricing data in all 9 regional binaries
- **FR-010**: System MUST provide thread-safe pricing lookups for concurrent gRPC calls

### Key Entities *(include if feature involves data)*

- **Lambda Function**: Represents a serverless compute resource with memory allocation, execution patterns (requests and duration), and regional pricing. Scale limits depend on user system capabilities as this is a local plugin.
- **Pricing Data**: Contains request pricing per million requests and GB-second pricing for compute duration, sourced from AWS Public Pricing API

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: System calculates Lambda costs within 1% accuracy compared to manual AWS pricing calculations
- **SC-002**: GetProjectedCost responds in under 100ms for Lambda resources with valid inputs
- **SC-003**: Supports() returns true for Lambda resource type without "Limited support" message
- **SC-004**: All unit tests pass covering memory sizes (128MB to 10GB), request volumes (1K to 100M), and durations (100ms to 15min). Includes boundary tests for minimum values (128MB, 1 request, 1ms), maximum values (10GB, 100M requests, 15min), and edge cases (invalid inputs, negative values, zero values).
  - Boundary tests: 128MB memory + 1 request + 1ms duration, 10GB memory + 100M requests + 15min duration
  - Edge cases: Invalid memory ("invalid"), negative requests (-1000), zero duration (0ms)
  - Error handling: Missing required tags, malformed resource descriptors
- **SC-005**: System handles concurrent requests without data races or pricing lookup failures
- **SC-006**: System supports 100 concurrent Lambda cost estimation requests per second
- **SC-007**: Pricing client handles up to 1000 concurrent requests with memory usage under 100MB