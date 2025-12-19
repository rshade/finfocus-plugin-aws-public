# Feature Specification: Lambda Cost Estimation

**Feature Branch**: `014-lambda-cost-estimation`
**Created**: 2025-12-18
**Status**: Draft
**Input**: Implement Lambda function cost estimation using AWS Public Pricing
data.

## Clarifications

### Session 2025-12-18

- Q: Memory SKU Format? → A: The SKU field will contain the memory size in MB
  as a numeric value (derived from Pulumi plan `memorySize`), e.g., '128' or
  '512'.
- Q: Unsupported Region Behavior? → A: Return $0 cost with "Region not
  supported" billing detail message.
- Q: Default Architecture Pricing? → A: Use architecture specified in resource
  description (Tags or properties).

## User Scenarios & Testing

### User Story 1 - Accurate Cost Estimation (Priority: P1)

As a cloud cost analyst, I want to see accurate monthly cost estimates for my
Lambda functions based on their configured memory and expected usage, so that
I can budget for serverless compute costs.

**Why this priority**: Core value of the feature; without this, the plugin
provides no value for Lambda.

**Independent Test**: Can be fully tested by providing a Lambda resource with
specific tags and verifying the output cost matches manual calculation.

**Acceptance Scenarios**:

1. **Given** a Lambda function with 512MB memory, **When** tagged with 1M
   requests and 200ms duration, **Then** the estimated cost should reflect
   both request and compute time charges.
2. **Given** a Lambda function with 128MB memory, **When** tagged with 1M
   requests and 100ms duration, **Then** the estimated cost should reflect the
   lower compute rate.
3. **Given** a Lambda function in `us-east-1`, **When** pricing is calculated,
   **Then** it uses the specific rates for that region.

---

### User Story 2 - Graceful Default Handling (Priority: P2)

As a user, I want the system to handle missing usage tags gracefully by
providing a zero-cost estimate with an explanation, rather than failing or
returning misleading defaults, so that I know I need to add tags for accurate
estimation.

**Why this priority**: Prevents user confusion and potential "bug" reports when
data is missing.

**Independent Test**: Verify behavior when providing a resource without the
expected usage tags.

**Acceptance Scenarios**:

1. **Given** a Lambda function without `requests_per_month` tag, **When** cost
   is estimated, **Then** it assumes 0 requests and returns $0 cost.
2. **Given** a Lambda function without `avg_duration_ms` tag, **When** cost is
   estimated, **Then** it defaults to a safe minimum (e.g., 100ms) but results
   in $0 cost if requests are also 0.
3. **Given** a Lambda function with missing tags, **When** result is returned,
   **Then** the billing detail string clearly indicates the missing inputs or
   default assumptions.

---

### User Story 3 - Regional Support (Priority: P2)

As a user operating in multiple AWS regions, I want Lambda pricing to be
available in all supported regions, so that my global infrastructure is
accurately costed.

**Why this priority**: Ensures consistency across the supported regions of the
plugin.

**Independent Test**: Verify Lambda pricing availability in a non-standard
region (e.g., `sa-east-1`).

**Acceptance Scenarios**:

1. **Given** a Lambda function in `sa-east-1`, **When** cost is calculated,
   **Then** it uses the pricing specific to South America (Sao Paulo).
2. **Given** a build for any of the 9 supported regions, **When** the plugin
   starts, **Then** it successfully loads the Lambda pricing data for that
   region.

### Edge Cases

- **Invalid Memory Size**: If the resource SKU cannot be parsed as a number
  (e.g., "unknown"), the system defaults to 128MB.
- **Negative Usage Values**: If tags contain negative numbers, they should be
  treated as 0 or absolute values (safest is 0) to avoid negative costs.
- **Unsupported Region**: If a resource is in a region not supported by the
  current build, it MUST return $0 cost with "Region not supported" in the
  billing detail message.
- **Extreme Usage**: Extremely large values (e.g., billions of requests) should
  calculate correctly without integer overflow (using floating point
  arithmetic).

## Requirements

### Functional Requirements

- **FR-001**: System MUST support `aws:lambda:function` (or equivalent)
  resource type validation.
- **FR-002**: System MUST calculate cost based on two dimensions: Request
  Count and Compute Duration (GB-seconds).
- **FR-003**: System MUST parse memory size from the resource SKU, expecting a
  raw numeric string representing MB (derived from Pulumi plan `memorySize`),
  defaulting to 128MB if parsing fails.
- **FR-004**: System MUST extract expected monthly request count from resource
  tags (e.g., `requests_per_month`).
- **FR-005**: System MUST extract average execution duration from resource tags
  (e.g., `avg_duration_ms`).
- **FR-006**: System MUST default request count to 0 if the tag is missing.
- **FR-007**: System MUST default duration to 100ms if the tag is missing.
- **FR-008**: System MUST provide a descriptive billing detail string
  explaining how the cost was calculated (e.g., "Lambda 512MB, 1M
  requests/month...").
- **FR-009**: System MUST support pricing lookups for both "Serverless" and
  "AWS Lambda" product families to cover all pricing components.
- **FR-010**: System MUST return $0 cost and "Region not supported" billing
  detail if the region is not supported by the pricing engine.
- **FR-011**: System MUST use the architecture specified in the resource
  description (via tags or properties) to select the correct pricing (x86 vs
  ARM), defaulting to x86 if unspecified.

### Key Entities

- **Lambda Resource**: Represents the function with its configuration (Memory)
  and usage metadata (Tags).
- **Pricing Record**: Represents the regional unit cost for Requests ($/million)
  and Compute ($/GB-second).

## Success Criteria

### Measurable Outcomes

- **SC-001**: Cost estimation is accurate to within $0.01 compared to AWS
  public pricing examples for standard usage scenarios.
- **SC-002**: 100% of supported regions (9 regions) provide Lambda pricing
  data.
- **SC-003**: Missing usage tags result in a valid response (no errors) with $0
  cost and explanatory message.
- **SC-004**: System supports memory configurations from 128MB up to 10GB
  (10240MB) for cost calculation.

### Assumptions

- Users will provide usage data via specific resource tags (`requests_per_month`,
  `avg_duration_ms`).
- AWS pricing model for Lambda (Requests + GB-seconds) remains consistent.
- The `sku` field reliably contains the memory size in MB or a value that can
  be mapped to it.
