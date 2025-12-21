# Feature Specification: Embed Raw AWS Pricing JSON Per Service

**Feature Branch**: `018-raw-pricing-embed`
**Created**: 2025-12-20
**Status**: Draft
**Input**: User description: "Embed each AWS service's pricing JSON exactly as it comes from the AWS Price List API. No combining, no processing."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Accurate Pricing Data Retrieval (Priority: P1)

As a PulumiCost user estimating AWS costs, I need the plugin to return accurate pricing for all supported AWS services so that my infrastructure cost estimates match real AWS pricing.

**Why this priority**: This is the core functionality. Without accurate pricing data, the entire plugin's value proposition fails. The v0.0.10 and v0.0.11 bugs demonstrated that combining/processing pricing data introduces risk of silent failures where prices return $0.

**Independent Test**: Can be tested by querying EC2 instance pricing (e.g., t3.micro in us-east-1) and verifying the returned price matches the AWS Price List API exactly.

**Acceptance Scenarios**:

1. **Given** the plugin binary for us-east-1, **When** a cost estimate is requested for t3.micro, **Then** the returned hourly rate matches the current AWS public pricing for t3.micro in us-east-1.
2. **Given** the plugin binary for us-east-1, **When** a cost estimate is requested for m5.large, **Then** the returned hourly rate matches the current AWS public pricing for m5.large in us-east-1.
3. **Given** the plugin binary for eu-west-1, **When** a cost estimate is requested for c5.xlarge, **Then** the returned hourly rate matches the current AWS public pricing for c5.xlarge in eu-west-1.

---

### User Story 2 - Service-Specific Pricing Isolation (Priority: P1)

As a plugin developer debugging pricing issues, I need each AWS service's pricing data stored in separate files so that I can inspect and validate individual service data without parsing combined blobs.

**Why this priority**: Debuggability is critical for preventing future v0.0.10-style bugs. When pricing data is combined, it's difficult to identify which service has corrupted data.

**Independent Test**: Can be tested by examining the generated pricing files and verifying each file contains only data for its specific AWS service with real AWS metadata (offerCode, version, publicationDate).

**Acceptance Scenarios**:

1. **Given** the pricing generation tool has run for us-east-1, **When** I examine ec2_us-east-1.json, **Then** the file contains `offerCode: "AmazonEC2"` and only EC2 products.
2. **Given** the pricing generation tool has run for us-east-1, **When** I examine elb_us-east-1.json, **Then** the file contains `offerCode: "AWSELB"` and only ELB products.
3. **Given** I need to debug EKS pricing issues, **When** I open eks_us-east-1.json, **Then** I can inspect EKS-specific pricing without parsing EC2, EBS, or other service data.

---

### User Story 3 - Preserved AWS Metadata (Priority: P2)

As a plugin operator, I need embedded pricing data to retain AWS version and publication metadata so that I can verify the data currency and troubleshoot pricing discrepancies.

**Why this priority**: When pricing seems wrong, operators need to verify which version of AWS pricing data is embedded. The current combined format loses this metadata.

**Independent Test**: Can be tested by parsing an embedded pricing file and extracting the AWS-provided metadata fields.

**Acceptance Scenarios**:

1. **Given** a generated ec2_us-east-1.json file, **When** I parse the JSON, **Then** I find `version` and `publicationDate` fields with AWS-provided timestamps.
2. **Given** two pricing updates a week apart, **When** I compare the version fields, **Then** I can determine which dataset is newer.

---

### User Story 4 - Consistent Build Verification (Priority: P2)

As a release engineer, I need immutable tests that verify each service's embedded pricing data meets minimum thresholds so that we never release a binary with broken pricing like v0.0.10.

**Why this priority**: The existing combined-data tests don't catch service-specific data loss. Per-service thresholds provide granular failure detection.

**Independent Test**: Can be tested by running unit tests with build tags and verifying tests fail when pricing data is below expected thresholds.

**Acceptance Scenarios**:

1. **Given** a properly built us-east-1 binary, **When** tests run with region_use1 tag, **Then** tests verify EC2 pricing data exceeds 100MB.
2. **Given** a properly built us-east-1 binary, **When** tests run with region_use1 tag, **Then** tests verify ELB pricing data exceeds 400KB.
3. **Given** a malformed build without embedded pricing, **When** tests run, **Then** the per-service threshold tests fail before release.

---

### Edge Cases

- What happens when AWS Price List API is unavailable during pricing generation?
  - The generation tool fails loudly and does not produce partial/empty files.
- What happens when one service's pricing file is corrupted but others are valid?
  - The plugin initializes successfully for working services and returns errors only for the corrupted service.
- How does the system handle a new AWS service that needs to be added?
  - Adding a new service requires: new embed directive, new JSON file, and updated generation tool (following existing patterns).
- What happens when the AWS API returns a different schema version?
  - The existing `awsPricing` struct in types.go parses it or fails with a clear error. Schema changes require code updates.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST fetch each AWS service's pricing data from its dedicated AWS Price List API endpoint without combining or processing.
- **FR-002**: System MUST save each service's pricing response as a separate JSON file named `{service}_{region}.json` (e.g., `ec2_us-east-1.json`).
- **FR-003**: System MUST preserve all AWS-provided metadata in each pricing file including `offerCode`, `version`, and `publicationDate`.
- **FR-004**: System MUST embed each service's pricing file using separate `//go:embed` directives in region-specific embed files.
- **FR-005**: System MUST parse each embedded service file independently during pricing client initialization.
- **FR-006**: System MUST provide per-service size/product-count validation tests that fail if data is below expected thresholds.
- **FR-007**: System MUST support the following AWS services: EC2 (AmazonEC2), EBS (AmazonEC2 - volumes), RDS (AmazonRDS), EKS (AmazonEKS), Lambda (AWSLambda), DynamoDB (AmazonDynamoDB), ELB (AWSELB), S3 (AmazonS3).
- **FR-008**: System MUST maintain backward compatibility with existing gRPC API consumers - no changes to ResourceDescriptor or response formats.
- **FR-009**: Generation tool MUST fail if any service's pricing fetch fails rather than generating partial data.
- **FR-010**: System MUST build region-specific binaries using existing build tag system (region_use1, region_usw2, etc.).

### Key Entities

- **Service Pricing File**: Raw JSON response from AWS Price List API for a single service in a single region. Contains `offerCode`, `version`, `publicationDate`, `products` map, and `terms` map.
- **Embed Registry**: Region-tagged Go files that declare `//go:embed` directives for each service's pricing file.
- **Pricing Client**: Thread-safe client that parses all embedded service files and provides lookup methods for EC2 instances, EBS volumes, EKS clusters, ELB load balancers, and other supported resources.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All supported EC2 instance types (t3, m5, c5, r5 families) return non-zero prices matching AWS public pricing within 0.01% tolerance.
- **SC-002**: Pricing generation produces 7 separate JSON files per region (EC2, S3, RDS, EKS, Lambda, DynamoDB, ELB).
- **SC-003**: Each generated pricing file contains valid AWS metadata fields (`offerCode`, `version`, `publicationDate`).
- **SC-004**: Per-service threshold tests verify: EC2 > 100MB, RDS > 10MB, EKS > 2MB, Lambda > 1MB, S3 > 500KB, DynamoDB > 400KB, ELB > 400KB.
- **SC-005**: All existing integration tests pass with identical results before and after the refactor.
- **SC-006**: Binary size remains under 250MB per region (approximately 160-170MB with embedded pricing data). CI alerts at 200MB (warning) and 240MB (critical).
- **SC-007**: Pricing client initialization time remains under 2 seconds (current performance baseline).
- **SC-008**: All 12 supported regions build successfully with proper pricing data embedded.

## Assumptions

- AWS Price List API endpoints remain stable and continue to return individual service pricing at documented URLs.
- The existing `awsPricing` struct in `internal/pricing/types.go` is compatible with raw AWS API responses (it was designed for this purpose).
- EBS pricing is included in the AmazonEC2 pricing file (products with `productFamily: "Storage"`) rather than a separate endpoint.
- Binary size increase from multiple embedded files vs. one combined file is acceptable (estimated similar total size due to no synthetic wrapper).
- Parsing 7 separate JSON files is not significantly slower than parsing one combined file of the same total size.
