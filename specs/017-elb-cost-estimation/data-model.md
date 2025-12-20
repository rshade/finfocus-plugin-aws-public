# Data Model: ELB Cost Estimation

## Entities

### elbPrice (Internal)
Cached pricing data for a single region.

| Field | Type | Description |
|-------|------|-------------|
| `ALBHourlyRate` | `float64` | Fixed hourly cost ($/hr) |
| `ALBLCURate` | `float64` | Cost per LCU-hour ($/LCU-hr) |
| `NLBHourlyRate` | `float64` | Fixed hourly cost ($/hr) |
| `NLBNLCURate` | `float64` | Cost per NLCU-hour ($/NLCU-hr) |
| `Currency` | `string` | Usually "USD" |

## Resource Mapping

Mapping from `pbc.ResourceDescriptor` to estimation logic:

| Descriptor Field | Use Case |
|-------------------|----------|
| `Sku` | Identifies load balancer type ("alb", "nlb") |
| `Tags["lcu_per_hour"]` | Specific ALB capacity |
| `Tags["nlcu_per_hour"]` | Specific NLB capacity |
| `Tags["capacity_units"]` | Generic fallback for either type |

## Validation Rules

1. **SKU Normalization**: Convert `Sku` to lowercase. Map "application" to "alb" and "network" to "nlb" if necessary.
2. **Numeric Tags**: Use `0.0` if tag value is non-numeric or missing.
3. **Region Match**: `pbc.ResourceDescriptor.Region` must match plugin's `p.region`.