# Quickstart: EKS Cluster Cost Estimation

**Date**: 2025-12-06
**Feature**: 010-eks-cost-estimation

## Overview

This guide walks through implementing EKS cluster cost estimation in the finfocus-plugin-aws-public. The implementation adds support for `resource_type: "eks"` to the gRPC service.

## Prerequisites

- Go 1.25.4 installed
- Access to AWS pricing API data
- Familiarity with existing EC2/S3 pricing patterns

## Implementation Steps

### 1. Extend Pricing Client Interface

**File**: `internal/pricing/client.go`

Add EKS pricing method to the PricingClient interface:

```go
type PricingClient interface {
    // ... existing methods ...
    EKSClusterPricePerHour() (float64, bool)
}
```

### 2. Add EKS Price Type

**File**: `internal/pricing/types.go`

Add eksPrice struct:

```go
type eksPrice struct {
    HourlyRate float64 `json:"hourly_rate"`
    Currency   string  `json:"currency"`
}
```

### 3. Update Pricing Client Implementation

**File**: `internal/pricing/client.go`

- Add eksPricing field to Client struct
- Initialize EKS pricing in NewClient() by filtering AmazonEKS service data
- Implement EKSClusterPricePerHour() method

### 4. Add EKS Support to Plugin

**File**: `internal/plugin/supports.go`

Add "eks" to supported resource types in Supports() method.

### 5. Implement EKS Cost Estimation

**File**: `internal/plugin/projected.go`

- Add estimateEKS() function following EC2 pattern
- Add "eks" case to GetProjectedCost() router
- Calculate: cost_per_month = hourly_rate × 730

### 6. Extend Pricing Generation Tool

**File**: `tools/generate-pricing/main.go`

Add "AmazonEKS" to supported AWS services list.

### 7. Generate EKS Pricing Data

Run pricing generation for all regions:

```bash
# Generate pricing data for us-east-1
go run tools/generate-pricing/main.go -region us-east-1

# Repeat for all 9 regions
```

### 8. Add Unit Tests

**Files**: `internal/plugin/projected_test.go`, `internal/pricing/client_test.go`

- Test estimateEKS() with mock pricing client
- Test EKSClusterPricePerHour() lookup
- Verify cost calculation: $0.10 × 730 = $73.00

### 9. Update Mock Client

**File**: `internal/plugin/plugin_test.go`

Add EKSClusterPricePerHour() to mock pricing client.

## Testing

### Unit Tests
```bash
go test ./internal/plugin -v -run TestEstimateEKS
go test ./internal/pricing -v -run TestEKSClusterPrice
```

### Integration Tests
```bash
go test ./internal/plugin -v -run TestProjectedEKS
```

### Manual Testing with grpcurl
```bash
# Test Supports
grpcurl -plaintext -d '{"provider":"aws","resource_type":"eks","region":"us-east-1"}' \
  localhost:PORT finfocus.v1.CostSourceService.Supports

# Test GetProjectedCost
grpcurl -plaintext -d '{"provider":"aws","resource_type":"eks","region":"us-east-1"}' \
  localhost:PORT finfocus.v1.CostSourceService.GetProjectedCost
```

## Build & Deploy

### Build Region Binaries
```bash
# Build for us-east-1
make build-region REGION=us-east-1

# Build for all regions
make build
```

### Verify Build
```bash
# Check binary size (< 10MB)
ls -lh bin/finfocus-plugin-aws-public-us-east-1

# Test gRPC functionality
grpcurl -plaintext localhost:PORT finfocus.v1.CostSourceService.Name
```

## Expected Output

For a valid EKS cluster request:

```json
{
  "unit_price": 0.10,
  "currency": "USD",
  "cost_per_month": 73.00,
  "billing_detail": "EKS cluster, 730 hrs/month (control plane only, excludes worker nodes)"
}
```

## Troubleshooting

### Common Issues

1. **Pricing data not found**: Ensure tools/generate-pricing includes AmazonEKS service
2. **Thread safety errors**: Verify sync.Once initialization pattern
3. **Build failures**: Check region-specific build tags are correct

### Debug Logging

Enable debug logging to see pricing lookups:

```bash
LOG_LEVEL=debug ./bin/finfocus-plugin-aws-public-us-east-1
```

## Next Steps

After implementation:
1. Run `make test` to verify all tests pass
2. Run `make lint` to check code quality
3. Create PR with conventional commit: `feat: add EKS cluster cost estimation`
4. Update CHANGELOG.md with new feature</content>
<parameter name="filePath">specs/010-eks-cost-estimation/quickstart.md