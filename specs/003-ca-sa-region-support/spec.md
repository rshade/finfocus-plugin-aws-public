# Feature Specification: Canada and South America Region Support

**Feature Branch**: `003-ca-sa-region-support`
**Created**: 2025-11-20
**Status**: Draft
**Input**: User description: "Add support for Canada and South America regions (ca-central-1, sa-east-1)"

## User Scenarios & Testing

### User Story 1 - Canada Region Cost Estimation (Priority: P1)

As a PulumiCost user with AWS resources in Canada (ca-central-1), I want to get accurate cost estimates for my EC2 instances and EBS volumes so that I can budget for my Canadian infrastructure deployments.

**Why this priority**: Canada is a popular region for businesses requiring data residency compliance in North America. Providing cost estimation enables users to make informed infrastructure decisions.

**Independent Test**: Can be fully tested by requesting cost estimates for EC2 and EBS resources in ca-central-1 and verifying accurate pricing data is returned.

**Acceptance Scenarios**:

1. **Given** a ca-central-1 region binary is built, **When** a user requests cost estimation for a t3.micro EC2 instance, **Then** the system returns the correct hourly rate and monthly cost with USD currency
2. **Given** a ca-central-1 region binary is built, **When** a user requests cost estimation for a gp3 EBS volume, **Then** the system returns the correct GB-month rate and calculated monthly cost

---

### User Story 2 - South America Region Cost Estimation (Priority: P1)

As a PulumiCost user with AWS resources in South America (sa-east-1), I want to get accurate cost estimates for my EC2 instances and EBS volumes so that I can budget for my São Paulo infrastructure deployments.

**Why this priority**: South America is an important region for latency-sensitive applications serving Latin American users. Cost estimation enables proper budgeting for this market.

**Independent Test**: Can be fully tested by requesting cost estimates for EC2 and EBS resources in sa-east-1 and verifying accurate pricing data is returned.

**Acceptance Scenarios**:

1. **Given** a sa-east-1 region binary is built, **When** a user requests cost estimation for a m5.large EC2 instance, **Then** the system returns the correct hourly rate and monthly cost with USD currency
2. **Given** a sa-east-1 region binary is built, **When** a user requests cost estimation for an io1 EBS volume, **Then** the system returns the correct GB-month rate and calculated monthly cost

---

### User Story 3 - Region Mismatch Rejection (Priority: P2)

As a PulumiCost system, I want to reject cost estimation requests for resources in regions not supported by the loaded binary so that users get clear feedback about using the correct regional binary.

**Why this priority**: Ensures system integrity by preventing incorrect pricing from being returned when the wrong regional binary is used.

**Independent Test**: Can be fully tested by sending a ca-central-1 resource request to an sa-east-1 binary and verifying proper error response.

**Acceptance Scenarios**:

1. **Given** a ca-central-1 binary is running, **When** a user requests cost estimation for a resource in us-east-1, **Then** the system returns ERROR_CODE_UNSUPPORTED_REGION with details about the mismatch
2. **Given** an sa-east-1 binary is running, **When** a user requests cost estimation for a resource in eu-west-1, **Then** the system returns ERROR_CODE_UNSUPPORTED_REGION with details about the mismatch

---

### Edge Cases

- What happens when pricing data for a specific instance type is not available in the region?
- How does the system handle concurrent cost estimation requests?
- What happens if the embedded pricing JSON is corrupted during build?

## Requirements

### Functional Requirements

- **FR-001**: System MUST provide build tags `region_cac1` for ca-central-1 and `region_sae1` for sa-east-1
- **FR-002**: System MUST create embed files `internal/pricing/embed_cac1.go` and `internal/pricing/embed_sae1.go` with region-specific pricing data
- **FR-003**: System MUST update `.goreleaser.yaml` with build configurations for both new regions
- **FR-004**: System MUST update the pricing generator tool to support ca-central-1 and sa-east-1 regions
- **FR-005**: System MUST return accurate EC2 instance pricing for supported instance types in both regions
- **FR-006**: System MUST return accurate EBS volume pricing for supported volume types (gp2, gp3, io1, io2) in both regions
- **FR-007**: System MUST return stub responses ($0 estimates) for unsupported services (S3, Lambda, RDS, DynamoDB) in both regions
- **FR-008**: System MUST reject requests for resources in regions not matching the binary's embedded region with ERROR_CODE_UNSUPPORTED_REGION
- **FR-009**: System MUST update the fallback embed file to exclude the new region tags
- **FR-010**: System MUST maintain thread-safety for concurrent pricing lookups

### Key Entities

- **Regional Pricing Data**: JSON files containing EC2 and EBS pricing for each region, embedded at build time
- **Build Tag**: Go build constraint (region_cac1, region_sae1) that selects the appropriate embed file
- **Regional Binary**: Compiled plugin binary containing only one region's pricing data

## Success Criteria

### Measurable Outcomes

- **SC-001**: Both ca-central-1 and sa-east-1 region binaries build successfully without errors
- **SC-002**: Each binary is under 20MB in size (consistent with existing regional binaries)
- **SC-003**: Region mismatch detection responds in under 100ms
- **SC-004**: All existing tests pass plus new region-specific tests pass
- **SC-005**: Concurrent cost estimation requests are handled correctly without race conditions
- **SC-006**: Cost estimates are returned with correct USD currency and billing details
- **SC-007**: 100% of region mismatch requests are correctly rejected with appropriate error codes
