# Quickstart: Testing Lambda Cost Estimation

**Feature**: 014-lambda-cost-estimation

## Overview

This feature adds cost estimation for `aws:lambda:function` resources.
Estimates are based on expected usage provided via resource tags.

## Prerequisites

- Built plugin binary for your region (e.g., `make build-region REGION=us-east-1`)
- A JSON file containing a ResourceDescriptor for a Lambda function (or use
  the provided testdata).

## Step 1: Create Test Data

Create a file named `lambda-test.json`:

```json
{
  "trace_id": "test-trace-1",
  "resource": {
    "provider": "aws",
    "resource_type": "aws:lambda:function",
    "sku": "512",
    "region": "us-east-1",
    "tags": {
      "requests_per_month": "1000000",
      "avg_duration_ms": "200"
    }
  }
}
```

## Step 2: Run the Plugin

Start the plugin and use `grpcurl` or a test client to invoke
`GetProjectedCost`.

*(Note: Since this is a gRPC plugin, you typically test it via `make test` or
integration tests. For manual verification, use the provided unit tests).*

## Step 3: Verify Output

The expected cost calculation for the above example (us-east-1 pricing):

- **Memory**: 512 MB = 0.5 GB
- **Duration**: 200 ms = 0.2 s
- **Requests**: 1,000,000
- **GB-Seconds**: 0.5 \* 0.2 \* 1,000,000 = 100,000 GB-seconds

**Costs**:

- **Requests**: 1M \* $0.20/M = $0.20
- **Compute**: 100k \* $0.0000166667 = $1.67
- **Total**: $1.87

The plugin should return approximately **$1.87**.
