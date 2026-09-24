# Quickstart: ASG Cost Estimator

**Feature**: 001-asg-estimator | **Date**: 2026-03-26

## Prerequisites

```bash
make develop  # Install deps + generate pricing + carbon data
```

## Build & Test

```bash
# Build with real pricing (us-east-1)
make build-default-region

# Run all tests
make test

# Run ASG-specific tests
go test -v ./internal/plugin/... -run TestASG
go test -v ./internal/carbon/... -run TestASG

# Run with region tag for pricing verification
go test -tags=region_use1 -v ./internal/plugin/... -run TestASG

# Lint
make lint
```

## Manual Verification

After building with `make build-default-region`:

```bash
# Start the plugin
./bin/finfocus-plugin-aws-public-us-east-1 &

# Test ASG support (via grpcurl)
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "aws:autoscaling/group:Group",
    "region": "us-east-1"
  }
}' localhost:<PORT> finfocus.v1.CostSourceService/Supports

# Test ASG cost estimation
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "aws:autoscaling/group:Group",
    "sku": "m5.large",
    "region": "us-east-1",
    "tags": {
      "desired_capacity": "3"
    }
  }
}' localhost:<PORT> finfocus.v1.CostSourceService/GetProjectedCost
```

## Key Files to Modify

| File | Change |
| ---- | ------ |
| `internal/plugin/constants.go` | Add `serviceASG` constant |
| `internal/plugin/projected.go` | Add switch case + `estimateASG()` |
| `internal/plugin/supports.go` | Add switch case + metrics |
| `internal/plugin/pricingspec.go` | Add switch case + `asgPricingSpec()` |
| `internal/plugin/actual.go` | Add switch case in `getProjectedForResource` |
| `internal/plugin/classification.go` | Add classification entry |
| `internal/plugin/arn.go` | Update autoscaling ARN case |
| `internal/plugin/asg_attrs.go` | NEW: Tag extraction helpers |
| `internal/carbon/asg_estimator.go` | NEW: Carbon estimator |
| `internal/carbon/types.go` | Add `ASGConfig` struct |
