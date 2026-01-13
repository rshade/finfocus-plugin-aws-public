# Data Model: DynamoDB Hardening Bundle

**Feature**: 020-dynamodb-hardening
**Date**: 2025-12-31

## Overview

No new entities are introduced. This feature modifies validation and error handling within existing data flows.

## Existing Entities (Unchanged)

### ResourceDescriptor (Input)

Proto-defined input from finfocus.v1:

| Field | Type | DynamoDB Usage |
|-------|------|----------------|
| provider | string | "aws" |
| resource_type | string | "dynamodb" or "aws:dynamodb/table:Table" |
| sku | string | "on-demand" or "provisioned" |
| region | string | AWS region (e.g., "us-east-1") |
| tags | map[string]string | Capacity/storage configuration |

**DynamoDB Tags**:

| Tag Key | Type | Mode | Description |
|---------|------|------|-------------|
| `storage_gb` | float64 | Both | Table storage in GB |
| `read_capacity_units` | int64 | Provisioned | Provisioned RCU |
| `write_capacity_units` | int64 | Provisioned | Provisioned WCU |
| `read_requests_per_month` | int64 | On-Demand | Monthly read requests |
| `write_requests_per_month` | int64 | On-Demand | Monthly write requests |

### GetProjectedCostResponse (Output)

Proto-defined output from finfocus.v1:

| Field | Type | Description |
|-------|------|-------------|
| unit_price | float64 | Primary unit price (RCU for provisioned, storage for on-demand) |
| currency | string | Always "USD" |
| cost_per_month | float64 | Total monthly cost |
| billing_detail | string | Human-readable breakdown |

### PricingClient Methods

Existing methods returning `(float64, bool)`:

| Method | Returns |
|--------|---------|
| `DynamoDBStoragePricePerGBMonth()` | Storage $/GB-month |
| `DynamoDBProvisionedRCUPrice()` | RCU $/hour |
| `DynamoDBProvisionedWCUPrice()` | WCU $/hour |
| `DynamoDBOnDemandReadPrice()` | Read $/request |
| `DynamoDBOnDemandWritePrice()` | Write $/request |

## Validation Rules (New)

These rules are now explicitly enforced:

| Tag | Rule | On Violation |
|-----|------|--------------|
| All numeric tags | Value >= 0 | Log warning, default to 0 |
| All numeric tags | Parseable number | Log warning, default to 0 |

## State Transitions

N/A - DynamoDB cost estimation is stateless (single RPC call).
