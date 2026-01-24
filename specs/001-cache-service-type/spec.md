# Feature Specification: Cache Normalized Service Type

**Feature Branch**: `001-cache-service-type`
**Created**: 2026-01-22
**Status**: Draft
**Input**: GitHub Issue #157 - perf: Cache normalized service type to avoid duplicate detectService() calls

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Batch Cost Estimation Performance (Priority: P1)

A FinFocus Core user runs `GetRecommendations` with 100 resources (the maximum batch size). Currently, each resource triggers 2-3 `detectService()` calls across validation, support checks, and cost routing, resulting in 200-300 redundant string parsing operations per batch.

**Why this priority**: `GetRecommendations` is the primary batch operation and directly benefits from reduced per-resource overhead. This is where the optimization has the most measurable impact.

**Independent Test**: Can be fully tested by benchmarking `GetRecommendations` with 100 identical resource descriptors before and after optimization. The optimization delivers value if per-resource overhead decreases measurably.

**Acceptance Scenarios**:

1. **Given** a batch request with 100 EC2 resources, **When** `GetRecommendations` processes the batch, **Then** each resource's service type is computed exactly once (not 2-3 times per resource).
2. **Given** a batch request with mixed resource types (EC2, EBS, RDS), **When** `GetRecommendations` processes the batch, **Then** each unique resource descriptor's service type is computed once for that descriptor.

---

### User Story 2 - Single Resource Request Optimization (Priority: P2)

A FinFocus Core user calls `GetProjectedCost` for a single EC2 instance. The request flows through validation (1-2 `detectService` calls) and cost calculation (1 call), resulting in 2-3 redundant computations for the same input.

**Why this priority**: Single-resource requests are more frequent than batch requests, but the absolute overhead per request is smaller. Still provides cumulative benefit across high-throughput scenarios.

**Independent Test**: Can be fully tested by benchmarking `GetProjectedCost` for a single resource before and after optimization. Expect ~2x reduction in string parsing operations.

**Acceptance Scenarios**:

1. **Given** a single EC2 resource request, **When** `GetProjectedCost` is called, **Then** `detectService` is invoked at most once for that request.
2. **Given** a Pulumi-format resource type (`aws:eks/cluster:Cluster`), **When** `GetProjectedCost` is called, **Then** the normalization and service detection happen once, not at each validation stage.

---

### User Story 3 - Actual Cost Calculation Optimization (Priority: P3)

A FinFocus Core user calls `GetActualCost` which also flows through validation and cost routing with multiple `detectService` calls.

**Why this priority**: Same pattern as `GetProjectedCost` but less frequently used. Benefits from the same caching mechanism.

**Independent Test**: Can be fully tested by benchmarking `GetActualCost` before and after optimization.

**Acceptance Scenarios**:

1. **Given** a single RDS resource request, **When** `GetActualCost` is called, **Then** `detectService` is invoked at most once for that request.

---

### Edge Cases

- What happens when the resource type is malformed or empty? Caching should not interfere with error handling - invalid inputs should still produce appropriate errors.
- How does caching behave with concurrent requests? The caching mechanism must be thread-safe for concurrent gRPC calls.
- What happens when the same resource type string appears in different request contexts? Each request should have isolated caching (no cross-request leakage).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST compute the normalized service type exactly once per resource within a single request lifecycle.
- **FR-002**: System MUST maintain identical behavior for all existing RPC methods (`GetProjectedCost`, `GetActualCost`, `GetRecommendations`, `Supports`, `GetPricingSpec`).
- **FR-003**: System MUST return identical results for all inputs before and after optimization (pure refactoring, no behavior change).
- **FR-004**: System MUST handle concurrent RPC calls without data races or cross-request contamination.
- **FR-005**: System MUST preserve all existing error handling semantics for malformed resource types.
- **FR-006**: System MUST NOT introduce additional heap allocations for single-resource requests beyond the caching overhead.
- **FR-007**: System SHOULD reduce string parsing operations by at least 50% per single-resource request.

### Key Entities

- **ResourceDescriptor**: The proto input containing `resource_type`, `provider`, `sku`, `region`, and `tags`. The `resource_type` field is the input to normalization and service detection.
- **serviceResolver (new)**: A lightweight wrapper that memoizes the result of `normalizeResourceType()` and `detectService()` for a given `ResourceDescriptor.ResourceType`. Created once per request and passed through validation and routing logic.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Benchmark for `GetRecommendations` with 100 resources shows measurable reduction in `detectService` call count (from ~300 calls to ~100 calls).
- **SC-002**: Benchmark for single `GetProjectedCost` shows reduction from 2-3 `detectService` calls to 1 call.
- **SC-003**: All existing unit tests and integration tests pass without modification (behavior preservation).
- **SC-004**: No new data races detected by `go test -race` across all affected packages.
- **SC-005**: Memory allocation profile shows no significant increase per request (acceptable: <100 bytes per resource for caching overhead).

## Assumptions

- The performance impact of 2-3 redundant `detectService()` calls per request is minor for single requests but compounds significantly for batch operations.
- The caching mechanism will use a memoized wrapper struct rather than context-based caching, as this is simpler to implement and maintain.
- Profile data should be collected before implementation to establish baseline performance (if not already done).
