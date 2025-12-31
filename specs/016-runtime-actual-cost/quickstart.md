# Quickstart: Testing Runtime-Based Actual Cost Estimation

**Feature Branch**: `016-runtime-actual-cost`
**Date**: 2025-12-31

## Prerequisites

- Go 1.25+ installed
- grpcurl (optional, for manual testing)
- Built plugin binary with region tag

## Build the Plugin

```bash
# Build for us-east-1 with embedded pricing data
make build-default-region

# Or build a specific region
make build-region REGION=us-east-1
```

## Run Unit Tests

```bash
# Run all unit tests (includes new timestamp resolution tests)
make test

# Run specific tests for this feature
go test -v -tags=region_use1 ./internal/plugin/... -run TestResolveTimestamps
go test -v -tags=region_use1 ./internal/plugin/... -run TestExtractPulumiCreated
go test -v -tags=region_use1 ./internal/plugin/... -run TestDetermineConfidence
go test -v -tags=region_use1 ./internal/plugin/... -run TestGetActualCost_WithPulumiCreated
```

## Run Integration Tests

```bash
# Full integration test suite
go test -tags=integration,region_use1 ./internal/plugin/... -v

# Specific integration tests for this feature
go test -tags=integration,region_use1 ./internal/plugin/... -run TestIntegration_ActualCost_RuntimeFromMetadata
```

## Manual Testing with grpcurl

### Start the Plugin

```bash
# Start the plugin (note the PORT output)
./pulumicost-plugin-aws-public-us-east-1
# Output: PORT=12345
```

### Test Case 1: Auto-Calculate from pulumi:created

```bash
# Resource created 7 days ago, using automatic timestamp resolution
START_TIME=$(date -u -d "7 days ago" +%Y-%m-%dT%H:%M:%SZ)

grpcurl -plaintext -d '{
  "resource_id": "{\"provider\":\"aws\",\"resource_type\":\"ec2\",\"sku\":\"t3.micro\",\"region\":\"us-east-1\"}",
  "tags": {
    "pulumi:created": "'$START_TIME'"
  }
}' localhost:12345 pulumicost.v1.CostSourceService/GetActualCost
```

**Expected Response**:

```json
{
  "results": [{
    "timestamp": "2025-12-24T00:00:00Z",
    "cost": 0.1234,
    "usageAmount": 168.0,
    "usageUnit": "hours",
    "source": "aws-public-fallback[confidence:HIGH]"
  }]
}
```

### Test Case 2: Imported Resource (Lower Confidence)

```bash
START_TIME=$(date -u -d "30 days ago" +%Y-%m-%dT%H:%M:%SZ)

grpcurl -plaintext -d '{
  "resource_id": "{\"provider\":\"aws\",\"resource_type\":\"ec2\",\"sku\":\"t3.micro\",\"region\":\"us-east-1\"}",
  "tags": {
    "pulumi:created": "'$START_TIME'",
    "pulumi:external": "true"
  }
}' localhost:12345 pulumicost.v1.CostSourceService/GetActualCost
```

**Expected Response** (note MEDIUM confidence):

```json
{
  "results": [{
    "timestamp": "2025-12-01T00:00:00Z",
    "cost": 0.528,
    "usageAmount": 720.0,
    "usageUnit": "hours",
    "source": "aws-public-fallback[confidence:MEDIUM] imported resource"
  }]
}
```

### Test Case 3: Explicit Timestamps Override

```bash
# Explicit last 7 days, even though resource is 30 days old
START_TIME=$(date -u -d "7 days ago" +%Y-%m-%dT%H:%M:%SZ)
END_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
CREATED_TIME=$(date -u -d "30 days ago" +%Y-%m-%dT%H:%M:%SZ)

grpcurl -plaintext -d '{
  "resource_id": "{\"provider\":\"aws\",\"resource_type\":\"ec2\",\"sku\":\"t3.micro\",\"region\":\"us-east-1\"}",
  "start": {"seconds": '$(date -d "$START_TIME" +%s)'},
  "end": {"seconds": '$(date -d "$END_TIME" +%s)'},
  "tags": {
    "pulumi:created": "'$CREATED_TIME'"
  }
}' localhost:12345 pulumicost.v1.CostSourceService/GetActualCost
```

**Expected**: Cost for 168 hours (7 days), NOT 720 hours (30 days)

### Test Case 4: Missing Timestamps (Error)

```bash
grpcurl -plaintext -d '{
  "resource_id": "{\"provider\":\"aws\",\"resource_type\":\"ec2\",\"sku\":\"t3.micro\",\"region\":\"us-east-1\"}",
  "tags": {}
}' localhost:12345 pulumicost.v1.CostSourceService/GetActualCost
```

**Expected**: gRPC error with code INVALID_ARGUMENT

## Test Fixtures

Test fixtures are provided in `test/fixtures/actual-cost/`:

| File | Description |
|------|-------------|
| `with-created.json` | Resource with valid pulumi:created |
| `with-external.json` | Imported resource (pulumi:external=true) |
| `explicit-override.json` | Both explicit times and pulumi:created |
| `missing-timestamps.json` | No timestamps (error case) |

## Verification Checklist

- [ ] Unit tests pass (`make test`)
- [ ] Integration tests pass (`go test -tags=integration,region_use1 ...`)
- [ ] Manual test case 1: Auto-calculate from pulumi:created
- [ ] Manual test case 2: Imported resource shows MEDIUM confidence
- [ ] Manual test case 3: Explicit timestamps override pulumi:created
- [ ] Manual test case 4: Missing timestamps returns error
- [ ] Lint passes (`make lint`)
- [ ] Build succeeds (`make build-default-region`)

## Troubleshooting

### "start_time is required" Error

This error occurs when:

1. No explicit `start` timestamp in request, AND
2. No valid `pulumi:created` in tags

**Solution**: Ensure either explicit timestamps or `pulumi:created` tag is provided.

### "cannot parse time" Log Warning

This warning appears when `pulumi:created` contains invalid RFC3339 format.
The plugin falls back to requiring explicit timestamps.

**Solution**: Verify timestamp format matches `2006-01-02T15:04:05Z`.

### Confidence Shows HIGH When Expected MEDIUM

Check that:

1. `pulumi:external` tag value is exactly `"true"` (case-sensitive)
2. The tag is in `tags` map, not `resource_id` JSON
