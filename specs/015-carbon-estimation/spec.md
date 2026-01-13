# Feature Specification: Carbon Emission Estimation via Cloud Carbon Footprint

**Feature Branch**: `015-carbon-estimation`
**Created**: 2025-12-19
**Status**: Draft
**Input**: User description: "Implement carbon emission estimation for AWS EC2 instances using Cloud Carbon Footprint methodology"
**Related Issue**: [#120](https://github.com/rshade/finfocus-plugin-aws-public/issues/120)

## Overview

Enable the AWS Public Pricing plugin to estimate carbon emissions (gCO2e) for AWS resources alongside financial cost estimates. This supports the FinFocus "GreenOps" initiative by providing sustainability metrics using the open-source Cloud Carbon Footprint (CCF) methodology.

## User Scenarios & Testing

### User Story 1 - View Carbon Footprint for EC2 Instance (Priority: P1)

As a cloud engineer using FinFocus, I want to see the estimated carbon footprint when I request cost estimates for EC2 instances, so that I can understand both the financial and environmental impact of my infrastructure decisions.

**Why this priority**: This is the core value proposition - providing carbon visibility alongside cost for the most common compute resource type. Without this, the feature delivers no value.

**Independent Test**: Can be fully tested by requesting a cost estimate for any EC2 instance type and verifying carbon metrics are returned in the response.

**Acceptance Scenarios**:

1. **Given** an EC2 instance type (e.g., t3.micro) in a supported region (e.g., us-east-1), **When** I request a projected cost estimate, **Then** the response includes both financial cost AND carbon footprint in gCO2e.

2. **Given** an EC2 instance type with known CPU specifications, **When** the carbon estimate is calculated, **Then** the calculation uses the correct vCPU count, min/max watts for that instance's processor architecture, and region-specific grid emission factor.

3. **Given** an EC2 instance in a low-carbon region (e.g., eu-north-1), **When** compared to the same instance in a high-carbon region (e.g., us-east-1), **Then** the carbon footprint for eu-north-1 is significantly lower (reflecting grid carbon intensity differences).

---

### User Story 2 - Discovery of Carbon Estimation Capabilities (Priority: P2)

As a FinFocus core engine, I want to discover which sustainability metrics a plugin supports, so that I can correctly aggregate and display carbon data from multiple plugins.

**Why this priority**: Essential for integration with the core engine, but depends on P1 being implemented first.

**Independent Test**: Can be tested by calling the Supports() method and verifying the response advertises carbon footprint capability.

**Acceptance Scenarios**:

1. **Given** a request to check support for an EC2 resource, **When** the plugin responds, **Then** the `supported_metrics` field includes `METRIC_KIND_CARBON_FOOTPRINT`.

2. **Given** a request to check support for an unsupported resource type (e.g., DynamoDB), **When** the plugin responds, **Then** the `supported_metrics` field does NOT include `METRIC_KIND_CARBON_FOOTPRINT`.

---

### User Story 3 - Custom Utilization Override (Priority: P3)

As a cloud engineer with knowledge of my actual workload patterns, I want to provide a custom CPU utilization percentage, so that the carbon estimate reflects my actual usage rather than the default 50% assumption.

**Why this priority**: Improves accuracy for power users but the feature works with defaults. Nice-to-have enhancement.

**Independent Test**: Can be tested by sending requests with different utilization percentages and verifying carbon estimates change proportionally.

**Acceptance Scenarios**:

1. **Given** a request with `utilization_percentage = 0.8` (80%), **When** the carbon estimate is calculated, **Then** the result is higher than with the default 50% utilization.

2. **Given** a request with `utilization_percentage = 0.2` (20%), **When** the carbon estimate is calculated, **Then** the result is lower than with the default 50% utilization.

3. **Given** a request with no utilization percentage specified, **When** the carbon estimate is calculated, **Then** the default 50% utilization is used.

4. **Given** a per-resource `utilization_percentage` override, **When** conflicting with a global request-level value, **Then** the per-resource value takes precedence.

---

### Edge Cases

- What happens when an instance type is not found in the CCF data?
  - Return financial cost normally, return carbon = 0 with explanatory message
- What happens when a region has no grid emission factor?
  - Use global average fallback (0.00039278 metric tons CO2eq/kWh)
- What happens when utilization_percentage is outside valid range (0.0-1.0)?
  - Clamp to valid range (0.0 minimum, 1.0 maximum)
- How does the system handle instance types with GPU workloads?
  - GPU power consumption is out of scope for v1; document limitation

## Requirements

### Functional Requirements

- **FR-001**: System MUST embed CCF instance specification data (vCPU count, CPU architecture, min/max watts) for at least 500 AWS instance types.

- **FR-002**: System MUST embed grid emission factors for all currently supported AWS regions (us-east-1, us-east-2, us-west-1, us-west-2, ca-central-1, eu-west-1, eu-north-1, ap-southeast-1, ap-southeast-2, ap-northeast-1, ap-south-1, sa-east-1).

- **FR-003**: System MUST calculate carbon emissions using the CCF formula:
  - Average Watts = Min Watts + (Utilization × (Max Watts - Min Watts))
  - Energy (kWh) = (Average Watts × vCPU Count × Hours) / 1000
  - Energy with PUE = Energy × 1.135 (AWS Power Usage Effectiveness)
  - Carbon (gCO2e) = Energy with PUE × Grid Intensity × 1,000,000

- **FR-004**: System MUST return carbon metrics in grams of CO2 equivalent (gCO2e) as the standardized unit.

- **FR-005**: System MUST include `METRIC_KIND_CARBON_FOOTPRINT` in the `supported_metrics` field of `SupportsResponse` for EC2 resources.

- **FR-006**: System MUST return carbon metrics in the `impact_metrics` array of `GetProjectedCostResponse` when carbon estimation succeeds.

- **FR-007**: System MUST use a default CPU utilization of 50% when no `utilization_percentage` is provided.

- **FR-008**: System MUST respect `utilization_percentage` from the request when provided (valid range: 0.0 to 1.0).

- **FR-009**: System MUST use per-resource `utilization_percentage` (from ResourceDescriptor) when it differs from the global request value.

- **FR-010**: System MUST return carbon = 0 with explanatory message for unknown instance types, without failing the financial cost calculation.

- **FR-011**: System MUST use the global average grid emission factor (0.00039278 metric tons/kWh) for regions without specific data.

- **FR-012**: System MUST include unit specification ("gCO2e") in the ImpactMetric response.

### Key Entities

- **Instance Specification**: Maps EC2 instance type to vCPU count, CPU microarchitecture, and power consumption characteristics (min/max watts)

- **Grid Emission Factor**: Maps AWS region to carbon intensity of local electricity grid (metric tons CO2eq per kWh)

- **Impact Metric**: Standardized sustainability measurement containing metric kind, numeric value, and unit of measurement

## Success Criteria

### Measurable Outcomes

- **SC-001**: Carbon estimates are returned for at least 95% of common EC2 instance types (t3, m5, c5, r5 families).

- **SC-002**: Carbon estimates for the same instance type vary by at least 10x between lowest-carbon region (eu-north-1) and highest-carbon region, reflecting real grid differences.

- **SC-003**: Carbon calculation adds less than 1ms to request processing time (lookup-based, no external API calls).

- **SC-004**: Financial cost estimates continue to work correctly when carbon data is unavailable for an instance type.

- **SC-005**: All supported regions have accurate grid emission factors matching CCF reference data.

## Assumptions

- CPU utilization of 50% is a reasonable default for cloud workloads (per CCF methodology for hyperscale data centers)
- AWS PUE of 1.135 is accurate for all AWS regions (CCF published value)
- Grid emission factors are static and updated infrequently (acceptable for v1; real-time API integration out of scope)
- GPU power consumption is not included in v1 estimates
- Memory/storage/networking carbon footprint is not included in v1 estimates
- Embodied emissions (manufacturing carbon cost) are not included in v1 estimates

## Out of Scope

- Embodied emissions (manufacturing carbon cost of hardware)
- Memory, storage, and networking carbon estimation
- Real-time grid intensity via Electricity Maps API
- GPU power consumption modeling
- Spot/Reserved instance carbon adjustments
- RDS, Lambda, EKS, or other service types (EC2 only for v1)

## Attribution Requirements

Cloud Carbon Footprint data is licensed under Apache 2.0. The following attribution must be included:

- Add to project NOTICE file or documentation:
  - [Cloud Carbon Footprint](https://www.cloudcarbonfootprint.org/)
  - Copyright 2021 Thoughtworks, Inc.
  - Licensed under the Apache License, Version 2.0

## Dependencies

- `finfocus-spec` v0.4.10+ (provides MetricKind enum, ImpactMetric message, supported_metrics field)
