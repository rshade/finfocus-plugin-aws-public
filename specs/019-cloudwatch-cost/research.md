# Research: CloudWatch Cost Estimation

**Feature**: 019-cloudwatch-cost
**Date**: 2025-12-30
**Status**: Complete

## Research Questions Resolved

### Q1: What is the AWS Price List API service code for CloudWatch?

**Decision**: `AmazonCloudWatch`

**Rationale**: Verified via AWS Price List API index at
`https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/index.json`.
The CloudWatch service uses offerCode `AmazonCloudWatch` with regionIndexUrl
`/offers/v1.0/aws/AmazonCloudWatch/current/region_index.json`.

**Alternatives Considered**:

- `AWSCloudWatch` - Not a valid service code
- `AmazonLogs` / `AmazonMetrics` - These are product families within CloudWatch, not separate services

### Q2: What are the CloudWatch pricing components?

**Decision**: Implement three pricing components for v1:

1. **Log Ingestion** - per GB ingested (tiered: $0.50 → $0.25 → $0.10 → $0.05)
2. **Log Storage** - per GB-month archived ($0.03/GB-month)
3. **Custom Metrics** - per metric/month (tiered: $0.30 → $0.10 → $0.05 → $0.02)

**Rationale**: These are the three most common cost drivers for CloudWatch usage,
covering the spec's user stories (logs + metrics). Other features (alarms, dashboards,
Contributor Insights, Application Insights) are explicitly out of scope per spec
assumptions.

**Alternatives Considered**:

- Include alarms/dashboards - Rejected: out of scope per spec
- Include Logs Insights queries - Rejected: harder to estimate (query volume varies widely)
- Include vended logs pricing - Rejected: AWS service logs are often free, hard to distinguish

### Q3: What is the exact tiered pricing for Custom Metrics?

**Decision**: Use AWS's official tiered structure (us-east-1 baseline):

| Tier | Metric Count Range | Price per Metric/Month |
|------|-------------------|----------------------|
| 1 | First 10,000 | $0.30 |
| 2 | 10,001 - 250,000 | $0.10 |
| 3 | 250,001 - 1,000,000 | $0.05 |
| 4 | Over 1,000,000 | $0.02 |

**Rationale**: AWS CloudWatch pricing page and pricing API confirm these tiers.
The spec (FR-007) explicitly requires "standard AWS tiered pricing logic".

**Alternatives Considered**:

- First-tier only ($0.30 flat) - Rejected: Would overestimate by 10x for large-scale users
- Average rate approach - Rejected: Less accurate than explicit tiers

**Sources**:

- [AWS CloudWatch Pricing](https://aws.amazon.com/cloudwatch/pricing/)
- AWS Price List API (`AmazonCloudWatch` service)

### Q4: What is the exact tiered pricing for Log Ingestion?

**Decision**: Use AWS's official tiered structure (us-east-1 baseline):

| Tier | Data Volume Range | Price per GB |
|------|-------------------|--------------|
| 1 | First 10 TB | $0.50 |
| 2 | 10 TB - 30 TB | $0.25 |
| 3 | 30 TB - 50 TB | $0.10 |
| 4 | Over 50 TB | $0.05 |

**Rationale**: AWS CloudWatch Logs pricing uses volume-based tiers similar to metrics.
Most users fall in Tier 1 ($0.50/GB), but high-volume users get significant discounts.

**Alternatives Considered**:

- First-tier only - Rejected for same reasons as metrics
- Infrequent Access logs class - Rejected: adds complexity; can be added later

### Q5: How should log storage be priced?

**Decision**: Use flat rate of $0.03/GB-month for archived logs storage.

**Rationale**: AWS CloudWatch Logs storage pricing is simpler than ingestion:

- First 5 GB: Free tier
- Beyond 5 GB: $0.03/GB-month (us-east-1)

For v1, we ignore the free tier (conservative estimate) and use the flat rate.

**Alternatives Considered**:

- Include free tier deduction - Rejected: adds complexity, free tier is small
- Tiered storage pricing - Rejected: AWS doesn't tier storage the same way as ingestion

### Q6: How should pricing data be parsed from AWS Price List API?

**Decision**: Parse `AmazonCloudWatch` JSON using existing pattern:

1. Filter by `productFamily`:
   - `Data Payload` → Log ingestion/storage
   - `Metric` → Custom metrics
2. Extract pricing via `Terms.OnDemand` structure
3. Build indexed maps for O(1) lookups:
   - `logsIngestionPrice map[string][]tierPrice` (region → tiered rates)
   - `logsStoragePrice map[string]float64` (region → rate)
   - `metricsPrice map[string][]tierPrice` (region → tiered rates)

**Rationale**: Follows established pattern from EC2, EBS, DynamoDB parsers.
Uses same `awsPricing` struct for JSON unmarshalling.

**Alternatives Considered**:

- Hardcoded pricing - Rejected: violates "No Dummy Data" constitution principle
- Runtime API fetch - Rejected: plugin has no runtime network calls per constitution

### Q7: What product attributes identify CloudWatch price components?

**Decision**: Use these AWS attribute combinations:

| Component | productFamily | group | usagetype pattern |
|-----------|---------------|-------|-------------------|
| Log Ingestion | Data Payload | Ingested Logs | `*-DataProcessing-Bytes` |
| Log Storage | Storage Snapshot | Archived Logs | `*-TimedStorage-ByteHrs` |
| Custom Metrics | Metric | Custom Metric | `*-MetricMonitorUsage` |

**Rationale**: AWS Price List API uses `productFamily` as primary classifier,
with `group` and `usagetype` for fine-grained identification.

### Q8: What resource type formats should be supported?

**Decision**: Support these resource type formats:

| Format | Example | Notes |
|--------|---------|-------|
| Pulumi full | `aws:cloudwatch/logGroup:LogGroup` | For log resources |
| Pulumi full | `aws:cloudwatch/metricAlarm:MetricAlarm` | For metric resources |
| Simple | `cloudwatch` | Generic CloudWatch estimation |
| Simple | `logs` | Logs-only estimation |
| Simple | `metrics` | Metrics-only estimation |

**Rationale**: Follows pattern established by other services (EC2 supports
`aws:ec2/instance:Instance`, `ec2`, `ec2/instance`). The `detectService()`
function normalizes all formats to a canonical form.

### Q9: How should the SKU field be used?

**Decision**: Use `sku` to determine pricing mode:

- `sku: "logs"` → Calculate only log costs
- `sku: "metrics"` → Calculate only metric costs
- `sku: "combined"` or empty → Calculate both (sum)

**Rationale**: Matches spec's user stories where logs and metrics can be
estimated separately or together. Similar to DynamoDB's `on-demand` vs
`provisioned` SKU pattern.

### Q10: What tags should be recognized?

**Decision**: Recognize these tags for usage input:

| Tag Name | Type | Description | Default |
|----------|------|-------------|---------|
| `log_ingestion_gb` | float64 | GB of logs ingested per month | 0.0 |
| `log_storage_gb` | float64 | GB of logs stored | 0.0 |
| `custom_metrics` | int | Number of custom metrics | 0 |

**Rationale**: Tag names match the spec's acceptance scenarios exactly.
Defaults to 0 (not an error) per FR-009 and edge case specification.

## Technology Decisions

### Pricing Calculation Implementation

**Decision**: Implement tiered pricing calculation as a pure function:

```go
// calculateTieredCost computes cost using AWS-style tiered pricing
func calculateTieredCost(usage float64, tiers []tierRate) float64 {
    remaining := usage
    total := 0.0
    for _, tier := range tiers {
        if remaining <= 0 {
            break
        }
        tierUsage := min(remaining, tier.UpTo - tier.From)
        total += tierUsage * tier.Rate
        remaining -= tierUsage
    }
    return total
}
```

**Rationale**: Pure function is easily testable, follows KISS principle,
and can be reused if other services need tiered pricing.

### Error Handling for Missing Pricing Data

**Decision**: Return `$0.00` cost with standardized error message (Soft Failure).

**Rationale**: Codebase audit revealed ALL services use soft failure pattern,
returning $0.00 with explanatory BillingDetail. Some specs (e.g., NAT Gateway)
documented "return error" but implementations all use soft failure.

**Normalization Required**: As part of this feature, standardize error messages
across all services using a constant:

```go
// internal/plugin/constants.go (new file)
const (
    // PricingNotFoundTemplate is the standard message for missing pricing data.
    // Use with fmt.Sprintf: fmt.Sprintf(PricingNotFoundTemplate, "EC2 instance type", "t3.micro")
    PricingNotFoundTemplate = "%s %q not found in pricing data"

    // PricingUnavailableTemplate is for region-level pricing unavailability.
    // Use with fmt.Sprintf: fmt.Sprintf(PricingUnavailableTemplate, "CloudWatch", "ap-northeast-3")
    PricingUnavailableTemplate = "%s pricing data not available for region %s"
)
```

**Services to Update**:

| Service | Current Message | Normalized |
|---------|-----------------|------------|
| EC2 | "EC2 instance type %q not found in pricing data for %s/%s" | Use template |
| EBS | "EBS volume type %q not found in pricing data" | Use template |
| S3 | "S3 storage class %q not found in pricing data" | Use template |
| RDS | "RDS instance type %q not found in pricing data for %s" | Use template |
| EKS | "EKS pricing data not found" | Use unavailable template |
| Lambda | "Lambda pricing data not found" | Use unavailable template |
| ELB | (implicit $0) | Add explicit message |
| NAT Gateway | "NAT Gateway pricing data not available for this region" | Use template |
| DynamoDB | (silent $0) | Add explicit message |
| CloudWatch | NEW | Use template |

**Benefits**:

1. Consistent user experience across all services
2. Easier to grep/search for pricing failures in logs
3. Single source of truth for error messages
4. Facilitates future i18n if needed

### Thread Safety

**Decision**: Same pattern as other services:

- `sync.Once` for parsing
- Read-only maps after initialization
- No locking needed for lookups

**Rationale**: Constitution mandates thread-safe lookups. The sync.Once +
immutable-after-init pattern is proven across EC2/EBS/EKS implementations.

## Best Practices Applied

### From Existing Codebase

1. **Service detection**: Use `detectService()` normalization pattern
2. **Tag extraction**: Check multiple tag key variations with fallback
3. **Billing detail**: Include explicit usage values and rates in human-readable format
4. **Logging**: Include traceID in all log entries using pluginsdk.FieldTraceID
5. **Non-critical service**: CloudWatch should NOT fail plugin startup if parsing errors

### From AWS Pricing API

1. **offerCode validation**: Verify JSON has expected `offerCode: "AmazonCloudWatch"`
2. **OnDemand terms only**: Filter out Reserved/SavingsPlan terms (already done in generate-pricing)
3. **Region extraction**: Parse region from product attributes

## References

- [AWS CloudWatch Pricing](https://aws.amazon.com/cloudwatch/pricing/)
- [AWS Price List API](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/price-changes.html)
- AWS Price List API Index: `https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/index.json`
- CloudWatch Pricing API: `https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonCloudWatch/current/{region}/index.json`
