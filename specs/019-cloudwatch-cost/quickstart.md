# Quickstart: CloudWatch Cost Estimation

**Feature**: 019-cloudwatch-cost
**Date**: 2025-12-30

## Prerequisites

- Go 1.25+
- `make` available
- AWS pricing data generated (via `make generate-pricing`)

## Quick Validation

### 1. Generate Pricing Data

```bash
# Generate pricing for us-east-1 (includes CloudWatch)
go run ./tools/generate-pricing --regions us-east-1 --out-dir ./internal/pricing/data
```

### 2. Build Region-Specific Binary

```bash
# Build us-east-1 binary with CloudWatch support
make build-region REGION=us-east-1
```

### 3. Run Tests

```bash
# Unit tests
make test

# Integration tests (requires built binary)
go test -tags=integration ./internal/plugin/... -run TestCloudWatch -v
```

## Usage Examples

### Example 1: Estimate Log Costs

```bash
# Start plugin
./pulumicost-plugin-aws-public-us-east-1 &
PORT=$(grep -m1 "PORT=" /proc/$!/fd/1 | cut -d= -f2)

# Query log costs (100 GB ingestion, 500 GB storage)
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "cloudwatch",
    "sku": "logs",
    "region": "us-east-1",
    "tags": {
      "log_ingestion_gb": "100",
      "log_storage_gb": "500"
    }
  }
}' localhost:$PORT pulumicost.v1.CostSourceService/GetProjectedCost
```

Expected response:

```json
{
  "unitPrice": 0.5,
  "currency": "USD",
  "costPerMonth": 65,
  "billingDetail": "CloudWatch Logs: 100 GB ingestion @ $0.50/GB + 500 GB storage @ $0.03/GB-mo"
}
```

### Example 2: Estimate Metric Costs

```bash
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "cloudwatch",
    "sku": "metrics",
    "region": "us-east-1",
    "tags": {
      "custom_metrics": "50"
    }
  }
}' localhost:$PORT pulumicost.v1.CostSourceService/GetProjectedCost
```

Expected response:

```json
{
  "unitPrice": 0.3,
  "currency": "USD",
  "costPerMonth": 15,
  "billingDetail": "CloudWatch Metrics: 50 custom metrics @ $0.30/metric"
}
```

### Example 3: Combined Estimation

```bash
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "cloudwatch",
    "sku": "combined",
    "region": "us-east-1",
    "tags": {
      "log_ingestion_gb": "100",
      "log_storage_gb": "500",
      "custom_metrics": "50"
    }
  }
}' localhost:$PORT pulumicost.v1.CostSourceService/GetProjectedCost
```

Expected response:

```json
{
  "unitPrice": 0,
  "currency": "USD",
  "costPerMonth": 80,
  "billingDetail": "CloudWatch: Logs + Metrics combined = $80.00/month"
}
```

## Testing Tiered Pricing

### High-Volume Log Ingestion (15 TB)

```bash
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "cloudwatch",
    "sku": "logs",
    "region": "us-east-1",
    "tags": {
      "log_ingestion_gb": "15360"
    }
  }
}' localhost:$PORT pulumicost.v1.CostSourceService/GetProjectedCost
```

Expected calculation:

- First 10 TB (10240 GB) @ $0.50 = $5,120
- Next 5 TB (5120 GB) @ $0.25 = $1,280
- Total: $6,400

### High-Volume Metrics (100,000 metrics)

```bash
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "cloudwatch",
    "sku": "metrics",
    "region": "us-east-1",
    "tags": {
      "custom_metrics": "100000"
    }
  }
}' localhost:$PORT pulumicost.v1.CostSourceService/GetProjectedCost
```

Expected calculation:

- First 10,000 @ $0.30 = $3,000
- Next 90,000 @ $0.10 = $9,000
- Total: $12,000

## Troubleshooting

### No Pricing Data

If you see `$0.00` with message "pricing unavailable":

1. Check if CloudWatch JSON exists: `ls internal/pricing/data/cloudwatch_*.json`
2. Regenerate: `go run ./tools/generate-pricing --regions us-east-1 --out-dir ./internal/pricing/data`
3. Rebuild: `make build-region REGION=us-east-1`

### Region Mismatch

If `Supports()` returns `false`:

- Verify the binary region matches the request region
- Binary name indicates region: `pulumicost-plugin-aws-public-us-east-1`

### Invalid Tag Values

Non-numeric tags are logged as warnings and treated as 0:

```bash
# This will log a warning and use 0 for ingestion
grpcurl -plaintext -d '{
  "resource": {
    "tags": { "log_ingestion_gb": "not-a-number" }
  }
}' localhost:$PORT pulumicost.v1.CostSourceService/GetProjectedCost
```
