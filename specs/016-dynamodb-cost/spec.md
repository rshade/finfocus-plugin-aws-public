# Feature Specification: DynamoDB Cost Estimation

**Feature Branch**: `016-dynamodb-cost`
**Created**: 2025-12-19
**Status**: Draft
**Input**: User description: (See prompt)

## Clarifications

### Session 2025-12-19
- **Q: Handling Unknown SKUs → A: Default to On-Demand pricing (treat unknown SKU as missing).**

## User Scenarios & Testing *(mandatory)*

### User Story 1 - On-Demand Cost Estimation (Priority: P1)

As a cloud cost analyst, I want accurate cost estimates for DynamoDB tables configured with On-Demand capacity, based on my projected usage, so that I can forecast spending for variable workloads.

**Why this priority**: On-Demand is the primary focus for v1 simplicity and is a common configuration for new or variable workloads.

**Independent Test**: Can be fully tested by providing a `ResourceDescriptor` with `Sku="on-demand"` and valid usage tags (read/write requests, storage) and verifying the returned cost matches the manual calculation (Units * Unit Price + Storage * Storage Price).

**Acceptance Scenarios**:

1. **Given** a DynamoDB resource with `Sku: "on-demand"`, **When** valid `read_requests_per_month`, `write_requests_per_month`, and `storage_gb` tags are provided, **Then** the system returns a projected cost matching the sum of throughput and storage costs.
2. **Given** an On-Demand resource, **When** `storage_gb` is provided but request counts are zero, **Then** the system returns only the storage cost.
3. **Given** an On-Demand resource, **When** `Sku` is missing/empty, **Then** the system defaults to On-Demand pricing.

---

### User Story 2 - Provisioned Capacity Cost Estimation (Priority: P2)

As a cloud cost analyst, I want accurate cost estimates for DynamoDB tables with Provisioned capacity, so that I can budget for steady-state workloads.

**Why this priority**: Provisioned capacity is critical for production workloads with predictable traffic and represents a significant portion of AWS spend.

**Independent Test**: Can be fully tested by providing a `ResourceDescriptor` with `Sku="provisioned"` and capacity tags (RCUs, WCUs) and verifying the returned cost matches the manual calculation (Capacity * Hours * Hourly Price + Storage).

**Acceptance Scenarios**:

1. **Given** a DynamoDB resource with `Sku: "provisioned"`, **When** valid `read_capacity_units` and `write_capacity_units` tags are provided, **Then** the system calculates cost based on 730 hours per month for each unit.
2. **Given** a Provisioned resource, **When** `storage_gb` is included, **Then** the storage cost is added to the throughput cost.

---

### User Story 3 - Handling Missing Data (Priority: P3)

As a user, I want clear explanations when my input data is insufficient for an estimate, so that I know why a cost might be zero or inaccurate.

**Why this priority**: Ensures users understand "zero cost" results are due to missing data, not free resources.

**Independent Test**: Submit a resource with no tags and check the response `BillingDetail`.

**Acceptance Scenarios**:

1. **Given** a DynamoDB resource (any mode), **When** required usage tags are missing, **Then** the system returns $0 cost.
2. **Given** the above scenario, **Then** the response `BillingDetail` field explicitly lists the detected mode and the missing/zero inputs.

### Edge Cases

- **Unknown Sku**: The system MUST default to On-Demand pricing if the `Sku` field is unrecognized or empty.
- What happens when tags contain non-numeric values? (Should handle gracefully, likely treating as 0).
- What happens if the Region is not supported by the plugin? (Standard plugin behavior, likely returns error or unsupported).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST identify resources with `ResourceType: "dynamodb"` as supported.
- **FR-002**: The system MUST support pricing lookups for all 12 supported regions using embedded pricing data.
- **FR-003**: The system MUST determine the pricing model based on the `Sku` field: "provisioned" for Provisioned Capacity, and "on-demand" (or default) for Pay-Per-Request.
- **FR-004**: For On-Demand pricing, the system MUST calculate costs using:
    - Read Request Units (per million)
    - Write Request Units (per million)
    - Storage (per GB-month)
- **FR-005**: For Provisioned pricing, the system MUST calculate costs using:
    - Read Capacity Units (per RCU-hour, assuming 730 hours/month)
    - Write Capacity Units (per WCU-hour, assuming 730 hours/month)
    - Storage (per GB-month)
- **FR-006**: The system MUST fallback to 0 values for any missing usage tags (Read/Write units, RCUs/WCUs, Storage) without failing the request.
- **FR-007**: The system MUST include a human-readable `BillingDetail` string in the response summarizing the inputs used (e.g., "DynamoDB on-demand, 10M reads...").

### Key Entities *(include if feature involves data)*

- **PricingClient**: Interface for retrieving region-specific pricing data (On-Demand rates, Provisioned hourly rates, Storage rates).
- **ResourceDescriptor**: The input object containing Region, SKU (Capacity Mode), and Tags (Usage volumes).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `GetProjectedCost` returns a non-zero value for valid DynamoDB inputs in all supported regions.
- **SC-002**: Cost calculations for On-Demand mode match the AWS pricing formula: `(Reads/1M * ReadPrice) + (Writes/1M * WritePrice) + (Storage * StoragePrice)`.
- **SC-003**: Cost calculations for Provisioned mode match the AWS pricing formula: `(RCUs * 730 * RCUPrice) + (WCUs * 730 * WCUPrice) + (Storage * StoragePrice)`.
- **SC-004**: The `Supports()` method returns `true` for `dynamodb` resources, enabling the core engine to route requests to this plugin.