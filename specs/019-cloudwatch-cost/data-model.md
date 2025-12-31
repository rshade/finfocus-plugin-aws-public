# Data Model: CloudWatch Cost Estimation

**Feature**: 019-cloudwatch-cost
**Date**: 2025-12-30

## Entities

### 1. CloudWatchPrice

Represents the pricing rates for CloudWatch in a specific region.

```go
// cloudWatchPrice holds all pricing rates for CloudWatch in a region.
// Parsed from AWS Price List API (AmazonCloudWatch service).
type cloudWatchPrice struct {
    // LogsIngestionTiers is tiered pricing for log data ingestion (per GB)
    // Tiers: 0-10TB @ $0.50, 10-30TB @ $0.25, 30-50TB @ $0.10, 50TB+ @ $0.05
    LogsIngestionTiers []tierRate

    // LogsStorageRate is flat rate for archived log storage (per GB-month)
    // Typically $0.03/GB-month in us-east-1
    LogsStorageRate float64

    // MetricsTiers is tiered pricing for custom metrics (per metric/month)
    // Tiers: 0-10K @ $0.30, 10K-250K @ $0.10, 250K-1M @ $0.05, 1M+ @ $0.02
    MetricsTiers []tierRate

    // Currency is always "USD" for this plugin
    Currency string
}

// tierRate represents a single tier in tiered pricing.
// Used for both log ingestion and metrics.
type tierRate struct {
    // From is the lower bound (inclusive) of this tier
    // For first tier, From = 0
    From float64

    // UpTo is the upper bound (exclusive) of this tier
    // For last tier, UpTo = math.MaxFloat64
    UpTo float64

    // Rate is the price per unit in this tier
    Rate float64
}
```

**Fields**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| LogsIngestionTiers | []tierRate | Yes | Tiered rates for log ingestion (GB) |
| LogsStorageRate | float64 | Yes | Flat rate for log storage (GB-month) |
| MetricsTiers | []tierRate | Yes | Tiered rates for custom metrics (count) |
| Currency | string | Yes | Always "USD" |

**Validation Rules**:

- All rates must be >= 0
- Tiers must be sorted by From value (ascending)
- Tier ranges must not overlap
- Last tier's UpTo should be math.MaxFloat64

**State Transitions**: N/A (immutable after parsing)

### 2. CloudWatchUsage (Input from Tags)

Represents the usage values extracted from ResourceDescriptor tags.

```go
// cloudWatchUsage represents usage metrics extracted from resource tags.
// All values default to 0.0 if tags are missing or invalid.
type cloudWatchUsage struct {
    // LogIngestionGB is monthly log data ingestion volume in gigabytes
    LogIngestionGB float64

    // LogStorageGB is current archived log storage in gigabytes
    LogStorageGB float64

    // CustomMetrics is the count of custom metrics being published
    CustomMetrics int64
}
```

**Fields**:

| Field | Type | Required | Default | Tag Name |
|-------|------|----------|---------|----------|
| LogIngestionGB | float64 | No | 0.0 | `log_ingestion_gb` |
| LogStorageGB | float64 | No | 0.0 | `log_storage_gb` |
| CustomMetrics | int64 | No | 0 | `custom_metrics` |

**Validation Rules**:

- All values must be >= 0 (negative values treated as 0)
- Non-numeric tag values default to 0 with warning log

### 3. CloudWatchCostBreakdown (Output)

Represents the calculated cost components (internal, not exposed via API).

```go
// cloudWatchCostBreakdown holds the calculated cost components.
// Used internally to build the GetProjectedCostResponse.
type cloudWatchCostBreakdown struct {
    // IngestionCost is the monthly cost for log ingestion
    IngestionCost float64

    // StorageCost is the monthly cost for log storage
    StorageCost float64

    // MetricsCost is the monthly cost for custom metrics
    MetricsCost float64

    // TotalCost is the sum of all components
    TotalCost float64

    // BillingDetail is the human-readable breakdown
    BillingDetail string
}
```

## Relationships

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                         GetProjectedCost Request                         │
│  ResourceDescriptor { sku: "logs"|"metrics"|"combined", tags: {...} }   │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           estimateCloudWatch()                           │
│                                                                          │
│  1. Extract usage from tags → cloudWatchUsage                           │
│  2. Lookup pricing → cloudWatchPrice                                    │
│  3. Calculate costs → cloudWatchCostBreakdown                           │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        GetProjectedCostResponse                          │
│  { unit_price, cost_per_month, currency, billing_detail }               │
└─────────────────────────────────────────────────────────────────────────┘
```

## Pricing Data Index

The pricing client maintains these indexes for CloudWatch lookups:

```go
type Client struct {
    // ... existing fields ...

    // cloudWatchPrices maps region to CloudWatch pricing
    // Key: AWS region code (e.g., "us-east-1")
    // Value: parsed cloudWatchPrice struct
    cloudWatchPrices map[string]*cloudWatchPrice
}
```

**Lookup Methods**:

```go
// CloudWatchLogsIngestionTiers returns tiered rates for log ingestion.
// Returns nil, false if region not found.
func (c *Client) CloudWatchLogsIngestionTiers(region string) ([]tierRate, bool)

// CloudWatchLogsStorageRate returns the flat rate for log storage.
// Returns 0, false if region not found.
func (c *Client) CloudWatchLogsStorageRate(region string) (float64, bool)

// CloudWatchMetricsTiers returns tiered rates for custom metrics.
// Returns nil, false if region not found.
func (c *Client) CloudWatchMetricsTiers(region string) ([]tierRate, bool)
```

## Tiered Pricing Calculation

```go
// calculateTieredCost computes total cost using AWS tiered pricing.
//
// Parameters:
//   - usage: the total usage amount (GB for logs, count for metrics)
//   - tiers: sorted slice of tier rates
//
// Returns: total cost for the given usage
//
// Example for 100 custom metrics with tiers [$0.30 first 10K, $0.10 next]:
//   calculateTieredCost(100, metricsTiers) = 100 * 0.30 = $30.00
//
// Example for 15,000 metrics:
//   calculateTieredCost(15000, metricsTiers)
//     = 10000 * 0.30 + 5000 * 0.10 = $3,000 + $500 = $3,500
func calculateTieredCost(usage float64, tiers []tierRate) float64 {
    remaining := usage
    total := 0.0

    for _, tier := range tiers {
        if remaining <= 0 {
            break
        }
        tierCapacity := tier.UpTo - tier.From
        tierUsage := math.Min(remaining, tierCapacity)
        total += tierUsage * tier.Rate
        remaining -= tierUsage
    }

    return total
}
```

## AWS Price List API Mapping

### Product Families and Attributes

| Component | productFamily | Identifying Attributes |
|-----------|---------------|----------------------|
| Log Ingestion | `Data Payload` | group: `Ingested Logs` |
| Log Storage | `Storage Snapshot` | group: `Archived Logs` |
| Custom Metrics | `Metric` | group: `Custom Metric` |

### Pricing JSON Structure

```json
{
  "offerCode": "AmazonCloudWatch",
  "products": {
    "SKU123": {
      "sku": "SKU123",
      "productFamily": "Data Payload",
      "attributes": {
        "group": "Ingested Logs",
        "usagetype": "USE1-DataProcessing-Bytes",
        "region": "us-east-1"
      }
    }
  },
  "terms": {
    "OnDemand": {
      "SKU123": {
        "SKU123.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "priceDimensions": {
            "SKU123.JRTCKXETXF.6YS6EN2CT7": {
              "pricePerUnit": { "USD": "0.5000000000" },
              "unit": "GB"
            }
          }
        }
      }
    }
  }
}
```

## Billing Detail Format

The `billing_detail` field follows established patterns:

**Logs Only**:

```text
CloudWatch Logs: 100 GB ingestion @ $0.50/GB ($50.00) + 500 GB storage @ $0.03/GB-mo ($15.00) = $65.00/month
```

**Metrics Only**:

```text
CloudWatch Metrics: 50 custom metrics @ $0.30/metric = $15.00/month
```

**Combined**:

```text
CloudWatch: Logs (100 GB ingestion, 500 GB storage) + Metrics (50 custom) = $80.00/month
```

**Missing Data (Soft Failure)**:

```text
CloudWatch cost estimation unavailable for region ap-northeast-3 - returning $0.00
```
