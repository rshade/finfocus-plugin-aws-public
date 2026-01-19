# Feature Specification: IAM Zero-Cost Resource Handling

**Feature Branch**: `035-iam-zero-cost`  
**Created**: 2026-01-19  
**Status**: Draft  
**Input**: User description provided in prompt.

## Clarifications

### Session 2026-01-19

- Q: Should the system use an explicit allowlist or a prefix match for IAM resource types? → A: Prefix Match (aws:iam/*)
- Q: How should the system handle casing for resource type matching? → A: Case-Insensitive (normalize to lowercase before checking)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Estimate Pulumi Stack with IAM Resources (Priority: P1)

As a DevOps engineer using FinFocus, I want the plugin to correctly identify and estimate AWS IAM resources in my Pulumi stack as having zero cost, so that my total cost estimate is accurate and I don't see "unsupported resource" warnings for standard infrastructure components.

**Why this priority**: IAM resources are fundamental to almost every AWS deployment. Currently, they may be flagged as unsupported or require unnecessary processing, which creates noise in the estimation report.

**Independent Test**: Create a Pulumi stack (or mock) with IAM Users, Roles, and Policies. Run the plugin and verify the output contains these resources with $0 cost.

**Acceptance Scenarios**:

1. **Given** a Pulumi stack containing an `aws:iam/user:User` resource, **When** the plugin estimates costs, **Then** the resource is marked as supported and has a projected cost of $0.00.
2. **Given** a Pulumi stack containing an `aws:iam/role:Role` resource, **When** the plugin estimates costs, **Then** the resource is marked as supported and has a projected cost of $0.00.
3. **Given** a Pulumi stack containing an `aws:iam/policy:Policy` resource, **When** the plugin estimates costs, **Then** the resource is marked as supported and has a projected cost of $0.00.
4. **Given** a Pulumi stack containing an `aws:iam/group:Group` or `aws:iam/instanceProfile:InstanceProfile`, **When** the plugin estimates costs, **Then** the resource is marked as supported and has a projected cost of $0.00.

### Edge Cases

- **What happens when an unknown IAM resource type is encountered?** If the normalization logic misses a specific IAM sub-type, it might fall through to default handling. The normalization must be robust enough to catch standard Pulumi IAM patterns.
- **How does system handle mixed-case resource types?** The normalization should handle case sensitivity appropriately to match Pulumi's output.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST recognize "iam" as a distinct, zero-cost resource category in the internal type map.
- **FR-002**: The system MUST normalize any Pulumi resource type starting with the case-insensitive prefix `aws:iam/` to the canonical "iam" service type. Normalization MUST convert the input type to lowercase before checking the prefix.
- **FR-003**: The `Supports()` method MUST return `true` for the "iam" service type.
- **FR-004**: The `GetProjectedCost()` method MUST return a cost estimate of $0.00 for identified IAM resources without performing pricing lookups.
- **FR-005**: The cost estimate response for IAM resources MUST include a billing detail message stating "IAM - no direct AWS charges" (or similar clear wording).
- **FR-006**: The system MUST NOT return carbon estimation metrics for IAM resources (effectively 0 or null).

### Key Entities

- **IAM Resource**: Represents an Identity and Access Management entity (User, Role, Policy, etc.) in the infrastructure state.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of standard Pulumi IAM resource types (User, Role, Policy, Group, InstanceProfile) are reported as "Supported" by the plugin.
- **SC-002**: IAM resources contribute exactly $0.00 to the total projected cost.
- **SC-003**: Execution time for IAM resources is negligible (<1ms) as it bypasses external pricing lookups.