# EKS Cost Estimation API Contracts

**Date**: 2025-12-06
**Feature**: 010-eks-cost-estimation

## Overview

EKS cost estimation extends the existing finfocus.v1 gRPC CostSourceService with EKS cluster pricing support.

## gRPC Method Contracts

### Supports() Method Extension

**Purpose**: Check if EKS resource type is supported

**Request**: ResourceDescriptor
```protobuf
message ResourceDescriptor {
  string provider = 1;      // "aws"
  string resource_type = 2; // "eks" (NEW)
  string sku = 3;           // "cluster"
  string region = 4;        // AWS region
  map<string, string> tags = 5; // Empty for EKS
}
```

**Response**: SupportsResponse
```protobuf
message SupportsResponse {
  bool supported = 1;       // true for resource_type="eks"
  string reason = 2;        // Empty on success
}
```

**Preconditions**:
- provider == "aws"
- resource_type == "eks"
- region is valid AWS region

**Postconditions**:
- Returns supported=true for EKS clusters
- Returns supported=false for unsupported resource types

### GetProjectedCost() Method Extension

**Purpose**: Calculate monthly EKS cluster cost

**Request**: ResourceDescriptor (same as Supports)

**Response**: GetProjectedCostResponse
```protobuf
message GetProjectedCostResponse {
  double unit_price = 1;        // Hourly cluster rate ($0.10)
  string currency = 2;          // "USD"
  double cost_per_month = 3;    // unit_price × 730
  string billing_detail = 4;    // "EKS cluster, 730 hrs/month (control plane only, excludes worker nodes)"
}
```

**Preconditions**:
- Supports() returns supported=true
- Pricing data available for region

**Postconditions**:
- cost_per_month = unit_price × 730 (hours in month)
- billing_detail clarifies control plane scope
- Thread-safe operation

## Pricing Client Interface Contract

**Purpose**: Abstract pricing data access for testability

**Interface**: PricingClient (extended)
```go
type PricingClient interface {
    // Existing methods...
    EKSClusterPricePerHour() (float64, bool)  // NEW: Returns ($0.10, true)
}
```

**Contract**:
- Returns current EKS cluster hourly rate
- Thread-safe for concurrent access
- Returns (0, false) if pricing unavailable

## Error Handling Contracts

**Invalid Resource**: ERROR_CODE_INVALID_RESOURCE (6)
- Trigger: Malformed ResourceDescriptor for EKS
- Response: gRPC status with error details

**Unsupported Region**: ERROR_CODE_UNSUPPORTED_REGION (9)
- Trigger: Region not in pricing data
- Response: gRPC status with error details

**Data Corruption**: ERROR_CODE_DATA_CORRUPTION (11)
- Trigger: Embedded pricing data load failure
- Response: gRPC status with error details

## Logging Contracts

**Success Logging**:
```json
{
  "level": "debug",
  "component": "finfocus-plugin-aws-public",
  "trace_id": "<request-id>",
  "operation": "GetProjectedCost",
  "aws_region": "us-east-1",
  "hourly_rate": 0.10,
  "message": "EKS pricing lookup successful"
}
```

**Error Logging**:
```json
{
  "level": "error",
  "component": "finfocus-plugin-aws-public",
  "trace_id": "<request-id>",
  "operation": "GetProjectedCost",
  "error": "pricing data unavailable for region",
  "message": "EKS pricing lookup failed"
}
```</content>
<parameter name="filePath">specs/010-eks-cost-estimation/contracts/eks-api-contracts.md