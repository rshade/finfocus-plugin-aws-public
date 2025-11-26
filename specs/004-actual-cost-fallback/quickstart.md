# Quickstart: Fallback GetActualCost

**Date**: 2025-11-25
**Feature**: 004-actual-cost-fallback

## Overview

This guide explains how to use the new GetActualCost fallback
functionality in the pulumicost-plugin-aws-public.

## Prerequisites

- pulumicost-plugin-aws-public binary for your region
- grpcurl or similar gRPC client for testing
- Understanding of ResourceDescriptor structure

## Basic Usage

### Starting the Plugin

```bash
# Start the us-east-1 region binary
./pulumicost-plugin-aws-public-us-east-1
# Output: PORT=12345
```

### Calling GetActualCost

```bash
# Calculate actual cost for an EC2 instance running 24 hours
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "ec2",
    "sku": "t3.micro",
    "region": "us-east-1"
  },
  "from": "2025-01-01T00:00:00Z",
  "to": "2025-01-02T00:00:00Z"
}' localhost:12345 pulumicost.v1.CostSourceService/GetActualCost
```

### Example Response

```json
{
  "cost": 0.2493,
  "currency": "USD",
  "billingDetail": "Fallback estimate: On-demand Linux, Shared × 24.00 hours / 730"
}
```

## Calculation Formula

The actual cost is calculated using:

```text
actual_cost = projected_monthly_cost × (runtime_hours / 730)
```

Where:

- `projected_monthly_cost` = hourly_rate × 730 (from GetProjectedCost)
- `runtime_hours` = (to - from) in hours
- `730` = standard hours per month

### Example Calculation

```text
t3.micro hourly rate: $0.0104
Projected monthly: $0.0104 × 730 = $7.592
Runtime: 24 hours
Actual cost: $7.592 × (24 / 730) = $0.2493
```

## Supported Resource Types

| Type | Support Level | Notes |
|------|---------------|-------|
| ec2 | Full | Hourly rate × time |
| ebs | Full | GB-month rate × time × size |
| s3 | Stub | Returns $0.00 |
| lambda | Stub | Returns $0.00 |
| rds | Stub | Returns $0.00 |
| dynamodb | Stub | Returns $0.00 |

## Error Handling

### Invalid Time Range

```bash
# from > to returns error
grpcurl -plaintext -d '{
  "resource": {...},
  "from": "2025-01-02T00:00:00Z",
  "to": "2025-01-01T00:00:00Z"
}' localhost:12345 pulumicost.v1.CostSourceService/GetActualCost

# Error: invalid time range: from is after to
```

### Region Mismatch

```bash
# Resource region doesn't match binary region
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "ec2",
    "sku": "t3.micro",
    "region": "eu-west-1"  # Mismatch with us-east-1 binary
  },
  "from": "2025-01-01T00:00:00Z",
  "to": "2025-01-02T00:00:00Z"
}' localhost:12345 pulumicost.v1.CostSourceService/GetActualCost

# Error: region mismatch (ERROR_CODE_UNSUPPORTED_REGION)
```

### Zero Duration

```bash
# from == to returns $0.00
grpcurl -plaintext -d '{
  "resource": {...},
  "from": "2025-01-01T00:00:00Z",
  "to": "2025-01-01T00:00:00Z"
}' localhost:12345 pulumicost.v1.CostSourceService/GetActualCost

# Response: {"cost": 0, "currency": "USD", "billingDetail": "Zero runtime hours"}
```

## Integration with PulumiCost Core

PulumiCost core automatically discovers and calls GetActualCost when
analyzing resource runtime costs. No additional configuration needed
once the plugin is available.

## Limitations

- This is a **fallback estimate** based on public pricing
- Actual AWS costs may vary due to:
  - Spot/Reserved pricing
  - Data transfer costs
  - Usage-based services (S3, Lambda)
  - Regional price differences
- Stub services always return $0.00

## Testing

Run unit tests to verify the implementation:

```bash
go test ./internal/plugin -v -run TestGetActualCost
```
