# CloudWatch Cost Estimation Contract

**Feature**: 019-cloudwatch-cost
**Date**: 2025-12-30

## gRPC Service Contract

CloudWatch cost estimation uses the existing `CostSourceService` from
`pulumicost.v1` proto. No new RPC methods are added.

### Supports Request

```protobuf
message SupportsRequest {
  ResourceDescriptor resource = 1;
}

message ResourceDescriptor {
  string provider = 1;       // "aws"
  string resource_type = 2;  // "cloudwatch", "logs", "metrics"
  string sku = 3;            // "logs", "metrics", "combined", or empty
  string region = 4;         // "us-east-1", etc.
  map<string, string> tags = 5;
}
```

**Supported Resource Types**:

| resource_type | Normalized To |
|---------------|---------------|
| `cloudwatch` | `cloudwatch` |
| `logs` | `cloudwatch` |
| `metrics` | `cloudwatch` |
| `aws:cloudwatch/logGroup:LogGroup` | `cloudwatch` |
| `aws:cloudwatch/metricAlarm:MetricAlarm` | `cloudwatch` |

### Supports Response

```protobuf
message SupportsResponse {
  bool supported = 1;
  string reason = 2;
  repeated MetricKind supported_metrics = 3;
}
```

**Response Examples**:

| Condition | supported | reason |
|-----------|-----------|--------|
| Valid CloudWatch resource | `true` | `""` |
| Region mismatch | `false` | `"Region not supported by this binary"` |
| Unknown resource type | `false` | `"Unknown resource type: xyz"` |

### GetProjectedCost Request

```protobuf
message GetProjectedCostRequest {
  ResourceDescriptor resource = 1;
  double utilization_percentage = 2;  // Unused for CloudWatch
}
```

**Required Tags**:

| Tag Name | Type | Description | Default |
|----------|------|-------------|---------|
| `log_ingestion_gb` | string (numeric) | Monthly log ingestion in GB | "0" |
| `log_storage_gb` | string (numeric) | Log storage in GB | "0" |
| `custom_metrics` | string (numeric) | Number of custom metrics | "0" |

**SKU Behavior**:

| sku value | Calculation |
|-----------|-------------|
| `"logs"` | Ingestion + storage only |
| `"metrics"` | Custom metrics only |
| `"combined"` or `""` | All components |

### GetProjectedCost Response

```protobuf
message GetProjectedCostResponse {
  double unit_price = 1;      // Varies by component
  string currency = 2;        // "USD"
  double cost_per_month = 3;  // Total monthly cost
  string billing_detail = 4;  // Human-readable breakdown
  ImpactMetrics impact_metrics = 5;  // nil for CloudWatch
}
```

**Response Examples**:

**Logs Only (100 GB ingestion, 500 GB storage)**:

```json
{
  "unit_price": 0.50,
  "currency": "USD",
  "cost_per_month": 65.00,
  "billing_detail": "CloudWatch Logs: 100 GB ingestion @ $0.50/GB + 500 GB storage @ $0.03/GB-mo"
}
```

**Metrics Only (50 custom metrics)**:

```json
{
  "unit_price": 0.30,
  "currency": "USD",
  "cost_per_month": 15.00,
  "billing_detail": "CloudWatch Metrics: 50 custom metrics @ $0.30/metric"
}
```

**Combined**:

```json
{
  "unit_price": 0.00,
  "currency": "USD",
  "cost_per_month": 80.00,
  "billing_detail": "CloudWatch: Logs + Metrics combined = $80.00/month"
}
```

**Missing Pricing Data (Soft Failure)**:

```json
{
  "unit_price": 0.00,
  "currency": "USD",
  "cost_per_month": 0.00,
  "billing_detail": "CloudWatch pricing unavailable for region - returning $0.00"
}
```

## Error Handling

CloudWatch estimation follows the soft failure pattern:

| Condition | Behavior |
|-----------|----------|
| Missing pricing data for region | Return $0.00, log warning |
| Invalid tag value (non-numeric) | Treat as 0, log warning |
| Negative usage value | Treat as 0 |
| Region mismatch | Return via Supports() = false |

**No gRPC errors** are returned for CloudWatch-specific issues.
ERROR_CODE_INVALID_RESOURCE is only used if ResourceDescriptor is malformed
(missing provider, etc.).

## Tiered Pricing Implementation

### Log Ingestion Tiers (us-east-1 baseline)

| Tier | Range | Rate per GB |
|------|-------|-------------|
| 1 | 0 - 10 TB | $0.50 |
| 2 | 10 TB - 30 TB | $0.25 |
| 3 | 30 TB - 50 TB | $0.10 |
| 4 | 50 TB+ | $0.05 |

### Custom Metrics Tiers (us-east-1 baseline)

| Tier | Range | Rate per Metric |
|------|-------|-----------------|
| 1 | 0 - 10,000 | $0.30 |
| 2 | 10,001 - 250,000 | $0.10 |
| 3 | 250,001 - 1,000,000 | $0.05 |
| 4 | 1,000,001+ | $0.02 |

### Log Storage Rate

Flat rate: $0.03 per GB-month (no tiers)
