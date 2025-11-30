# Feature Specification: Add support for additional US regions (us-west-1, us-gov-west-1, us-gov-east-1)

**Feature Branch**: `006-add-us-regions`  
**Created**: 2025-11-30  
**Status**: Draft  
**Input**: User description: "title:	Add support for additional US regions (us-west-1, us-gov-west-1, us-gov-east-1)
state:	OPEN
author:	rshade
labels:	enhancement
comments:	0
assignees:	
projects:	
milestone:	
number:	4
--
### Description
Add pricing data and build configurations for:

- **us-west-1** (N. California)
- **us-gov-west-1** (AWS GovCloud US-West)
- **us-gov-east-1** (AWS GovCloud US-East)

**Note:** GovCloud regions may have different pricing than commercial regions.

### Implementation Tasks
- [ ] Add build tags: `region_usw1`, `region_govw1`, `region_gove1`
- [ ] Create embed files
- [ ] Update `.goreleaser.yaml`
- [ ] Research GovCloud pricing differences
- [ ] Update pricing generator
- [ ] Add tests
- [ ] Update documentation

### Acceptance Criteria
- All US region binaries build successfully
- GovCloud pricing is accurate
- Tests pass"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Get pricing for us-west-1 region (Priority: P1)

As a user estimating AWS costs, I want to get accurate pricing data for the us-west-1 (N. California) region so I can plan my infrastructure costs effectively.

**Why this priority**: This is a core commercial region that users frequently deploy to, enabling immediate value for cost estimation.

**Independent Test**: Can be fully tested by requesting projected costs for resources in us-west-1 and verifying accurate pricing data is returned.

**Acceptance Scenarios**:

1. **Given** a resource configuration in us-west-1, **When** requesting projected costs, **Then** accurate pricing data for us-west-1 is returned
2. **Given** an invalid region request, **When** requesting projected costs, **Then** appropriate error is returned

---

### User Story 2 - Get pricing for us-gov-west-1 region (Priority: P1)

As a user in government or regulated industries, I want to get accurate pricing data for the us-gov-west-1 (AWS GovCloud US-West) region so I can estimate costs for compliant infrastructure.

**Why this priority**: GovCloud regions serve specialized users with compliance requirements, providing essential functionality for this user segment.

**Independent Test**: Can be fully tested by requesting projected costs for resources in us-gov-west-1 and verifying GovCloud-specific pricing data is returned.

**Acceptance Scenarios**:

1. **Given** a resource configuration in us-gov-west-1, **When** requesting projected costs, **Then** accurate GovCloud pricing data for us-gov-west-1 is returned
2. **Given** a request for GovCloud pricing, **When** the pricing differs from commercial regions, **Then** the correct GovCloud rates are applied

---

### User Story 3 - Get pricing for us-gov-east-1 region (Priority: P1)

As a user in government or regulated industries, I want to get accurate pricing data for the us-gov-east-1 (AWS GovCloud US-East) region so I can estimate costs for compliant infrastructure.

**Why this priority**: GovCloud regions serve specialized users with compliance requirements, providing essential functionality for this user segment.

**Independent Test**: Can be fully tested by requesting projected costs for resources in us-gov-east-1 and verifying GovCloud-specific pricing data is returned.

**Acceptance Scenarios**:

1. **Given** a resource configuration in us-gov-east-1, **When** requesting projected costs, **Then** accurate GovCloud pricing data for us-gov-east-1 is returned
2. **Given** a request for GovCloud pricing, **When** the pricing differs from commercial regions, **Then** the correct GovCloud rates are applied

### Edge Cases

- What happens when GovCloud pricing data is unavailable or outdated?
- How does the system handle requests for regions not yet supported?
- What if commercial and GovCloud pricing structures differ significantly?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide accurate pricing data for us-west-1 region
- **FR-002**: System MUST provide accurate pricing data for us-gov-west-1 region  
- **FR-003**: System MUST provide accurate pricing data for us-gov-east-1 region
- **FR-004**: System MUST use GovCloud-specific pricing for GovCloud regions when it differs from commercial pricing
- **FR-005**: System MUST build successfully for all new US region configurations
- **FR-006**: System MUST pass all tests for the new regions

### Key Entities *(include if feature involves data)*

- **Region**: Represents an AWS region with its pricing data and configuration
- **Pricing Data**: Contains cost information for AWS services within a specific region
- **Build Configuration**: Defines how binaries are built for specific regions

## Assumptions

- GovCloud pricing data is available and can be obtained from AWS pricing APIs
- The pricing structure for GovCloud regions follows similar patterns to commercial regions
- Build configurations can be extended to support additional regions without major architectural changes

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All US region binaries build successfully without errors
- **SC-002**: GovCloud pricing data is accurate and reflects actual AWS pricing
- **SC-003**: All automated tests pass for the new regions
- **SC-004**: Users can successfully request and receive projected costs for all three new regions