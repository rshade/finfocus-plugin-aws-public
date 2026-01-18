# Carbon Estimation

This document describes the carbon footprint estimation capabilities of the
finfocus-plugin-aws-public plugin.

## Overview

Carbon estimation uses the [Cloud Carbon Footprint (CCF)](https://www.cloudcarbonfootprint.org/)
methodology to calculate operational carbon emissions (gCO2e) for AWS resources.

**Supported services:**

| Service | Carbon Estimation | Method |
|---------|------------------|--------|
| EC2 | ✅ Full | CPU power × utilization × grid factor |
| EC2 (GPU) | ✅ Full | CPU + GPU power × utilization × grid factor |
| EBS | ✅ Full | Storage energy × replication × grid factor |
| S3 | ✅ Full | Storage energy × replication × grid factor |
| Lambda | ✅ Full | vCPU equivalent × duration × grid factor |
| RDS | ✅ Full | Compute + storage carbon |
| DynamoDB | ✅ Full | Storage-based (SSD × 3× replication) |
| ElastiCache | ✅ Full | EC2-equivalent node carbon × cluster size |
| EKS | ⚠️ Control plane only | Returns 0 (shared infrastructure) |

## Carbon Formula

### EC2 Instances (CPU)

```text
avgWatts = minWatts + (utilization × (maxWatts - minWatts))
energyKWh = (avgWatts × vCPUs × hours) / 1000
energyWithPUE = energyKWh × 1.135  (AWS PUE)
carbonGrams = energyWithPUE × gridIntensity × 1,000,000
```

### GPU Instances

GPU power is added to the CPU carbon:

```text
gpuPowerWatts = numGPUs × gpuTDP × utilization
gpuEnergyKWh = (gpuPowerWatts × hours) / 1000
gpuEnergyWithPUE = gpuEnergyKWh × 1.135
gpuCarbonGrams = gpuEnergyWithPUE × gridIntensity × 1,000,000
totalCarbon = cpuCarbon + gpuCarbon
```

### Storage Services (EBS, S3, DynamoDB)

```text
energyWhPerTB = storageCoefficient  // 1.2 Wh/TB for SSD, 0.65 for HDD
totalEnergy = (sizeGB / 1000) × energyWhPerTB × replicationFactor × hours
carbonGrams = (totalEnergy / 1000) × gridIntensity × 1,000,000 × PUE
```

### Lambda Functions

```text
vCPUEquivalent = memoryMB / 1792  // AWS allocates 1 vCPU per 1792 MB
computeHours = (durationMs × invocations) / 3,600,000
carbonGrams = computeCarbonPerVCPU × vCPUEquivalent × efficiencyFactor
```

ARM64 efficiency factor: 0.80 (20% more efficient than x86_64)

## Usage Examples

### EC2 Carbon Estimation

Request:

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "ec2",
    "sku": "m5.large",
    "region": "us-east-1"
  }
}
```

Response includes carbon in `impact_metrics`:

```json
{
  "cost_per_month": 70.08,
  "impact_metrics": [
    {
      "kind": "METRIC_KIND_CARBON_FOOTPRINT",
      "value": 12543.7,
      "unit": "gCO2e"
    }
  ]
}
```

### GPU Instance Carbon

Request:

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "ec2",
    "sku": "p4d.24xlarge",
    "region": "us-east-1"
  }
}
```

Response includes both CPU and GPU carbon:

```json
{
  "impact_metrics": [
    {
      "kind": "METRIC_KIND_CARBON_FOOTPRINT",
      "value": 1547832.5,
      "unit": "gCO2e"
    }
  ],
  "billing_detail": "... CPU: 1045438 gCO2e, GPU: 502394 gCO2e ..."
}
```

### EBS Volume Carbon

Request:

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "ebs",
    "sku": "gp3",
    "region": "us-west-2",
    "tags": {
      "size": "500"
    }
  }
}
```

Response includes carbon in `impact_metrics`:

```json
{
  "cost_per_month": 40.00,
  "impact_metrics": [
    {
      "kind": "METRIC_KIND_CARBON_FOOTPRINT",
      "value": 287.4,
      "unit": "gCO2e"
    }
  ],
  "billing_detail": "EBS gp3 storage, 500GB, carbon: 287.4 gCO2e/month"
}
```

### Lambda Function Carbon

Request:

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "lambda",
    "sku": "arm64",
    "region": "eu-west-1",
    "tags": {
      "memory_mb": "1024",
      "duration_ms": "500",
      "requests_per_month": "1000000"
    }
  }
}
```

### RDS Instance Carbon

Request:

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "rds",
    "sku": "db.m5.large",
    "region": "us-east-1",
    "tags": {
      "engine": "mysql",
      "storage_size_gb": "100",
      "storage_type": "gp3",
      "multi_az": "true"
    }
  }
}
```

Multi-AZ doubles both compute and storage carbon.

## Regional Grid Factors

Carbon estimates vary significantly by AWS region due to different electricity
generation mixes:

| Region | Grid Factor | Relative |
|--------|-------------|----------|
| eu-north-1 (Sweden) | 0.0000088 | Cleanest (hydro) |
| sa-east-1 (Brazil) | 0.0000617 | Very clean (hydro) |
| ca-central-1 (Canada) | 0.00012 | Clean |
| eu-west-1 (Ireland) | 0.0002786 | Moderate |
| us-west-2 (Oregon) | 0.000322 | Moderate |
| us-east-1 (Virginia) | 0.000379 | Average |
| ap-southeast-1 (Singapore) | 0.000408 | Above average |
| ap-south-1 (Mumbai) | 0.000708 | High (coal) |

**Implication:** Running the same workload in eu-north-1 (~0.0000088 tCO2e/kWh, hydro)
vs ap-south-1 (~0.000708 tCO2e/kWh, coal) can result in **80× less carbon emissions**.

## Utilization

Carbon estimation uses a utilization factor (0.0 to 1.0) representing average
CPU/GPU usage:

**Priority order:**

1. Per-resource: `ResourceDescriptor.utilization_percentage`
2. Request-level: `GetProjectedCostRequest.utilization_percentage`
3. Default: 50% (0.5)

Higher utilization = more power consumption = more carbon.

## Embodied Carbon

Embodied carbon represents the manufacturing emissions of hardware, amortized
over the expected server lifespan (48 months).

### Why Disabled by Default

Embodied carbon is **disabled by default** for several reasons:

1. **Conservative estimates**: Only operational carbon is reported by default since it
   can be measured with higher confidence (actual power consumption × grid factor)
2. **Comparability**: Most carbon calculators (AWS Customer Carbon Footprint Tool,
   Google Cloud Carbon Footprint) report only operational (Scope 2) emissions,
   making comparison easier
3. **User expectation**: Enterprise users typically expect operational-only baselines
4. **Uncertainty**: Embodied carbon values (1000 kgCO2e/server) are rough estimates
   that vary significantly by manufacturer and server configuration

### Enabling Embodied Carbon

To include embodied carbon in estimates, set `IncludeEmbodiedCarbon: true` in the
EC2InstanceConfig:

```go
config := carbon.EC2InstanceConfig{
    InstanceType:          "m5.large",
    Region:                "us-east-1",
    IncludeEmbodiedCarbon: true,
    EmbodiedConfig: carbon.EmbodiedCarbonConfig{
        Enabled:                 true,
        ServerLifespanMonths:    48,   // 4 years (default)
        EmbodiedCarbonPerServer: 1000, // kgCO2e (default)
    },
}
```

### Supported Services

Currently supported for:

- EC2 instances (using CCF methodology: 1000 kgCO2e per server)

### Formula

```text
monthlyEmbodiedCarbon = (1000 kgCO2e / 48 months) × (instanceVCPUs / maxFamilyVCPUs)
```

The vCPU scaling factor accounts for shared server resources—a smaller instance
consumes a proportionally smaller share of the server's embodied carbon.

**API Note:** The `EC2InstanceConfig` struct has a top-level `IncludeEmbodiedCarbon`
boolean for convenience. When `true`, the `EmbodiedConfig.Enabled` field is
automatically honored. Use `IncludeEmbodiedCarbon: true` as the primary toggle;
the nested `EmbodiedConfig` struct allows overriding default values (lifespan,
carbon per server) when needed.

## Limitations

1. **Scope 2 only**: Estimates cover operational (electricity) emissions only
2. **No data transfer**: Network carbon not included
3. **Steady-state**: Assumes constant utilization over the period
4. **Grid factors**: Updated annually; may lag real-world grid changes
5. **EKS control plane**: Returns 0 (shared infrastructure)

### RDS Multi-AZ Limitations

Multi-AZ carbon calculation uses a 2× multiplier assuming same-region deployment
where both availability zones share the same grid emission factor. Cross-region
Multi-AZ deployments (if applicable) may have different actual carbon footprints
due to varying grid intensities between regions.

Regional grid factors can vary significantly—up to 80× between eu-north-1 (Sweden,
hydro) and ap-south-1 (Mumbai, coal). If AWS ever supports cross-region Multi-AZ
for RDS, this assumption would need to be revisited to use region-specific grid
factors for each replica.

## Data Sources

- **Instance power data**: [cloud-carbon-coefficients](https://github.com/cloud-carbon-footprint/cloud-carbon-coefficients)
- **Grid emission factors**: CCF + EPA eGRID (US regions)
- **GPU specifications**: CCF GPU power data
- **Storage coefficients**: CCF storage methodology

## Related Documentation

- [Grid Factor Updates](./grid-factor-updates.md) - Annual update process
- [CCF Methodology](https://www.cloudcarbonfootprint.org/docs/methodology)
