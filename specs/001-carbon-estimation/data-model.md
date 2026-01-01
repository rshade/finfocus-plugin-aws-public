# Data Model: Comprehensive Carbon Estimation

**Feature**: 001-carbon-estimation  
**Date**: 2025-12-31  
**Phase**: Design & Contracts

This document defines the core data entities, relationships, and validation rules for carbon estimation across AWS services.

---

## Core Entities

### 1. InstanceSpec (existing)

**Purpose**: Stores EC2 instance power characteristics from CCF data.

**Location**: `internal/carbon/instance_specs.go`

**Fields**:
| Field | Type | Description | Source |
|-------|------|-------------|--------|
| InstanceType | string | EC2 instance type (e.g., "t3.micro") | CCF aws-instances.csv |
| VCPUCount | int | Number of virtual CPUs | CCF aws-instances.csv |
| MinWatts | float64 | Power consumption at idle (watts per vCPU) | CCF aws-instances.csv |
| MaxWatts | float64 | Power consumption at 100% utilization (watts per vCPU) | CCF aws-instances.csv |

**Validation Rules**:
- `VCPUCount >= 1`
- `MinWatts >= 0`
- `MaxWatts >= MinWatts`

**Relationships**: Used by `CarbonEstimator.EstimateCarbonGrams()`

---

### 2. GPUSpec (new)

**Purpose**: Stores GPU accelerator specifications for GPU-enabled instances.

**Location**: `internal/carbon/gpu_specs.go` (to be created)

**Fields**:
| Field | Type | Description | Source |
|-------|------|-------------|--------|
| InstanceType | string | GPU instance type (e.g., "p4d.24xlarge") | AWS Instance Families |
| GPUModel | string | GPU model name (e.g., "A100") | NVIDIA/AWS Specifications |
| GPUCount | int | Number of GPUs per instance | AWS Instance Types |
| TDPPerGPU | float64 | Thermal Design Power per GPU (watts) | Manufacturer Datasheets |

**Validation Rules**:
- `GPUCount >= 0` (0 for non-GPU instances)
- `TDPPerGPU >= 0` (watts)

**Embedded Data Format** (CSV):
```csv
instance_type,gpu_model,gpu_count,tdp_per_gpu_watts
p4d.24xlarge,A100,8,400
p4de.24xlarge,A100,8,400
g5.xlarge,A10G,1,150
g5.4xlarge,A10G,1,150
g5.8xlarge,A10G,1,150
g5.12xlarge,A10G,4,150
g5.16xlarge,A10G,1,150
g5.24xlarge,A10G,4,150
inf2.xlarge,Inferentia2,1,175
inf2.8xlarge,Inferentia2,2,175
inf2.24xlarge,Inferentia2,6,175
inf2.48xlarge,Inferentia2,12,175
trn1.2xlarge,Trainium,1,175
trn1.32xlarge,Trainium,8,175
```

**Relationships**: Used by new `GPUEstimator.EstimateCarbonGrams()`

---

### 3. StorageSpec (new)

**Purpose**: Maps storage technologies (SSD/HDD) to power coefficients and replication factors.

**Location**: `internal/carbon/storage_specs.go` (to be created)

**Fields**:
| Field | Type | Description | Source |
|-------|------|-------------|--------|
| ServiceType | string | AWS service ("ebs" or "s3") | Feature Spec |
| StorageClass | string | Storage class/type (e.g., "gp3", "STANDARD") | AWS Documentation |
| Technology | string | Storage technology ("SSD" or "HDD") | CCF Methodology |
| ReplicationFactor | int | Replication factor for durability (1×, 2×, 3×) | AWS Architecture |
| PowerCoefficient | float64 | Power coefficient (Watt-Hours per TB-Hour) | CCF Methodology |

**Embedded Data Format** (CSV):
```csv
service_type,storage_class,technology,replication_factor,power_coefficient_wh_per_tbh
ebs,gp2,SSD,2,1.2
ebs,gp3,SSD,2,1.2
ebs,io1,SSD,2,1.2
ebs,io2,SSD,2,1.2
ebs,st1,HDD,2,0.65
ebs,sc1,HDD,2,0.65
s3,STANDARD,SSD,3,1.2
s3,STANDARD_IA,SSD,3,1.2
s3,INTELLIGENT_TIERING,SSD,3,1.2
s3,ONEZONE_IA,SSD,1,1.2
s3,GLACIER,HDD,3,0.65
s3,DEEP_ARCHIVE,HDD,3,0.65
```

**Validation Rules**:
- `ReplicationFactor >= 1` and `ReplicationFactor <= 3`
- `PowerCoefficient > 0`

**Relationships**: Used by `StorageEstimator.EstimateCarbonGrams()`

---

### 4. GridFactor (existing)

**Purpose**: Maps AWS regions to grid carbon emission factors.

**Location**: `internal/carbon/grid_factors.go`

**Fields** (map[string]float64):
| Key | Value | Description | Source |
|-----|-------|-------------|--------|
| Region Code | float64 | Grid intensity (metric tons CO2e/kWh) | CCF Methodology |

**Current Regions**: us-east-1, us-east-2, us-west-1, us-west-2, ca-central-1, eu-west-1, eu-north-1, ap-southeast-1, ap-southeast-2, ap-northeast-1, ap-south-1, sa-east-1

**Default**: `DefaultGridFactor = 0.00039278` (global average)

**Validation Rules**:
- Grid factor between 0.0 and 2.0 metric tons CO2e/kWh

**Relationships**: Used by all carbon estimators

---

### 5. TotalCarbonEstimate (new)

**Purpose**: Composite carbon result containing operational, embodied, and total carbon.

**Location**: `internal/carbon/types.go` (to be created)

**Fields**:
| Field | Type | Description | Unit |
|-------|------|-------------|-------|
| OperationalCarbon | float64 | Carbon from energy consumption (compute + storage) | gCO2e |
| EmbodiedCarbon | float64 | Carbon from hardware manufacturing (amortized monthly) | gCO2e |
| TotalCarbon | float64 | Sum of operational + embodied carbon | gCO2e |

**Calculation**:
```
TotalCarbon = OperationalCarbon + EmbodiedCarbon
```

**Validation Rules**:
- `OperationalCarbon >= 0`
- `EmbodiedCarbon >= 0` (0 if not enabled)
- `TotalCarbon >= OperationalCarbon`

**Relationships**: Returned by `CarbonEstimator.EstimateCarbonGrams()` when embodied carbon enabled

---

### 6. EmbodiedCarbonConfig (new)

**Purpose**: Configuration for embodied carbon calculation.

**Location**: `internal/carbon/types.go` (to be created)

**Fields**:
| Field | Type | Description | Default |
|-------|------|-------------|----------|
| Enabled | bool | Whether to include embodied carbon in estimates | false |
| ServerLifespanMonths | int | Server lifespan for amortization (months) | 48 (4 years) |
| EmbodiedCarbonPerServer | float64 | Total embodied carbon per server (kgCO2e) | 1000 |

**Validation Rules**:
- `ServerLifespanMonths > 0`
- `EmbodiedCarbonPerServer > 0`

**Relationships**: Used by `CarbonEstimator.EstimateCarbonGrams()` to calculate embodied carbon

---

## Service-Specific Entities

### 7. EC2InstanceConfig (new)

**Purpose**: Configuration for EC2 instance carbon estimation.

**Location**: `internal/carbon/types.go` (to be created)

**Fields**:
| Field | Type | Description | Required |
|-------|------|-------------|----------|
| InstanceType | string | EC2 instance type | Yes |
| Region | string | AWS region | Yes |
| Utilization | float64 | CPU utilization (0.0 to 1.0) | Yes (default 0.50) |
| Hours | float64 | Operating hours | Yes |
| IncludeGPU | bool | Include GPU power consumption | Yes (default true) |
| IncludeEmbodiedCarbon | bool | Include embodied carbon | Yes (default false) |

**Derived Calculations**:
- CPU Power = `InstanceSpec.MinWatts + (Utilization × (InstanceSpec.MaxWatts - InstanceSpec.MinWatts))`
- GPU Power (if applicable) = `GPUSpec.TDPPerGPU × GPUSpec.GPUCount × Utilization`
- Total Power = `CPU Power × VCPUCount + GPU Power`

---

### 8. EBSVolumeConfig (new)

**Purpose**: Configuration for EBS volume carbon estimation.

**Location**: `internal/carbon/types.go` (to be created)

**Fields**:
| Field | Type | Description | Required |
|-------|------|-------------|----------|
| VolumeType | string | EBS volume type (gp2, gp3, io1, io2, st1, sc1) | Yes |
| SizeGB | float64 | Volume size in gigabytes | Yes |
| Region | string | AWS region | Yes |
| Hours | float64 | Storage duration (hours) | Yes |

**Derived Calculations**:
- SizeTB = `SizeGB / 1024`
- Energy = `(SizeTB × Hours × StorageSpec.PowerCoefficient × StorageSpec.ReplicationFactor) / 1000`
- Carbon = `Energy × GridFactor × AWSPUE × 1,000,000`

---

### 9. S3StorageConfig (new)

**Purpose**: Configuration for S3 storage carbon estimation.

**Location**: `internal/carbon/types.go` (to be created)

**Fields**:
| Field | Type | Description | Required |
|-------|------|-------------|----------|
| StorageClass | string | S3 storage class | Yes |
| SizeGB | float64 | Storage size in gigabytes | Yes |
| Region | string | AWS region | Yes |
| Hours | float64 | Storage duration (hours) | Yes |

**Derived Calculations**:
- SizeTB = `SizeGB / 1024`
- Energy = `(SizeTB × Hours × StorageSpec.PowerCoefficient × StorageSpec.ReplicationFactor) / 1000`
- Carbon = `Energy × GridFactor × AWSPUE × 1,000,000`

---

### 10. LambdaFunctionConfig (new)

**Purpose**: Configuration for Lambda function carbon estimation.

**Location**: `internal/carbon/types.go` (to be created)

**Fields**:
| Field | Type | Description | Required |
|-------|------|-------------|----------|
| MemoryMB | int | Allocated memory in megabytes | Yes |
| DurationMs | int | Average invocation duration in milliseconds | Yes |
| Invocations | int64 | Total number of invocations | Yes |
| Architecture | string | CPU architecture (x86_64, arm64) | Yes (default x86_64) |
| Region | string | AWS region | Yes |

**Derived Calculations**:
- vCPU Equivalent = `MemoryMB / 1792`
- Running Time (Hours) = `DurationMs × Invocations / 3,600,000`
- Average Watts = `MinWatts + 0.50 × (MaxWatts - MinWatts)` (50% utilization)
- Energy = `(Average Watts × vCPU Equivalent × Running Time) / 1000`
- Carbon = `Energy × GridFactor × AWSPUE × 1,000,000`
- ARM64 Multiplier = `0.80` (20% efficiency improvement)

---

### 11. RDSInstanceConfig (new)

**Purpose**: Configuration for RDS instance carbon estimation (compute + storage).

**Location**: `internal/carbon/types.go` (to be created)

**Fields**:
| Field | Type | Description | Required |
|-------|------|-------------|----------|
| InstanceType | string | RDS instance class (EC2-equivalent) | Yes |
| Region | string | AWS region | Yes |
| MultiAZ | bool | Multi-AZ deployment | Yes (default false) |
| StorageType | string | Storage type (gp3, io1, io2, etc.) | Yes |
| StorageSizeGB | float64 | Storage size in gigabytes | Yes |
| Utilization | float64 | CPU utilization (0.0 to 1.0) | Yes (default 0.50) |
| Hours | float64 | Operating hours | Yes |

**Derived Calculations**:
- Compute Carbon = EC2 instance carbon × MultiAZ Multiplier (2× if MultiAZ)
- Storage Carbon = EBS volume carbon × Storage Replication Factor (2× if MultiAZ)
- Total Carbon = Compute Carbon + Storage Carbon

---

### 12. DynamoDBTableConfig (new)

**Purpose**: Configuration for DynamoDB table carbon estimation (storage only).

**Location**: `internal/carbon/types.go` (to be created)

**Fields**:
| Field | Type | Description | Required |
|-------|------|-------------|----------|
| SizeGB | float64 | Table storage size in gigabytes | Yes |
| Region | string | AWS region | Yes |
| Hours | float64 | Storage duration (hours) | Yes |

**Derived Calculations**:
- SizeTB = `SizeGB / 1024`
- Energy = `(SizeTB × Hours × 1.2 Wh/TB × 3× Replication) / 1000` (SSD, 3× replication)
- Carbon = `Energy × GridFactor × AWSPUE × 1,000,000`

---

### 13. EKSClusterConfig (new)

**Purpose**: Configuration for EKS cluster carbon estimation (worker nodes only).

**Location**: `internal/carbon/types.go` (to be created)

**Fields**:
| Field | Type | Description | Required |
|-------|------|-------------|----------|
| Region | string | AWS region | Yes |

**Behavior**: Returns zero carbon for control plane. Billing detail directs users to estimate worker nodes as EC2 instances.

---

## Constants

### PUE and Utilization

| Constant | Value | Description | Source |
|----------|-------|-------------|--------|
| AWSPUE | 1.135 | AWS Power Usage Effectiveness | CCF Methodology |
| DefaultUtilization | 0.50 | Default CPU utilization (50%) | CCF Methodology |
| HoursPerMonth | 730.0 | Standard hours per month | Billing Standard |
| VCPUPer1792MB | 1.0 | Lambda vCPU equivalent (1792 MB = 1 vCPU) | AWS Lambda |

### Embodied Carbon Defaults

| Constant | Value | Description | Source |
|----------|-------|-------------|--------|
| EmbodiedCarbonPerServer | 1000.0 | Embodied carbon per server (kgCO2e) | CCF Methodology |
| ServerLifespanMonths | 48 | Server lifespan for amortization (months) | CCF Methodology |

### Grid Emission Factors

| Region | Grid Factor (metric tons CO2e/kWh) | Description |
|--------|--------------------------------------|-------------|
| us-east-1 | 0.000379 | Virginia (SERC) |
| us-east-2 | 0.000411 | Ohio (RFC) |
| us-west-1 | 0.000322 | N. California (WECC) |
| us-west-2 | 0.000322 | Oregon (WECC) |
| ca-central-1 | 0.00012 | Canada |
| eu-west-1 | 0.0002786 | Ireland |
| eu-north-1 | 0.0000088 | Sweden (very low carbon) |
| ap-southeast-1 | 0.000408 | Singapore |
| ap-southeast-2 | 0.00079 | Sydney |
| ap-northeast-1 | 0.000506 | Tokyo |
| ap-south-1 | 0.000708 | Mumbai |
| sa-east-1 | 0.0000617 | São Paulo (very low carbon) |
| DefaultGridFactor | 0.00039278 | Global average (for unknown regions) |

---

## State Transitions

### Carbon Estimator Lifecycle

```
Init → Parse Embedded Data (sync.Once) → Ready → EstimateCarbonGrams() → Complete
                ↓
            Error (invalid data)
```

### Grid Factor Update Lifecycle (Phase 3)

```
Fetch Latest Grid Factors → Validate Range → Update Map → Test → Deploy
           ↓                    ↓              ↓         ↓        ↓
    Error/Warning          Out of Range   Updated   Pass     Live
```

---

## Validation Rules Summary

### Input Validation
- All instance types must exist in embedded data
- All storage classes must exist in embedded data
- All regions must have a grid factor (or use default)
- Utilization must be between 0.0 and 1.0
- Hours must be >= 0
- Storage sizes must be >= 0

### Output Validation
- Carbon values must be >= 0
- Total carbon must equal sum of operational + embodied (if enabled)
- Grid factors must be within valid range (0.0 to 2.0)

### Embedded Data Validation
- Instance specs: vCPU >= 1, MinWatts >= 0, MaxWatts >= MinWatts
- GPU specs: GPUCount >= 0, TDPPerGPU >= 0
- Storage specs: ReplicationFactor >= 1 and <= 3, PowerCoefficient > 0
- Grid factors: 0.0 <= factor <= 2.0

---

## Relationships

### Estimator Interfaces

```
CarbonEstimator (interface)
    ├─ EstimateCarbonGrams(instanceType, region, utilization, hours) → float64, bool
    │   └─ Uses: InstanceSpec, GPUSpec, GridFactor, EmbodiedCarbonConfig
    │
    ├─ StorageEstimator (new)
    │   ├─ EstimateEBSVolumeCarbon(volumeType, sizeGB, region, hours) → float64, bool
    │   └─ Uses: StorageSpec, GridFactor
    │
    ├─ S3Estimator (new)
    │   ├─ EstimateS3StorageCarbon(storageClass, sizeGB, region, hours) → float64, bool
    │   └─ Uses: StorageSpec, GridFactor
    │
    ├─ LambdaEstimator (new)
    │   ├─ EstimateLambdaCarbon(memoryMB, durationMs, invocations, architecture, region) → float64, bool
    │   └─ Uses: InstanceSpec (for power values), GridFactor
    │
    ├─ RDSEstimator (new)
    │   ├─ EstimateRDSCarbon(instanceType, multiAZ, storageType, sizeGB, region, utilization, hours) → float64, bool
    │   └─ Uses: InstanceSpec, StorageSpec, GridFactor
    │
    ├─ DynamoDBEstimator (new)
    │   ├─ EstimateDynamoDBCarbon(sizeGB, region, hours) → float64, bool
    │   └─ Uses: StorageSpec, GridFactor
    │
    └─ EKSEstimator (new)
        └─ EstimateEKSCarbon(region) → float64 (always 0), string (billing detail)
```

### Data Flow

```
ResourceDescriptor (gRPC) → Service Router → Specific Estimator → Calculate → TotalCarbonEstimate
                                     ↓                                     ↓
                             Embedded Data (specs)                      gRPC Response
```

---

## File Structure

```
internal/carbon/
├── constants.go              # AWSPUE, DefaultUtilization, etc.
├── estimator.go              # Existing EC2 estimator
├── estimator_test.go         # Existing EC2 tests
├── instance_specs.go         # InstanceSpec (existing)
├── instance_specs_test.go    # InstanceSpec tests
├── grid_factors.go           # GridFactor (existing)
├── gpu_specs.go             # GPUSpec (new)
├── gpu_specs_test.go        # GPU specs tests (new)
├── storage_specs.go         # StorageSpec (new)
├── storage_specs_test.go    # Storage specs tests (new)
├── types.go                 # Service-specific configs, TotalCarbonEstimate (new)
├── ebs_estimator.go         # EBS estimator (new)
├── ebs_estimator_test.go    # EBS tests (new)
├── s3_estimator.go          # S3 estimator (new)
├── s3_estimator_test.go     # S3 tests (new)
├── lambda_estimator.go       # Lambda estimator (new)
├── lambda_estimator_test.go  # Lambda tests (new)
├── rds_estimator.go         # RDS estimator (new)
├── rds_estimator_test.go    # RDS tests (new)
├── dynamodb_estimator.go    # DynamoDB estimator (new)
├── dynamodb_estimator_test.go # DynamoDB tests (new)
├── eks_estimator.go         # EKS estimator (new)
├── eks_estimator_test.go    # EKS tests (new)
└── embodied_carbon.go       # Embodied carbon calculator (new)
```

---

## Testing Strategy

### Unit Tests (per estimator)
- Table-driven tests for all input combinations
- Edge cases: zero hours, zero utilization, unknown instance types
- Validation tests: invalid inputs return false
- Reference tests: compare against CCF reference calculations

### Integration Tests
- gRPC service methods with mock pricing clients
- End-to-end carbon estimation with real embedded data
- Concurrent access tests (thread safety)

### Performance Tests
- Benchmark estimators to ensure <100ms target
- Verify lazy initialization with sync.Once
- Test memory footprint with embedded data
