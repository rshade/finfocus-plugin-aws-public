# Feature Specification: E2E Test Support and Validation

**Feature Branch**: `001-e2e-test-support`
**Created**: 2025-12-02
**Status**: Draft
**Input**: GitHub Issue #26 - Add E2E test support and validation endpoints

## Clarifications

### Session 2025-12-02

- Q: How should the plugin handle invalid FINFOCUS_TEST_MODE values? →
  A: Treat as disabled (production mode) and log warning at startup

## Problem Statement

The finfocus-plugin-aws-public plugin needs to support end-to-end testing
scenarios within the finfocus ecosystem. Currently, there is no standardized
way to:

- Validate cost calculations against expected ranges during automated tests
- Enable test-specific behaviors without affecting production usage
- Document expected costs for common test resources
- Provide enhanced diagnostics for test failure debugging

This feature enables reliable E2E testing of the cost estimation pipeline by
ensuring deterministic, validatable, and well-documented behavior for test
scenarios.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Validate Projected Costs in E2E Tests (Priority: P1)

As an E2E test framework (finfocus-core), I need to verify that projected
cost calculations for standard test resources fall within expected ranges, so
that I can detect regressions in cost estimation accuracy.

**Why this priority**: Core validation capability - without accurate cost
validation, E2E tests cannot verify the primary function of the plugin.

**Independent Test**: Can be fully tested by requesting projected costs for a
t3.micro EC2 instance and verifying the response matches documented expected
ranges.

**Acceptance Scenarios**:

1. **Given** a t3.micro EC2 instance in us-east-1, **When** requesting
   projected cost, **Then** the hourly rate is approximately $0.0104
   (within 1% tolerance)
2. **Given** an 8GB gp2 EBS volume in us-east-1, **When** requesting projected
   cost, **Then** the monthly cost is approximately $0.80 (within 5% tolerance)
3. **Given** any supported test resource, **When** requesting projected cost,
   **Then** the response includes sufficient detail for validation (unit price,
   currency, monthly cost, billing detail)

---

### User Story 2 - Validate Actual Cost Fallback in E2E Tests (Priority: P2)

As an E2E test framework, I need to verify that actual cost fallback
calculations correctly prorate monthly costs based on runtime duration, so that
I can validate time-based cost estimation.

**Why this priority**: Validates the fallback actual cost feature which uses
projected costs to estimate runtime costs.

**Independent Test**: Can be tested by requesting actual costs for a 30-minute
runtime and verifying the prorated calculation.

**Acceptance Scenarios**:

1. **Given** a t3.micro EC2 instance that ran for 30 minutes, **When**
   requesting actual cost fallback, **Then** the cost is approximately $0.0052
   (30/60 * hourly rate)
2. **Given** an 8GB gp2 EBS volume with 30-minute runtime, **When** requesting
   actual cost, **Then** the cost is correctly prorated from monthly rate
3. **Given** a zero-duration runtime, **When** requesting actual cost,
   **Then** the cost is $0 with appropriate explanation

---

### User Story 3 - Enhanced Test Diagnostics (Priority: P3)

As a test developer debugging a failing E2E test, I need enhanced logging and
error messages when test mode is enabled, so that I can quickly identify the
root cause of test failures.

**Why this priority**: Improves developer experience and reduces debugging
time, but not required for basic test functionality.

**Independent Test**: Can be tested by enabling test mode and verifying that
log output includes additional context for requests and errors.

**Acceptance Scenarios**:

1. **Given** test mode is enabled, **When** a cost request is processed,
   **Then** logs include trace ID, resource details, and calculation breakdown
2. **Given** test mode is enabled and an error occurs, **When** the error is
   returned, **Then** the error message includes specific diagnostic information
3. **Given** test mode is disabled, **When** the same operations occur,
   **Then** behavior matches production (no extra overhead)

---

### User Story 4 - Access Expected Cost Ranges (Priority: P4)

As an E2E test framework, I need to query expected cost ranges for test
resources, so that I can set appropriate validation tolerances in my tests.

**Why this priority**: Convenience feature that improves test maintainability;
tests can still hard-code expected values without this.

**Independent Test**: Can be tested by requesting expected cost range for a
known resource and verifying the response includes min, max, and expected
values.

**Acceptance Scenarios**:

1. **Given** a t3.micro EC2 instance configuration, **When** querying expected
   cost range, **Then** the response includes minimum, maximum, and expected
   cost values
2. **Given** an unsupported resource type, **When** querying expected cost
   range, **Then** the response indicates no expected range is available
3. **Given** any test resource, **When** querying expected cost range,
   **Then** tolerance percentages are included (1% for EC2, 5% for EBS)

---

### Edge Cases

- What happens when test mode is enabled but resource pricing data is
  unavailable?
- How does the system handle requests for resources not in the expected cost
  range database?
- What happens when pricing data changes and expected ranges become stale?
- When FINFOCUS_TEST_MODE is set to invalid values (e.g., "maybe", "1",
  "yes", empty string), the system treats it as disabled (production mode)
  and logs a warning at startup

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST maintain backward compatibility - all existing
  production behaviors remain unchanged when test mode is disabled
- **FR-002**: System MUST support test mode activation via
  FINFOCUS_TEST_MODE environment variable
- **FR-003**: System MUST return deterministic cost calculations for the same
  resource configuration (no randomness or external dependencies in test mode)
- **FR-004**: System MUST support projected cost requests for t3.micro EC2
  instances in us-east-1
- **FR-005**: System MUST support projected cost requests for gp2 EBS volumes
  (default 8GB) in us-east-1
- **FR-006**: System MUST support actual cost fallback calculations with
  runtime-based proration
- **FR-007**: System MUST provide expected cost range data for standard test
  resources
- **FR-008**: System MUST provide enhanced logging when test mode is enabled
  (including trace IDs, calculation details)
- **FR-009**: System MUST include tolerance information with expected cost
  ranges (1% for EC2, 5% for EBS)
- **FR-010**: System MUST not make external network calls in test mode (all
  pricing data embedded)
- **FR-011**: System MUST treat invalid FINFOCUS_TEST_MODE values as disabled
  (production mode) and log a warning at startup; only "true" enables test mode

### Key Entities

- **Test Resource Configuration**: Represents a standard resource used in E2E
  tests (EC2 t3.micro, EBS gp2 8GB)
- **Expected Cost Range**: Defines minimum, maximum, and expected cost values
  with tolerance percentage for a test resource
- **Test Mode Context**: Environment-based flag that enables test-specific
  behaviors and diagnostics

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All E2E test scenarios from finfocus-core pass with the plugin
  integration
- **SC-002**: Cost calculation responses return in under 100ms in test mode
- **SC-003**: Projected cost calculations for t3.micro EC2 are within 1% of
  documented expected values
- **SC-004**: Projected cost calculations for gp2 EBS volumes are within 5% of
  documented expected values
- **SC-005**: Actual cost fallback calculations match the formula:
  `projected_monthly_cost * (runtime_hours / 730)`
- **SC-006**: Test mode does not impact production performance when disabled
  (zero overhead)
- **SC-007**: Memory usage remains under 50MB during test execution
- **SC-008**: 100% of test resource configurations documented in the expected
  cost ranges are supported

## Assumptions

- The pricing data embedded in the plugin is kept reasonably up-to-date with
  AWS public pricing
- E2E tests will primarily use us-east-1 region for cost validation
- The 730 hours/month assumption for monthly cost calculations is acceptable
  for testing
- Test mode activation via environment variable is sufficient (no runtime API
  needed)
- Expected cost ranges can be based on current AWS public pricing with
  documented reference dates

## Dependencies

- E2E test implementation in finfocus-core (#177)
- Fallback actual cost implementation (#24) - completed
- Zerolog structured logging (#005) - completed
- Plugin release v0.0.1 (#23)

## Out of Scope

- Real-time pricing updates from AWS (uses embedded pricing data)
- Multi-region test resource configurations (focus on us-east-1)
- Test mode for services beyond EC2 and EBS (S3, Lambda, RDS, DynamoDB remain
  stubs)
- Performance benchmarking framework (separate concern)
- Test data generation or fixture management
