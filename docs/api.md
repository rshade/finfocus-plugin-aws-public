# FinFocus AWS Public Plugin API Documentation

## Overview

The FinFocus AWS Public Plugin implements the `CostSourceService` gRPC service
defined in the `finfocus.v1` protocol buffer specification. This document provides
detailed API documentation for developers integrating with or extending the plugin.

## Service Definition

```protobuf
service CostSourceService {
  rpc Name(NameRequest) returns (NameResponse);
  rpc Supports(SupportsRequest) returns (SupportsResponse);
  rpc GetProjectedCost(GetProjectedCostRequest) returns (GetProjectedCostResponse);
  rpc GetActualCost(GetActualCostRequest) returns (GetActualCostResponse);
  rpc GetRecommendations(GetRecommendationsRequest) returns (GetRecommendationsResponse);
  rpc GetPluginInfo(GetPluginInfoRequest) returns (GetPluginInfoResponse);
}
```

## RPC Methods

### Name

Returns the plugin identifier.

**Request:** `NameRequest` (empty message)
**Response:** `NameResponse`

```json
{
  "name": "finfocus-plugin-aws-public"
}
```

### GetPluginInfo

Returns metadata about the plugin.

**Request:** `GetPluginInfoRequest` (empty message)
**Response:** `GetPluginInfoResponse`

```json
{
  "name": "finfocus-plugin-aws-public",
  "version": "0.0.3",
  "spec_version": "v0.4.14",
  "providers": ["aws"],
  "metadata": {
    "region": "us-east-1",
    "type": "public-pricing-fallback"
  }
}
```

### Supports

Checks if the plugin can provide cost estimates for a given resource.

**Request:** `SupportsRequest`

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "ec2",
    "sku": "t3.micro",
    "region": "us-east-1"
  }
}
```

**Response:** `SupportsResponse`

```json
{
  "supported": true,
  "supported_metrics": ["METRIC_KIND_CARBON_FOOTPRINT"]
}
```

### GetProjectedCost

Estimates monthly cost for a resource.

**Request:** `GetProjectedCostRequest`

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "ec2",
    "sku": "t3.micro",
    "region": "us-east-1"
  },
  "utilization_percentage": 0.8
}
```

**Response:** `GetProjectedCostResponse`

```json
{
  "cost_per_month": 7.592,
  "unit_price": 0.0104,
  "currency": "USD",
  "billing_detail": "On-demand Linux, Shared tenancy, 730 hrs/month",
  "impact_metrics": [
    {
      "kind": "METRIC_KIND_CARBON_FOOTPRINT",
      "value": 3507.6,
      "unit": "gCO2e"
    }
  ]
}
```

### GetActualCost

Retrieves actual historical cost data for a resource.

**Request:** `GetActualCostRequest`

```json
{
  "resource_id": "i-abc123",
  "tags": {
    "provider": "aws",
    "resource_type": "ec2",
    "sku": "t3.micro",
    "region": "us-east-1"
  },
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-01-31T23:59:59Z"
}
```

**Response:** `GetActualCostResponse`

```json
{
  "cost_per_month": 7.592,
  "unit_price": 0.0104,
  "currency": "USD",
  "billing_detail": "Actual usage: 744 hours at $0.0104/hour"
}
```

### GetRecommendations

Provides cost optimization recommendations for resources.

**Request:** `GetRecommendationsRequest`

```json
{
  "target_resources": [
    {
      "provider": "aws",
      "resource_type": "ec2",
      "sku": "m5.large",
      "region": "us-east-1"
    }
  ]
}
```

**Response:** `GetRecommendationsResponse`

```json
{
  "recommendations": [
    {
      "action": "MODIFY",
      "resource": {
        "provider": "aws",
        "resource_type": "ec2",
        "sku": "m5.large",
        "region": "us-east-1"
      },
      "modify": {
        "recommended_config": {
          "instance_type": "t3.medium"
        }
      },
      "impact": {
        "current_cost": 96.0,
        "recommended_cost": 28.8,
        "savings": 67.2,
        "savings_percentage": 70.0
      }
    }
  ]
}
```

## Resource Types

### EC2 Instances

- **Resource Type:** `ec2`
- **SKU:** Instance type (e.g., `t3.micro`, `m5.large`)
- **Required Tags:** None
- **Optional Tags:** `platform` (windows/linux), `tenancy` (shared/dedicated/host)

### EBS Volumes

- **Resource Type:** `ebs`
- **SKU:** Volume type (e.g., `gp2`, `gp3`, `io1`)
- **Required Tags:** `size` (in GB)
- **Default Size:** 8GB if not specified

### Lambda Functions

- **Resource Type:** `lambda`
- **SKU:** Memory allocation (MB)
- **Required Tags:** `requests_per_month`, `avg_duration_ms`
- **Optional Tags:** `arch` (x86_64/arm64)

### S3 Storage

- **Resource Type:** `s3`
- **SKU:** Storage class (e.g., `STANDARD`, `STANDARD_IA`)
- **Required Tags:** `size` (in GB)
- **Default Size:** 1GB if not specified

### DynamoDB

- **Resource Type:** `dynamodb`
- **SKU:** Capacity mode (`provisioned` or `on-demand`)
- **Provisioned Mode Tags:** `read_capacity_units`, `write_capacity_units`, `storage_gb`
- **On-Demand Mode Tags:** `read_requests_per_month`, `write_requests_per_month`, `storage_gb`

### ELB Load Balancers

- **Resource Type:** `elb`
- **SKU:** Load balancer type (`alb` or `nlb`)
- **Required Tags (ALB):** `lcu_per_hour`
- **Required Tags (NLB):** `nlcu_per_hour`

## Error Codes

### ERROR_CODE_UNSUPPORTED_REGION

- **gRPC Code:** `FailedPrecondition`
- **Cause:** Resource region doesn't match plugin binary region
- **Details:** `pluginRegion`, `requiredRegion`

### ERROR_CODE_INVALID_RESOURCE

- **gRPC Code:** `InvalidArgument`
- **Cause:** ResourceDescriptor missing required fields

### ERROR_CODE_PRICING_UNAVAILABLE

- **gRPC Code:** `NotFound`
- **Cause:** Pricing data not available for requested resource

## Transport

### gRPC Transport

- **Protocol:** gRPC over HTTP/2
- **Port:** Announced on stdout (e.g., `PORT=50051`)
- **Security:** No TLS (local communication only)

### Plugin Lifecycle

1. **Startup:** Core starts plugin binary as subprocess
2. **Discovery:** Core reads PORT from stdout
3. **Connection:** Core establishes gRPC connection to `127.0.0.1:<port>`
4. **Shutdown:** Core cancels context to trigger graceful shutdown

## Testing

### Unit Tests

Run plugin unit tests:

```bash
go test ./internal/plugin/... -v
```

### Integration Tests

Run end-to-end tests:

```bash
go test ./internal/plugin/... -tags=integration -v
```

### Manual Testing

Use grpcurl for manual API testing:

```bash
grpcurl -plaintext localhost:50051 \
  finfocus.v1.CostSourceService/GetProjectedCost \
  -d '{"resource": {"provider": "aws", "resource_type": "ec2", "sku": "t3.micro", "region": "us-east-1"}}'
```

Or for `GetPluginInfo`:

```bash
grpcurl -plaintext localhost:50051 \
  finfocus.v1.CostSourceService/GetPluginInfo
```

## Rate Limiting

The plugin implements rate limiting to ensure fair usage and prevent abuse:

### Request Limits
- **Per Client**: 1000 requests per minute
- **Burst Limit**: 100 concurrent requests
- **Global Limit**: 10,000 requests per minute across all clients

### Rate Limit Headers
The plugin includes rate limiting information in gRPC metadata:

```
x-ratelimit-limit: 1000
x-ratelimit-remaining: 950
x-ratelimit-reset: 1640995200
x-ratelimit-retry-after: 60
```

### Handling Rate Limits
When rate limited, the plugin returns:

```json
{
  "code": 8,
  "message": "rate limit exceeded",
  "details": {
    "retry_after_seconds": 60,
    "limit": 1000,
    "window_seconds": 60
  }
}
```

### Best Practices
- Implement exponential backoff for retries
- Cache results when possible to reduce API calls
- Monitor rate limit headers to adjust request patterns
- Consider upgrading service tiers for higher limits

## Extending the Plugin

### Adding New Resource Types

1. Implement cost estimation logic in `internal/plugin/projected.go`
2. Add test cases in `internal/plugin/projected_test.go`
3. Update resource type documentation
4. Regenerate pricing data if needed

### Adding New Regions

1. Update `internal/pricing/regions.yaml`
2. Run `make generate-embeds`
3. Run `make generate-goreleaser`
4. Build and test the new region

### Custom Pricing Sources

The plugin supports extensible pricing sources through the `PricingClient` interface. Implement custom pricing clients for alternative data sources.
