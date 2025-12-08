# Feature Specification: S3 Storage Cost Estimation

**Feature Branch**: `011-s3-cost-estimation`  
**Created**: 2025-12-07  
**Status**: Draft  
**Input**: User description: "title:	feat(s3): implement S3 storage cost estimation
state:	OPEN
author:	rshade
labels:	aws-service, enhancement, priority: high
comments:	0
assignees:	
projects:	
milestone:	
number:	51
--
## Overview

Implement real S3 storage cost estimation using AWS Public Pricing data. S3 is currently a stub service returning $0 and needs full pricing support for storage classes (Standard, IA, Glacier) and request-based charges.

## User Story

As a cloud cost analyst,
I want accurate S3 storage cost estimates based on storage class and bucket size,
So that I can understand the storage component of my AWS spending.

## Problem Statement

S3 is one of the most widely used AWS services. The current stub implementation returns $0, making cost projections incomplete. S3 has predictable per-GB-month pricing that maps well to the existing EBS pricing pattern, making it an ideal candidate for full implementation.

As a cloud cost analyst, I want accurate S3 storage cost estimates based on storage class and bucket size, so that I can understand the storage component of my AWS spending.

### ResourceDescriptor Mapping

```go
// Input
ResourceDescriptor{
    Provider:     "aws",
    ResourceType: "s3",
    Sku:          "STANDARD",        // Storage class
    Region:       "us-east-1",
    Tags: map[string]string{
        "size": "100",              // GB
    },
}

// Output
GetProjectedCostResponse{
    UnitPrice:     0.023,           // $/GB-month for Standard
    Currency:      "USD",
    CostPerMonth:  2.30,            // 100 GB * $0.023
    BillingDetail: "S3 Standard storage, 100 GB, $0.0230/GB-month",
}
```

## Acceptance Criteria

- [ ] `Supports()` returns `supported=true` without "Limited support" reason for S3
- [ ] `GetProjectedCost()` returns accurate per-GB-month rates for Standard storage class
- [ ] `GetProjectedCost()` returns accurate rates for Standard-IA storage class
- [ ] Size extracted from `tags["size"]` with default of 1 GB if missing
- [ ] Unknown storage classes return $0 with explanatory `BillingDetail`
- [ ] All 9 regional binaries include S3 pricing data
- [ ] Thread-safe pricing lookups (concurrent RPC test)
- [ ] Pricing lookup < 50ms (performance requirement from existing code)
- [ ] Unit tests cover all storage classes and edge cases
- [ ] Mock pricing client updated in test files
- [ ] `GetActualCost()` automatically inherits S3 support via router

## Out of Scope

- Request-based charges (PUT, GET, DELETE operations)
- Data transfer costs
- S3 Intelligent-Tiering automatic optimization
- Lifecycle policy cost modeling
- S3 Select and Glacier retrieval fees

## Technical Notes

### AWS Pricing API

S3 pricing is available at:
```
https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonS3/current/<region>/index.json
```

Filter criteria for storage:
- `ProductFamily == "Storage"`
- `servicecode == "AmazonS3"`
- Key attributes: `storageClass`, `volumeType`, `location`

### Storage Class Mapping

AWS API values → User-friendly SKU:
- `"Standard"` → `"STANDARD"`
- `"Standard - Infrequent Access"` → `"STANDARD_IA"`
- `"One Zone - Infrequent Access"` → `"ONEZONE_IA"`
- `"Glacier Flexible Retrieval"` → `"GLACIER"`
- `"Glacier Deep Archive"` → `"DEEP_ARCHIVE"`

### Thread Safety

Follow the existing `sync.Once` pattern for initialization:
```go
c.s3Index = make(map[string]s3Price)
// Build index during init()
```

### Logging Requirements

Debug log for pricing lookup:
```go
p.logger.Debug().
    Str(pluginsdk.FieldTraceID, traceID).
    Str(pluginsdk.FieldOperation, "GetProjectedCost").
    Str("storage_class", storageClass).
    Str("aws_region", p.region).
    Float64("unit_price", ratePerGBMonth).
    Msg("S3 pricing lookup successful")
```

## Testing Strategy

- [ ] Unit tests for `estimateS3()` with mock pricing client
- [ ] Unit tests for `S3PricePerGBMonth()` lookup method
- [ ] Table-driven tests for all storage classes
- [ ] Test default size behavior when tag missing
- [ ] Test unknown storage class returns $0
- [ ] Concurrent access test (multiple goroutines)
- [ ] Integration test with real embedded pricing data

## Related

- Current stub implementation: `internal/plugin/projected.go:73-74`, `internal/plugin/projected.go:255-264`
- EBS pattern to follow: `internal/plugin/projected.go:182-253`
- Pricing client pattern: `internal/pricing/client.go:180-223`
- AWS S3 Pricing: https://aws.amazon.com/s3/pricing/"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Estimate S3 Storage Costs (Priority: P1)

As a cloud cost analyst, I want to get accurate projected monthly costs for S3 storage based on storage class and bucket size, so that I can understand the storage component of my AWS spending.

**Why this priority**: This is the core functionality requested, enabling cost analysts to make informed decisions about S3 usage.

**Independent Test**: Can be fully tested by providing a ResourceDescriptor with S3 resource type, storage class SKU, and size tag, and verifying the returned cost estimate matches expected AWS pricing.

**Acceptance Scenarios**:

1. **Given** a ResourceDescriptor with resourceType="s3", Sku="STANDARD", region="us-east-1", and tags["size"]="100", **When** GetProjectedCost is called, **Then** returns UnitPrice=0.023, CostPerMonth=2.30, Currency="USD", BillingDetail="S3 Standard storage, 100 GB, $0.0230/GB-month"
2. **Given** a ResourceDescriptor with resourceType="s3", Sku="STANDARD_IA", region="us-east-1", and tags["size"]="100", **When** GetProjectedCost is called, **Then** returns accurate Standard-IA pricing for the region
3. **Given** a ResourceDescriptor with resourceType="s3", Sku="UNKNOWN", region="us-east-1", and tags["size"]="100", **When** GetProjectedCost is called, **Then** returns CostPerMonth=0.00 with explanatory BillingDetail
4. **Given** a ResourceDescriptor with resourceType="s3", Sku="STANDARD", region="us-east-1", and no size tag, **When** GetProjectedCost is called, **Then** uses sensible default size (e.g., 1 GB) and calculates cost accordingly

---

### Edge Cases

- What happens when storage class is not recognized? (Return $0 with explanation)
- How does system handle missing size tag? (Use default size)
- What if pricing data is unavailable for a region? (Fallback behavior)
- How to handle concurrent requests for pricing lookups? (Thread-safe)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support S3 storage cost estimation for all defined storage classes (STANDARD, STANDARD_IA, ONEZONE_IA, GLACIER, DEEP_ARCHIVE)
- **FR-002**: System MUST extract storage size from tags["size"] in GB, with sensible default if missing
- **FR-003**: System MUST return accurate per-GB-month rates based on AWS Public Pricing data
- **FR-004**: System MUST handle unknown storage classes by returning $0 with explanatory billing detail
- **FR-005**: System MUST include S3 pricing data in all 9 regional binaries
- **FR-006**: System MUST perform thread-safe pricing lookups under concurrent load
- **FR-007**: System MUST complete pricing lookups in under 50ms
- **FR-008**: System MUST log debug information for pricing lookups including trace ID, storage class, region, and unit price
- **FR-009**: System MUST update Supports() to return supported=true for S3 without "Limited support" reason
- **FR-010**: System MUST automatically inherit S3 support for GetActualCost via the router

### Key Entities *(include if feature involves data)*

- **ResourceDescriptor**: Input containing provider="aws", resourceType="s3", Sku (storage class), region, tags["size"] (GB)
- **GetProjectedCostResponse**: Output containing UnitPrice ($/GB-month), Currency, CostPerMonth, BillingDetail
- **S3Price**: Internal pricing data with Unit, RatePerGBMonth, Currency

### Non-Functional Requirements

- **NFR-001**: System MUST support at least 100 concurrent GetProjectedCost calls without data races or deadlocks

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Supports() returns supported=true for S3 without "Limited support" reason
- **SC-002**: GetProjectedCost() returns accurate per-GB-month rates for Standard storage class (verified against AWS pricing)
- **SC-003**: GetProjectedCost() returns accurate rates for Standard-IA storage class (verified against AWS pricing)
- **SC-004**: Size is correctly extracted from tags["size"] with default applied when missing
- **SC-005**: Unknown storage classes return $0 with explanatory BillingDetail
- **SC-006**: All 9 regional binaries include S3 pricing data (build verification)
- **SC-007**: Pricing lookups complete in under 50ms (performance test)
- **SC-008**: Thread-safe under concurrent RPC calls (stress test with multiple goroutines)
- **SC-009**: Unit tests cover all storage classes and edge cases (test coverage >90%)
- **SC-010**: GetActualCost() automatically supports S3 via router inheritance