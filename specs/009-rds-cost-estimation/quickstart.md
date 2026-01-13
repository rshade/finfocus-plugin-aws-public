# Quickstart: RDS Instance Cost Estimation

**Feature**: 009-rds-cost-estimation
**Date**: 2025-12-02

## Overview

This feature adds RDS database instance cost estimation to the aws-public plugin. After
implementation, the plugin will return accurate cost estimates for RDS instances including
compute (instance hours) and storage costs.

## Prerequisites

- Go 1.25+
- make, golangci-lint
- grpcurl (for manual testing)

## Build & Test

```bash
# Run linting
make lint

# Run all tests
make test

# Build specific region binary (e.g., us-east-1)
go build -tags region_use1 -o finfocus-plugin-aws-public-us-east-1 \
    ./cmd/finfocus-plugin-aws-public

# Generate pricing data (includes RDS)
go run ./tools/generate-pricing \
    --regions us-east-1 \
    --service AmazonEC2,AmazonRDS \
    --out-dir ./data
```

## Usage Examples

### Basic RDS Query (MySQL)

```bash
# Start the plugin
./finfocus-plugin-aws-public-us-east-1 &
# Capture: PORT=12345

# Query RDS cost
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "rds",
    "sku": "db.t3.medium",
    "region": "us-east-1",
    "tags": {
      "engine": "mysql",
      "storage_type": "gp3",
      "storage_size": "100"
    }
  }
}' localhost:12345 finfocus.v1.CostSourceService/GetProjectedCost
```

**Expected Response**:

```json
{
  "unitPrice": 0.068,
  "currency": "USD",
  "costPerMonth": 59.64,
  "billingDetail": "RDS db.t3.medium MySQL, 730 hrs/month + 100GB gp3 storage"
}
```

### PostgreSQL with Defaults

```bash
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "rds",
    "sku": "db.m5.large",
    "region": "us-east-1",
    "tags": {
      "engine": "postgres"
    }
  }
}' localhost:12345 finfocus.v1.CostSourceService/GetProjectedCost
```

**Expected Response** (with defaults: gp2, 20GB):

```json
{
  "unitPrice": 0.171,
  "currency": "USD",
  "costPerMonth": 127.13,
  "billingDetail": "RDS db.m5.large PostgreSQL, 730 hrs/month + 20GB gp2 storage (defaulted)"
}
```

### Supports Query

```bash
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "rds",
    "sku": "db.t3.medium",
    "region": "us-east-1"
  }
}' localhost:12345 finfocus.v1.CostSourceService/Supports
```

**Expected Response**:

```json
{
  "supported": true,
  "reason": "RDS instances supported with instance and storage cost estimation"
}
```

## Testing Checklist

- [ ] `make lint` passes
- [ ] `make test` passes (all existing + new RDS tests)
- [ ] Manual grpcurl test for MySQL returns expected cost
- [ ] Manual grpcurl test for PostgreSQL returns expected cost
- [ ] Supports() returns `supported=true` without "Limited support"
- [ ] Unknown instance type returns $0 with explanation
- [ ] Missing engine tag defaults to MySQL
- [ ] Missing storage tags default to gp2/20GB

## Key Files Modified

| File | Changes |
|------|---------|
| `internal/pricing/client.go` | Add RDS interface methods, indexes |
| `internal/pricing/types.go` | Add `rdsInstancePrice`, `rdsStoragePrice` |
| `internal/plugin/projected.go` | Add `estimateRDS()`, update router |
| `internal/plugin/supports.go` | Move RDS to fully-supported |
| `tools/generate-pricing/main.go` | Support AmazonRDS service |

## Troubleshooting

### "RDS instance type not found in pricing data"

- Verify the instance type starts with `db.` prefix
- Check the region binary matches the resource region
- Ensure pricing data was regenerated with `--service AmazonEC2,AmazonRDS`

### "Resource region does not match plugin region"

- Use the correct region-specific binary (e.g., `finfocus-plugin-aws-public-us-east-1`)
- Ensure `resource.region` matches the binary's embedded region

### Storage cost seems wrong

- Verify `storage_size` is a valid positive integer string
- Check `storage_type` is one of: gp2, gp3, io1, io2, standard
