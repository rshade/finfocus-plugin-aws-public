# Feature Specification: Canada and South America Region Support

**Feature Branch**: `003-ca-sa-region-support`
**Created**: 2025-11-29
**Status**: Draft
**Input**: User description: "Add pricing data and build configurations for: ca-central-1 (Canada Central / Montreal) sa-east-1 (South America / São Paulo) Implementation Tasks Add build tags: region_cac1, region_sae1 Create embed files: internal/pricing/embed_cac1.go internal/pricing/embed_sae1.go Update .goreleaser.yaml Update pricing generator Add tests Update documentation Acceptance Criteria Both region binaries build successfully Pricing data is region-specific All tests pass"

## User Scenarios & Testing

### User Story 1 - Canada Region Support (Priority: P1)

As a PulumiCost user deploying infrastructure in Canada, I want to use a dedicated `ca-central-1` plugin binary so that I can estimate costs using accurate, region-specific pricing data without downloading unnecessary data for other regions.

**Why this priority**: Expanding region support is critical for users operating in specific geographies to get accurate cost estimates.

**Independent Test**: Can be tested by building the `ca-central-1` binary and verifying it correctly estimates costs for resources in that region.

**Acceptance Scenarios**:

1. **Given** the `pulumicost-plugin-aws-public-ca-central-1` binary is built, **When** it is started, **Then** it should run successfully and listen on a port.
2. **Given** the running `ca-central-1` plugin, **When** a cost estimation request for a `ca-central-1` resource is made, **Then** it returns a valid cost estimate using Canadian pricing.
3. **Given** the running `ca-central-1` plugin, **When** a cost estimation request for a `us-east-1` resource is made, **Then** it returns an `ERROR_CODE_UNSUPPORTED_REGION` error.

---

### User Story 2 - South America Region Support (Priority: P1)

As a PulumiCost user deploying infrastructure in South America, I want to use a dedicated `sa-east-1` plugin binary so that I can estimate costs using accurate, region-specific pricing data for the São Paulo region.

**Why this priority**: Validates the plugin architecture's ability to support South American AWS regions.

**Independent Test**: Can be tested by building the `sa-east-1` binary and verifying it correctly estimates costs for resources in that region.

**Acceptance Scenarios**:

1. **Given** the `pulumicost-plugin-aws-public-sa-east-1` binary is built, **When** it is started, **Then** it should run successfully and listen on a port.
2. **Given** the running `sa-east-1` plugin, **When** a cost estimation request for a `sa-east-1` resource is made, **Then** it returns a valid cost estimate using South American pricing.
3. **Given** the running `sa-east-1` plugin, **When** a cost estimation request for a `ca-central-1` resource is made, **Then** it returns an `ERROR_CODE_UNSUPPORTED_REGION` error.

---

### Edge Cases

- What happens if pricing data for the new regions cannot be fetched during generation? The generation process should fail immediately (Exit code 1).
- How does the system behave if a user tries to use the `ca-central-1` binary for a region that is technically close but different (e.g. `us-east-1`)? (Should fail as per Acceptance Scenarios).

## Clarifications
### Session 2025-11-29
- Q: What happens if pricing data for the new regions cannot be fetched during generation? → A: Fail the generation process immediately (Exit code 1)
- Q: What are the logging requirements for the new region binaries? → A: Use existing `rs/zerolog` and follow established patterns (stderr, prefixed, trace_id)


## Requirements

### Functional Requirements

- **FR-001**: System MUST support building a standalone binary for the Canada Central (`ca-central-1`) region.
- **FR-002**: System MUST support building a standalone binary for the South America (`sa-east-1`) region.
- **FR-003**: The Canada Central binary MUST include pricing data specific to the `ca-central-1` region.
- **FR-004**: The South America binary MUST include pricing data specific to the `sa-east-1` region.
- **FR-005**: The release process MUST automatically generate and publish artifacts for both new regions.
- **FR-006**: The pricing data generation tools MUST be capable of fetching and formatting data for the new regions.
- **FR-007**: Documentation MUST be updated to reflect support for the new regions.
- FR-008: The system MUST provide automated tests to verify support for the new regions.
- FR-009: The system MUST use existing `rs/zerolog` for logging, following established patterns (stderr, prefixed, trace_id).

### Key Entities

- **Region Binary**: The resulting executable artifact for a specific region.
- **Pricing Data**: The region-specific cost information embedded in the binary.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Binaries for `ca-central-1` and `sa-east-1` build successfully without compilation errors.
- **SC-002**: Generated pricing data files for new regions are non-empty and contain valid JSON.
- **SC-003**: All unit and integration tests, including new region-specific tests, pass (exit code 0).
- **SC-004**: The release artifacts list includes `pulumicost-plugin-aws-public-ca-central-1` and `pulumicost-plugin-aws-public-sa-east-1`.
