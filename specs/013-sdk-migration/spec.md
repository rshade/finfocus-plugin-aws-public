# Feature Specification: SDK Migration and Code Consolidation

**Feature Branch**: `013-sdk-migration`
**Created**: 2025-12-16
**Status**: Draft
**Input**: User description: "SDK Migration and Code Consolidation"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Plugin Developer Eliminates Duplicate Code (Priority: P1)

As a plugin maintainer, I want duplicate code patterns consolidated into shared
helpers so that bug fixes and enhancements only need to be made in one place,
reducing maintenance burden and preventing code drift.

**Why this priority**: Code duplication is the root cause of inconsistent
behavior, missed bug fixes, and increased testing overhead. Eliminating it
first provides immediate maintainability benefits with no external dependencies.

**Independent Test**: Can be fully tested by verifying that EC2 attribute
extraction and RegionConfig loading work identically in all call sites, and
that tests cover the unified implementations.

**Acceptance Scenarios**:

1. **Given** a plugin with duplicate platform/tenancy detection in estimate.go
   and projected.go, **When** I update the platform normalization logic,
   **Then** both EC2 estimation paths use the updated logic from a single
   shared helper.

2. **Given** a plugin with duplicate RegionConfig structs in generate-embeds
   and generate-goreleaser tools, **When** I add a new field to RegionConfig,
   **Then** only one struct definition needs to be modified and both tools
   automatically use the updated definition.

3. **Given** a shared EC2 attribute extractor, **When** I call it with a tags
   map containing `platform=windows` and `tenancy=dedicated`, **Then** it
   returns `OS=Windows` and `Tenancy=Dedicated` with consistent normalization.

---

### User Story 2 - Plugin Integrates SDK Validation Helpers (Priority: P2)

As a plugin maintainer, I want to use SDK-provided request validation helpers
so that validation logic is consistent across the PulumiCost ecosystem and
error messages follow standard formats.

**Why this priority**: Request validation is called on every RPC, and
inconsistent validation leads to confusing error messages. SDK helpers ensure
ecosystem-wide consistency and reduce per-plugin maintenance.

**Independent Test**: Can be tested by sending invalid requests to
GetProjectedCost, GetActualCost, and GetPricingSpec RPCs and verifying error
codes and messages match SDK standards.

**Acceptance Scenarios**:

1. **Given** a GetProjectedCost request with a nil resource descriptor,
   **When** the request is processed, **Then** the response returns
   ERROR_CODE_INVALID_RESOURCE with a standardized error message from the
   SDK validator.

2. **Given** a GetActualCost request missing the Start timestamp, **When** the
   request is processed, **Then** the response returns an appropriate error
   code with trace_id preserved in ErrorDetail.

3. **Given** all three RPC methods (GetProjectedCost, GetActualCost,
   GetPricingSpec), **When** validation fails, **Then** error messages are
   consistent and logging includes trace_id correlation.

---

### User Story 3 - Plugin Uses SDK Environment Variable Handling (Priority: P2)

As a plugin operator, I want the plugin to correctly read environment variables
using SDK-standardized helpers so that configuration works consistently with
pulumicost-core regardless of which variable name is used.

**Why this priority**: Environment variable inconsistencies between core and
plugin caused E2E test failures. SDK helpers with proper fallback logic prevent
integration issues.

**Independent Test**: Can be tested by starting the plugin with
PULUMICOST_PLUGIN_PORT, PORT, or neither and verifying correct behavior in
each case.

**Acceptance Scenarios**:

1. **Given** PULUMICOST_PLUGIN_PORT=8080 is set, **When** the plugin starts,
   **Then** it listens on port 8080.

2. **Given** only PORT=9000 is set (legacy fallback), **When** the plugin
   starts, **Then** it listens on port 9000.

3. **Given** neither port variable is set, **When** the plugin starts,
   **Then** it uses an ephemeral port and announces it via stdout.

---

### User Story 4 - Plugin Uses SDK Property Mapping (Priority: P3)

As a plugin maintainer, I want to use SDK-provided property extraction helpers
so that tag parsing is consistent across plugins and supports features like
default tracking and key aliases.

**Why this priority**: Property mapping reduces ~300 lines of boilerplate and
enables consistent default tracking for billing_detail messages. However, it
depends on mapping package availability.

**Independent Test**: Can be tested by providing various tag combinations to
estimation functions and verifying extracted values and "(defaulted)"
annotations match expected behavior.

**Acceptance Scenarios**:

1. **Given** an EBS resource with tags containing `volume_size=100`, **When**
   estimating cost, **Then** the extractor returns sizeGB=100 using the
   volume_size alias.

2. **Given** an EC2 resource with no platform tag, **When** estimating cost,
   **Then** the extractor returns OS=Linux and the billing_detail indicates
   the default was used.

3. **Given** an RDS resource with partial tag information, **When** estimating
   cost, **Then** the extractor applies reasonable defaults and tracks which
   values were defaulted.

---

### User Story 5 - Plugin Supports ARN-Based Resource Identification (P3)

As a PulumiCost core developer, I want to identify resources using their AWS
ARN so that integration with AWS-native systems (CUR, Config) is simplified
without requiring JSON construction or tag population.

**Why this priority**: ARN support enables richer AWS integration scenarios
but is the newest capability with the most external dependencies.

**Independent Test**: Can be tested by calling GetActualCost with various ARN
formats and verifying correct resource type, region, and service extraction.

**Acceptance Scenarios**:

1. **Given** a GetActualCost request with ARN
   `arn:aws:ec2:us-east-1:123456789012:instance/i-abc123` and tags containing
   `sku=t3.micro`, **When** the request is processed, **Then** the resource
   is identified as EC2 in us-east-1 with instance type t3.micro.

2. **Given** a GetActualCost request with `arn=arn:aws:s3:::my-bucket` (global
   service, no region in ARN), **When** the request is processed, **Then**
   the region is extracted from tags or defaults appropriately.

3. **Given** a GetActualCost request with an invalid ARN format, **When** the
   request is processed, **Then** the system falls back to JSON ResourceId
   parsing, then Tags extraction.

---

### Edge Cases

- What happens when SDK validators return different error formats?
  - Error responses must preserve trace_id, ErrorCode, and ErrorDetail.details
- How does the plugin handle SDK dependency version mismatches?
  - go.mod constraints ensure compatible SDK versions
- What happens when ARN region doesn't match plugin binary's region?
  - Returns ERROR_CODE_UNSUPPORTED_REGION with details
- How does the system handle partially populated ARN+Tags combinations?
  - ARN provides base identification; Tags supplement with SKU
- What happens when the mapping package doesn't support a transformation?
  - Plugin can define custom transformations wrapping SDK primitives

## Requirements *(mandatory)*

### Functional Requirements

#### Internal Code Consolidation (No Dependencies)

- **FR-001**: System MUST provide a single EC2Attributes type with OS and
  Tenancy fields for pricing lookups
- **FR-002**: System MUST provide ExtractEC2AttributesFromTags() that
  normalizes platform to "Linux"/"Windows" and tenancy to
  "Shared"/"Dedicated"/"Host"
- **FR-003**: System MUST provide ExtractEC2AttributesFromStruct() for
  EstimateCost path compatibility
- **FR-004**: System MUST consolidate RegionConfig struct into
  internal/regionsconfig package with Load() and Validate() functions
- **FR-005**: System MUST have generate-embeds and generate-goreleaser tools
  import shared RegionConfig package

#### SDK Environment Variable Handling

- **FR-006**: System MUST use pluginsdk.GetPort() for port configuration with
  PULUMICOST_PLUGIN_PORT taking precedence over PORT
- **FR-007**: System MUST use pluginsdk.GetLogLevel() for logging configuration
- **FR-008**: System MUST eliminate direct os.Getenv() calls for
  PulumiCost-related variables

#### SDK Request Validation

- **FR-009**: System MUST use pluginsdk.ValidateProjectedCostRequest() and
  pluginsdk.ValidateActualCostRequest() for request validation in
  GetProjectedCost, GetActualCost, and GetPricingSpec RPCs
- **FR-010**: System MUST retain custom region validation (plugin binary region
  check) after SDK validation passes
- **FR-011**: System MUST preserve trace_id in all error responses using
  ErrorDetail.details

#### SDK Property Mapping

- **FR-012**: System MUST use mapping package extractors where available for
  tag-to-property conversion
- **FR-013**: System MUST track which property values were defaulted for
  billing_detail annotations
- **FR-014**: System MUST support key aliases (e.g., "size" and "volume_size"
  for EBS)

#### ARN Support

- **FR-015**: System MUST parse AWS ARNs to extract partition, service, region,
  account, and resource components
- **FR-016**: System MUST map ARN service names to Pulumi resource type format
  (e.g., ec2 -> aws:ec2/instance:Instance)
- **FR-017**: System MUST handle global services (S3) where ARN region may be
  empty
- **FR-018**: System MUST try ARN parsing first, then JSON ResourceId, then
  Tags fallback
- **FR-019**: System MUST require SKU from tags when using ARN identification
  (ARN doesn't contain instance type)

### Key Entities

- **EC2Attributes**: Contains OS (Linux/Windows) and Tenancy
  (Shared/Dedicated/Host) for EC2 pricing lookups
- **RegionConfig**: Contains ID, Name, and Tag for AWS region configuration
  with validation rules
- **ARNComponents**: Contains Partition, Service, Region, AccountID,
  ResourceType, and ResourceID parsed from AWS ARN strings

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Duplicate code elimination reduces total lines of
  platform/tenancy detection code by 50%
- **SC-002**: RegionConfig consolidation eliminates 100% of duplicate struct
  definitions (2 definitions -> 1)
- **SC-003**: All existing unit tests pass without modification after SDK
  integration
- **SC-004**: Error response format remains backward compatible - existing
  error handling in pulumicost-core continues to work
- **SC-005**: Plugin starts successfully with any combination of
  PULUMICOST_PLUGIN_PORT, PORT, or neither set
- **SC-006**: ARN parsing correctly identifies resource type and region for 7
  supported AWS services (EC2, EBS, RDS, S3, Lambda, DynamoDB, EKS)
- **SC-007**: Property extraction code reduced by at least 40% (from ~300
  lines to ~180 lines or fewer)
- **SC-008**: Manual E2E verification with pulumicost-core passes using all
  three resource identification methods (ARN, JSON, Tags) - verified post-PR
  via existing CI workflow

## Clarifications

### Session 2025-12-16

- Q: How should implementation proceed when an SDK dependency is not yet
  available? → A: Use pulumicost-spec v0.4.8 (already released with all
  required features)

## Assumptions

- pulumicost-spec v0.4.8 is available and includes all required SDK features:
  env.go helpers, mapping package, validation helpers, and ARN field support
- SDK validation helpers return gRPC status errors compatible with current
  error handling
- All functional requirements (FR-001 through FR-019) can proceed immediately
  with no external blockers

## Scope Boundaries

### In Scope

- Consolidating duplicate EC2 attribute extraction into shared helper
- Consolidating duplicate RegionConfig into shared internal package
- Adopting SDK environment variable helpers
- Adopting SDK request validation helpers
- Adopting SDK property mapping package
- Implementing ARN parsing for GetActualCost

### Out of Scope

- Adding new AWS services beyond current supported set
- Changing the gRPC protocol or adding new RPCs
- Modifying pricing data format or generation
- Adding AWS API calls (plugin remains public-pricing-only)
- Supporting non-AWS ARN formats
