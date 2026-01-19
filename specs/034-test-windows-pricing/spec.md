# Feature Specification: Windows vs Linux Pricing Differentiation Integration Test

**Feature Branch**: `034-test-windows-pricing`  
**Created**: 2026-01-18  
**Status**: Draft  
**Input**: User description: "test: add integration test for Windows vs Linux pricing differentiation"

## Clarifications

### Session 2026-01-18
- Q: For tenancy differentiation, should we focus on "Dedicated Instance" or "Dedicated Host"? → A: Dedicated Instance
- Q: How should the integration test handle cases where the embedded pricing data is missing a specific platform/tenancy combination? → A: Fail test (Enforce data availability)
- Q: Which AWS region should be used as the primary target for these integration tests? → A: us-east-1 (Primary test region)
- Q: Should the integration test suite only focus on Linux and Windows, or should it verify all platforms supported by the plugin (e.g., RHEL, SUSE)? → A: Test all supported platforms (Comprehensive)
- Q: Should the integration tests cover both x86 and ARM (Graviton) architectures? → A: Test both x86 and ARM (Graviton)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Operating System Price Differentiation (Priority: P1)

As a cloud cost analyst, I want the system to accurately reflect the higher cost of Windows EC2 instances compared to Linux instances, so that I don't underestimate my cloud infrastructure budget.

**Why this priority**: Correctly identifying platform-specific costs is critical because Windows instances typically cost 30-50% more than Linux. Under-estimating this is a significant risk for financial planning.

**Independent Test**: Can be fully tested by requesting pricing for the same instance type (e.g., t3.medium) in `us-east-1` with different platform tags (linux vs windows) and comparing the monthly cost.

**Acceptance Scenarios**:

1. **Given** a request for a t3.medium instance on Linux in `us-east-1`, **When** comparing it to a t3.medium instance on Windows in the same region, **Then** the Windows instance monthly cost must be higher.
2. **Given** a Windows EC2 cost estimate, **When** reviewing the billing details, **Then** the details must explicitly mention "Windows" as the operating system.

---

### User Story 2 - Dedicated Instance Price Differentiation (Priority: P2)

As a cloud architect, I want to see the cost impact of choosing Dedicated tenancy over Shared tenancy for my EC2 instances, so that I can make informed decisions about resource isolation vs cost.

**Why this priority**: Tenancy is a major cost driver in AWS. Validating that the system accounts for this premium ensures the accuracy of architectural cost-benefit analyses.

**Independent Test**: Can be fully tested by requesting pricing for the same instance type and platform (e.g., m5.large on Linux) in `us-east-1` with different tenancy tags (shared vs dedicated) and comparing the monthly cost.

**Acceptance Scenarios**:

1. **Given** a request for a shared tenancy instance in `us-east-1`, **When** comparing it to a dedicated tenancy instance of the same type, **Then** the dedicated tenancy monthly cost must be higher.
2. **Given** a dedicated tenancy cost estimate, **When** reviewing the billing details, **Then** the details must explicitly mention the "Dedicated" tenancy.

---

### User Story 3 - CPU Architecture Price Differentiation (Priority: P3)

As a performance engineer, I want to see the cost savings of moving to ARM-based Graviton instances, so that I can optimize our infrastructure costs.

**Why this priority**: Architecture is a key cost optimization lever in AWS. Linux on ARM is generally ~20% cheaper than x86.

**Independent Test**: Can be fully tested by requesting pricing for an ARM-based instance (e.g., t4g.medium) vs an x86-based instance (e.g., t3.medium) in `us-east-1`.

**Acceptance Scenarios**:

1. **Given** a request for an ARM64 instance, **When** comparing it to the equivalent x86_64 instance, **Then** the pricing must reflect the architecture-specific rate.
2. **Given** an ARM64 estimate, **When** reviewing the billing details, **Then** the details must explicitly mention "arm64" or "Graviton".

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST identify the "platform" tag in resource descriptors to differentiate between Windows and Linux operating systems.
- **FR-002**: System MUST identify the "tenancy" tag in resource descriptors to differentiate between Shared and Dedicated Instance hosting.
- **FR-003**: System MUST return platform-specific pricing when valid platform tags (e.g., "windows", "linux", "rhel", "suse") are specified for an EC2 resource.
- **FR-004**: System MUST return higher costs for Windows than for Linux on identical instance types and regions.
- **FR-005**: System MUST return Dedicated Instance pricing when "dedicated" tenancy is specified.
- **FR-006**: System MUST include the platform name and tenancy type in the human-readable billing breakdown.
- **FR-007**: System MUST differentiate pricing based on CPU architecture (x86_64 vs arm64/Graviton) when applicable.

### Assumptions

- **AS-001**: The pricing data source contains valid, non-zero on-demand prices for all supported platform and tenancy combinations for common EC2 instance types.
- **AS-002**: The platform detection logic uses tags as the primary source of truth for resource metadata.
- **AS-003**: The integration tests primarily target the `us-east-1` region for consistency and data coverage.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of tested instance types show a higher monthly cost for Windows than for Linux when all other parameters are identical.
- **SC-002**: Windows pricing for common instance types (e.g., T3, M5) falls within the expected ratio of 1.3x to 2.0x of the equivalent Linux pricing.
- **SC-003**: Dedicated tenancy monthly costs are consistently higher than Shared tenancy costs for the same instance configuration.
- **SC-004**: 100% of generated cost breakdowns explicitly mention the requested platform (e.g., "Windows", "Linux", "RHEL") and tenancy.
- **SC-005**: Integration tests MUST fail if requested platform/tenancy pricing data is missing or returns $0, ensuring no silent regressions.