# Feature Specification: Pre-allocate Map Capacity for Pricing Indexes

**Feature Branch**: `021-map-prealloc`
**Created**: 2026-01-04
**Status**: Draft
**GitHub Issue**: #176
**Input**: Performance optimization to pre-allocate map capacities in pricing client initialization

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Faster Plugin Initialization (Priority: P1)

As a PulumiCost user running cost estimates on large infrastructure deployments, I want the pricing plugin to initialize as quickly as possible so that my CI/CD pipelines have minimal overhead when estimating costs.

**Why this priority**: Plugin initialization happens on every invocation. Reducing initialization time directly improves user experience and reduces CI/CD pipeline duration for all users.

**Independent Test**: Can be measured by running `BenchmarkNewClient` before and after the change and comparing initialization time and memory allocations.

**Acceptance Scenarios**:

1. **Given** a fresh plugin process starting, **When** the pricing client initializes, **Then** map allocations should occur upfront without repeated growth/rehashing during parsing.
2. **Given** the pricing client is initializing with EC2 data containing 90,000+ products, **When** products are indexed, **Then** no map rehashing should occur during the parsing loop.

---

### User Story 2 - Reduced Memory Churn (Priority: P2)

As a platform engineer running PulumiCost in memory-constrained environments (e.g., serverless, containers with memory limits), I want the plugin to have predictable memory allocation patterns so that garbage collection pauses are minimized.

**Why this priority**: Unpredictable GC pauses can cause latency spikes in cost estimation responses, impacting user-perceived performance.

**Independent Test**: Can be verified by running `BenchmarkNewClient` with `-benchmem` flag and comparing allocations (allocs/op) before and after.

**Acceptance Scenarios**:

1. **Given** the plugin is processing pricing data, **When** maps grow during parsing, **Then** the total number of allocations should be reduced compared to the baseline.
2. **Given** a memory-constrained environment, **When** the pricing client initializes, **Then** memory usage should follow a predictable, single-allocation pattern per map.

---

### Edge Cases

- What happens when pricing data volume changes significantly between AWS API updates? The capacity estimates should be conservative enough to handle 20-30% data growth without performance regression.
- How does the system handle regions with fewer products than estimated? Pre-allocation may waste some memory, but this is acceptable given memory is reclaimed after initialization.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST pre-allocate `ec2Index` map with capacity for approximately 100,000 entries to accommodate the ~90,000+ EC2 product SKUs per region.
- **FR-002**: System MUST pre-allocate `ebsIndex` map with capacity for approximately 50 entries to accommodate ~20-30 EBS volume types.
- **FR-003**: System MUST pre-allocate `s3Index` map with capacity for approximately 100 entries to accommodate ~50-100 S3 storage classes.
- **FR-004**: System MUST pre-allocate `rdsInstanceIndex` map with capacity for approximately 5,000 entries to accommodate RDS instance type combinations.
- **FR-005**: System MUST pre-allocate `rdsStorageIndex` map with capacity for approximately 100 entries to accommodate RDS storage types.
- **FR-006**: System MUST pre-allocate `elasticacheIndex` map with capacity for approximately 1,000 entries to accommodate ElastiCache node type and engine combinations.
- **FR-007**: System MUST document capacity estimates in code comments explaining the reasoning for each value.
- **FR-008**: System MUST NOT break existing functionality - all pricing lookups must return identical results before and after this change.

### Key Entities

- **Pricing Index Maps**: Six map types in the Client struct that are pre-allocated during initialization:
  - `ec2Index`: Stores EC2 instance pricing keyed by instance type
  - `ebsIndex`: Stores EBS volume pricing keyed by volume type
  - `s3Index`: Stores S3 storage pricing keyed by storage class
  - `rdsInstanceIndex`: Stores RDS instance pricing keyed by instance type/engine
  - `rdsStorageIndex`: Stores RDS storage pricing keyed by volume type
  - `elasticacheIndex`: Stores ElastiCache pricing keyed by node type/engine

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `BenchmarkNewClient` shows reduction in total allocations (allocs/op), targeting at least 10% fewer allocations.
- **SC-002**: `BenchmarkNewClient` shows no regression in initialization time (ns/op should remain within 5% of baseline or improve).
- **SC-003**: All existing unit tests pass without modification.
- **SC-004**: All existing integration tests pass without modification.
- **SC-005**: Memory usage per initialization (B/op in benchmark) does not increase by more than 10% (pre-allocation trades slight upfront memory for fewer total allocations).
- **SC-006**: Benchmark comparison results are documented in PR description for validation. (Note: Automated CI benchmark comparison is a future enhancement - manual comparison via `benchstat` is acceptable for v1.)

## Assumptions

- Capacity estimates are based on typical us-east-1 pricing data volumes, which represents the largest AWS region.
- A 20-30% buffer above current observed counts is included to accommodate future AWS service expansion.
- The slight memory overhead from over-allocation is acceptable given it occurs once per plugin lifecycle and is bounded.
- Other regions with fewer products will simply have unused map capacity, which is an acceptable trade-off for consistent performance.

## Out of Scope

- Dynamic capacity adjustment based on actual data volume (would add complexity for minimal benefit).
- Pre-allocation for non-map data structures (pointers like `eksPricing`, `lambdaPricing` are single allocations).
- Memory pooling or object reuse patterns (beyond the scope of this optimization).

## Clarifications

### Session 2026-01-04

- Q: How should benchmark baseline be captured and compared for validation? → A: Capture baseline in CI (run benchmark on `main`, then on PR branch, compare in PR comments)
