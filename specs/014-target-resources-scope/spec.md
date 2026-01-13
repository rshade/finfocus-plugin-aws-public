# Feature Specification: Support target_resources Scope

**Feature Branch**: `014-target-resources-scope`
**Created**: 2025-12-19
**Status**: Draft
**Input**: User description provided via CLI.

## Clarifications

### Session 2025-12-19
- Q: For batch processing, we need to link each output `Recommendation` to its specific input `ResourceDescriptor`. How should correlation be handled? → A: Use resource_id, arn, and then fallback to name in `ResourceRecommendationInfo`.
- Q: What is the logging strategy for batch requests to balance observability and volume? → A: Log one summary line (batch size, total savings) + only individual warnings/errors.

## User Scenarios & Testing

### User Story 1 - Batch Resource Analysis (Priority: P1)

As a client system (like FinFocus Core), I want to submit a list of multiple resources in a single request so that I can efficiently get cost recommendations for all of them without making sequential calls.

**Why this priority**: This is the core purpose of the feature, enabling the new batch-oriented workflow supported by the protocol update.

**Independent Test**: Can be tested by sending a `GetRecommendationsRequest` with a populated `target_resources` list and verifying the response contains recommendations for those resources.

**Acceptance Scenarios**:

1. **Given** a request with 5 distinct valid resources in `target_resources` and no filter, **When** `GetRecommendations` is called, **Then** the response contains 5 corresponding recommendation results.
2. **Given** a request with `target_resources` containing a mix of EC2 instances and EBS volumes, **When** processed, **Then** the plugin returns valid cost estimates for each supported resource type.

---

### User Story 2 - Filtered Batch Analysis (Priority: P2)

As a client system, I want to provide a list of resources AND a filter (e.g., Region) so that I only receive recommendations for the subset of resources that match the criteria.

**Why this priority**: Allows the client to refine the scope of analysis dynamically without manually filtering the list before sending.

**Independent Test**: Send a mixed batch of resources (e.g., different regions) with a `Filter` applied.

**Acceptance Scenarios**:

1. **Given** a request with 10 resources (5 in `us-east-1`, 5 in `us-west-2`) and a `Filter` specifying `Region="us-east-1"`, **When** processed, **Then** the response contains only the 5 recommendations for the `us-east-1` resources.
2. **Given** a request with resources of mixed types and a `Filter` specifying `ResourceType="aws:ec2:Instance"`, **When** processed, **Then** only EC2 instance recommendations are returned.

---

### User Story 3 - Legacy Single-Resource Fallback (Priority: P3)

As an older client system, I want to continue sending requests with only `Filter.Sku` (and no `target_resources`) so that my existing integration continues to work without code changes.

**Why this priority**: Backward compatibility is essential to prevent breaking existing deployments.

**Independent Test**: Send a request with empty `target_resources` but a valid `Filter.Sku` and `Filter.ResourceType`.

**Acceptance Scenarios**:

1. **Given** a request with empty `target_resources` but `Filter` populated with a specific SKU and ResourceType, **When** processed, **Then** the plugin treats this as a single-resource scope and returns the recommendation.
2. **Given** a request with empty `target_resources` AND empty/invalid `Filter` for legacy lookup, **When** processed, **Then** the plugin returns 0 recommendations (stateless behavior).

---

### Edge Cases

- **Large Batch**: If `target_resources` exceeds 100 items, the system must return an `InvalidArgument` error to prevent performance degradation or timeouts.
- **Unsupported Provider**: Resources with `provider` other than `aws` in the batch should be silently ignored or return no recommendation, without failing the entire batch.
- **Unknown Resource Type**: Resources with unknown `resource_type` should be ignored.

## Requirements

### Functional Requirements

- **FR-001**: The plugin MUST update its dependency to `github.com/rshade/finfocus-spec` v0.4.9 or later.
- **FR-002**: The `GetRecommendations` method MUST accept and process `req.TargetResources` as the primary list of resources to analyze.
- **FR-003**: The system MUST implement a legacy fallback: if `req.TargetResources` is empty, it MUST check `req.Filter` for a specific `Sku` and `ResourceType` and treat it as a single-item scope.
- **FR-004**: If both `req.TargetResources` and the legacy filter fallback are empty, the system MUST return 0 recommendations.
- **FR-005**: The system MUST apply any criteria present in `req.Filter` (e.g., Region, ResourceType) to **each** resource in the determined scope (AND operation). Resources not matching the filter MUST be excluded from the response.
- **FR-006**: The system MUST validate that `len(req.TargetResources)` does not exceed 100. If it does, it MUST return an `InvalidArgument` error.
- **FR-007**: The system MUST filter out resources where `provider` is not `aws`.
- **FR-008**: Every generated `Recommendation` MUST include correlation metadata in `ResourceRecommendationInfo` derived from the input `ResourceDescriptor` (priority: `ResourceId` > `Arn` > `Name`).
- **FR-009**: The system MUST log a single summary entry for batch requests containing total resources processed, matched count, and aggregated potential savings. Individual resource details MUST only be logged for warnings (e.g., unsupported type) or errors.

### Key Entities

- **ResourceDescriptor**: Represents a resource in the scope, containing `Sku`, `ResourceType`, `Region`, `Tags`, and `Provider`.
- **GetRecommendationsRequest**: The gRPC request object containing `TargetResources` (scope) and `Filter` (criteria).
- **Recommendation**: The cost estimation result for a specific resource.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Batch requests with up to 100 resources are processed successfully, returning a recommendation for every supported, matching resource.
- **SC-002**: Requests exceeding 100 resources are strictly rejected with an error.
- **SC-003**: Legacy requests (empty target, populated filter) produce identical results to the previous version (0 regression).
- **SC-004**: Filtering logic correctly excludes non-matching resources from a batch (e.g., 0 false positives when filtering by Region).