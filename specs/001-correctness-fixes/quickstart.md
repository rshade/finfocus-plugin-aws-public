# Quickstart: Correctness Fixes Batch

**Branch**: `001-correctness-fixes`

## Prerequisites

- Go 1.25+
- `make` available
- Pricing data generated (`make develop`)

## Build and Test

```bash
# Build with us-east-1 pricing (for real cost testing)
make build-default-region

# Run all unit tests
make test

# Run linter (extended timeout recommended)
make lint

# Run specific test for zero-cost resources
go test -v ./internal/plugin/... -run TestZeroCost

# Run specific test for confidence signal
go test -v ./internal/plugin/... -run TestConfidence
```

## Verify Zero-Cost Fix (#294, #293)

### Projected Cost Path

```bash
# Build and start plugin
make build-default-region
./dist/finfocus-plugin-aws-public-us-east-1 &

# Test LaunchTemplate returns $0
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "aws:ec2/launchTemplate:LaunchTemplate",
    "region": "us-east-1"
  }
}' localhost:<PORT> finfocus.v1.CostSourceService/GetProjectedCost
# Expected: cost_per_month = 0, billing_detail contains "zero-cost"
```

### Actual Cost Path

```bash
# Test VPC ARN returns $0 without error
grpcurl -plaintext -d '{
  "arn": "arn:aws:ec2:us-east-1:123456789012:vpc/vpc-abc123",
  "start": "2026-01-01T00:00:00Z",
  "end": "2026-02-01T00:00:00Z"
}' localhost:<PORT> finfocus.v1.CostSourceService/GetActualCost
# Expected: empty results array (no error), no "tags missing sku" log
```

## Verify Confidence Signal (#292)

```bash
# RDS with missing engine tag (should show defaults)
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "rds",
    "sku": "db.t3.micro",
    "region": "us-east-1",
    "tags": {"allocatedStorage": "100"}
  }
}' localhost:<PORT> finfocus.v1.CostSourceService/GetProjectedCost
# Expected: billing_detail contains [defaults:engine=mysql,storageType=gp2]
#           and [confidence:medium]

# EBS with all properties explicit (high confidence)
grpcurl -plaintext -d '{
  "resource": {
    "provider": "aws",
    "resource_type": "ebs",
    "sku": "gp3",
    "region": "us-east-1",
    "tags": {"size": "100"}
  }
}' localhost:<PORT> finfocus.v1.CostSourceService/GetProjectedCost
# Expected: billing_detail contains [confidence:high]
#           NO [defaults:...] tag
```

## Verify Lint Compliance (#289)

```bash
# Must report zero issues
make lint

# Verify .golangci.yml is tracked
git ls-files .golangci.yml
# Expected: .golangci.yml (listed)
```

## Key Files to Review

| Area                    | Files                                     |
|-------------------------|-------------------------------------------|
| Zero-cost patterns      | `internal/plugin/constants.go`            |
| Projected cost defaults | `internal/plugin/projected.go`            |
| Actual cost validation  | `internal/plugin/validation.go`           |
| Actual cost handler     | `internal/plugin/actual.go`               |
| Lint configuration      | `.golangci.yml`                           |
