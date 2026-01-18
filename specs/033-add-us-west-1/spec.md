# Feature Specification: Add us-west-1 (N. California) Region Support

**Feature Branch**: `033-add-us-west-1`
**Created**: 2026-01-18
**Status**: Draft
**Input**: User description: (See original prompt)

## Clarifications

### Session 2026-01-18
- Q: How does system handle resources that are available in other regions but not in `us-west-1`? → A: Return explicit error (e.g., `Code: UnsupportedResource`)
- Q: What happens when AWS Pricing API for `us-west-1` is unavailable during build? → A: Fail build immediately

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Estimate Costs in us-west-1 (Priority: P1)

As a user, I want to estimate costs for AWS resources deployed in the `us-west-1` (N. California) region so that I can manage my cloud spend in that region.

**Why this priority**: This is the core purpose of the feature - to enable cost estimation for this specific region.

**Independent Test**: Can be tested by running the plugin with a resource located in `us-west-1` and verifying a cost estimate is returned.

**Acceptance Scenarios**:

1. **Given** the plugin is running, **When** I request a cost estimate for an EC2 instance in `us-west-1`, **Then** I receive a valid cost estimate based on N. California pricing.
2. **Given** the plugin is running, **When** I request a cost estimate for an unknown region (e.g., `us-west-9`), **Then** I receive an error indicating the region is not supported.

---

### User Story 2 - Deploy Plugin with us-west-1 Support (Priority: P1)

As a DevOps engineer, I want a Docker image that includes the `us-west-1` regional binary so that I can deploy the plugin in my environment and support all US commercial regions.

**Why this priority**: Users consume the plugin primarily via Docker images, and incomplete region support limits utility.

**Independent Test**: Can be tested by pulling the Docker image and verifying the `us-west-1` binary exists, runs, and responds to health checks.

**Acceptance Scenarios**:

1. **Given** the Docker image `finfocus-plugin-aws-public`, **When** I inspect the installed binaries, **Then** I see `finfocus-plugin-aws-public-us-west-1`.
2. **Given** the Docker container is running, **When** I check the health endpoint, **Then** it reports the `us-west-1` service is healthy.

---

### User Story 3 - Carbon Estimation for us-west-1 (Priority: P2)

As a user, I want carbon estimates for `us-west-1` to accurately reflect the N. California grid intensity (WECC grid factor: 0.000322 metric tons CO2e/kWh).

**Why this priority**: Carbon estimation is a key differentiation of the plugin, and using incorrect grid factors would lead to inaccurate reporting.

**Independent Test**: Can be tested by requesting an estimate and checking the carbon emission factors used in the calculation.

**Acceptance Scenarios**:

1. **Given** a resource in `us-west-1`, **When** I request a cost/carbon estimate, **Then** the carbon calculation uses the CAISO (California) grid emission factor.

### Edge Cases

- **Resolved**: If the AWS Pricing API is unreachable during the build process for `us-west-1`, the build MUST fail immediately (do not use fallback/stale data).
- **Resolved**: When a resource type is available in other regions but not in `us-west-1`, the system returns an `UnsupportedResource` error code (not $0.00).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support `us-west-1` as a valid region identifier in configuration and API calls.
- **FR-002**: System MUST provide accurate pricing data for supported AWS services (EC2, EBS, RDS, etc.) specifically for the `us-west-1` region.
- **FR-003**: System MUST include a dedicated regional binary for `us-west-1` in the distribution artifacts (Docker image).
- **FR-004**: System MUST use the correct grid emission factor (CAISO) for carbon estimation in `us-west-1`.
- **FR-005**: System MUST expose the `us-west-1` service on a unique port (8010) within the Docker container to avoid conflicts.
- **FR-006**: System MUST include `us-west-1` in health checks and verification loops to ensure service reliability.
- **FR-007**: System MUST return an explicit error code (`ERROR_CODE_INVALID_RESOURCE`, proto code 6) if a user requests a cost estimate for a valid AWS resource type that is not available in the `us-west-1` region. This maps to the semantic concept of "UnsupportedResource" in user-facing messages.

### Key Entities *(include if feature involves data)*

- **Region**: Represents an AWS geographical area (specifically `us-west-1`). Attributes include ID, Name, and Build Tag.
- **Pricing Data**: The set of costs for resources in a specific region, embedded into the binary.
- **Grid Factor**: The carbon intensity (gCO2e/kWh) of the power grid in the region.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can successfully retrieve cost estimates for at least 5 different resource types (EC2, EBS, RDS, S3, Lambda) in `us-west-1`.
- **SC-002**: The Docker image successfully starts all 12 regional binaries (including `us-west-1`) within 30 seconds.
- **SC-003**: Carbon estimates for `us-west-1` differ from `us-east-1` and `us-west-2` for identical resources, reflecting the specific CAISO grid factor.
- **SC-004**: Build process generates a valid executable for `us-west-1` that passes all internal self-tests and is comparable in size to other regional binaries.
