# Feature Specification: Comprehensive Carbon Estimation Expansion

**Feature Branch**: `001-carbon-estimation`
**Created**: 2025-12-31
**Status**: Draft
**Input**: User description: "Expand carbon footprint estimation from EC2-only to all supported AWS services, add GPU power consumption, implement embodied carbon, and establish a sustainable update process. This bundle consolidates six related carbon issues (#135-140) into a phased implementation plan."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - GPU Instance Carbon Estimation (Priority: P1)

As a sustainability-focused infrastructure engineer, I want to receive accurate carbon footprint estimates for GPU instances (p4d, g5, inf2, trn1 families) that include accelerator power consumption, so that I can make informed decisions about ML/AI workload placement considering both cost and environmental impact.

**Why this priority**: GPU instances are high-power consumers with significantly underestimated carbon if only CPU is calculated. ML/AI workloads are growing rapidly, making this a critical gap for accurate sustainability reporting.

**Independent Test**: Can be fully tested by requesting carbon estimates for GPU instance types and validating that reported carbon reflects both CPU and GPU power consumption based on manufacturer TDP specifications.

**Acceptance Scenarios**:

1. **Given** a p4d.24xlarge instance request, **When** carbon estimation is performed, **Then** the estimate includes power consumption from 8x A100 GPUs (400W each) plus CPU power
2. **Given** a g5.xlarge instance (1 GPU), **When** carbon estimation is performed, **Then** the GPU power (150W for A10G) is included in the total carbon footprint
3. **Given** an inference instance (inf2.24xlarge with 6 Inferentia2 chips), **When** carbon estimation is performed, **Then** the accelerator power (175W each) is factored into the estimate

---

### User Story 2 - EBS Storage Carbon Estimation (Priority: P1)

As a cloud cost and sustainability analyst, I want to receive carbon footprint estimates for EBS volumes that account for storage technology (SSD vs HDD) and replication factors, so that I can include storage carbon in infrastructure sustainability reports.

**Why this priority**: EBS is one of the most commonly used AWS services and represents significant infrastructure footprint. Storage carbon is currently missing entirely from estimates.

**Independent Test**: Can be fully tested by requesting carbon estimates for different EBS volume types (gp3, io2, st1) and validating estimates reflect appropriate power coefficients per the CCF methodology.

**Acceptance Scenarios**:

1. **Given** a 500GB gp3 SSD volume in us-east-1, **When** carbon estimation is requested, **Then** the system returns a carbon footprint in gCO2e based on SSD power coefficient (1.2 Wh/TB) with 2x replication factor
2. **Given** a 2TB st1 HDD volume in eu-west-1, **When** carbon estimation is requested, **Then** the system returns a carbon footprint based on HDD power coefficient (0.65 Wh/TB)
3. **Given** an EBS volume request, **When** the Supports() check is called with carbon metrics, **Then** the system advertises carbon estimation capability for EBS resources

---

### User Story 3 - S3 Storage Carbon Estimation (Priority: P2)

As an enterprise sustainability officer, I want carbon estimates for S3 storage that account for different storage classes and their replication factors, so that I can accurately report the environmental impact of data storage across the organization.

**Why this priority**: S3 is ubiquitous in AWS deployments and different storage classes have significantly different carbon footprints due to replication and storage technology differences.

**Independent Test**: Can be fully tested by requesting carbon estimates for S3 buckets with different storage classes and validating the estimates reflect the correct storage technology and replication factor.

**Acceptance Scenarios**:

1. **Given** 100GB of S3 STANDARD storage, **When** carbon estimation is requested, **Then** the estimate reflects SSD storage with 3x replication factor
2. **Given** 1TB of S3 GLACIER storage, **When** carbon estimation is requested, **Then** the estimate reflects HDD storage with 3x replication factor
3. **Given** 500GB of S3 ONEZONE_IA storage, **When** carbon estimation is requested, **Then** the estimate reflects SSD storage with 1x replication factor (no cross-AZ replication)

---

### User Story 4 - Lambda Function Carbon Estimation (Priority: P2)

As a serverless application developer, I want to understand the carbon footprint of my Lambda functions based on memory allocation, duration, and invocation count, so that I can optimize for both performance and sustainability.

**Why this priority**: Serverless is increasingly adopted, and Lambda carbon depends on memory allocation (proxy for compute) and architecture choice (ARM vs x86).

**Independent Test**: Can be fully tested by providing Lambda function parameters (memory, duration, invocations, architecture) and validating the carbon estimate reflects the CCF compute methodology.

**Acceptance Scenarios**:

1. **Given** a Lambda function with 1792MB memory, 500ms duration, 1M invocations, **When** carbon estimation is requested, **Then** the estimate reflects approximately 1 vCPU equivalent of compute carbon
2. **Given** a Lambda function using arm64 architecture, **When** carbon estimation is requested, **Then** the estimate reflects ~20% efficiency improvement compared to x86_64
3. **Given** Lambda parameters, **When** the system calculates carbon, **Then** the estimate converts millisecond duration to hours for the CCF formula

---

### User Story 5 - RDS Instance Carbon Estimation (Priority: P2)

As a database administrator tracking sustainability metrics, I want carbon estimates for RDS instances that include both compute and storage components, with consideration for Multi-AZ deployments, so that I can report comprehensive database infrastructure carbon footprint.

**Why this priority**: RDS is a composite resource (compute + storage) and Multi-AZ doubles the physical footprint, making accurate estimation important for database-heavy deployments.

**Independent Test**: Can be fully tested by requesting carbon estimates for RDS instances with different configurations (instance class, storage size, Multi-AZ setting) and validating compute + storage components are summed correctly.

**Acceptance Scenarios**:

1. **Given** a db.m5.large RDS instance with 100GB storage, **When** carbon estimation is requested, **Then** the estimate includes both compute carbon (based on m5.large equivalent) and storage carbon (SSD-backed)
2. **Given** a Multi-AZ RDS deployment, **When** carbon estimation is requested, **Then** the estimate reflects 2x the single-instance carbon (synchronous replica)
3. **Given** an RDS instance request, **When** storage type is gp3, **Then** storage carbon uses SSD coefficients

---

### User Story 6 - DynamoDB Table Carbon Estimation (Priority: P2)

As a NoSQL database user, I want carbon estimates for DynamoDB tables based on storage consumption, so that I can include managed database carbon in sustainability reporting.

**Why this priority**: DynamoDB compute is managed/opaque, but storage carbon can be estimated using the same methodology as S3 Standard (SSD with 3x replication).

**Independent Test**: Can be fully tested by providing DynamoDB storage parameters and validating the carbon estimate matches S3 Standard methodology (SSD, 3x replication).

**Acceptance Scenarios**:

1. **Given** a DynamoDB table with 50GB storage, **When** carbon estimation is requested, **Then** the estimate reflects SSD storage with 3x replication factor
2. **Given** a DynamoDB table in any supported region, **When** carbon estimation is requested, **Then** the regional grid factor is applied correctly

---

### User Story 7 - Embodied Carbon Estimation (Priority: P3)

As a comprehensive sustainability analyst, I want to include embodied carbon (manufacturing, transportation, disposal) in infrastructure estimates, amortized over the equipment lifespan, so that I can report total lifecycle carbon footprint.

**Why this priority**: Embodied carbon is a significant but often overlooked component. Including it provides complete lifecycle visibility, but operational carbon is more actionable for day-to-day decisions.

**Independent Test**: Can be fully tested by requesting total carbon estimates with embodied carbon enabled and validating the breakdown shows operational + embodied components separately.

**Acceptance Scenarios**:

1. **Given** an EC2 instance request with embodied carbon enabled, **When** carbon estimation is performed, **Then** the response includes both operational carbon and monthly amortized embodied carbon as separate values
2. **Given** a t3.micro instance (2 vCPUs) vs m5.24xlarge (96 vCPUs), **When** embodied carbon is calculated, **Then** the m5.24xlarge receives proportionally more embodied carbon based on vCPU share of server capacity
3. **Given** embodied carbon calculation, **When** the server lifespan assumption is applied, **Then** the system uses 4 years (48 months) for amortization per CCF methodology

---

### User Story 8 - Grid Factor Update Process (Priority: P3)

As a plugin maintainer, I want an automated or semi-automated process to update regional grid emission factors annually, so that carbon estimates remain accurate as electricity grids evolve.

**Why this priority**: Grid factors change as energy sources evolve. Outdated factors lead to inaccurate estimates, but this is a maintenance concern rather than core functionality.

**Independent Test**: Can be fully tested by running the grid factor update tool and validating it fetches data from authoritative sources and produces valid grid factor data.

**Acceptance Scenarios**:

1. **Given** the grid factor update tool is executed, **When** it completes, **Then** it produces updated grid factors from CCF or EPA eGRID sources
2. **Given** updated grid factors, **When** they are integrated, **Then** the values fall within reasonable ranges for each region
3. **Given** the annual update process, **When** documented, **Then** the documentation includes calendar reminders and validation steps

---

### Edge Cases

- What happens when an unknown GPU instance family is requested?
  - System falls back to CPU-only estimation with a warning in billing_detail
- How does the system handle unknown EBS volume types?
  - Default to SSD coefficient with a note in billing_detail
- What happens when a region has no grid factor defined?
  - Use global average grid factor with warning
- How does the system handle instance types not in the CCF specs database?
  - Extrapolate from similar instance family or return error with guidance
- What happens when storage size is zero or negative?
  - Return zero carbon with validation warning
- How does Multi-AZ interact with read replicas for RDS?
  - Each component (primary, standby, replicas) is estimated separately

## Requirements *(mandatory)*

### Functional Requirements

**GPU Power Consumption (Phase 1)**

- **FR-001**: System MUST include GPU accelerator power consumption when estimating carbon for GPU instance families (p4d, p4de, p5, g4dn, g5, inf1, inf2, trn1)
- **FR-002**: System MUST use manufacturer-specified TDP (Thermal Design Power) values for GPU power calculations
- **FR-003**: System MUST calculate total power as CPU power + (GPU power × GPU count × utilization)
- **FR-004**: System MUST report whether an instance has GPU acceleration in the estimation response

**EBS Storage Carbon (Phase 1)**

- **FR-005**: System MUST estimate carbon for EBS volumes based on storage technology (SSD vs HDD)
- **FR-006**: System MUST apply different power coefficients: 1.2 Wh/TB for SSD, 0.65 Wh/TB for HDD
- **FR-007**: System MUST apply 2x replication factor for EBS (AZ-level replication)
- **FR-008**: System MUST map volume types to storage technology: gp2/gp3/io1/io2 → SSD, st1/sc1 → HDD
- **FR-009**: System MUST advertise carbon estimation capability for EBS in the Supports() response

**S3 Storage Carbon (Phase 2)**

- **FR-010**: System MUST estimate carbon for S3 storage based on storage class
- **FR-011**: System MUST apply appropriate replication factors: 3x for STANDARD/GLACIER, 1x for ONEZONE_IA
- **FR-012**: System MUST map storage classes to technology: STANDARD/IA → SSD, GLACIER/DEEP_ARCHIVE → HDD

**Lambda Carbon (Phase 2)**

- **FR-013**: System MUST estimate Lambda carbon based on memory allocation (memory ÷ 1792MB = vCPU equivalent)
- **FR-014**: System MUST apply 20% efficiency factor for arm64 architecture
- **FR-015**: System MUST convert duration (milliseconds) × invocations to hours for energy calculation

**RDS Carbon (Phase 2)**

- **FR-016**: System MUST estimate RDS carbon as composite of compute + storage
- **FR-017**: System MUST double carbon estimate for Multi-AZ deployments
- **FR-018**: System MUST use EC2-equivalent power specs for RDS instance classes

**DynamoDB Carbon (Phase 2)**

- **FR-019**: System MUST estimate DynamoDB carbon based on storage using SSD coefficients with 3x replication

**EKS Carbon (Phase 2)**

- **FR-020**: System MUST return zero carbon for EKS control plane with documentation that worker nodes should be estimated as EC2 instances
- **FR-021**: System MUST document the rationale for excluding control plane carbon in billing_detail

**Embodied Carbon (Phase 3)**

- **FR-022**: System MUST calculate embodied carbon using CCF methodology: 1000 kgCO2e per server, 4-year lifespan
- **FR-023**: System MUST scale embodied carbon proportionally based on instance vCPUs relative to maximum family vCPUs
- **FR-024**: System MUST provide separate operational and embodied carbon values in the response
- **FR-025**: System MUST allow embodied carbon to be optionally included (not required by default)

**Grid Factor Updates (Phase 3)**

- **FR-026**: System MUST provide a tool to fetch and update regional grid emission factors
- **FR-027**: System MUST validate that updated grid factors fall within reasonable ranges (0.0 to 2.0 metric tons CO2e/kWh)
- **FR-028**: System MUST document the annual update process with validation steps

**General Requirements**

- **FR-029**: All carbon calculations MUST use the CCF methodology with proper source references
- **FR-030**: All carbon values MUST be returned in gCO2e (grams CO2 equivalent)
- **FR-031**: All carbon estimators MUST apply AWS PUE factor of 1.135
- **FR-032**: System MUST apply regional grid emission factors based on the resource's region

### Key Entities

- **InstanceSpec**: Instance type power characteristics including minWatts, maxWatts, vCPU count, sourced from CCF coefficients
- **GPUSpec**: GPU accelerator specifications including model name, TDP watts per GPU, GPU count per instance type
- **StorageSpec**: Storage carbon parameters including technology type (SSD/HDD), power coefficient (Wh/TB), replication factor
- **GridFactor**: Regional grid emission intensity in metric tons CO2e per kWh, mapped to AWS regions
- **TotalCarbonEstimate**: Composite carbon result containing operational carbon, embodied carbon, and total carbon

### Assumptions

- AWS PUE (Power Usage Effectiveness) is 1.135 per CCF methodology
- Default CPU utilization is 50% when not specified
- Server embodied carbon is 1000 kgCO2e per CCF methodology
- Server lifespan for amortization is 4 years (48 months)
- GPU utilization equals CPU utilization when not separately specified
- EBS replication factor is 2x (within-AZ)
- S3 STANDARD/GLACIER replication factor is 3x (cross-AZ)
- Lambda executes at 100% utilization during invocation
- RDS storage is SSD-backed (gp3 equivalent) by default

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: GPU instance carbon estimates reflect accelerator power consumption, with p4d.24xlarge showing significantly higher carbon than comparable non-GPU instances
- **SC-002**: EBS volume carbon estimates are returned for all supported volume types (gp2, gp3, io1, io2, st1, sc1)
- **SC-003**: All services (EC2, EBS, S3, Lambda, RDS, DynamoDB) return carbon metrics in the GetProjectedCost response
- **SC-004**: Carbon estimation coverage expands from 1 service (EC2) to 6+ services
- **SC-005**: Supports() response correctly advertises METRIC_KIND_CARBON_FOOTPRINT for all services with carbon estimation
- **SC-006**: Embodied carbon can be optionally included, with clear separation between operational and embodied components
- **SC-007**: Grid factor update process is documented and can be executed successfully to refresh emission factors
- **SC-008**: All carbon calculations include source references to CCF methodology documentation
- **SC-009**: Unit test coverage for the carbon package achieves 80% or higher
- **SC-010**: Carbon estimates fall within reasonable ranges validated against CCF reference calculations
