# Research: Carbon Emission Estimation

**Feature**: 015-carbon-estimation
**Date**: 2025-12-19

## Research Topics

### 1. CCF Data Source Selection

**Decision**: Use `cloud-carbon-coefficients` repository CSV data

**Rationale**:

- Canonical source maintained by Cloud Carbon Footprint project
- Contains 500+ AWS instance types with vCPU, CPU architecture, min/max watts
- Apache 2.0 license (compatible with this project)
- Static data - no API dependency at runtime
- CSV format is easily parsed and embedded via `//go:embed`

**Alternatives Considered**:

| Alternative | Rejected Because |
|-------------|------------------|
| AWS Sustainability API | Requires AWS credentials, runtime dependency |
| Hardcoded constants | Limited instance coverage, maintenance burden |
| SPECpower database directly | Requires complex processing, CCF already did this work |

**Source URL**: `https://github.com/cloud-carbon-footprint/cloud-carbon-coefficients/blob/main/data/aws-instances.csv`

### 2. Grid Emission Factor Strategy

**Decision**: Embed static grid emission factors as Go constants

**Rationale**:

- Grid factors change infrequently (annually at most)
- 12 regions × 1 float64 = negligible memory overhead
- No external API dependency (Electricity Maps requires subscription)
- CCF constants are well-researched and documented

**Grid Emission Factors (metric tons CO2eq/kWh)**:

```go
var GridEmissionFactors = map[string]float64{
    "us-east-1":      0.000379,    // Virginia (SERC)
    "us-east-2":      0.000411,    // Ohio (RFC)
    "us-west-1":      0.000322,    // N. California (WECC)
    "us-west-2":      0.000322,    // Oregon (WECC)
    "ca-central-1":   0.00012,     // Canada
    "eu-west-1":      0.0002786,   // Ireland
    "eu-north-1":     0.0000088,   // Sweden
    "ap-southeast-1": 0.000408,    // Singapore
    "ap-southeast-2": 0.00079,     // Sydney
    "ap-northeast-1": 0.000506,    // Tokyo
    "ap-south-1":     0.000708,    // Mumbai
    "sa-east-1":      0.0000617,   // São Paulo
    "_default":       0.00039278,  // Global average fallback
}
```

**Alternatives Considered**:

| Alternative | Rejected Because |
|-------------|------------------|
| Electricity Maps API | Requires API key, runtime dependency, cost |
| Real-time grid data | Out of scope for v1, adds complexity |
| Regional averages only | Less accurate, CCF provides region-specific data |

### 3. Formula Implementation

**Decision**: Implement CCF formula exactly as specified

**Formula**:

```go
const (
    AWS_PUE             = 1.135   // Power Usage Effectiveness for AWS datacenters
    DefaultUtilization  = 0.50    // 50% CPU utilization assumption
    HoursPerMonth       = 730.0   // Standard month
)

func CalculateCarbonGrams(
    minWatts, maxWatts float64,
    vCPUCount int,
    utilization float64,
    gridIntensity float64,  // metric tons CO2eq/kWh
    hours float64,
) float64 {
    // Step 1: Average watts based on utilization
    avgWatts := minWatts + (utilization * (maxWatts - minWatts))

    // Step 2: Energy consumption (kWh)
    energyKWh := (avgWatts * float64(vCPUCount) * hours) / 1000.0

    // Step 3: Apply PUE overhead
    energyWithPUE := energyKWh * AWS_PUE

    // Step 4: Carbon emissions (gCO2e)
    // gridIntensity is metric tons/kWh, multiply by 1,000,000 for grams
    carbonGrams := energyWithPUE * gridIntensity * 1_000_000

    return carbonGrams
}
```

**Rationale**:

- Matches CCF methodology exactly for consistency
- Linear interpolation between idle/max watts is industry standard
- 50% utilization default is CCF's hyperscale datacenter assumption
- PUE of 1.135 is AWS's published value

### 4. Data Embedding Strategy

**Decision**: Embed CSV at build time, parse once via sync.Once

**Rationale**:

- Consistent with existing pricing data pattern (`internal/pricing/client.go`)
- Thread-safe initialization
- No runtime file I/O
- Single source of truth for all region binaries

**Implementation Pattern**:

```go
//go:embed data/ccf_instance_specs.csv
var instanceSpecsCSV []byte

type InstanceSpec struct {
    InstanceType string
    VCPUCount    int
    MinWatts     float64
    MaxWatts     float64
}

var (
    instanceSpecs map[string]InstanceSpec
    specsOnce     sync.Once
)

func GetInstanceSpec(instanceType string) (InstanceSpec, bool) {
    specsOnce.Do(parseInstanceSpecs)
    spec, ok := instanceSpecs[instanceType]
    return spec, ok
}
```

### 5. Proto Integration (finfocus-spec v0.4.10)

**Decision**: Use proto-defined types directly

**Available in v0.4.10**:

```protobuf
enum MetricKind {
    METRIC_KIND_UNSPECIFIED = 0;
    METRIC_KIND_CARBON_FOOTPRINT = 1;  // gCO2e
    METRIC_KIND_ENERGY_CONSUMPTION = 2; // kWh
    METRIC_KIND_WATER_USAGE = 3;        // L
}

message ImpactMetric {
    MetricKind kind = 1;
    double value = 2;
    string unit = 3;
}

message SupportsResponse {
    bool supported = 1;
    string reason = 2;
    map<string, bool> capabilities = 3;
    repeated MetricKind supported_metrics = 4;  // NEW
}

message GetProjectedCostResponse {
    double unit_price = 1;
    string currency = 2;
    double cost_per_month = 3;
    string billing_detail = 4;
    repeated ImpactMetric impact_metrics = 5;   // NEW
}

message GetProjectedCostRequest {
    ResourceDescriptor resource = 1;
    double utilization_percentage = 2;          // NEW
}

message ResourceDescriptor {
    // ... existing fields ...
    optional double utilization_percentage = 6; // NEW
}
```

**Implementation**:

```go
import pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"

// In estimateEC2:
resp := &pbc.GetProjectedCostResponse{
    // ... existing financial fields ...
    ImpactMetrics: []*pbc.ImpactMetric{
        {
            Kind:  pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
            Value: carbonGrams,
            Unit:  "gCO2e",
        },
    },
}

// In Supports for EC2:
resp := &pbc.SupportsResponse{
    Supported:        true,
    SupportedMetrics: []pbc.MetricKind{pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT},
}
```

### 6. Utilization Percentage Handling

**Decision**: Priority order for utilization: per-resource > request-level > default

**Rationale**:

- Matches proto design intent (per-resource overrides global)
- Default of 50% is reasonable for most workloads
- Clamping to 0.0-1.0 prevents invalid calculations

**Implementation**:

```go
func getUtilization(req *pbc.GetProjectedCostRequest, resource *pbc.ResourceDescriptor) float64 {
    // Priority 1: Per-resource override
    if resource.UtilizationPercentage != nil && *resource.UtilizationPercentage > 0 {
        return clamp(*resource.UtilizationPercentage, 0.0, 1.0)
    }

    // Priority 2: Request-level value
    if req.UtilizationPercentage > 0 {
        return clamp(req.UtilizationPercentage, 0.0, 1.0)
    }

    // Priority 3: Default
    return DefaultUtilization // 0.50
}

func clamp(v, min, max float64) float64 {
    if v < min { return min }
    if v > max { return max }
    return v
}
```

## Dependencies Update

**Required**: Update `go.mod` to use `finfocus-spec v0.4.10`:

```bash
go get github.com/rshade/finfocus-spec@v0.4.10
go mod tidy
```

## Attribution Requirements

Per Apache 2.0 license, add to NOTICE file or README:

```text
Cloud Carbon Footprint
https://www.cloudcarbonfootprint.org/
Copyright 2021 Thoughtworks, Inc.
Licensed under the Apache License, Version 2.0
```

## Validation Complete

All NEEDS CLARIFICATION items resolved:

| Item | Resolution |
|------|------------|
| Data source | CCF cloud-carbon-coefficients CSV |
| Grid factors | Static constants from CCF |
| Formula | CCF methodology exactly |
| Proto types | finfocus-spec v0.4.10 |
| Utilization | Priority: per-resource > request > default (0.5) |
