# Data Model: Plugin Rename to FinFocus

**Feature**: 001-plugin-rename
**Date**: 2026-01-11
**Status**: Unchanged

## Overview

The data model for the plugin remains **unchanged** by this rename operation. All entities, relationships, and data structures are identical; only import paths and type references change from `finfocus.v1` to `finfocus.v1`.

## Core Entities

### PricingData

Embedded AWS pricing data loaded from JSON files at build time.

**Purpose**: Contains complete AWS pricing information for supported services and regions

**Fields**:
- `Products`: Map of product code to Product definitions
- `Terms`: Map of product code to PriceTerm definitions
- `ServiceCode`: Service identifier (e.g., "AmazonEC2", "AmazonS3")

**Lifecycle**: Loaded once at startup via `sync.Once`, cached for lifetime of process

**Storage**: Memory-mapped JSON files embedded in binary

---

### Product

AWS service product definition from pricing data.

**Purpose**: Represents an AWS service product with attributes and pricing terms

**Fields**:
- `ProductCode`: Unique product identifier (e.g., "HVM64", "NAT-Gateway")
- `Attributes`: Map of attribute name to value (e.g., "instanceType": "t3.micro", "region": "us-east-1")
- `SKU`: Stock Keeping Unit identifier from AWS

**Validation Rules**:
- ProductCode must be non-empty
- Required attributes vary by service type

**Relationships**:
- One-to-many with PriceTerm (via product code)

---

### PriceTerm

On-demand pricing term for a product.

**Purpose**: Defines cost per unit for a product

**Fields**:
- `PriceDimensions`: Map of dimension name to price details
- `Unit`: Pricing unit (e.g., "Hours", "GB-Month", "Requests")
- `Currency`: Currency code (e.g., "USD")

**Validation Rules**:
- Price must be numeric and >= 0
- Unit must be non-empty
- Currency must be supported (typically USD)

**Relationships**:
- Many-to-one with Product

---

### ResourceDescriptor (Proto: finfocus.v1.ResourceDescriptor)

Input from FinFocus core describing a cloud resource.

**Purpose**: Provides resource metadata for cost estimation

**Fields** (from proto):
- `region`: AWS region (e.g., "us-east-1")
- `resource_type`: Pulumi resource type (e.g., "aws:ec2/instance:Instance")
- `properties`: Map of resource properties (e.g., instanceType, vpcId)
- `provider`: Cloud provider (e.g., "aws")

**Validation Rules**:
- region must be supported (us-east-1, us-west-2, eu-west-1)
- resource_type must match supported services
- properties must include required fields for resource type

**Source**: Passed via gRPC from FinFocus core

---

### CostResponse (Proto: finfocus.v1.GetProjectedCostResponse)

Output to FinFocus core with cost estimates.

**Purpose**: Provides projected cost calculations for a resource

**Fields** (from proto):
- `unit_price`: Cost per unit of resource usage
- `currency`: Currency code (e.g., "USD")
- `cost_per_month`: Estimated monthly cost based on 730 hours
- `billing_detail`: Detailed breakdown by cost components

**Validation Rules**:
- unit_price must be numeric and >= 0
- currency must match pricing data
- cost_per_month is calculated as unit_price * usage_multiplier

**Destination**: Returned via gRPC to FinFocus core

---

## Data Flow

```
┌─────────────────┐
│ FinFocus Core │
└────────┬────────┘
         │ gRPC: ResourceDescriptor
         ▼
┌─────────────────────────────────┐
│ finfocus-plugin-aws-public     │
│                                 │
│  1. Parse ResourceDescriptor    │
│  2. Lookup pricing data         │
│     (by service + properties)   │
│  3. Calculate costs            │
│  4. Return CostResponse         │
└────────┬────────────────────────┘
         │ gRPC: GetProjectedCostResponse
         ▼
┌─────────────────┐
│ FinFocus Core │
└─────────────────┘
```

## State Transitions

### Plugin Lifecycle

```
Not Started → Loading Pricing Data → Ready → Serving Requests
                              ↓ (error)
                         Failed to Load
```

### Loading Pricing Data (via sync.Once)

```
Uninitialized → Loading → Loaded (cached)
                      ↓ (error)
                 Load Failed (log error, return gRPC error)
```

### Request Processing (per gRPC call)

```
Receive Request → Validate Input → Lookup Pricing → Calculate Cost → Return Response
                       ↓                           ↓ (not found)
                   Invalid Input            Price Not Found (return error)
```

## Indexing Strategy

To meet performance targets (<100ms GetProjectedCost), pricing data is indexed:

**Primary Index**: Map[ServiceCode][]Product - Products grouped by AWS service

**Secondary Indexes**:
- Map[Region][]Product - Products filtered by region
- Map[Attribute:Value][]Product - Products by specific attribute values

**Thread Safety**: Indexes are read-only after initialization, safe for concurrent access.

## Validation Rules

### Input Validation (ResourceDescriptor)

- Region must be: us-east-1, us-west-2, or eu-west-1
- resource_type must match supported AWS services:
  - EC2 (aws:ec2/instance:Instance)
  - S3 (aws:s3/bucket:BucketV2)
  - RDS (aws:rds/instance:Instance)
  - EKS (aws:eks/cluster:Cluster)
  - Lambda (aws:lambda/function:Function)
  - DynamoDB (aws:dynamodb/table:Table)
  - ELB (aws:elb/loadBalancer:LoadBalancer)
- Required properties must be present (varies by resource type)

### Output Validation (CostResponse)

- unit_price >= 0
- cost_per_month calculated as unit_price * 730 (hours per month)
- Currency must be "USD"
- billing_detail must include cost components

## Constraints

### Performance

- Pricing data load: <500ms (one-time, via sync.Once)
- Lookup via indexed maps: O(1) average case
- Cost calculation: O(1) for standard resources
- Total GetProjectedCost: <100ms

### Storage

- Binary size: <250MB per region (includes ~150MB pricing data)
- Memory footprint: <400MB per region (memory-mapped pricing data)
- No runtime storage or persistence

### Data Integrity

- Pricing data is embedded at build time (no network calls)
- Full AWS pricing data retained (no filtering per constitution)
- Data validation on load (malformed data = load failure)

## Changes from Rename Operation

### What Changed

- Import paths: `finfocus.v1` → `finfocus.v1`
- Module name: `finfocus-plugin-aws-public` → `finfocus-plugin-aws-public`
- Logging prefix: `[finfocus-plugin-aws-public]` → `[finfocus-plugin-aws-public]`
- Dependency: `github.com/rshade/finfocus-spec` → `github.com/rshade/finfocus-spec`

### What Did NOT Change

- Entity structure and field names
- Data flow and state transitions
- Indexing strategy and performance characteristics
- Validation rules and constraints
- gRPC protocol interface (proto message structure identical)
- Pricing data content and format

## Testing Strategy

### Unit Tests

- Test pricing lookup logic with mock pricing data
- Test cost calculation formulas
- Test input validation for ResourceDescriptor

### Integration Tests

- Test gRPC handlers with in-memory mock pricing
- Test full request flow (ResourceDescriptor → CostResponse)
- Test error handling (invalid input, pricing not found)

### Concurrency Tests

- Test thread-safe pricing lookups (100 concurrent calls)
- Test sync.Once behavior (ensure single load)

## Notes

- This data model is **read-only** for cost estimation (no write operations)
- No user-facing data storage or persistence
- All data is embedded at build time (no runtime dependencies)
- Proto definitions are provided by finfocus-spec v0.5.0