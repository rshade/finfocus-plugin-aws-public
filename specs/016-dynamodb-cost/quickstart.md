# Quickstart: DynamoDB Cost Estimation

## Overview

This feature enables cost estimation for AWS DynamoDB resources based on their capacity mode (On-Demand or Provisioned) and storage usage.

## Estimating a Table

To estimate a DynamoDB table, provide a `ResourceDescriptor` with the following fields:

### On-Demand Example

```json
{
  "provider": "aws",
  "resource_type": "dynamodb",
  "sku": "on-demand",
  "region": "us-east-1",
  "tags": {
    "read_requests_per_month": "10000000",
    "write_requests_per_month": "1000000",
    "storage_gb": "50"
  }
}
```

### Provisioned Example

```json
{
  "provider": "aws",
  "resource_type": "dynamodb",
  "sku": "provisioned",
  "region": "us-east-1",
  "tags": {
    "read_capacity_units": "100",
    "write_capacity_units": "50",
    "storage_gb": "50"
  }
}
```

## Running Tests

### Unit Tests
```bash
go test ./internal/plugin -v -run TestProjectedCost_DynamoDB
```

### Integration Tests
Ensure you build for the specific region first:
```bash
make build-region REGION=us-east-1
# Then run against the binary using a gRPC client (e.g., grpcurl)
```
