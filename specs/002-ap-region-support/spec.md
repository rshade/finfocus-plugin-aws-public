# Feature Specification: Asia Pacific Region Support

**Feature Branch**: `002-ap-region-support`
**Created**: 2025-11-18
**Status**: Draft
**Input**: User description: "Add support for Asia Pacific regions (ap-southeast-1, ap-southeast-2, ap-northeast-1, ap-south-1)"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Singapore Region Cost Estimation (Priority: P1)

A FinFocus user deploying AWS resources to ap-southeast-1 (Singapore) needs accurate cost estimates for their infrastructure. They should receive regional pricing data specific to Singapore, not default US pricing.

**Why this priority**: Singapore (ap-southeast-1) is one of the most heavily used Asia Pacific regions for global SaaS applications, making it critical for users with Asian customer bases.

**Independent Test**: Can be fully tested by deploying the ap-southeast-1 binary, requesting cost estimates for EC2 and EBS resources in that region, and verifying the returned pricing matches AWS's published Singapore pricing.

**Acceptance Scenarios**:

1. **Given** a FinFocus deployment with the ap-southeast-1 plugin binary, **When** a user requests cost estimate for a t3.micro EC2 instance in ap-southeast-1, **Then** the system returns the correct Singapore hourly rate and monthly cost projection
2. **Given** a FinFocus deployment with the ap-southeast-1 plugin binary, **When** a user requests cost estimate for a 100GB gp3 EBS volume in ap-southeast-1, **Then** the system returns the correct Singapore GB-month rate and total monthly cost
3. **Given** a user attempts to estimate resources in ap-southeast-1, **When** they have the wrong regional binary loaded (e.g., us-east-1), **Then** they receive a clear error message indicating region mismatch with details about which binary is needed

---

### User Story 2 - Sydney Region Cost Estimation (Priority: P2)

A FinFocus user deploying AWS resources to ap-southeast-2 (Sydney) needs accurate cost estimates for their Australian infrastructure deployments.

**Why this priority**: Sydney is the primary AWS region for Australian customers and often has distinct pricing from Singapore due to local infrastructure costs.

**Independent Test**: Can be tested by deploying the ap-southeast-2 binary and verifying cost estimates for Sydney resources return correct regional pricing independent of other region binaries.

**Acceptance Scenarios**:

1. **Given** a FinFocus deployment with the ap-southeast-2 plugin binary, **When** a user requests cost estimate for an m5.large EC2 instance in ap-southeast-2, **Then** the system returns the correct Sydney hourly rate
2. **Given** a FinFocus deployment with the ap-southeast-2 plugin binary, **When** a user requests cost estimate for an io2 EBS volume in ap-southeast-2, **Then** the system returns the correct Sydney GB-month rate

---

### User Story 3 - Tokyo Region Cost Estimation (Priority: P2)

A FinFocus user deploying AWS resources to ap-northeast-1 (Tokyo) needs accurate cost estimates for their Japanese infrastructure deployments.

**Why this priority**: Tokyo is a critical region for Asia-Pacific operations and often has unique pricing due to being one of AWS's longest-running AP regions.

**Independent Test**: Can be tested by deploying the ap-northeast-1 binary and verifying Tokyo-specific pricing is returned for resources in that region.

**Acceptance Scenarios**:

1. **Given** a FinFocus deployment with the ap-northeast-1 plugin binary, **When** a user requests cost estimate for a t3.medium EC2 instance in ap-northeast-1, **Then** the system returns the correct Tokyo hourly rate
2. **Given** resources spread across multiple regions including ap-northeast-1, **When** costs are estimated, **Then** Tokyo resources use Tokyo pricing while other regions use their respective pricing

---

### User Story 4 - Mumbai Region Cost Estimation (Priority: P3)

A FinFocus user deploying AWS resources to ap-south-1 (Mumbai) needs accurate cost estimates for their Indian infrastructure deployments.

**Why this priority**: Mumbai serves the Indian market and has distinct pricing, but typically sees lower usage volume compared to Singapore, Sydney, and Tokyo.

**Independent Test**: Can be tested by deploying the ap-south-1 binary and verifying Mumbai-specific pricing is returned.

**Acceptance Scenarios**:

1. **Given** a FinFocus deployment with the ap-south-1 plugin binary, **When** a user requests cost estimate for a c5.xlarge EC2 instance in ap-south-1, **Then** the system returns the correct Mumbai hourly rate
2. **Given** a FinFocus deployment with the ap-south-1 plugin binary, **When** a user requests cost estimate for a gp2 EBS volume in ap-south-1, **Then** the system returns the correct Mumbai GB-month rate

---

### Edge Cases

- What happens when a user requests pricing for an AP region but has loaded the wrong regional binary (e.g., requesting ap-southeast-1 pricing with the us-east-1 binary)?
- How does the system handle unavailable instance types or volume types in specific AP regions if AWS doesn't offer them in that region?
- What happens if pricing data is missing for a specific resource type in one of the AP regions?
- How does the system behave if a user attempts to build a binary without the required pricing data files present?
- What happens when concurrent gRPC calls request pricing for different AP regions (testing thread safety)?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support building separate plugin binaries for ap-southeast-1, ap-southeast-2, ap-northeast-1, and ap-south-1
- **FR-002**: Each AP region binary MUST embed only its region-specific pricing data and not include data from other regions
- **FR-003**: System MUST map AP region identifiers to build tags (ap-southeast-1 → region_apse1, ap-southeast-2 → region_apse2, ap-northeast-1 → region_apne1, ap-south-1 → region_aps1)
- **FR-004**: Each AP region binary MUST correctly identify its supported region and return ERROR_CODE_UNSUPPORTED_REGION with region details when asked to estimate costs for other regions
- **FR-005**: System MUST fetch and embed AWS public pricing data for EC2 instances and EBS volumes in all four AP regions
- **FR-006**: System MUST return accurate hourly rates for EC2 instances based on AP region-specific pricing
- **FR-007**: System MUST return accurate GB-month rates for EBS volumes based on AP region-specific pricing
- **FR-008**: Build process MUST generate pricing data files for all four AP regions (aws_pricing_ap-southeast-1.json, aws_pricing_ap-southeast-2.json, aws_pricing_ap-northeast-1.json, aws_pricing_ap-south-1.json)
- **FR-009**: Build configuration MUST produce correctly named binaries (finfocus-plugin-aws-public-ap-southeast-1, etc.)
- **FR-010**: System MUST maintain thread-safe pricing lookups for concurrent gRPC calls within each AP region binary
- **FR-011**: Test suite MUST verify region-specific behavior for each AP region binary
- **FR-012**: Error messages for region mismatches MUST clearly identify both the plugin's supported region and the requested resource region

### Key Entities

- **AP Region Binary**: A compiled plugin executable that supports exactly one Asia Pacific AWS region, contains embedded pricing data for that region only, and implements the CostSourceService gRPC protocol
- **Region Build Tag**: A Go build constraint (e.g., region_apse1) that controls which pricing data file is embedded into a binary during compilation
- **Pricing Data File**: A JSON file (e.g., aws_pricing_ap-southeast-1.json) containing AWS public on-demand pricing for EC2 instances and EBS volumes in a specific AP region
- **Region Mapping**: The relationship between AWS region identifiers (ap-southeast-1), build tags (region_apse1), and binary names (finfocus-plugin-aws-public-ap-southeast-1)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can successfully build all four AP region binaries (ap-southeast-1, ap-southeast-2, ap-northeast-1, ap-south-1) without errors
- **SC-002**: Each AP region binary is under 20MB in size, confirming it embeds only its region's pricing data
- **SC-003**: Cost estimates for EC2 t3.micro instances return different hourly rates across AP regions, confirming region-specific pricing (typically within 10-20% variance)
- **SC-004**: All existing test suites pass for all four AP region binaries with 100% success rate
- **SC-005**: Region mismatch errors occur within 100ms and include both plugin region and required region in error details
- **SC-006**: Concurrent gRPC calls (10+ simultaneous requests) to an AP region binary complete successfully without data corruption or crashes
- **SC-007**: Building any single AP region binary completes in under 2 minutes (excluding initial dependency downloads)
- **SC-008**: Each AP region binary correctly rejects requests for other regions with ERROR_CODE_UNSUPPORTED_REGION 100% of the time

## Assumptions *(mandatory)*

### Pricing Data Assumptions

- AWS public pricing APIs provide consistent data formats across all AP regions (same JSON structure as current US regions)
- Pricing data for EC2 and EBS services is available and complete for all four AP regions
- On-demand pricing (not spot or reserved) is sufficient for initial AP region support
- Pricing assumptions remain consistent with existing implementation (Linux OS, shared tenancy, 730 hours/month for EC2)

### Build Process Assumptions

- The existing build tag pattern (region_XXX) scales to accommodate AP regions without conflicts
- GoReleaser configuration can handle four additional region-specific builds without performance degradation
- The generate-pricing tool can be extended to fetch AP region pricing without requiring AWS credentials (public pricing API)

### Technical Assumptions

- Thread safety mechanisms in existing pricing client work correctly for AP region data
- The gRPC service protocol and error handling patterns remain consistent across regions
- Existing embed file patterns (embed_*.go) can be replicated for AP regions
- The pluginsdk.Serve() lifecycle management works identically for AP region binaries

### Testing Assumptions

- Test infrastructure can validate region-specific behavior without requiring actual AWS resources
- Dummy pricing data generation (--dummy flag) can be extended to AP regions for development
- Region mismatch error scenarios can be tested without building all regional binaries

## Out of Scope *(mandatory)*

### Explicitly Not Included

- Support for AWS China regions (cn-north-1, cn-northwest-1) - requires separate AWS partition handling
- Support for AWS GovCloud regions - requires different compliance and access patterns
- Spot pricing or reserved instance pricing for AP regions - only on-demand pricing in scope
- Cost estimation for AWS services beyond EC2 and EBS in AP regions - S3, Lambda, RDS, DynamoDB remain stubbed
- Historical pricing data or pricing trend analysis for AP regions
- Automatic region detection or recommendation based on resource location
- Multi-region cost aggregation or comparison features
- Custom pricing adjustments or discount factors specific to AP regions
- Integration with AWS Cost Explorer or CUR data for AP regions

### Future Considerations

- Support for additional AP regions (ap-northeast-2, ap-northeast-3, ap-south-2, etc.)
- Extended service support (RDS, DynamoDB pricing) in AP regions
- Reserved instance and spot pricing for cost optimization scenarios
- Cross-region price comparison tools
- Pricing data update mechanisms for keeping AP region pricing current
