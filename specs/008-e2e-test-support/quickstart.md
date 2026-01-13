# Quickstart: E2E Test Support and Validation

**Feature**: 001-e2e-test-support
**Date**: 2025-12-02

## Overview

This feature adds E2E test support to the finfocus-plugin-aws-public plugin,
enabling reliable integration testing with finfocus-core.

## Prerequisites

- Go 1.25+
- Plugin binary built (`make build` or `goreleaser build`)
- us-east-1 region binary for testing

## Enabling Test Mode

Set the environment variable before starting the plugin:

```bash
export FINFOCUS_TEST_MODE=true
./finfocus-plugin-aws-public-us-east-1
```

**Valid Values**:

- `true` - Enable test mode (enhanced logging, validation support)
- `false` or unset - Production mode (standard behavior)
- Other values - Treated as disabled with warning logged

## Expected Cost Ranges

Use these values to validate cost calculation accuracy in E2E tests:

### EC2 t3.micro (us-east-1)

| Metric | Value | Tolerance |
|--------|-------|-----------|
| Hourly Rate | $0.0104 | ±1% |
| Monthly Cost | $7.592 (730 hours) | ±1% |
| 30-min Runtime | $0.0052 | ±1% |

### EBS gp2 8GB (us-east-1)

| Metric | Value | Tolerance |
|--------|-------|-----------|
| GB-Month Rate | $0.10 | ±5% |
| Monthly Cost (8GB) | $0.80 | ±5% |
| 30-min Runtime | $0.00137 | ±5% |

**Reference Date**: 2025-12-01 (AWS public pricing)

## Validation Example

```go
// In your E2E test
resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
    Resource: &pbc.ResourceDescriptor{
        Provider:     "aws",
        ResourceType: "ec2",
        Sku:          "t3.micro",
        Region:       "us-east-1",
    },
})

// Validate within tolerance
expected := 7.592
tolerance := 0.01 // 1%
minCost := expected * (1 - tolerance)
maxCost := expected * (1 + tolerance)

if resp.CostPerMonth < minCost || resp.CostPerMonth > maxCost {
    t.Errorf("Cost %f outside expected range [%f, %f]",
        resp.CostPerMonth, minCost, maxCost)
}
```

## Enhanced Logging

When test mode is enabled, additional debug logs are emitted:

```json
{"level":"debug","resource_type":"ec2","sku":"t3.micro","region":"us-east-1",
 "message":"Test mode: request details"}
{"level":"debug","unit_price":0.0104,"cost_per_month":7.592,
 "message":"Test mode: calculation result"}
```

Enable full debug output:

```bash
LOG_LEVEL=debug FINFOCUS_TEST_MODE=true ./finfocus-plugin-aws-public-us-east-1
```

## Actual Cost Fallback Validation

Test the fallback actual cost calculation:

```go
// 30-minute runtime
start := time.Now().Add(-30 * time.Minute)
end := time.Now()

resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
    ResourceId: `{"provider":"aws","resource_type":"ec2","sku":"t3.micro","region":"us-east-1"}`,
    Start:      timestamppb.New(start),
    End:        timestamppb.New(end),
})

// Fallback formula: projected_monthly × (runtime_hours / 730)
// 7.592 × (0.5 / 730) = 0.0052
expectedCost := 0.0052
tolerance := 0.01

// Validate result in resp.Results[0].Cost
```

## Testing Checklist

- [ ] Plugin starts with `FINFOCUS_TEST_MODE=true`
- [ ] Invalid env values log warning and disable test mode
- [ ] t3.micro EC2 cost within 1% of $7.592/month
- [ ] gp2 EBS 8GB cost within 5% of $0.80/month
- [ ] Actual cost fallback matches proration formula
- [ ] Enhanced logging visible at debug level
- [ ] Production mode unchanged when test mode disabled

## Troubleshooting

### Cost Outside Expected Range

1. Check reference date - AWS pricing may have changed
2. Verify region matches (us-east-1 expected)
3. Check tolerance percentage (1% EC2, 5% EBS)
4. Regenerate pricing data if significantly outdated

### No Enhanced Logs

1. Verify `FINFOCUS_TEST_MODE=true` (exact string)
2. Set `LOG_LEVEL=debug` for detailed output
3. Check stderr (logs go to stderr, not stdout)

### Plugin Won't Start

1. Check for PORT conflict
2. Verify binary matches region (us-east-1)
3. Check embedded pricing data integrity
