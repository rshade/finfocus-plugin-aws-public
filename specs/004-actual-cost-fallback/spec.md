# Feature Specification: Fallback GetActualCost Implementation

**Feature Branch**: `004-actual-cost-fallback`
**Created**: 2025-11-25
**Status**: Draft
**Input**: GitHub Issue #24 - Implement fallback GetActualCost

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Calculate Actual Cost for Running Resources (Priority: P1)

As a FinFocus user, I want to get actual cost estimates for resources
that have been running for a specific time period, so I can understand
my cloud spending without requiring AWS Cost Explorer access.

**Why this priority**: This is the core functionality of the feature.
Without actual cost calculation, the plugin cannot provide cost insights
for real deployments, blocking E2E tests and production use cases.

**Independent Test**: Can be fully tested by calling GetActualCost with
a resource descriptor and time range, and verifying it returns a
non-zero cost based on the runtime hours.

**Acceptance Scenarios**:

1. **Given** an EC2 t3.micro instance with projected monthly cost of
   $7.59, **When** GetActualCost is called with a 24-hour runtime,
   **Then** the system returns approximately $0.25 (= $7.59 × 24/730)
2. **Given** an EBS gp3 volume with 100GB, **When** GetActualCost is
   called with a 168-hour runtime (1 week), **Then** the system returns
   the proportional cost (monthly_cost × 168/730)
3. **Given** a valid resource and valid time range, **When**
   GetActualCost is called, **Then** the response includes cost in USD
   with billing details explaining the calculation basis

---

### User Story 2 - Handle Invalid Time Ranges Gracefully (Priority: P2)

As a FinFocus user, I want clear error messages when I provide invalid
time ranges, so I can correct my inputs without confusion.

**Why this priority**: Error handling is essential for usability but
secondary to the core calculation functionality.

**Independent Test**: Can be tested by calling GetActualCost with
various invalid time ranges and verifying appropriate error responses.

**Acceptance Scenarios**:

1. **Given** a time range where "start" is after "end", **When**
   GetActualCost is called, **Then** the system returns an error
   indicating the invalid time range
2. **Given** nil Start or End timestamp fields, **When** GetActualCost is
   called, **Then** the system returns ERROR_CODE_INVALID_RESOURCE with
   details indicating the missing timestamp
3. **Given** a zero-duration time range (start equals end), **When**
   GetActualCost is called, **Then** the system returns a cost of $0.00
   with appropriate billing details

---

### User Story 3 - Support Stub Services with Zero Cost (Priority: P3)

As a FinFocus user, I want GetActualCost to handle stub services (S3,
Lambda, RDS, DynamoDB) consistently by returning zero costs with clear
explanations.

**Why this priority**: Maintains consistency with existing
GetProjectedCost stub behavior, but is not core functionality.

**Independent Test**: Can be tested by calling GetActualCost for stub
service types and verifying $0 responses with appropriate billing
details.

**Acceptance Scenarios**:

1. **Given** an S3 resource type, **When** GetActualCost is called,
   **Then** the system returns $0 with billing_detail explaining the
   service is not yet implemented
2. **Given** any stub service (S3, Lambda, RDS, DynamoDB), **When**
   GetActualCost is called, **Then** the response format matches
   GetProjectedCost stub responses

---

### Edge Cases

- What happens when the time range spans multiple months?
  - The calculation uses total hours regardless of month boundaries
- What happens when the time range exceeds one month (730+ hours)?
  - Costs are calculated proportionally; 730 hours = one monthly cost
- How does the system handle fractional hours?
  - Runtime is calculated as floating-point hours for precision
- What happens with same-day time ranges (e.g., 9am to 5pm)?
  - Calculates the fractional hours (8 hours in this example)
- What happens when the resource region doesn't match the plugin binary?
  - Returns ERROR_CODE_UNSUPPORTED_REGION, consistent with
    GetProjectedCost

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST calculate actual cost using the formula:
  `actual_cost = projected_monthly_cost × (runtime_hours / 730)`
- **FR-002**: System MUST parse protobuf Timestamp fields from the
  request's Start and End fields (v0.3.0 proto)
- **FR-003**: System MUST calculate runtime in hours as a floating-point
  value for precision
- **FR-004**: System MUST return costs in USD currency
- **FR-005**: System MUST include billing detail in ActualCostResult.Source
  explaining the calculation basis (e.g., "Fallback estimate: $X.XX
  monthly rate × Y.YY hours / 730")
- **FR-006**: System MUST return ERROR_CODE_INVALID_RESOURCE when
  required ResourceDescriptor fields are missing from ResourceId JSON
- **FR-007**: System MUST return ERROR_CODE_UNSUPPORTED_REGION when
  resource region doesn't match plugin binary region
- **FR-008**: System MUST return an error for invalid time ranges
  (start > end)
- **FR-009**: System MUST return $0.00 cost for zero-duration time
  ranges (start = end)
- **FR-010**: System MUST return $0.00 with explanation for stub
  services (S3, Lambda, RDS, DynamoDB)
- **FR-011**: System MUST reuse existing GetProjectedCost logic to
  obtain monthly rates
- **FR-012**: System MUST parse ResourceId as JSON-encoded
  ResourceDescriptor (v0.3.0 proto)

### Key Entities (finfocus-spec v0.3.0)

- **GetActualCostRequest**: Contains resource_id (JSON-encoded
  ResourceDescriptor), Start/End protobuf Timestamps, and tags map
- **GetActualCostResponse**: Contains Results array of ActualCostResult
- **ActualCostResult**: Contains timestamp, cost, usage_amount,
  usage_unit, and source (billing detail)
- **ResourceDescriptor**: Provider, resource_type, sku, region, and
  tags identifying the resource (JSON-encoded in resource_id)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: GetActualCost returns non-nil responses for all supported
  resource types (EC2, EBS)
- **SC-002**: Cost calculations are accurate within 0.01% of the
  expected formula result
- **SC-003**: Response latency for GetActualCost is under 10ms for
  single resource requests
- **SC-004**: All unit tests pass with 100% coverage of the new actual
  cost logic
- **SC-005**: E2E tests can successfully retrieve actual costs for
  sample deployments
- **SC-006**: Error responses include appropriate error codes and
  descriptive messages

## Assumptions

- The 730 hours/month constant is acceptable for all calculations
  (industry standard for on-demand pricing)
- EC2 instances use Linux, shared tenancy, on-demand pricing (same as
  GetProjectedCost)
- EBS volumes use standard on-demand pricing (same as GetProjectedCost)
- Time ranges are expected to be reasonable (not spanning years)
- The formula provides a fallback estimate; actual AWS costs may vary
  due to usage patterns, data transfer, etc.

## Dependencies

This feature blocks the following issues in finfocus-core that require
functional actual cost calculations for comprehensive testing workflows:

- finfocus-core#177
- finfocus-core#178
- finfocus-core#180
- finfocus-core#182

## Out of Scope

- Real AWS Cost Explorer integration (this is a fallback estimation)
- Spot instance pricing adjustments
- Reserved instance pricing adjustments
- Savings plans calculations
- Data transfer costs
- Multi-resource batch calculations (one resource per RPC call)
