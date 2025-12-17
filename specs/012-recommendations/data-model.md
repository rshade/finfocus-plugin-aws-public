# Data Model: GetRecommendations RPC

**Date**: 2025-12-15
**Feature**: 012-recommendations

## Instance Family Mappings

### Generation Upgrade Map

Maps older EC2 instance families to their newer generation equivalents.

```go
// generationUpgradeMap maps old instance families to newer generations.
// Only includes mappings where the newer generation is typically the same price
// or cheaper with better performance.
var generationUpgradeMap = map[string]string{
    // T-series (burstable general purpose)
    "t2":  "t3",   // 2014 → 2018
    "t3":  "t3a",  // Intel → AMD (often cheaper)

    // M-series (general purpose)
    "m4":  "m5",   // 2015 → 2017
    "m5":  "m6i",  // 2017 → 2021
    "m5a": "m6a",  // AMD 2018 → AMD 2022

    // C-series (compute optimized)
    "c4":  "c5",   // 2015 → 2017
    "c5":  "c6i",  // 2017 → 2021
    "c5a": "c6a",  // AMD 2020 → AMD 2022

    // R-series (memory optimized)
    "r4":  "r5",   // 2016 → 2018
    "r5":  "r6i",  // 2018 → 2021
    "r5a": "r6a",  // AMD 2019 → AMD 2022

    // I-series (storage optimized)
    "i3":  "i3en", // 2017 → 2019

    // D-series (dense storage)
    "d2":  "d3",   // 2015 → 2020
}
```

**Validation Rule**: Before recommending, verify:
1. `newType` exists in pricing data for the region
2. `newPrice <= currentPrice`

### Graviton Migration Map

Maps x86 instance families to ARM/Graviton equivalents.

```go
// gravitonMap maps x86 instance families to Graviton (ARM) equivalents.
// Graviton instances typically offer ~20% cost savings with comparable performance.
var gravitonMap = map[string]string{
    // M-series → M6g
    "m5":   "m6g",
    "m5a":  "m6g",
    "m5n":  "m6g",
    "m6i":  "m6g",
    "m6a":  "m6g",

    // C-series → C6g/C6gn
    "c5":   "c6g",
    "c5a":  "c6g",
    "c5n":  "c6gn",
    "c6i":  "c6g",
    "c6a":  "c6g",

    // R-series → R6g
    "r5":   "r6g",
    "r5a":  "r6g",
    "r5n":  "r6g",
    "r6i":  "r6g",
    "r6a":  "r6g",

    // T-series → T4g
    "t3":   "t4g",
    "t3a":  "t4g",
}
```

**Confidence Score**: 0.7 (medium) - Requires ARM compatibility validation.

## Recommendation Types

### EC2 Generation Upgrade

| Field | Value |
|-------|-------|
| Category | `RECOMMENDATION_CATEGORY_COST_OPTIMIZATION` |
| ActionType | `RECOMMENDATION_ACTION_TYPE_MODIFY` |
| ModificationType | `"generation_upgrade"` |
| Priority | `RECOMMENDATION_PRIORITY_MEDIUM` |
| ConfidenceScore | `0.9` |
| Source | `"aws-public"` |

**CurrentConfig**:
```json
{
  "instance_type": "t2.medium"
}
```

**RecommendedConfig**:
```json
{
  "instance_type": "t3.medium"
}
```

**Reasoning**:
- "Newer t3 instances offer better performance at same or lower cost"
- "Drop-in replacement with no architecture changes required"

### EC2 Graviton Migration

| Field | Value |
|-------|-------|
| Category | `RECOMMENDATION_CATEGORY_COST_OPTIMIZATION` |
| ActionType | `RECOMMENDATION_ACTION_TYPE_MODIFY` |
| ModificationType | `"graviton_migration"` |
| Priority | `RECOMMENDATION_PRIORITY_LOW` |
| ConfidenceScore | `0.7` |
| Source | `"aws-public"` |

**CurrentConfig**:
```json
{
  "instance_type": "m5.large",
  "architecture": "x86_64"
}
```

**RecommendedConfig**:
```json
{
  "instance_type": "m6g.large",
  "architecture": "arm64"
}
```

**Reasoning**:
- "Graviton instances are ~20% cheaper with comparable performance"
- "Requires validation that application supports ARM architecture"

**Metadata**:
```json
{
  "architecture_change": "x86_64 -> arm64",
  "requires_validation": "Application must support ARM architecture"
}
```

### EBS Volume Type Upgrade

| Field | Value |
|-------|-------|
| Category | `RECOMMENDATION_CATEGORY_COST_OPTIMIZATION` |
| ActionType | `RECOMMENDATION_ACTION_TYPE_MODIFY` |
| ModificationType | `"volume_type_upgrade"` |
| Priority | `RECOMMENDATION_PRIORITY_MEDIUM` |
| ConfidenceScore | `0.9` |
| Source | `"aws-public"` |

**CurrentConfig**:
```json
{
  "volume_type": "gp2",
  "size_gb": "100"
}
```

**RecommendedConfig**:
```json
{
  "volume_type": "gp3",
  "size_gb": "100"
}
```

**Reasoning**:
- "gp3 volumes are ~20% cheaper than gp2"
- "gp3 provides better baseline performance (3000 IOPS, 125 MB/s)"
- "API-compatible change with no data migration required"

**Metadata**:
```json
{
  "baseline_iops": "gp2: 100 IOPS/GB, gp3: 3000 IOPS (included)",
  "baseline_throughput": "gp2: 128-250 MB/s, gp3: 125 MB/s (included)"
}
```

## Impact Calculation

### RecommendationImpact Structure

```go
impact := &pbc.RecommendationImpact{
    EstimatedSavings:   monthlySavings,        // currentCost - projectedCost
    Currency:           "USD",
    ProjectionPeriod:   "monthly",
    CurrentCost:        currentMonthlyCost,    // price × 730 for EC2
    ProjectedCost:      recommendedMonthlyCost,
    SavingsPercentage:  savingsPercent,        // (1 - projected/current) × 100
}
```

### EC2 Cost Calculation

```go
hoursPerMonth := 730.0
currentMonthlyCost := currentHourlyPrice * hoursPerMonth
projectedMonthlyCost := recommendedHourlyPrice * hoursPerMonth
monthlySavings := currentMonthlyCost - projectedMonthlyCost
savingsPercent := (monthlySavings / currentMonthlyCost) * 100
```

### EBS Cost Calculation

```go
sizeGB := extractSizeFromTags(resource.Tags, 100) // default 100GB
currentMonthlyCost := gp2PricePerGB * float64(sizeGB)
projectedMonthlyCost := gp3PricePerGB * float64(sizeGB)
monthlySavings := currentMonthlyCost - projectedMonthlyCost
savingsPercent := (monthlySavings / currentMonthlyCost) * 100
```

## Resource Extraction

### From ResourceDescriptor (Supports/GetProjectedCost pattern)

```go
type ResourceDescriptor struct {
    Provider     string            // "aws"
    ResourceType string            // "ec2", "ebs", "aws:ec2/instance:Instance"
    Sku          string            // instance type or volume type
    Region       string            // "us-east-1"
    Tags         map[string]string // additional attributes
}
```

### From RecommendationFilter (GetRecommendations pattern)

The `GetRecommendationsRequest.Filter` may contain:
- `ResourceIds []string` - specific resources to analyze
- Metadata in custom fields

For this plugin, if no filter is provided, return empty recommendations
(plugin doesn't maintain resource inventory).

## Validation Rules

### Pre-recommendation Checks

1. **Instance type exists**: `pricing.EC2OnDemandPricePerHour(type, "Linux", "Shared")` returns `found=true`
2. **Price comparison**: `recommendedPrice <= currentPrice`
3. **Region match**: Resource region matches plugin's embedded region

### Post-generation Validation

Use SDK validators:
```go
if err := pluginsdk.ValidateRecommendation(rec); err != nil {
    // Log and skip this recommendation
}
if err := pluginsdk.ValidateRecommendationImpact(rec.Impact); err != nil {
    // Log and skip this recommendation
}
```
