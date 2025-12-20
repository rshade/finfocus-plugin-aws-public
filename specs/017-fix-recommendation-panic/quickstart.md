# Quickstart: Bug Fix and Documentation Sprint - Dec 2025

**Feature**: 017-fix-recommendation-panic

This sprint addresses critical bugs and improves documentation clarity for the PulumiCost AWS Public Plugin.

## Testing Bug Fixes

### 1. Recommendation Panic Fix
To verify the panic fix, run a batch recommendation request where at least one resource has missing pricing data (triggering a nil Impact).

```bash
# Start the plugin
./pulumicost-plugin-aws-public-us-east-1

# In another terminal, call GetRecommendations with mixed resources
grpcurl -plaintext -d '{
  "target_resources": [
    {"resource_type": "ec2", "sku": "t2.micro", "region": "us-east-1"},
    {"resource_type": "ec2", "sku": "invalid-type", "region": "us-east-1"}
  ]
}' localhost:<PORT> pulumicost.v1.CostSourceService/GetRecommendations
```
**Expected**: The plugin returns a recommendation for `t2.micro` and does not panic for `invalid-type`.

### 2. Carbon CSV Logging
Corrupt the embedded CSV (locally for testing) or provide an invalid path to verify logging.
**Expected**: Structured JSON logs in stderr showing the CSV parsing error.

### 3. PORT Deprecation Warning
Start the plugin using the legacy `PORT` environment variable.
```bash
PORT=9000 ./pulumicost-plugin-aws-public-us-east-1
```
**Expected**: A warning log appears: `"PORT environment variable is deprecated and will be removed in v0.x.x. Please use PULUMICOST_PLUGIN_PORT instead."`

## New Documentation

- **Troubleshooting**: See [TROUBLESHOOTING.md](../../TROUBLESHOOTING.md) for common error scenarios and solutions.
- **EC2 OS Mapping**: Documented in `internal/plugin/ec2_attrs.go`.
- **Correlation IDs**: Documented via GoDoc in `internal/plugin/recommendations.go`.

## Verification Checklist

1. [ ] Run `make test` to ensure all regression tests pass.
2. [ ] Run `make lint` to verify code style and newlines.
3. [ ] Verify `TROUBLESHOOTING.md` exists and is formatted correctly.
4. [ ] Verify `GetUtilization` docstrings are consolidated.
