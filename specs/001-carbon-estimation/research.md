# Research: Comprehensive Carbon Estimation Expansion

**Feature**: 001-carbon-estimation  
**Date**: 2025-12-31  
**Research Status**: Complete

This document consolidates research findings for expanding carbon footprint estimation from EC2-only to all supported AWS services, including GPU power consumption, storage carbon, Lambda, RDS, DynamoDB, embodied carbon, and grid factor updates.

---

## 1. CCF Methodology and Carbon Estimation Formulas

### Decision
Use Cloud Carbon Footprint (CCF) methodology as the primary calculation framework.

### Rationale
- CCF is the industry standard for cloud carbon estimation
- Already partially implemented in the codebase (`internal/carbon/estimator.go`)
- Provides clear formulas for operational and embodied carbon
- Well-documented with ongoing community maintenance

### Key Formulas

**Operational Carbon Formula**:
```
Total CO2e = operational emissions + embodied emissions

Operational emissions = (Cloud provider service usage) × (Cloud energy conversion factors [kWh]) × (Cloud provider PUE) × (Grid emissions factors [metric tons CO2e])
```

**EC2 Instance Carbon Formula** (already implemented):
```
Step 1: Average watts = MinWatts + (utilization × (MaxWatts - MinWatts))
Step 2: Energy (kWh) = (Average watts × vCPU count × hours) / 1000
Step 3: Energy with PUE = Energy × AWS_PUE (1.135)
Step 4: Carbon (gCO2e) = Energy with PUE × grid intensity × 1,000,000
```

**AWS PUE**: 1.135 (from CCF methodology)  
**Default CPU Utilization**: 50% (CCF hyperscale datacenter assumption)

### Alternatives Considered
- AWS Customer Carbon Footprint Tool methodology: More complex allocation model, but aligned with CCF principles. Chose CCF for simplicity and community adoption.

---

## 2. GPU Power Consumption Specifications

### Decision
Create embedded GPU specifications table with TDP values per accelerator type.

### Rationale
- GPU instances have significantly higher power consumption than CPU-only instances
- Accurate GPU power data is essential for carbon estimation of ML/AI workloads
- Manufacturers publish TDP specifications that can be embedded at build time

### GPU TDP Specifications

| GPU Model | TDP (Watts) | AWS Instance Families | Notes |
|-----------|--------------|----------------------|-------|
| NVIDIA A100 | 400 | p4d, p4de, p5 | SXM form factor (8 GPUs per p4d.24xlarge) |
| NVIDIA A10G | 150 | g4dn, g5 | AWS data sheet shows 300W, but independent sources confirm 150W. Use 150W for conservative estimates. |
| AWS Inferentia2 | 175 | inf2 | Estimated based on "50% better performance/watt" than G5 (A10G @ 150W) |
| AWS Trainium | 175 | trn1 | Estimated based on Inference/Trainium power parity |

**Calculation Approach**:
```
Total GPU Power = TDP per GPU × GPU count × utilization
Total Instance Power = CPU Power + Total GPU Power
```

### Alternatives Considered
- Runtime power monitoring via CloudWatch: Not feasible for cost estimation (requires instance running)
- Power estimation based on pricing: No direct correlation between price and power consumption
- Dynamic power scaling models: Too complex for cost estimation use case

---

## 3. Storage Carbon Coefficients (SSD vs HDD)

### Decision
Use CCF methodology coefficients: 1.2 Wh/TB for SSD, 0.65 Wh/TB for HDD.

### Rationale
- CCF provides widely accepted industry-standard values
- Coefficients are embedded in existing CCF methodology
- Values align with third-party research and AWS documentation

### Storage Technology Mapping

**EBS Volume Types**:
- **SSD**: gp2, gp3, io1, io2 (Use 1.2 Wh/TB coefficient)
- **HDD**: st1, sc1 (Use 0.65 Wh/TB coefficient)

**S3 Storage Classes**:
- **SSD**: STANDARD, STANDARD_IA, INTELLIGENT_TIERING, ONEZONE_IA (Use 1.2 Wh/TB coefficient)
- **HDD**: GLACIER, DEEP_ARCHIVE (Use 0.65 Wh/TB coefficient)

**Storage Carbon Formula**:
```
Storage Energy (kWh) = (Size in TB × Hours × Power Coefficient × Replication Factor) / 1000
Storage Carbon (gCO2e) = Storage Energy × Grid Factor × AWS_PUE × 1,000,000
```

### Alternatives Considered
- Custom power measurements: Too complex and varies by hardware generation
- AWS-specific values: Not publicly documented; CCF values are the industry standard
- Dynamic coefficients based on tier: Unnecessary complexity for cost estimation

---

## 4. AWS Service Replication Factors

### Decision
Use standard replication factors based on AWS service architecture and CCF methodology.

### Rationale
- Replication multiplies physical footprint for carbon estimation
- AWS documentation confirms replication strategies for each service
- CCF methodology validates these factors

### Replication Factors

| AWS Service | Default Replication Factor | Notes |
|--------------|---------------------------|-------|
| **EBS** | 2× | Replicated across multiple servers within an Availability Zone for durability |
| **S3 STANDARD** | 3× | Replicated across Availability Zones for durability and availability |
| **S3 GLACIER** | 3× | Same replication as STANDARD (archival tier) |
| **S3 ONEZONE_IA** | 1× | Single Availability Zone replication (lower durability, lower carbon) |
| **S3 DEEP_ARCHIVE** | 3× | Same replication as STANDARD (deep archival tier) |
| **DynamoDB** | 3× | Managed NoSQL, replication matches S3 Standard methodology |

**RDS Special Cases**:
- **Single-AZ**: 1× for compute + storage replication factor (based on EBS volume type)
- **Multi-AZ**: 2× for compute (standby replica) + storage replication factor
- **Read Replicas**: Each replica adds 1× compute + storage footprint

### Alternatives Considered
- Dynamic replication based on configuration: Too complex; standard factors cover most use cases
- User-configurable replication factors: Not supported in gRPC protocol
- Ignoring replication: Would significantly underestimate carbon footprint

---

## 5. Embodied Carbon Calculation Methodology

### Decision
Use CCF methodology: 1000 kgCO2e per server, amortized over 4-year (48-month) lifespan, scaled proportionally by vCPU share.

### Rationale
- CCF provides industry-standard baseline for server embodied carbon
- 4-year lifespan aligns with AWS server refresh cycles
- Proportional vCPU scaling enables fair allocation across instance types
- Embodied carbon is a significant but often overlooked component (20-30% of total footprint)

### Embodied Carbon Formula

**Monthly Amortized Embodied Carbon**:
```
Embodied Carbon per Server = 1000 kgCO2e (CCF baseline)
Monthly Embodied Carbon = (Embodied Carbon per Server / 48 months) × (Instance vCPUs / Max Family vCPUs)
```

**Example Calculations**:
- **t3.micro (2 vCPUs)**: (1000 / 48) × (2 / 2) = 20.83 kgCO2e/month
- **m5.large (2 vCPUs)**: (1000 / 48) × (2 / 96) = 0.43 kgCO2e/month (max vCPUs for m5 family is 96 for m5.24xlarge)
- **p4d.24xlarge (96 vCPUs)**: (1000 / 48) × (96 / 96) = 20.83 kgCO2e/month

**Total Carbon**:
```
Total Carbon = Operational Carbon + Embodied Carbon
```

### Alternatives Considered
- Per-instance embodied carbon (Dell R740: 970 kgCO2e): More accurate but requires extensive database lookup
- No embodied carbon: Underestimates total footprint by 20-30%
- User-configurable server lifespan: Adds complexity; 4 years is standard in industry

---

## 6. Regional Grid Emission Factors

### Decision
Use CCF methodology grid factors, already partially implemented in `internal/carbon/grid_factors.go`. Establish annual update process.

### Rationale
- Grid factors vary significantly by region (0.0000088 to 0.00079 metric tons CO2e/kWh)
- CCF provides regularly updated factors based on EPA eGRID and international sources
- Existing implementation already includes key regions
- Annual updates are necessary as electricity grids evolve

### Current Grid Factors (from codebase)

| Region | Grid Factor (metric tons CO2e/kWh) | Source Location |
|--------|--------------------------------------|-----------------|
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

**Default Grid Factor**: 0.00039278 (global average)

### Update Process (Phase 3)

**Sources**:
- Cloud Carbon Footprint repository: https://github.com/cloud-carbon-footprint/cloud-carbon-coefficients/tree/main/data
- EPA eGRID (US regions): https://www.epa.gov/egrid

**Update Steps**:
1. Fetch latest grid factors from CCF repository
2. Validate values fall within reasonable range (0.0 to 2.0 metric tons CO2e/kWh)
3. Update `GridEmissionFactors` map in `internal/carbon/grid_factors.go`
4. Run tests to ensure no regressions
5. Document update in changelog

### Alternatives Considered
- Real-time grid factor API: Too complex for cost estimation use case
- AWS Customer Carbon Footprint Tool API: Not publicly available
- Static factors indefinitely: Would lead to inaccurate estimates as grids evolve

---

## 7. Lambda Carbon Estimation

### Decision
Use CCF methodology for Lambda: estimate vCPUs from memory allocation, apply 50% utilization assumption.

### Rationale
- Lambda does not provide direct vCPU or CPU utilization metrics
- Memory allocation is the primary resource allocation parameter
- CCF methodology provides well-tested approach for serverless carbon estimation

### Lambda Carbon Formula

**vCPU Estimation**:
```
vCPU Equivalent = Lambda Memory Allocated (MB) / 1792 MB
```
(1792 MB = 1 vCPU in AWS Lambda)

**Carbon Calculation**:
```
Step 1: Average Watts = MinWatts + 0.50 × (MaxWatts - MinWatts)  [50% utilization assumption]
Step 2: Running Time (Hours) = Duration (milliseconds) × Invocations / 3,600,000
Step 3: Energy (kWh) = (Average Watts × vCPU Equivalent × Running Time) / 1000
Step 4: Energy with PUE = Energy × AWS_PUE (1.135)
Step 5: Carbon (gCO2e) = Energy with PUE × Grid Factor × 1,000,000
```

**Example**:
- Lambda: 1792 MB memory, 500ms duration, 1M invocations
- vCPU Equivalent: 1792 / 1792 = 1 vCPU
- Running Time: 500 × 1,000,000 / 3,600,000 = 138.89 hours
- Using average EC2 power: 2.12 + 0.50 × (4.5 - 2.12) = 3.31 Watts
- Energy: (3.31 × 1 × 138.89) / 1000 = 0.460 kWh
- Carbon (us-east-1): 0.460 × 1.135 × 0.000379 × 1,000,000 = 197.8 gCO2e

**ARM64 Efficiency Factor**:
```
ARM64 Carbon = x86_64 Carbon × 0.80  [20% efficiency improvement]
```

### Alternatives Considered
- Memory-based power estimation (1.8W per GB): More complex, CCF methodology is simpler
- CloudWatch Lambda Insights metrics: Requires runtime data collection (not available for cost estimation)
- Ignoring Lambda: Major coverage gap for serverless workloads

---

## 8. RDS Carbon Estimation

### Decision
Treat RDS as composite resource: compute (EC2-equivalent) + storage (EBS-equivalent).

### Rationale
- RDS uses EC2 instance classes for compute
- RDS storage uses EBS technology
- Multi-AZ deployments have synchronous replicas

### RDS Carbon Formula

**Compute Component** (EC2-equivalent):
```
Use existing EC2 carbon estimation with RDS instance type
Apply Multi-AZ multiplier: 2× for Multi-AZ deployments (primary + standby)
```

**Storage Component** (EBS-equivalent):
```
Use EBS storage carbon formula with RDS volume size and type
Apply replication factor: 1× for Single-AZ, 2× for Multi-AZ
```

**Total RDS Carbon**:
```
Total = Compute Carbon + Storage Carbon
```

### Alternatives Considered
- Single combined formula: More complex, better to reuse EC2 and EBS estimators
- Separate RDS power coefficients: Redundant with EC2 coefficients
- Ignoring storage: Underestimates carbon footprint for storage-heavy databases

---

## 9. DynamoDB Carbon Estimation

### Decision
Estimate DynamoDB carbon based on storage using SSD coefficients with 3× replication.

### Rationale
- DynamoDB compute is fully managed/opaque
- Storage is the only dimensionable resource
- Managed NoSQL services use SSD storage with cross-AZ replication similar to S3 Standard

### DynamoDB Carbon Formula

```
Storage Energy (kWh) = (Size in TB × Hours × 1.2 Wh/TB × 3× Replication) / 1000
Storage Carbon (gCO2e) = Storage Energy × Grid Factor × AWS_PUE × 1,000,000
```

**Note**: DynamoDB on-demand capacity mode (billing by read/write units) cannot be converted to carbon; only provisioned storage is estimable.

### Alternatives Considered
- RCUs/WCUs to carbon conversion: No direct mapping; requires complex modeling
- Fixed per-table carbon: Inaccurate for variable table sizes
- Ignoring DynamoDB: Misses significant NoSQL carbon footprint

---

## 10. EKS Carbon Estimation

### Decision
Return zero carbon for EKS control plane; document that worker nodes should be estimated as EC2 instances.

### Rationale
- EKS control plane is shared across customers (multi-tenant)
- AWS Customer Carbon Footprint Tool excludes control plane from customer allocations
- Worker nodes are EC2 instances and should be estimated separately

### EKS Response

```
Carbon: 0 gCO2e
Billing Detail: "EKS control plane carbon is shared and not allocated. Estimate worker nodes as EC2 instances."
```

### Alternatives Considered
- Allocate control plane carbon: Complex multi-tenant allocation; not standard in CCF methodology
- Estimate cluster management carbon: Minimal footprint compared to compute
- Return error: Would break cost estimation workflow

---

## Summary of Decisions

| Research Area | Decision | Implementation Priority |
|---------------|----------|------------------------|
| CCF Methodology | Use existing CCF formulas | Phase 1 (P0) |
| GPU TDP Specs | Create embedded GPU table (A100: 400W, A10G: 150W, etc.) | Phase 1 (P0) |
| Storage Coefficients | SSD: 1.2 Wh/TB, HDD: 0.65 Wh/TB | Phase 1 (P0) |
| Replication Factors | EBS: 2×, S3: 3×/1×, DynamoDB: 3× | Phase 1 (P0) |
| Embodied Carbon | 1000 kgCO2e/server, 4-year amortization, vCPU scaling | Phase 2 (P1) |
| Grid Factors | Use CCF factors, annual update process | Phase 3 (P2) |
| Lambda | Memory→vCPU (1792 MB = 1 vCPU), 50% utilization | Phase 2 (P1) |
| RDS | Compute (EC2) + Storage (EBS), Multi-AZ 2× | Phase 2 (P1) |
| DynamoDB | Storage only, SSD 1.2 Wh/TB, 3× replication | Phase 2 (P1) |
| EKS | Control plane: 0, worker nodes as EC2 | Phase 2 (P1) |

---

## Implementation Notes

### Embedded Data Requirements

**GPU Specifications Table** (to be created in `internal/carbon/gpu_specs.go`):
```
instance_type, gpu_model, gpu_count, tdp_per_gpu_watts
p4d.24xlarge, A100, 8, 400
p4de.24xlarge, A100, 8, 400
g5.xlarge, A10G, 1, 150
inf2.24xlarge, Inferentia2, 6, 175
trn1.32xlarge, Trainium, 8, 175
```

**Storage Technology Mapping** (to be created in `internal/carbon/storage_specs.go`):
```
ebs_volume_type, technology, replication_factor
gp2, SSD, 2
gp3, SSD, 2
io1, SSD, 2
io2, SSD, 2
st1, HDD, 2
sc1, HDD, 2
```

```
s3_storage_class, technology, replication_factor
STANDARD, SSD, 3
STANDARD_IA, SSD, 3
INTELLIGENT_TIERING, SSD, 3
ONEZONE_IA, SSD, 1
GLACIER, HDD, 3
DEEP_ARCHIVE, HDD, 3
```

### Testing Strategy

- Unit tests for each carbon estimator with table-driven test cases
- Integration tests for gRPC service methods
- Validation against CCF reference calculations
- Performance tests to ensure <100ms latency target

### Performance Considerations

- All lookup tables (instance specs, GPU specs, storage specs, grid factors) embedded at build time
- Use map lookups (O(1)) instead of linear scans
- Lazy initialization with sync.Once
- Thread-safe concurrent access (required for gRPC)

---

## References

### Primary Sources
- Cloud Carbon Footprint Methodology: https://cloudcarbonfootprint.org/docs/methodology
- CCF GitHub Repository: https://github.com/cloud-carbon-footprint/cloud-carbon-coefficients
- AWS Customer Carbon Footprint Tool: https://aws.amazon.com/customer-carbon-footprint-tool/

### GPU Specifications
- NVIDIA A100 Datasheet: https://www.nvidia.com/content/dam/en-zz/Solutions/Data-Center/a100/pdf/nvidia-a100-datasheet-us-nvidia-1758950-r4-web.pdf
- NVIDIA A10G Datasheet: https://d1.awsstatic.com/product-marketing/ec2/NVIDIA_AWS_A10G_DataSheet_FINAL_02_17_2022.pdf
- AWS Inf2 Instances: https://aws.amazon.com/ec2/instance-types/inf2/
- AWS Trn1 Instances: https://aws.amazon.com/ec2/instance-types/trn1/

### Storage Research
- Cloud Carbon Footprint Storage Coefficients: https://cloudcarbonfootprint.org/docs/methodology
- Backblaze HDD Energy Studies: https://www.backblaze.com/blog/hard-drive-energy-efficiency/
- SSD vs HDD Embodied Carbon: https://blog.purestorage.com/perspectives/how-does-the-embodied-carbon-dioxide-equivalent-of-flash-compare-to-hdds/

### Embodied Carbon
- HPE Server Carbon Footprint: https://cdn.accentuate.io/10102297198904/8155358527573/HPEproductcarbonfootprintE28093HPESynergy480Gen10ComputeModuledatasheet-a50005191enw-v1752161114217.pdf
- Boavizta Server Manufacturing GWP: https://boavizta.org/en/blog/empreinte-de-la-fabrication-d-un-serveur
- Tech Carbon Standard Server Example: https://www.techcarbonstandard.org/technology-categories/lifecycle/example/server

### Grid Emission Factors
- EPA eGRID: https://www.epa.gov/egrid
- CCF Grid Factors: https://github.com/cloud-carbon-footprint/cloud-carbon-coefficients/tree/main/data
- Climatiq Emission Factors: https://www.climatiq.io/data

### AWS Documentation
- AWS EBS Features: https://aws.amazon.com/ebs/features/
- AWS EBS FAQs: https://aws.amazon.com/ebs/faqs/
- AWS S3 Replication: https://aws.amazon.com/s3/features/replication/
- AWS Lambda Pricing: https://aws.amazon.com/lambda/pricing/
