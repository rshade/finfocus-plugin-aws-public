# Data Model: Lambda Cost Estimation

**Feature**: 014-lambda-cost-estimation

## Entities

### Pricing Entities (Go Structs)

These structures will be added to `internal/pricing/types.go`.

```go
// lambdaPrice holds the regional pricing configuration for AWS Lambda.
// Derived from AWS Pricing API product families "Serverless" and "AWS Lambda".
type lambdaPrice struct {
    // RequestPrice is the cost per request (typically per 1M requests).
    // Source: Product Family "AWS Lambda", Group "AWS-Lambda-Requests"
    RequestPrice float64

    // GBSecondPrice is the cost per GB-second of compute duration.
    // Source: Product Family "Serverless", Group "AWS-Lambda-Duration"
    GBSecondPrice float64

    // Currency code (e.g., "USD")
    Currency string
}
```

### Resource Descriptor Mapping

Mapping from Pulumi `ResourceDescriptor` to cost factors.

| Field    | Source                                | Default |
|----------|---------------------------------------|---------|
| Memory   | `resource.Sku`                        | 128 MB  |
| Requests | `resource.Tags["requests_per_month"]` | 0       |
| Duration | `resource.Tags["avg_duration_ms"]`    | 100 ms  |
| Region   | `resource.Region`                     | N/A     |
| Arch     | `resource.Tags["arch"]`               | x86     |

### Cost Calculation Formula

```text
Memory (GB) = Memory (MB) / 1024
Duration (Seconds) = Duration (ms) / 1000
Total GB-Seconds = Memory (GB) * Duration (Seconds) * Request Count

Request Cost = Request Count * PricePerRequest
Compute Cost = Total GB-Seconds * PricePerGBSecond

Total Monthly Cost = Request Cost + Compute Cost
```
