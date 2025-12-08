# Quickstart: Lambda Function Cost Estimation

**Date**: 2025-12-07
**Feature**: 001-lambda-cost-estimation

## Overview

This guide provides step-by-step instructions for implementing Lambda function cost estimation in the PulumiCost AWS Public plugin. The feature adds support for calculating AWS Lambda costs based on request volume and compute duration.

## Prerequisites

- Go 1.25.4 installed
- Access to AWS Public Pricing API (for pricing data generation)
- Plugin codebase checked out and functional

## Implementation Steps

### 1. Extend Pricing Client Interface

Add Lambda pricing methods to `internal/pricing/client.go`:

```go
type PricingClient interface {
    // ... existing methods ...
    LambdaPricePerRequest() (float64, bool)    // $/request
    LambdaPricePerGBSecond() (float64, bool)   // $/GB-second
}
```

### 2. Add Lambda Price Types

Add to `internal/pricing/types.go`:

```go
type lambdaPrice struct {
    RequestPrice   float64  // $/request (typically $0.20/million = $0.0000002)
    GBSecondPrice  float64  // $/GB-second (typically $0.0000166667)
    Currency       string
}
```

### 3. Update Pricing Client Implementation

Modify `internal/pricing/client.go`:

- Add `lambdaPricing *lambdaPrice` field to Client struct
- Update initialization to parse Lambda pricing from AWS API data
- Filter by `servicecode == "AWSLambda"` or `ProductFamily == "Serverless"`
- Implement `LambdaPricePerRequest()` and `LambdaPricePerGBSecond()` methods

### 4. Create Lambda Estimator Function

Add to `internal/plugin/projected.go`:

```go
func (p *AWSPublicPlugin) estimateLambda(traceID string, resource *pbc.ResourceDescriptor) (*pbc.GetProjectedCostResponse, error) {
    // Extract memory size from resource.Sku (default 128MB)
    // Extract requests from tags["requests_per_month"]
    // Extract duration from tags["avg_duration_ms"]
    // Calculate GB-seconds and total cost
    // Return formatted response
}
```

### 5. Update Router and Supports

Modify `internal/plugin/projected.go`:
- Add `"lambda":` case to call `p.estimateLambda()`

Modify `internal/plugin/supports.go`:
- Move "lambda" from stub cases to fully supported cases

### 6. Update Pricing Data Generation

Modify `tools/generate-pricing/main.go`:
- Add `AWSLambda` service code support
- Extract request and duration pricing from API response

### 7. Add Tests

Create comprehensive tests in `internal/plugin/projected_test.go`:
- Unit tests for `estimateLambda()` with various inputs
- Tests for default values and error conditions
- Integration tests with mock pricing client

## Testing the Implementation

### Unit Tests

Run Lambda-specific unit tests:

```bash
go test ./internal/plugin -v -run TestEstimateLambda
go test ./internal/pricing -v -run TestLambdaPricing
```

### Integration Tests

Test end-to-end Lambda cost estimation:

```bash
go test ./internal/plugin -v -run TestLambdaIntegration
```

### Manual Testing

Use grpcurl to test the gRPC service:

```bash
# Start the plugin
go run cmd/pulumicost-plugin-aws-public/main.go

# In another terminal, test Lambda support
grpcurl -plaintext -d '{
  "provider": "aws",
  "resource_type": "lambda",
  "sku": "512",
  "region": "us-east-1",
  "tags": {
    "requests_per_month": "1000000",
    "avg_duration_ms": "200"
  }
}' localhost:50051 pulumicost.v1.CostSourceService/GetProjectedCost
```

Expected response:
```json
{
  "unit_price": 0.0000166667,
  "currency": "USD",
  "cost_per_month": 1.87,
  "billing_detail": "Lambda 512MB, 1M requests/month, 200ms avg duration, 100K GB-seconds"
}
```

## Validation Checklist

- [ ] `make lint` passes without errors
- [ ] `make test` passes all tests
- [ ] Lambda returns `supported: true` in Supports() method
- [ ] Cost calculation matches AWS pricing calculator within 1%
- [ ] Default values applied correctly for missing inputs
- [ ] Error handling works for invalid inputs
- [ ] Thread-safe concurrent access verified
- [ ] All 9 regional binaries build successfully

## Common Issues

### Pricing Data Not Found
- **Symptom**: Lambda pricing methods return `(0, false)`
- **Cause**: AWS pricing API response format changed or service code incorrect
- **Fix**: Check `tools/generate-pricing/main.go` for correct service code and attribute filtering

### Incorrect Cost Calculations
- **Symptom**: Costs don't match AWS calculator
- **Cause**: Wrong formula or unit conversions
- **Fix**: Verify GB-seconds calculation: `(memoryMB/1024) * (durationMs/1000) * requests`

### Build Failures
- **Symptom**: Region-specific builds fail
- **Cause**: Missing Lambda pricing data in embedded files
- **Fix**: Run `tools/generate-pricing/main.go` for all regions and regenerate embeds

## Performance Validation

### Latency Testing
```bash
# Benchmark Lambda cost estimation
go test -bench=BenchmarkEstimateLambda ./internal/plugin
```

### Memory Testing
```bash
# Check memory usage during concurrent requests
go test -v -run TestConcurrentLambda ./internal/plugin
```

## Deployment

### Build All Regions
```bash
make build-region REGION=us-east-1
make build-region REGION=us-west-2
# ... repeat for all 9 regions
```

### Release Process
```bash
make release  # Uses GoReleaser for all regions
```

## Next Steps

After implementation:
1. Create pull request with comprehensive tests
2. Update CHANGELOG.md with Lambda support
3. Consider adding Lambda to integration test suite
4. Monitor performance in production environment

## Related Documentation

- [AWS Lambda Pricing](https://aws.amazon.com/lambda/pricing/)
- [Plugin Architecture](../README.md)
- [Testing Guidelines](../../AGENTS.md)</content>
<parameter name="filePath">/mnt/c/GitHub/go/src/github.com/rshade/pulumicost-plugin-aws-public/specs/001-lambda-cost-estimation/quickstart.md