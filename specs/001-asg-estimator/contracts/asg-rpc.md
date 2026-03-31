# API Contract: ASG Cost Estimation RPCs

**Feature**: 001-asg-estimator | **Date**: 2026-03-26

## Supports RPC

### Request

```text
ResourceDescriptor:
  provider: "aws"
  resource_type: "aws:autoscaling/group:Group" | "asg" | "autoscaling"
  region: "<must match plugin binary region>"
```

### Response

```text
SupportsResponse:
  supported: true
  reason: ""
  supported_metrics: [METRIC_KIND_CARBON_FOOTPRINT]
```

### Error Cases

| Condition | Response |
| --------- | -------- |
| Region mismatch | supported: false, reason: "Region not supported by this binary" |

## GetProjectedCost RPC

### Request

```text
ResourceDescriptor:
  provider: "aws"
  resource_type: "aws:autoscaling/group:Group"
  sku: "<instance_type>"            # Optional, highest priority
  region: "us-east-1"
  tags:
    instance_type: "m5.large"       # Fallback 1
    desired_capacity: "3"           # Instance count (default: 1)
    min_size: "1"                   # Fallback for capacity
    operating_system: "Linux"       # OS for pricing (default: Linux)
    launch_template.instance_type: "m5.large"  # Fallback 2
    launch_configuration.instance_type: "m5.large"  # Fallback 3
```

### Response (Success)

```text
GetProjectedCostResponse:
  unit_price: 0.096                 # Per-instance hourly rate
  currency: "USD"
  cost_per_month: 210.24            # 0.096 × 3 × 730
  billing_detail: "ASG: 3× m5.large On-demand Linux, 730 hrs/month"
  metadata:
    estimate_quality: "high"        # or "medium" if defaults applied
  impact_metrics:
    - kind: METRIC_KIND_CARBON_FOOTPRINT
      value: 45678.9                # gCO2e (per-instance carbon × 3)
      unit: "gCO2e"
```

### Response (Zero Capacity)

```text
GetProjectedCostResponse:
  unit_price: 0.096
  currency: "USD"
  cost_per_month: 0.0
  billing_detail: "ASG: 0× m5.large On-demand Linux, 730 hrs/month (zero instances)"
  metadata:
    estimate_quality: "high"
```

### Error Cases

| Condition | Error |
| --------- | ----- |
| No instance type resolvable | PricingUnavailableError: "cannot determine instance type for ASG" |
| Instance type not in pricing data | PricingUnavailableError: "no pricing for instance type X in region Y" |
| Region mismatch | gRPC error with ERROR_CODE_UNSUPPORTED_REGION |

## GetPricingSpec RPC

### Response

```text
GetPricingSpecResponse:
  billing_mode: "on_demand"
  rate_per_unit: 0.096              # Per-instance hourly rate
  rate_unit: "per-instance-hour"
  currency: "USD"
  notes: "ASG cost = instance hourly rate × desired_capacity × 730 hrs/month"
```

## GetActualCost RPC

Delegates to GetProjectedCost via `getProjectedForResource`. Same pricing
logic applies. Actual cost calculation uses runtime hours when available:
`projected_monthly_cost × runtime_hours / 730`.
