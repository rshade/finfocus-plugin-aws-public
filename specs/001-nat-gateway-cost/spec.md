# Feature Specification: VPC NAT Gateway Cost Estimation

**Feature Branch**: `001-nat-gateway-cost`  
**Created**: 2025-12-21  
**Status**: Draft  
**Input**: User description: "title:	feat(natgw): implement VPC NAT Gateway cost estimation
state:	OPEN
author:	rshade
labels:	aws-service, enhancement
comments:	0
assignees:	
projects:	
milestone:	
number:	56
--
## Overview

Implement VPC NAT Gateway cost estimation. NAT Gateways are a notorious source of "surprise" AWS bills due to their combined hourly and data processing charges. This is a new resource type requiring full implementation.

## User Story

As a cloud cost analyst,
I want accurate NAT Gateway cost estimates based on gateway hours and data processing,
So that I can anticipate and control this commonly underestimated cost component.

## Problem Statement

NAT Gateways are essential for private subnet internet access but are frequently a source of unexpected costs. They combine:
- Fixed hourly charges (similar to EC2)
- Data processing charges (often overlooked)

This dual pricing model makes NAT Gateways particularly important for cost visibility.

## Proposed Solution

Extend the cost estimation system to include NAT Gateway resources, calculating costs based on gateway hours and data processing volumes to help users avoid unexpected bills.

## Clarifications

### Session 2025-12-22

- Q: If the data_processed_gb tag contains a non-numeric string, how should the system behave? → A: Return an error (InvalidArgument)
- Q: How should the system handle a negative value for the data_processed_gb tag? → A: Return an error (InvalidArgument)
- Q: If pricing data for a specific region is missing from the embedded data, how should the system react during cost estimation? → A: Return an error (NotFound/Internal)
- Q: What is the preferred tag name for specifying the data volume in GB? → A: data_processed_gb
- Q: If the data_processed_gb tag is present but its value is empty, how should the system behave? → A: Return an error (InvalidArgument)

## Acceptance Criteria

- [X] System supports NAT Gateway resource types for cost estimation
- [X] Cost calculations include both fixed hourly charges and variable data processing fees
- [X] Data volume can be specified via resource tags
- [X] System defaults to zero data cost when no data volume is provided
- [X] Pricing data is available across all supported AWS regions
- [X] System handles concurrent cost estimation requests safely
- [X] Comprehensive test coverage for various usage scenarios

## Out of Scope

- NAT instances (EC2-based alternative)
- Cross-AZ data transfer costs (Note: `data_processed_gb` only affects NAT Gateway processing fees)
- VPC endpoint pricing
- Multiple NAT Gateways per resource descriptor

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Get Accurate NAT Gateway Cost Estimates (Priority: P1)

As a cloud cost analyst, I want accurate NAT Gateway cost estimates based on gateway hours and data processing, so that I can anticipate and control this commonly underestimated cost component.

**Why this priority**: This is the core value proposition - providing visibility into a major source of unexpected AWS costs.

**Independent Test**: Can be fully tested by providing a NAT Gateway resource descriptor and verifying the returned cost matches expected calculations.

**Acceptance Scenarios**:

1. **Given** a NAT Gateway resource with data_processed_gb tag, **When** GetProjectedCost is called, **Then** returns combined hourly and data processing costs.
2. **Given** a NAT Gateway resource without data_processed_gb tag, **When** GetProjectedCost is called, **Then** returns hourly cost only (data cost = 0).
3. **Given** a NAT Gateway resource with high data volume (e.g., 10000 GB), **When** GetProjectedCost is called, **Then** calculates and returns correct total cost.
4. **Given** an unsupported resource type, **When** Supports is called with "natgw", **Then** returns supported=true.

---

### Edge Cases

- **Non-numeric tags**: If `data_processed_gb` contains a non-numeric value, the system returns an `InvalidArgument` error.
- **Negative values**: If `data_processed_gb` is negative, the system returns an `InvalidArgument` error.
- **Empty tags**: If `data_processed_gb` is present but empty, the system returns an `InvalidArgument` error.
- **Missing pricing data**: If NAT Gateway pricing data is missing for the target region, the system returns a `NotFound` or `Internal` error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support NAT Gateway resource types ("natgw", "nat_gateway", "nat-gateway") in Supports() method
- **FR-002**: System MUST calculate projected monthly cost combining hourly rate and data processing rate
- **FR-003**: System MUST extract data processed in GB from resource tags["data_processed_gb"]
- **FR-004**: System MUST default to 0GB data processed if tags["data_processed_gb"] is missing
- **FR-005**: System MUST include NAT Gateway pricing data in all 9 regional binaries
- **FR-006**: System MUST provide thread-safe pricing lookups for concurrent gRPC calls
- **FR-007**: System MUST log successful pricing lookups with debug information
- **FR-008**: System MUST validate that `data_processed_gb` is a numeric string and return `InvalidArgument` otherwise.
- **FR-009**: System MUST validate that `data_processed_gb` is non-negative and return `InvalidArgument` otherwise.
- **FR-010**: System MUST return an error if pricing data lookup fails for the specified region.
- **FR-011**: System MUST return `InvalidArgument` if `data_processed_gb` tag is present but empty.

### Key Entities *(include if feature involves data)*

- **NAT Gateway Resource**: Represents an AWS NAT Gateway with provider="aws", resource_type="natgw", and optional data_processed_gb tag
- **Pricing Data**: Hourly rate and data processing rate per GB for NAT Gateways, stored per region

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Supports() returns supported=true for all NAT Gateway resource type variants
- **SC-002**: GetProjectedCost() accurately calculates combined costs for various data volumes (0, 100, 1000, 10000 GB)
- **SC-003**: System handles missing data tags gracefully by defaulting to hourly-only costs
- **SC-004**: All regional binaries include NAT Gateway pricing data without filtering
- **SC-005**: Unit tests cover all logic branches and validated edge cases for NAT Gateway estimation logic
- **SC-006**: Concurrent pricing lookups complete without race conditions

## Assumptions

- NAT Gateway pricing follows the dual-rate model (hourly + per-GB data processing)
- Pricing data is available via AWS Pricing API for all supported regions
- Resource descriptors use standard tag naming conventions
- Monthly calculation assumes 730 hours (standard industry assumption)

## Dependencies

- Existing pricing client infrastructure
- AWS Pricing API access for VPC service
- Regional build matrix for embedding pricing data
