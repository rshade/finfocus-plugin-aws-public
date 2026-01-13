# gRPC Contract Changes: Carbon Emission Estimation

**Feature**: 015-carbon-estimation
**Date**: 2025-12-19
**Proto Version**: finfocus-spec v0.4.10

## Overview

This feature uses new proto fields added in finfocus-spec v0.4.10. No proto changes are required in this plugin - only implementation of the new fields.

## New Proto Fields Used

### MetricKind Enum (v0.4.10)

```protobuf
enum MetricKind {
    METRIC_KIND_UNSPECIFIED = 0;
    METRIC_KIND_CARBON_FOOTPRINT = 1;  // gCO2e - used by this feature
    METRIC_KIND_ENERGY_CONSUMPTION = 2; // kWh - reserved for future
    METRIC_KIND_WATER_USAGE = 3;        // L - reserved for future
}
```

### ImpactMetric Message (v0.4.10)

```protobuf
message ImpactMetric {
    MetricKind kind = 1;   // METRIC_KIND_CARBON_FOOTPRINT
    double value = 2;      // Carbon in gCO2e (e.g., 831.5)
    string unit = 3;       // "gCO2e"
}
```

### SupportsResponse Changes (v0.4.10)

```protobuf
message SupportsResponse {
    bool supported = 1;                           // existing
    string reason = 2;                            // existing
    map<string, bool> capabilities = 3;           // existing
    repeated MetricKind supported_metrics = 4;    // NEW - advertise carbon capability
}
```

**Implementation for EC2**:

```go
&pbc.SupportsResponse{
    Supported: true,
    SupportedMetrics: []pbc.MetricKind{
        pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
    },
}
```

### GetProjectedCostRequest Changes (v0.4.10)

```protobuf
message GetProjectedCostRequest {
    ResourceDescriptor resource = 1;              // existing
    double utilization_percentage = 2;            // NEW - global default (0.0-1.0)
}
```

### ResourceDescriptor Changes (v0.4.10)

```protobuf
message ResourceDescriptor {
    string provider = 1;                          // existing
    string resource_type = 2;                     // existing
    string sku = 3;                               // existing
    string region = 4;                            // existing
    map<string, string> tags = 5;                 // existing
    optional double utilization_percentage = 6;   // NEW - per-resource override
}
```

### GetProjectedCostResponse Changes (v0.4.10)

```protobuf
message GetProjectedCostResponse {
    double unit_price = 1;                        // existing
    string currency = 2;                          // existing
    double cost_per_month = 3;                    // existing
    string billing_detail = 4;                    // existing
    repeated ImpactMetric impact_metrics = 5;     // NEW - carbon metrics
}
```

**Implementation for EC2**:

```go
&pbc.GetProjectedCostResponse{
    // existing financial fields...
    ImpactMetrics: []*pbc.ImpactMetric{
        {
            Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
            Value: carbonGrams,  // e.g., 831.5
            Unit:  "gCO2e",
        },
    },
}
```

## Backward Compatibility

All new fields are additive and optional:

- Older clients ignoring `impact_metrics` will continue to work
- Older clients ignoring `supported_metrics` will continue to work
- `utilization_percentage` defaults to 0 (plugin uses internal default of 0.5)

## Service Methods Affected

### CostSourceService.Supports

**Before (v0.4.9)**:

```go
&pbc.SupportsResponse{
    Supported: true,
    Reason:    "",
}
```

**After (v0.4.10)**:

```go
&pbc.SupportsResponse{
    Supported: true,
    Reason:    "",
    SupportedMetrics: []pbc.MetricKind{
        pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
    },
}
```

### CostSourceService.GetProjectedCost

**Before (v0.4.9)**:

```go
&pbc.GetProjectedCostResponse{
    UnitPrice:     0.0104,
    Currency:      "USD",
    CostPerMonth:  7.59,
    BillingDetail: "On-demand Linux, shared tenancy, 730 hrs/month",
}
```

**After (v0.4.10)**:

```go
&pbc.GetProjectedCostResponse{
    UnitPrice:     0.0104,
    Currency:      "USD",
    CostPerMonth:  7.59,
    BillingDetail: "On-demand Linux, shared tenancy, 730 hrs/month",
    ImpactMetrics: []*pbc.ImpactMetric{
        {
            Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
            Value: 831.5,
            Unit:  "gCO2e",
        },
    },
}
```

## Error Handling

Carbon estimation errors do NOT fail the financial cost calculation:

- Unknown instance type → return financial cost only, no `ImpactMetrics`
- Unknown region grid factor → use default factor, still return `ImpactMetrics`

The `ImpactMetrics` array is empty (not present) when carbon cannot be calculated.
