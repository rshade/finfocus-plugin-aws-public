# API Contracts: Plugin Rename to FinFocus

**Feature**: 001-plugin-rename
**Date**: 2026-01-11
**Status**: Unchanged

## Overview

The gRPC API contracts for the plugin remain **unchanged** by this rename operation. The proto interface is identical between `pulumicost.v1` (pulumicost-spec v0.4.14) and `finfocus.v1` (finfocus-spec v0.5.0). Only the proto package name has changed; message structure, field names, and service methods are identical.

## gRPC Service Definition

The plugin implements the `CostSourceService` interface from `finfocus.v1` (formerly `pulumicost.v1`):

```protobuf
service CostSourceService {
  rpc Name(NameRequest) returns (NameResponse);
  rpc Supports(ResourceDescriptor) returns (SupportsResponse);
  rpc GetProjectedCost(ResourceDescriptor) returns (GetProjectedCostResponse);
  rpc GetActualCost(GetActualCostRequest) returns (GetActualCostResponse);
  rpc GetPricingSpec(GetPricingSpecRequest) returns (GetPricingSpecResponse);
}
```

## Proto Package Change

### Before (pulumicost-spec v0.4.14)

```protobuf
syntax = "proto3";
package pulumicost.v1;

// Messages...
message ResourceDescriptor { ... }
message GetProjectedCostResponse { ... }
```

### After (finfocus-spec v0.5.0)

```protobuf
syntax = "proto3";
package finfocus.v1;

// Messages... (unchanged)
message ResourceDescriptor { ... }
message GetProjectedCostResponse { ... }
```

**Note**: Message definitions are identical; only the package name changed.

## Message Definitions

### NameRequest / NameResponse

**Purpose**: Get plugin name

```protobuf
message NameRequest {}

message NameResponse {
  string name = 1;  // Returns "aws-public"
}
```

### ResourceDescriptor

**Purpose**: Describe a cloud resource for cost estimation

```protobuf
message ResourceDescriptor {
  string region = 1;           // AWS region (e.g., "us-east-1")
  string resource_type = 2;     // Pulumi resource type
  string provider = 3;          // Cloud provider (e.g., "aws")
  map<string, string> properties = 4;  // Resource properties
}
```

### SupportsRequest / SupportsResponse

**Purpose**: Check if plugin supports a resource type

```protobuf
message SupportsRequest {
  ResourceDescriptor descriptor = 1;
}

message SupportsResponse {
  bool supported = 1;
  string reason = 2;  // Optional: Reason if not supported
}
```

### GetProjectedCostResponse

**Purpose**: Return projected cost for a resource

```protobuf
message GetProjectedCostResponse {
  double unit_price = 1;        // Cost per unit
  string currency = 2;          // Currency code (e.g., "USD")
  double cost_per_month = 3;     // Estimated monthly cost
  BillingDetail billing_detail = 4;  // Detailed breakdown
}

message BillingDetail {
  repeated CostComponent components = 1;
}

message CostComponent {
  string name = 1;              // Component name (e.g., "Compute")
  double amount = 2;             // Cost amount
  string unit = 3;              // Pricing unit (e.g., "Hours")
}
```

### GetActualCostRequest / GetActualCostResponse

**Purpose**: Get actual cost (with fallback to projected)

```protobuf
message GetActualCostRequest {
  ResourceDescriptor descriptor = 1;
  string start_time = 2;        // ISO 8601 timestamp
  string end_time = 3;          // ISO 8601 timestamp
}

message GetActualCostResponse {
  double actual_cost = 1;        // Actual cost (if available)
  double projected_cost = 2;     // Fallback projected cost
  string currency = 3;
  bool has_actual_data = 4;      // True if actual data available
}
```

### GetPricingSpecRequest / GetPricingSpecResponse

**Purpose**: Get detailed pricing specification (optional)

```protobuf
message GetPricingSpecRequest {}

message GetPricingSpecResponse {
  string version = 1;           // Pricing data version
  repeated string supported_regions = 2;
  repeated string supported_services = 3;
  string last_updated = 4;      // ISO 8601 timestamp
}
```

## Error Handling

The plugin uses standard gRPC status codes:

| Status Code | When Used |
|-------------|-----------|
| `OK` | Request successful |
| `INVALID_ARGUMENT` | Invalid ResourceDescriptor (missing required fields) |
| `NOT_FOUND` | Pricing data not found for resource |
| `UNAVAILABLE` | Plugin not ready (pricing data loading) |
| `INTERNAL` | Unexpected error (e.g., data corruption) |

### Error Details

Errors include structured details via proto ErrorCode enum:

```protobuf
enum ErrorCode {
  ERROR_CODE_UNSPECIFIED = 0;
  ERROR_CODE_INVALID_RESOURCE = 6;        // Missing required fields
  ERROR_CODE_UNSUPPORTED_REGION = 9;     // Region not supported
  ERROR_CODE_DATA_CORRUPTION = 11;       // Pricing data load failed
}
```

## Concurrency Model

The plugin is designed for concurrent gRPC calls:

- **Thread-safe**: All gRPC method handlers are safe for concurrent execution
- **Stateless**: Each request is independent (no request-scoped state)
- **Read-only**: Pricing data is loaded once and cached (read-only after initialization)

## Performance Targets

| Operation | Target | Notes |
|-----------|--------|-------|
| Name() | <10ms | Simple static response |
| Supports() | <10ms | Region + resource type check |
| GetProjectedCost() | <100ms | Indexed pricing lookup + calculation |
| GetActualCost() | <100ms | Same as GetProjectedCost (fallback) |
| GetPricingSpec() | <10ms | Static pricing metadata |

## Version Compatibility

### Proto Package Names

| Spec Version | Proto Package | Status |
|--------------|---------------|--------|
| pulumicost-spec v0.4.14 | `pulumicost.v1` | Deprecated |
| finfocus-spec v0.5.0 | `finfocus.v1` | Current |

### Breaking Changes

**No breaking changes to message structure** between pulumicost.v1 and finfocus.v1:
- Message names unchanged
- Field names unchanged
- Field numbers unchanged
- Field types unchanged
- Service methods unchanged

**Only change**: Proto package name (`pulumicost.v1` → `finfocus.v1`)

## Implementation Notes

### Go Client Code

**Before (pulumicost):**
```go
import pulumicostv1 "github.com/rshade/pulumicost-spec/proto/v1"

desc := &pulumicostv1.ResourceDescriptor{
    Region: "us-east-1",
    ResourceType: "aws:ec2/instance:Instance",
}
```

**After (finfocus):**
```go
import finfocusv1 "github.com/rshade/finfocus-spec/proto/v1"

desc := &finfocusv1.ResourceDescriptor{
    Region: "us-east-1",
    ResourceType: "aws:ec2/instance:Instance",
}
```

**Note**: Only the import path changes; usage is identical.

## Validation Rules

### ResourceDescriptor Validation

- `region` must be one of: `us-east-1`, `us-west-2`, `eu-west-1`
- `resource_type` must be supported AWS service type
- `provider` must be: `aws`
- `properties` must contain required fields for resource type

### GetProjectedCostResponse Validation

- `unit_price` must be >= 0
- `currency` must be: `USD`
- `cost_per_month` is calculated as `unit_price * 730`

## Testing the gRPC Interface

### Using grpcurl

**List services:**
```bash
grpcurl -plaintext 127.0.0.1:50051 list
```

**Get plugin name:**
```bash
grpcurl -plaintext 127.0.0.1:50051 \
  finfocus.v1.CostSourceService/Name
```

**Test Supports:**
```bash
grpcurl -plaintext -d '{
  "region": "us-east-1",
  "resource_type": "aws:ec2/instance:Instance"
}' 127.0.0.1:50051 \
  finfocus.v1.CostSourceService/Supports
```

**Test GetProjectedCost:**
```bash
grpcurl -plaintext -d '{
  "region": "us-east-1",
  "resource_type": "aws:ec2/instance:Instance",
  "properties": {
    "instanceType": "t3.micro"
  }
}' 127.0.0.1:50051 \
  finfocus.v1.CostSourceService/GetProjectedCost
```

### Using Go Client

```go
import (
    "context"
    finfocusv1 "github.com/rshade/finfocus-spec/proto/v1"
    "google.golang.org/grpc"
)

func main() {
    conn, err := grpc.Dial("127.0.0.1:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := finfocusv1.NewCostSourceServiceClient(conn)

    desc := &finfocusv1.ResourceDescriptor{
        Region: "us-east-1",
        ResourceType: "aws:ec2/instance:Instance",
        Properties: map[string]string{
            "instanceType": "t3.micro",
        },
    }

    resp, err := client.GetProjectedCost(context.Background(), desc)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Unit price: $%.4f", resp.UnitPrice)
    log.Printf("Monthly cost: $%.2f", resp.CostPerMonth)
}
```

## References

- **Proto Definitions**: finfocus-spec v0.5.0
- **Data Model**: `../data-model.md` - Entity documentation
- **Constitution**: `.specify/memory/constitution.md` - Protocol consistency requirements
- **RENAME-PLAN.md**: Phase 3 migration details