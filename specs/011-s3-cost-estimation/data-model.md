# Data Model: S3 Storage Cost Estimation

**Feature**: 011-s3-cost-estimation
**Date**: 2025-12-07

## Entities

### ResourceDescriptor (Input)

**Purpose**: Describes the AWS resource for cost estimation.

**Fields**:
- `provider` (string): Must be "aws"
- `resource_type` (string): Must be "s3" for S3 buckets
- `sku` (string): Storage class SKU (e.g., "STANDARD", "STANDARD_IA")
- `region` (string): AWS region (e.g., "us-east-1")
- `tags` (map[string]string): Resource tags, including "size" for storage size in GB

**Validation Rules**:
- `provider` must equal "aws"
- `resource_type` must equal "s3"
- `sku` must be a valid storage class (STANDARD, STANDARD_IA, ONEZONE_IA, GLACIER, DEEP_ARCHIVE)
- `region` must be a supported AWS region
- `tags["size"]` must be parseable as positive float64, defaults to 1.0 if missing/invalid

**Relationships**: Input to GetProjectedCost RPC method.

### GetProjectedCostResponse (Output)

**Purpose**: Contains the calculated monthly cost estimate.

**Fields**:
- `unit_price` (float64): Cost per GB per month in USD
- `currency` (string): Always "USD"
- `cost_per_month` (float64): Total monthly cost (unit_price × size_in_gb)
- `billing_detail` (string): Human-readable description (e.g., "S3 Standard storage, 100 GB, $0.0230/GB-month")

**Validation Rules**:
- `unit_price` >= 0
- `currency` == "USD"
- `cost_per_month` >= 0
- `billing_detail` non-empty and descriptive

**Relationships**: Output from GetProjectedCost RPC method.

### S3Price (Internal)

**Purpose**: Represents pricing data for a specific storage class.

**Fields**:
- `unit` (string): Pricing unit, always "GB-Mo"
- `rate_per_gb_month` (float64): USD per GB per month
- `currency` (string): Always "USD"

**Validation Rules**:
- `unit` == "GB-Mo"
- `rate_per_gb_month` > 0
- `currency` == "USD"

**Relationships**: Stored in pricing client's s3Index map, keyed by storage class.

## State Transitions

None - stateless service.

## Data Volume / Scale Assumptions

- Pricing data: < 50MB per region binary (constitution requirement)
- Concurrent requests: Support 100+ simultaneous GetProjectedCost calls
- Storage classes: 5 supported (STANDARD, STANDARD_IA, ONEZONE_IA, GLACIER, DEEP_ARCHIVE)
- Regions: 9 supported regions

## Identity & Uniqueness Rules

- ResourceDescriptor: Identified by (provider, resource_type, sku, region) combination
- S3Price: Unique per (region, storage_class) pair
- GetProjectedCostResponse: Unique per input ResourceDescriptor