# Feature Specification: Implement Elastic Load Balancing (ALB/NLB) cost estimation

**Feature Branch**: `017-elb-cost-estimation`  
**Created**: 2025-12-19  
**Status**: Draft  
**Input**: User description provided via CLI.

## Clarifications

### Session 2025-12-19
- Q: If the ResourceType is "elb" and the Sku is missing or generic, what should be the default behavior? → A: Default to Application Load Balancer (ALB)
- Q: Should the plugin support alternative or more generic tag names for capacity units? → A: Support both specific (`lcu_per_hour`, `nlcu_per_hour`) and generic (`capacity_units`) tags.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Application Load Balancer (ALB) Cost Estimation (Priority: P1)

As a cloud cost analyst, I want to estimate the monthly cost of an Application Load Balancer based on its fixed hourly rate and its average Load Balancer Capacity Unit (LCU) consumption.

**Why this priority**: ALB is the most common load balancer type in AWS environments and representing its cost accurately (including variable LCU charges) is critical for infrastructure cost visibility.

**Independent Test**: Can be tested by providing an ALB resource descriptor with a specific LCU-per-hour tag and verifying the total monthly cost matches (Fixed Rate * 730) + (LCU Rate * LCU Count * 730).

**Acceptance Scenarios**:

1. **Given** an ALB resource in us-east-1 with `lcu_per_hour` tag set to "5", **When** cost is estimated, **Then** the result should be approximately $45.63 (assuming $0.0225/hr fixed and $0.008/LCU-hr).
2. **Given** an ALB resource with no tags, **When** cost is estimated, **Then** only the fixed hourly cost should be charged.

---

### User Story 2 - Network Load Balancer (NLB) Cost Estimation (Priority: P1)

As a cloud cost analyst, I want to estimate the monthly cost of a Network Load Balancer based on its fixed hourly rate and its average Network Load Balancer Capacity Unit (NLCU) consumption.

**Why this priority**: NLB is used for high-performance networking and has a different capacity unit pricing model (NLCU) which must be supported.

**Independent Test**: Can be tested by providing an NLB resource descriptor with a `nlcu_per_hour` tag and verifying the total monthly cost.

**Acceptance Scenarios**:

1. **Given** an NLB resource in us-east-1 with `nlcu_per_hour` tag set to "2", **When** cost is estimated, **Then** the result should reflect both fixed and NLCU charges.

---

### User Story 3 - Regional Pricing Support (Priority: P2)

As a global cloud user, I want to see ELB costs calculated using the pricing data for the specific region where the load balancer is deployed.

**Why this priority**: AWS pricing varies by region.

**Independent Test**: Can be tested by running the plugin for different regions and verifying the unit prices and total costs match regional AWS pricing.

**Acceptance Scenarios**:

1. **Given** an ALB resource in a region with higher pricing (e.g., sa-east-1), **When** cost is estimated, **Then** the unit price should reflect that region's pricing.

### Edge Cases

- **Missing SKU/LB Type**: If the resource is provided but the LB type (ALB vs NLB) cannot be determined (missing or generic Sku), the system MUST default to Application Load Balancer (ALB).
- **Non-numeric Tags**: If `lcu_per_hour`, `nlcu_per_hour`, or `capacity_units` tags contain non-numeric values, the system should default to 0 capacity units and log a warning.
- **Unsupported LB Types**: If a Classic Load Balancer (CLB) or Gateway Load Balancer (GWLB) is requested, it should be marked as unsupported or estimated as a fixed rate if applicable (though CLB is out of scope).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support "elb", "alb", and "nlb" resource types in the `Supports()` method.
- **FR-002**: System MUST identify the load balancer type (ALB vs NLB) from the resource SKU or descriptor, defaulting to "alb" if undetermined.
- **FR-003**: System MUST calculate fixed monthly costs based on a standard 730-hour month.
- **FR-004**: System MUST calculate variable capacity costs using specific tags (`lcu_per_hour` for ALB, `nlcu_per_hour` for NLB) or a generic `capacity_units` tag as a fallback.
- **FR-005**: System MUST default capacity units to 0 if all relevant tags are missing or invalid.
- **FR-006**: System MUST fetch pricing data specifically for ALB (Load Balancer-Application) and NLB (Load Balancer-Network) from the AWS Pricing API.
- **FR-007**: System MUST provide a detailed `BillingDetail` string explaining the cost breakdown.
    - Example ALB: `ALB, 730 hrs/month, 5 LCU avg/hr`
    - Example NLB: `NLB, 730 hrs/month, 2 NLCU avg/hr`

### Key Entities *(include if feature involves data)*

- **Load Balancer**: Represents an AWS ELB resource.
    - `Type`: ALB or NLB.
    - `Region`: AWS region (e.g., us-east-1).
    - `CapacityUnitsPerMonth`: Calculated average consumption.
- **ELB Pricing**: Data entity containing rates.
    - `FixedHourlyRate`: Cost per LB-hour.
    - `CapacityUnitRate`: Cost per LCU-hour or NLCU-hour.

## Success Criteria *(mandatory)*

### Measurable Outcomes



- **SC-001**: 100% accuracy in cost estimation for ALB and NLB against known AWS public pricing for all supported regions.

- **SC-002**: Successful extraction and application of capacity unit tags in 100% of valid test cases.

- **SC-003**: Plugin identifies ELB resources as "supported" across all 9 standard regional builds.

- **SC-004**: Cost estimation response time remains within established plugin performance benchmarks (under 100ms per resource).



## Assumptions

- **A-001**: Monthly costs are calculated based on a fixed 730-hour month.
- **A-002**: Specific tags (`lcu_per_hour` for ALB, `nlcu_per_hour` for NLB) take precedence over the generic `capacity_units` tag.
- **A-003**: If a capacity unit tag is provided but is not a valid number, it will be treated as 0.
- **A-004**: Application Load Balancers (ALB) and Network Load Balancers (NLB) are the only ELB types supported in this phase.
- **A-005**: If both `lcu_per_hour` and `nlcu_per_hour` tags are provided for a single resource, the system will only use the one corresponding to the identified load balancer type.
