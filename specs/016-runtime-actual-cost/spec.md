# Feature Specification: Runtime-Based Actual Cost Estimation

**Feature Branch**: `016-runtime-actual-cost`
**Created**: 2025-12-31
**Status**: Draft
**GitHub Issue**: #196

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Auto-Calculate Cost from Creation Time (Priority: P1)

As a cloud cost analyst, I want actual cost calculations to automatically use the
resource's creation timestamp from Pulumi state, so I can get accurate runtime-based
cost estimates without manually specifying start times.

**Why this priority**: This is the core value proposition. Using `pulumi:created`
timestamps enables automatic runtime detection, making cost tracking effortless
for resources managed by Pulumi.

**Independent Test**: Can be fully tested by providing a resource with
`pulumi:created` in its properties and verifying the cost is calculated from
that timestamp to the specified end time.

**Acceptance Scenarios**:

1. **Given** an EC2 instance with `pulumi:created` set to 7 days ago, **When**
   actual cost is requested with end time of now, **Then** the system returns
   cost based on 168 hours of runtime.
2. **Given** a resource with `pulumi:created` and a request with explicit start
   time after creation, **When** actual cost is calculated, **Then** the explicit
   start time takes precedence over creation time.
3. **Given** a resource without `pulumi:created`, **When** actual cost is
   requested, **Then** the system falls back to requiring explicit start time
   in the request.

---

### User Story 2 - Identify Imported Resources with Lower Confidence (Priority: P2)

As a cloud cost analyst, I want to know when cost estimates may be less accurate
due to resource imports, so I can make informed decisions about the reliability
of the data.

**Why this priority**: Imported resources have `pulumi:created` set to import
time, not actual cloud creation time. Users need visibility into estimate accuracy.

**Independent Test**: Can be tested by providing a resource with
`pulumi:external=true` and verifying the response includes a lower confidence
indicator.

**Acceptance Scenarios**:

1. **Given** a resource with `pulumi:external=true`, **When** actual cost is
   calculated, **Then** the response indicates lower confidence with an
   explanatory note.
2. **Given** a resource without `pulumi:external`, **When** actual cost is
   calculated, **Then** the response indicates standard confidence.

---

### User Story 3 - Prioritize Runtime from Request Metadata (Priority: P2)

As a cost analyst using automation, I want to optionally override the creation
timestamp with explicit request parameters, so I can calculate costs for specific
time windows regardless of resource age.

**Why this priority**: Flexibility for reporting and auditing use cases where
specific date ranges matter more than actual resource age.

**Independent Test**: Can be tested by providing both `pulumi:created` and
explicit request timestamps, verifying the request parameters take precedence.

**Acceptance Scenarios**:

1. **Given** a resource created 30 days ago, **When** actual cost is requested
   for only the last 7 days, **Then** the cost reflects only those 7 days.
2. **Given** explicit start and end times in the request, **When** actual cost
   is calculated, **Then** `pulumi:created` is ignored in favor of request times.

---

### Edge Cases

- What happens when `pulumi:created` contains an invalid or unparseable timestamp?
  System falls back to requiring explicit request timestamps.
- What happens when the requested end time is before `pulumi:created`?
  System returns zero cost with an explanatory note.
- What happens when `pulumi:modified` exists but `pulumi:created` is missing?
  System does not use `pulumi:modified` as a creation time substitute.
- What happens for resources that can be stopped (EC2)?
  System cannot track stop/start events; estimates assume continuous runtime
  with a documented limitation note.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST parse `pulumi:created` from resource properties when
  present (RFC3339 format).
- **FR-002**: System MUST use `pulumi:created` as the default start time when
  no explicit start time is provided in the request.
- **FR-003**: System MUST allow explicit request start/end times to override
  `pulumi:created`.
- **FR-004**: System MUST detect `pulumi:external=true` and flag the estimate
  as lower confidence.
- **FR-005**: System MUST include confidence level in the response (HIGH for
  native resources, MEDIUM for imported resources).
- **FR-006**: System MUST include explanatory notes in the response describing
  the estimation basis and any limitations.
- **FR-007**: System MUST handle missing `pulumi:created` gracefully by
  requiring explicit timestamps or returning an appropriate error.
- **FR-008**: System MUST calculate actual cost using the formula:
  `actual_cost = projected_monthly_cost × (runtime_hours / 730)`.
- **FR-009**: System MUST return zero cost with explanation when end time
  precedes start time (after validation).

### Key Entities

- **Resource Properties**: Metadata injected by pulumicost-core from Pulumi state
  - `pulumi:created`: RFC3339 timestamp of resource creation
  - `pulumi:modified`: RFC3339 timestamp of last modification
  - `pulumi:external`: Boolean flag indicating imported resource
- **Actual Cost Result**: Response containing calculated cost with metadata
  - Cost amount and currency
  - Time range used for calculation
  - Confidence level (HIGH/MEDIUM/LOW)
  - Explanatory notes

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of resources with valid `pulumi:created` timestamps
  automatically use that time for cost calculation when no explicit start
  is provided.
- **SC-002**: Imported resources (`pulumi:external=true`) are correctly
  identified and flagged in 100% of cases.
- **SC-003**: Cost calculation accuracy matches the existing projected cost
  system (same hourly rate derivation).
- **SC-004**: Response includes confidence indicator for every actual cost
  calculation.
- **SC-005**: Users receive clear explanatory notes describing the estimation
  basis and any limitations.

## Assumptions

- **A-001**: pulumicost-core injects `pulumi:created`, `pulumi:modified`, and
  `pulumi:external` into resource properties before calling the plugin.
- **A-002**: Timestamps are in RFC3339 format as specified by pulumicost-core.
- **A-003**: The existing `GetActualCost` calculation formula remains unchanged;
  this feature adds smarter timestamp resolution.
- **A-004**: Stop/start tracking for EC2 instances is out of scope; estimates
  assume continuous runtime.
- **A-005**: The 730-hour month constant is used consistently with projected
  cost calculations.

## Limitations (Must Document to Users)

- **L-001**: Imported resources have inaccurate creation times (import time,
  not cloud creation time).
- **L-002**: Stop/start events are not tracked; estimates assume 100% uptime.
- **L-003**: Estimates do not reflect Reserved Instance, Spot, or Savings Plans
  pricing.
- **L-004**: For accurate billing data, users should use FinOps plugins with
  access to real billing APIs.
