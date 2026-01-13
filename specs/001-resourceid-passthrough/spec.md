# Feature Specification: Resource ID Passthrough in GetRecommendations

**Feature Branch**: `001-resourceid-passthrough`
**Created**: 2025-12-26
**Status**: Draft
**GitHub Issue**: #198
**Input**: Pass through the native resource ID from ResourceDescriptor to recommendation responses for proper correlation in finfocus-core.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Resource Correlation in Batch Requests (Priority: P1)

As a finfocus-core developer, I need recommendation responses to include the resource ID from the original request so I can correctly correlate recommendations back to their source resources when processing batch responses.

**Why this priority**: This is the core value proposition. Without proper ID correlation, batch recommendation requests become unusable because the caller cannot determine which resource each recommendation applies to.

**Independent Test**: Can be fully tested by sending a batch of resources with unique IDs and verifying each recommendation includes the correct source resource ID.

**Acceptance Scenarios**:

1. **Given** a GetRecommendationsRequest with TargetResources containing resources with `Id` field populated, **When** GetRecommendations is called, **Then** each returned recommendation's `Resource.Id` field MUST contain the same ID from the input resource.

2. **Given** a GetRecommendationsRequest with a single resource where `Id = "urn:pulumi:stack::project::aws:ec2/instance:Instance::myserver"`, **When** GetRecommendations returns a generation upgrade recommendation, **Then** the recommendation's `Resource.Id` MUST equal `"urn:pulumi:stack::project::aws:ec2/instance:Instance::myserver"`.

---

### User Story 2 - Backward Compatibility with Tag-Based Correlation (Priority: P2)

As an existing finfocus-core user, I need the plugin to continue supporting correlation via the `resource_id` tag when the native ID field is not populated, so my existing integrations keep working.

**Why this priority**: Maintains backward compatibility during the transition period. Existing callers may still use the tag-based approach.

**Independent Test**: Can be tested by sending resources without the `Id` field but with `resource_id` in tags, and verifying recommendations use the tag value.

**Acceptance Scenarios**:

1. **Given** a resource with empty `Id` field but `tags["resource_id"] = "my-resource-123"`, **When** GetRecommendations returns recommendations, **Then** the recommendation's `Resource.Id` MUST equal `"my-resource-123"`.

2. **Given** a resource with both `Id = "native-id"` AND `tags["resource_id"] = "tag-id"`, **When** GetRecommendations returns recommendations, **Then** the recommendation's `Resource.Id` MUST equal `"native-id"` (native ID takes precedence).

---

### Edge Cases

- What happens when a resource has no ID (empty `Id` field and no `resource_id` tag)?
  - The recommendation is still generated; `Resource.Id` remains empty
- What happens when a resource generates multiple recommendations (e.g., both generation upgrade and Graviton)?
  - Each recommendation MUST have the same `Resource.Id` from the source resource
- What happens when `Id` field is whitespace-only?
  - Treat as empty; fall back to `resource_id` tag if present

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST pass through the `Id` field from `ResourceDescriptor` to `Recommendation.Resource.Id` for each generated recommendation
- **FR-002**: System MUST prioritize the native `Id` field over the `resource_id` tag when both are present
- **FR-003**: System MUST fall back to `tags["resource_id"]` when the native `Id` field is empty or whitespace-only
- **FR-004**: System MUST preserve existing behavior for the `name` tag correlation (continue using `tags["name"]` for `Resource.Name`)
- **FR-005**: System MUST populate the same `Resource.Id` on all recommendations generated from a single input resource

### Key Entities

- **ResourceDescriptor**: Input resource with new `Id` field (from finfocus-spec#200)
  - Key attributes: `Id` (new), `Provider`, `ResourceType`, `Sku`, `Region`, `Tags`
- **Recommendation**: Output containing cost optimization suggestions
  - Key attributes: `Resource.Id` (populated from input), `Resource.Name`, `Impact`, `ActionDetail`
- **ResourceRecommendationInfo**: Nested in Recommendation, contains correlation info
  - Key attributes: `Id` (to be populated), `Name`, `Provider`, `ResourceType`, `Region`, `Sku`

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of recommendations generated from resources with `Id` field populated include that ID in `Recommendation.Resource.Id`
- **SC-002**: Zero breaking changes to existing callers using tag-based correlation
- **SC-003**: All existing GetRecommendations tests continue to pass
- **SC-004**: New unit tests achieve 100% coverage of ID passthrough logic

## Dependencies

### External Dependencies

- **Requires**: finfocus-spec v0.4.11+ (rshade/finfocus-spec#200 merged)
  - Status: Available - v0.4.11 released with `Id` field on `ResourceDescriptor`
  - Action: Update `go.mod` to `github.com/rshade/finfocus-spec v0.4.11` or later

## Assumptions

- The `Id` field in `ResourceDescriptor` will be a string (based on typical Pulumi URN format)
- Whitespace-only strings are treated as empty for the `Id` field
- The existing `resource_id` and `name` tag logic will be preserved for backward compatibility
- No performance impact expected as this is a simple field copy operation
