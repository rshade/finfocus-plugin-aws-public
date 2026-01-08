# Protocol Extension: UsageProfile Enum

**Feature**: 030-core-protocol-intelligence (P1 - Dev Mode)
**Status**: PROPOSED - Requires PR to rshade/pulumicost-spec
**Date**: 2026-01-05

## Purpose

Add `UsageProfile` enum to `GetProjectedCostRequest` to enable Dev Mode realistic cost estimates. This allows users to specify operational context (production vs development) so the plugin can apply appropriate hour multipliers.

## Proto Definition

```protobuf
// UsageProfile indicates operational context for cost estimation
enum UsageProfile {
  // Default: production assumptions (730 hours/month = 24/7)
  USAGE_PROFILE_UNSPECIFIED = 0;
  // 24/7 operation (730 hours/month)
  USAGE_PROFILE_PRODUCTION = 1;
  // Business hours only (160 hours/month = 8hrs * 5days * 4weeks)
  USAGE_PROFILE_DEVELOPMENT = 2;
  // Reserved for future use (currently same as PRODUCTION)
  USAGE_PROFILE_BURST = 3;
}
```

## Integration

Add to `GetProjectedCostRequest` message:

```protobuf
message GetProjectedCostRequest {
  // ... existing fields ...

  // OPTIONAL: Operational context for cost estimation
  // Default: USAGE_PROFILE_UNSPECIFIED (treats as PRODUCTION)
  UsageProfile usage_profile = 10;
}
```

## Backward Compatibility

- Field is **optional** with default value `USAGE_PROFILE_UNSPECIFIED`
- Existing plugins without this field will ignore it (treated as UNSPECIFIED)
- No breaking changes to existing `GetProjectedCostRequest` structure

## Implementation Notes

### Plugin Behavior

| UsageProfile | Hours/Month | Multiplier | Billing Detail |
|--------------|-------------|------------|----------------|
| UNSPECIFIED | 730 | 1.0x | (no annotation) |
| PRODUCTION | 730 | 1.0x | (no annotation) |
| DEVELOPMENT | 160 | 0.219x | "(dev profile)" appended |
| BURST | 730 | 1.0x | (no annotation) |

### Affected Services

Only time-based services (not usage-based):

- ✅ EC2 instances (hours)
- ✅ EKS clusters (hours)
- ✅ ELB (load balancer hours)
- ✅ NAT Gateway (gateway hours)
- ✅ ElastiCache (node hours)
- ✅ RDS (instance hours)
- ❌ S3 (storage - not time-based)
- ❌ EBS (storage - not time-based)
- ❌ Lambda (usage-based - requests/compute)
- ❌ DynamoDB (usage-based - throughput/storage)
- ❌ CloudWatch (usage-based - ingestion)

### Cost Calculation

```go
if usage_profile == DEVELOPMENT && service.AffectedByDevMode {
    cost_per_month = cost_per_month * 160 / 730  // ~21.9% of production
    billing_detail += " (dev profile)"
}
```

## Example Usage

### Request

```protobuf
{
  "resource_type": "aws:ec2:instance:Instance",
  "region": "us-east-1",
  "instance_type": "t3.medium",
  "usage_profile": "DEVELOPMENT"
}
```

### Response

```protobuf
{
  "unit_price": 0.0416,
  "currency": "USD",
  "cost_per_month": 6.66,  // ~22% of $30.37 production
  "billing_detail": "$0.0416/hr × 160 hrs/month (dev profile)",
  "usage_profile": "DEVELOPMENT"  // Echoed for confirmation
}
```

## Testing

### Unit Tests

```go
func TestUsageProfileDevMode(t *testing.T) {
    req := &GetProjectedCostRequest{
        ResourceType: "aws:ec2:instance:Instance",
        UsageProfile: UsageProfile_DEVELOPMENT,
    }

    resp := plugin.EstimateCost(req)

    assert.Equal(t, 6.66, resp.CostPerMonth)  // 30.37 * 160/730
    assert.Contains(t, resp.BillingDetail, "(dev profile)")
}

func TestUsageProfileProduction(t *testing.T) {
    req := &GetProjectedCostRequest{
        ResourceType: "aws:ec2:instance:Instance",
        UsageProfile: UsageProfile_PRODUCTION,
    }

    resp := plugin.EstimateCost(req)

    assert.Equal(t, 30.37, resp.CostPerMonth)  // Full production cost
    assert.NotContains(t, resp.BillingDetail, "(dev profile)")
}

func TestUsageProfileUnspecified(t *testing.T) {
    req := &GetProjectedCostRequest{
        ResourceType: "aws:ec2:instance:Instance",
        UsageProfile: UsageProfile_UNSPECIFIED,
    }

    resp := plugin.EstimateCost(req)

    assert.Equal(t, 30.37, resp.CostPerMonth)  // Treated as production
}
```

## References

- Feature spec: spec.md (User Story 1 - Dev Mode Cost Estimates)
- Data model: data-model.md (UsageProfile enum)
- Implementation guide: quickstart.md (Phase 4: Dev Mode)
