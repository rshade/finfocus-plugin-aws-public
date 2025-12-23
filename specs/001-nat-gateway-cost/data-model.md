# Data Model: VPC NAT Gateway

## Entities

### NAT Gateway Resource
**Type**: `pbc.ResourceDescriptor`
**Fields**:
- `ResourceType`: `"natgw"`, `"nat_gateway"`, or `"nat-gateway"`
- `Provider`: `"aws"`
- `Region`: Standard AWS region (e.g., `"us-east-1"`)
- `Tags`:
    - `data_processed_gb`: (Required for non-zero data cost) Numeric string, non-negative.

### NAT Gateway Pricing
**Type**: `natGatewayPrice` struct (internal/pricing/types.go)
**Fields**:
- `HourlyRate`: `float64`
- `DataProcessingRate`: `float64`
- `Currency`: `string` (fixed to "USD")

## Validation Rules

1. **Resource Type**: Must be one of the three supported variants.
2. **Region**: Must match the region binary being used.
3. **Data Processed Tag**:
    - If missing: Default to 0.0 GB.
    - If present but empty: Return `InvalidArgument`.
    - If present but non-numeric: Return `InvalidArgument`.
    - If present but negative: Return `InvalidArgument`.

## State Transitions
N/A (Stateless gRPC plugin)
