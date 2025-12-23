# API Contracts: VPC NAT Gateway Cost Estimation

**Feature**: VPC NAT Gateway Cost Estimation
**Date**: 2025-12-21

## gRPC Service Extensions

### PricingClient Interface

Extended with NAT Gateway pricing method:

```go
// NATGatewayPrice contains pricing information for VPC NAT Gateways.
type NATGatewayPrice struct {
    // HourlyRate is the hourly rate for NAT Gateways in the current region.
    HourlyRate float64
    // DataProcessingRate is the data processing rate per GB.
    DataProcessingRate float64
    // Currency is the pricing currency (always "USD").
    Currency string
}

type PricingClient interface {
    // Existing methods...

    // NATGatewayPrice returns NAT Gateway pricing for the current region.
    // Returns (price, true) if available, (nil, false) if not supported.
    NATGatewayPrice() (*NATGatewayPrice, bool)
}
```

### CostSourceService Methods (Existing)

No new gRPC methods required. NAT Gateway support is added via:

- `Supports(ResourceDescriptor)`: Returns supported=true for NAT Gateway
- `GetProjectedCost(ResourceDescriptor)`: Returns NAT Gateway costs

### ResourceDescriptor Schema

```protobuf
message ResourceDescriptor {
  string provider = 1;           // "aws"
  string resource_type = 2;      // "natgw", "nat_gateway", "nat-gateway"
  string region = 3;             // e.g., "us-east-1"
  map<string, string> tags = 4;  // optional: {"data_processed_gb": "1000"}
}
```

### GetProjectedCostResponse Schema

```protobuf
message GetProjectedCostResponse {
  double unit_price = 1;         // Hourly rate in USD
  string currency = 2;           // "USD"
  double cost_per_month = 3;     // Total monthly cost
  string billing_detail = 4;     // Cost breakdown description
}
```

## Data Processing Rules

- `data_processed_gb` tag is optional
- If missing, defaults to 0 (hourly cost only)
- Must be parseable as float64 >= 0
- Invalid values (empty, non-numeric, negative) result in gRPC error

## Error Conditions

- `ERROR_CODE_UNSUPPORTED_REGION`: Region not supported
- `ERROR_CODE_INVALID_RESOURCE`: Invalid resource descriptor or tag value
- `ERROR_CODE_DATA_CORRUPTION`: Pricing data load failure
- gRPC `INVALID_ARGUMENT`: Invalid `data_processed_gb` tag value

## Performance Contracts

- RPC latency: <100ms per call
- Concurrent calls: Support >=100 simultaneous requests
- Memory usage: <400MB per region binary
