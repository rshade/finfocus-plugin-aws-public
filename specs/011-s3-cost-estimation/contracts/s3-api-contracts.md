# API Contracts: S3 Storage Cost Estimation

**Feature**: 011-s3-cost-estimation
**Date**: 2025-12-07

## gRPC Service: CostSourceService

### Method: GetProjectedCost

**Purpose**: Calculate projected monthly cost for S3 storage.

**Request**: ResourceDescriptor
```protobuf
message ResourceDescriptor {
  string provider = 1;
  string resource_type = 2;
  string sku = 3;
  string region = 4;
  map<string, string> tags = 5;
}
```

**Response**: GetProjectedCostResponse
```protobuf
message GetProjectedCostResponse {
  double unit_price = 1;
  string currency = 2;
  double cost_per_month = 3;
  string billing_detail = 4;
}
```

**Preconditions**:
- provider == "aws"
- resource_type == "s3"
- sku in ["STANDARD", "STANDARD_IA", "ONEZONE_IA", "GLACIER", "DEEP_ARCHIVE"]
- region in supported regions
- tags["size"] parseable as positive float64

**Postconditions**:
- unit_price >= 0
- currency == "USD"
- cost_per_month = unit_price * size_gb
- billing_detail contains storage class, size, and rate

**Error Cases**:
- INVALID_RESOURCE: Invalid provider/resource_type
- UNSUPPORTED_REGION: Region not supported
- DATA_CORRUPTION: Pricing data load failed

### Method: Supports

**Purpose**: Check if S3 resource type is supported.

**Request**: ResourceDescriptor

**Response**: SupportsResponse
```protobuf
message SupportsResponse {
  bool supported = 1;
  string reason = 2;
}
```

**Preconditions**: Valid ResourceDescriptor

**Postconditions**:
- supported == true for resource_type == "s3"
- reason == "" (no "Limited support")

### Method: GetActualCost

**Purpose**: Calculate actual cost with fallback to projected.

**Request**: GetActualCostRequest

**Response**: GetActualCostResponse

**Implementation**: Uses projected monthly cost × (runtime_hours / 730) as fallback.

**Note**: S3 support inherited automatically via router.