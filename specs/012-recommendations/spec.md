# Feature Specification: GetRecommendations RPC for Cost Optimization

**Feature Branch**: `012-recommendations`
**Created**: 2025-12-15
**Status**: Draft
**GitHub Issue**: #105
**Input**: User description: "Implement GetRecommendations RPC from finfocus-spec v0.4.7 for cost optimization recommendations"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - EC2 Instance Generation Upgrade Suggestions (Priority: P1)

As an infrastructure engineer using FinFocus, I want to receive recommendations when my EC2 instances use older generation types so that I can modernize to newer, more cost-effective instances without manual research.

**Why this priority**: Generation upgrades are the most common and safest optimization opportunity. Newer generations (t2→t3, m4→m5) are drop-in replacements that provide better performance at the same or lower price. This delivers immediate, actionable value with minimal risk.

**Independent Test**: Can be fully tested by requesting recommendations for a t2.medium instance and verifying the system suggests t3.medium with price comparison data.

**Acceptance Scenarios**:

1. **Given** an EC2 instance using t2.medium, **When** GetRecommendations is called, **Then** the system returns a recommendation to upgrade to t3.medium with current cost, recommended cost, and monthly savings calculated.
2. **Given** an EC2 instance using the latest generation (e.g., t3a.micro), **When** GetRecommendations is called, **Then** no generation upgrade recommendation is returned.
3. **Given** an EC2 instance type where the newer generation is more expensive in the current region, **When** GetRecommendations is called, **Then** no upgrade recommendation is returned (avoid false positives).

---

### User Story 2 - AWS Graviton/ARM Migration Suggestions (Priority: P2)

As an infrastructure engineer, I want to receive recommendations for migrating x86 instances to ARM-based Graviton instances so that I can evaluate potential cost savings while being aware of compatibility requirements.

**Why this priority**: Graviton migrations offer significant cost savings (~20%) but require ARM compatibility validation. This is high value but needs user judgment, hence medium confidence level.

**Independent Test**: Can be fully tested by requesting recommendations for an m5.large instance and verifying the system suggests m6g.large with savings percentage and architecture change warnings.

**Acceptance Scenarios**:

1. **Given** an EC2 instance using m5.large, **When** GetRecommendations is called, **Then** the system returns a Graviton recommendation with m6g.large, showing ~20% savings and "medium" confidence level.
2. **Given** the Graviton recommendation, **Then** the recommendation details include a clear warning that ARM architecture compatibility must be validated.
3. **Given** an instance type with no Graviton equivalent (e.g., mac1.metal), **When** GetRecommendations is called, **Then** no Graviton recommendation is returned.

---

### User Story 3 - EBS Volume Type Upgrade (gp2 to gp3) (Priority: P3)

As an infrastructure engineer, I want to receive recommendations to migrate gp2 volumes to gp3 so that I can reduce storage costs while improving baseline performance.

**Why this priority**: EBS gp2→gp3 migration is a straightforward, API-compatible change that provides both cost savings and performance improvements. Lower priority than EC2 because storage costs are typically smaller than compute costs.

**Independent Test**: Can be fully tested by requesting recommendations for a gp2 volume and verifying the system suggests gp3 with cost comparison and performance improvement details.

**Acceptance Scenarios**:

1. **Given** an EBS volume of type gp2 with 100GB size, **When** GetRecommendations is called, **Then** the system returns a volume type upgrade recommendation to gp3 with cost savings calculated.
2. **Given** an EBS volume of type gp3, io1, or io2, **When** GetRecommendations is called, **Then** no volume type upgrade recommendation is returned.
3. **Given** the gp2→gp3 recommendation, **Then** the details include baseline performance comparison (IOPS and throughput).

---

### User Story 4 - Graceful Handling of Unsupported Services (Priority: P4)

As a FinFocus user, I want GetRecommendations to return an empty list for services where recommendations aren't supported so that my automation doesn't break and I understand the limitation.

**Why this priority**: Important for API completeness and robustness, but doesn't directly deliver cost optimization value.

**Independent Test**: Can be fully tested by calling GetRecommendations for S3, Lambda, RDS, or DynamoDB resources and verifying an empty recommendations list is returned without errors.

**Acceptance Scenarios**:

1. **Given** a resource type that is not EC2 or EBS (e.g., S3, Lambda, RDS), **When** GetRecommendations is called, **Then** an empty recommendations list is returned (not an error).
2. **Given** a request with a missing or nil resource descriptor, **When** GetRecommendations is called, **Then** an appropriate error is returned with ERROR_CODE_INVALID_RESOURCE.

---

### Edge Cases

- What happens when the instance type doesn't exist in pricing data? (Return no recommendations)
- What happens when the recommended instance type isn't available in the region? (Return no recommendation for that type)
- What happens when EBS volume size is not specified in tags? (Use reasonable default for calculation, e.g., 100GB)
- What happens when both generation upgrade AND Graviton are available? (Return both recommendations, letting user choose)
- What happens when the region doesn't support Graviton instances? (Return no Graviton recommendation)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST implement the GetRecommendations RPC method as defined in finfocus-spec v0.4.7
- **FR-002**: System MUST return generation upgrade recommendations for EC2 instances when newer generations offer same or lower price
- **FR-003**: System MUST return Graviton/ARM migration recommendations for compatible x86 instance families
- **FR-004**: System MUST return gp2→gp3 volume type upgrade recommendations for EBS volumes
- **FR-005**: System MUST calculate monthly savings as (current_cost - recommended_cost) based on 730 hours/month for EC2
- **FR-006**: System MUST set confidence level to 0.9 (high) for generation upgrades and EBS volume changes
- **FR-007**: System MUST set confidence level to 0.7 (medium) for Graviton recommendations (requires user validation)
- **FR-008**: System MUST return empty recommendations list (not error) for unsupported resource types
- **FR-009**: System MUST return ERROR_CODE_INVALID_RESOURCE when request or resource descriptor is nil
- **FR-010**: System MUST include trace_id in all log entries for distributed tracing
- **FR-011**: System MUST only recommend when new option price is less than or equal to current price
- **FR-012**: System MUST include relevant metadata in recommendation details (instance types, performance info, architecture warnings)

### Key Entities

- **Recommendation**: Represents a cost optimization suggestion with type, title, description, current cost, recommended cost, monthly savings, confidence level, and details map
- **Instance Family**: The category portion of EC2 instance type (e.g., "t2" from "t2.medium") used for mapping to newer generations
- **Instance Size**: The size portion of EC2 instance type (e.g., "medium" from "t2.medium") preserved during recommendations
- **Generation Upgrade Map**: Mapping of older instance families to their newer generation equivalents (t2→t3, m4→m5, etc.)
- **Graviton Map**: Mapping of x86 instance families to ARM/Graviton equivalents (m5→m6g, c5→c6g, t3→t4g)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: GetRecommendations returns valid proto response for all supported resource types within 100ms
- **SC-002**: 100% of generation upgrade recommendations result in equal or lower monthly cost
- **SC-003**: 100% of Graviton recommendations include architecture change warning in details
- **SC-004**: All recommendations include accurate monthly savings calculation (verified against pricing data)
- **SC-005**: Zero false positives - no recommendations where new option costs more than current
- **SC-006**: Empty list returned for unsupported services (S3, Lambda, RDS, DynamoDB) without errors
- **SC-007**: All log entries for GetRecommendations operations include trace_id for distributed tracing

## Scope Boundaries *(mandatory)*

### In Scope

- EC2 instance generation upgrade recommendations (t2→t3, m4→m5, c4→c5, r4→r5, etc.)
- EC2 Graviton/ARM migration recommendations (m5→m6g, c5→c6g, t3→t4g, etc.)
- EBS gp2→gp3 volume type recommendations
- Price comparison using embedded public AWS pricing data
- Confidence levels indicating recommendation safety

### Out of Scope (Intentionally Excluded)

| Feature                           | Reason                                                             |
| --------------------------------- | ------------------------------------------------------------------ |
| Reserved Instance recommendations | Cannot know if resource is already covered by RI/Savings Plan      |
| Rightsizing (downsize)            | No utilization data - could cause outages                          |
| Spot Instance recommendations     | No fault-tolerance context                                         |
| Multi-AZ recommendations          | No availability requirements context                               |
| Storage class tiering (S3)        | No access pattern data                                             |
| io1/io2 IOPS optimization         | Requires workload analysis                                         |

## Assumptions

- AWS public pricing data is embedded in the binary at build time
- The pricing client already provides methods for EC2 and EBS price lookups
- The finfocus-spec v0.4.7+ proto definitions include GetRecommendationsRequest/Response messages
- Instance type format is consistently "family.size" (e.g., "t2.medium", "m5.large")
- 730 hours per month is the standard calculation for on-demand pricing
- Region-specific pricing is already handled by the region-specific binary builds

## Dependencies

- finfocus-spec v0.4.7 or later (proto definitions)
- Existing pricing client with EC2OnDemandPricePerHour and EBSPricePerGBMonth methods
- pluginsdk for trace_id extraction and logging field constants
