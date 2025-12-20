# Data Model: Carbon Emission Estimation

**Feature**: 015-carbon-estimation
**Date**: 2025-12-19

## Entities

### InstanceSpec

Maps EC2 instance type to power consumption characteristics derived from CCF data.

| Field | Type | Description | Source |
|-------|------|-------------|--------|
| InstanceType | string | EC2 instance type (e.g., "t3.micro") | CCF CSV column 1 |
| VCPUCount | int | Number of virtual CPUs | CCF CSV column 3 |
| MinWatts | float64 | Power consumption at idle (watts per vCPU) | CCF CSV "PkgWatt @ Idle" |
| MaxWatts | float64 | Power consumption at 100% (watts per vCPU) | CCF CSV "PkgWatt @ 100%" |
| Architecture | string | CPU microarchitecture (e.g., "Skylake") | CCF CSV column 5 (informational) |

**Validation Rules**:

- InstanceType must be non-empty
- VCPUCount must be >= 1
- MinWatts must be >= 0
- MaxWatts must be >= MinWatts

**Uniqueness**: InstanceType is unique key

### GridEmissionFactor

Maps AWS region to grid carbon intensity.

| Field | Type | Description |
|-------|------|-------------|
| Region | string | AWS region code (e.g., "us-east-1") |
| Intensity | float64 | Carbon intensity (metric tons CO2eq/kWh) |

**Predefined Values** (from CCF):

| Region | Intensity | Notes |
|--------|-----------|-------|
| us-east-1 | 0.000379 | Virginia (SERC) |
| us-east-2 | 0.000411 | Ohio (RFC) |
| us-west-1 | 0.000322 | N. California (WECC) |
| us-west-2 | 0.000322 | Oregon (WECC) |
| ca-central-1 | 0.00012 | Canada |
| eu-west-1 | 0.0002786 | Ireland |
| eu-north-1 | 0.0000088 | Sweden (very low) |
| ap-southeast-1 | 0.000408 | Singapore |
| ap-southeast-2 | 0.00079 | Sydney |
| ap-northeast-1 | 0.000506 | Tokyo |
| ap-south-1 | 0.000708 | Mumbai |
| sa-east-1 | 0.0000617 | São Paulo (very low) |
| _default | 0.00039278 | Global average fallback |

### CarbonEstimate

Result of carbon calculation for a single resource.

| Field | Type | Description |
|-------|------|-------------|
| CarbonGrams | float64 | Estimated carbon emissions (gCO2e) |
| EnergyKWh | float64 | Estimated energy consumption (kWh) - optional, for future use |
| Utilization | float64 | CPU utilization used in calculation (0.0-1.0) |
| Region | string | AWS region used for grid factor |
| InstanceType | string | EC2 instance type |

## Data Flow

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                         GetProjectedCostRequest                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │ ResourceDescriptor                                               │   │
│  │   - resource_type: "aws:ec2/instance:Instance"                  │   │
│  │   - sku: "t3.micro"                                             │   │
│  │   - region: "us-east-1"                                         │   │
│  │   - utilization_percentage: 0.7 (optional)                      │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│  utilization_percentage: 0.5 (request-level default)                   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Carbon Estimator                               │
│                                                                         │
│  1. Get utilization: per-resource (0.7) > request (0.5) > default (0.5)│
│  2. Lookup InstanceSpec for "t3.micro":                                │
│     - VCPUCount: 2, MinWatts: 0.47, MaxWatts: 1.69                    │
│  3. Lookup GridEmissionFactor for "us-east-1": 0.000379               │
│  4. Apply CCF formula:                                                 │
│     avgWatts = 0.47 + (0.7 × (1.69 - 0.47)) = 1.324                   │
│     energyKWh = (1.324 × 2 × 730) / 1000 = 1.933                      │
│     energyPUE = 1.933 × 1.135 = 2.194                                 │
│     carbonGrams = 2.194 × 0.000379 × 1,000,000 = 831.5                │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                       GetProjectedCostResponse                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │ Financial Cost (existing)                                        │   │
│  │   - cost_per_month: $7.59                                       │   │
│  │   - unit_price: $0.0104/hr                                      │   │
│  │   - currency: "USD"                                             │   │
│  │   - billing_detail: "On-demand Linux, shared tenancy, 730 hrs"  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │ ImpactMetrics (NEW)                                              │   │
│  │   - [0] kind: METRIC_KIND_CARBON_FOOTPRINT                      │   │
│  │         value: 831.5                                            │   │
│  │         unit: "gCO2e"                                           │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

## State Transitions

Not applicable - carbon estimation is stateless. Each request is independent.

## Indexes (In-Memory)

### instanceSpecs (map[string]InstanceSpec)

- **Key**: Instance type (e.g., "t3.micro")
- **Value**: InstanceSpec struct
- **Initialization**: Parsed from embedded CSV via sync.Once
- **Size**: ~500 entries, ~50KB memory

### gridFactors (map[string]float64)

- **Key**: AWS region code (e.g., "us-east-1")
- **Value**: Grid emission factor (metric tons CO2eq/kWh)
- **Initialization**: Static Go constants
- **Size**: 13 entries, negligible memory

## Relationships

```text
ResourceDescriptor ──────┬───────> InstanceSpec
  (sku = instance_type)  │            (VCPUCount, MinWatts, MaxWatts)
                         │
                         └───────> GridEmissionFactor
  (region)                           (Intensity)
                         │
                         ▼
                   CarbonEstimate ──────> ImpactMetric
                         │                  (proto message)
                         │
                         └───────> GetProjectedCostResponse.impact_metrics
```

## CSV Schema (CCF aws-instances.csv)

Columns used from CCF data:

| Column Index | Column Name | Maps To |
|--------------|-------------|---------|
| 0 | Instance type | InstanceSpec.InstanceType |
| 2 | Instance vCPU | InstanceSpec.VCPUCount |
| 4 | Platform CPU Name | InstanceSpec.Architecture |
| 14 | PkgWatt @ Idle | InstanceSpec.MinWatts |
| 17 | PkgWatt @ 100% | InstanceSpec.MaxWatts |

Note: Column indices are 0-based. CSV parsing should handle header row and skip empty/malformed lines.
