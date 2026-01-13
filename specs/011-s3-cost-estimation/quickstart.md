# Quickstart: S3 Storage Cost Estimation

**Feature**: 011-s3-cost-estimation
**Date**: 2025-12-07

## Overview

S3 storage cost estimation is now supported in the finfocus-plugin-aws-public. This guide shows how to test and use the new functionality.

## Prerequisites

- Go 1.25.4 installed
- Plugin built with S3 pricing data (run `make build-region REGION=us-east-1`)

## Testing S3 Cost Estimation

### Unit Tests

Run S3-specific tests:

```bash
go test ./internal/plugin -v -run TestEstimateS3
go test ./internal/pricing -v -run TestS3PricePerGBMonth
```

### Integration Test

Test via gRPC using grpcurl:

```bash
# Start the plugin
./bin/finfocus-plugin-aws-public-us-east-1

# In another terminal, test S3 cost estimation
grpcurl -plaintext -d '{
  "provider": "aws",
  "resourceType": "s3",
  "sku": "STANDARD",
  "region": "us-east-1",
  "tags": {"size": "100"}
}' 127.0.0.1:PORT finfocus.v1.CostSourceService.GetProjectedCost
```

Expected response:
```json
{
  "unitPrice": 0.023,
  "currency": "USD",
  "costPerMonth": 2.3,
  "billingDetail": "S3 Standard storage, 100 GB, $0.0230/GB-month"
}
```

## Usage Examples

### Standard Storage

```go
resource := &pbc.ResourceDescriptor{
    Provider:     "aws",
    ResourceType: "s3",
    Sku:          "STANDARD",
    Region:       "us-east-1",
    Tags:         map[string]string{"size": "100"},
}
// Returns ~$2.30/month
```

### Standard-IA Storage

```go
resource := &pbc.ResourceDescriptor{
    Provider:     "aws",
    ResourceType: "s3",
    Sku:          "STANDARD_IA",
    Region:       "us-east-1",
    Tags:         map[string]string{"size": "500"},
}
// Returns cost based on Standard-IA pricing
```

### Unknown Storage Class

```go
resource := &pbc.ResourceDescriptor{
    Provider:     "aws",
    ResourceType: "s3",
    Sku:          "UNKNOWN",
    Region:       "us-east-1",
    Tags:         map[string]string{"size": "100"},
}
// Returns $0.00 with explanatory billing detail
```

## Supported Storage Classes

- `STANDARD`: Standard storage
- `STANDARD_IA`: Standard Infrequent Access
- `ONEZONE_IA`: One Zone Infrequent Access
- `GLACIER`: Glacier Flexible Retrieval
- `DEEP_ARCHIVE`: Glacier Deep Archive

## Performance Characteristics

- RPC latency: < 100ms
- Pricing lookup: < 50ms
- Concurrent requests: Supports 100+ simultaneous calls
- Memory footprint: < 50MB per region binary

## Troubleshooting

### Zero Cost Returned

- Check storage class SKU spelling
- Verify region is supported
- Confirm pricing data is embedded (check build logs)

### High Latency

- Check zerolog debug logs for lookup timing
- Ensure pricing data is properly initialized
- Verify thread safety (no deadlocks)

### Build Issues

- Run `tools/generate-pricing/main.go` to fetch S3 pricing
- Ensure build tags are correct for region
- Check GoReleaser config for S3 service inclusion