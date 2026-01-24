# Strategic Roadmap: finfocus-plugin-aws-public

## Mission Statement

To provide the most comprehensive, air-gapped cost and carbon estimation engine
for AWS, enabling continuous governance and pre-deployment planning without the
security overhead of cloud credentials.

---

## Past Milestones [Done]

- **Core Infrastructure:** gRPC `CostSourceService` implementation, regional
  build matrix (12 regions), and `zerolog` trace propagation.
- **Compute:** EC2 On-Demand cost estimation, Lambda (requests + GB-seconds,
  x86_64/arm64), and CCF-based Carbon Footprint (gCO2e) metrics.
- **Storage:** EBS (Basic Storage GB-month pricing), S3 (Storage by storage class).
- **Managed Services:** EKS Control Plane, DynamoDB (On-Demand/Provisioned with
  validation and hardening), ELB (ALB/NLB with LCU/NLCU support), RDS (instance +
  storage, multi-engine), and ElastiCache (Redis/Memcached/Valkey node pricing).
- **Networking:** NAT Gateway (hourly + data processing per GB), CloudWatch
  (Logs ingestion/storage with tiered pricing, custom metrics).
- **Optimization:** `GetRecommendations` batch processing for `target_resources`
  (up to 100 items), SDK mapping package integration for configurable
  recommendation rules.
- **Architecture:** Transition to per-service raw JSON embedding to manage
  binary size and initialization speed.
- **Performance:** `go-json` integration and map pre-allocation for faster
  pricing data initialization.
- **Cost Standards:** FOCUS 1.2 cost record format support with standardized
  pricing specifications.
- **Actual Cost:** Runtime-based `GetActualCost` using Pulumi state metadata,
  with intelligent fallback to 730-hour monthly projections when usage is absent.
- **Carbon Estimation (Comprehensive):** Full carbon footprint estimation suite:
  - EC2 instances with CPU/GPU power consumption (CCF methodology)
  - EBS volumes (SSD/HDD coefficients with replication factors)
  - RDS instances (compute + storage carbon, Multi-AZ 2× multiplier)
  - S3 storage (by storage class with replication factors)
  - Lambda functions (vCPU-equivalent + ARM64 efficiency adjustment)
  - DynamoDB tables (storage-based with 3× SSD replication)
  - EKS clusters (control plane guidance, worker nodes as EC2)
  - ElastiCache nodes (EC2-equivalent mapping for cache node types)
  - Embodied carbon (server manufacturing amortization per CCF)
  - GPU-specific power specs for P/G series instances
  - Storage specs embedded from CCF cloud-carbon-coefficients
- **Multi-Region Docker:** Single Docker image containing all regional
  binaries with tini init and Prometheus metrics aggregation.
- **us-west-1 Region Support:** Added N. California region to the supported
  region matrix (#273).
- **Zero-Cost Resource Handling:** Graceful handling for AWS resources
  with no direct cost (VPC, Security Groups, Subnets) - return $0 estimates
  instead of SKU errors (#237).
- **Carbon Metrics Advertisement:** `getSupportedMetrics` now accurately
  reflects carbon estimation availability per service (#257).
- **IAM Zero-Cost Resources:** Added IAM users, roles, policies, groups, and
  instance profiles to zero-cost handling (#274).

---

## Immediate Focus [In Progress / Planned]

- **[Planned] Service Breadth Expansion:**
  - **Route53:** Hosted zones and basic query volume estimation.
  - **CloudFront:** Basic data transfer and request pricing (based on regional
    estimates).
- **[Planned] Build Infrastructure:**
  - **Region Mapping Consolidation:** Consolidate all region-to-tag mappings
    to use `regions.yaml` as single source of truth, eliminating hardcoded
    duplicates in shell scripts (#287).

---

## Future Vision [Researching / Planned]

- **[Researching] Memory Optimization:** Implementing lazy-loading or
  memory-mapped access for embedded JSON files to reduce the runtime memory
  footprint without moving to an external database.
- **[Planned] Service Depth (Phase 2):**
  - **EBS Depth:** Adding IOPS and Throughput pricing for `gp3`, `io1`, and
    `io2`.
- **[Researching] Cross-Service Recommendations:** Static lookup logic to
  suggest move-to-managed alternatives (e.g., self-managed DB on EC2 -> RDS)
  based on Resource Tags.
- **[Planned] Additional Regions:** Expansion to include GovCloud
  (us-gov-west-1, us-gov-east-1) and specialized regions (Beijing/Ningxia,
  EU-North-1) as public pricing data parity allows. Infrastructure exists but
  regions.yaml catalog incomplete (#271, #272).
- **[Planned] Forecasting Intelligence:**
  - **Growth Hints:** Implement logic to return `GrowthType` (Linear) for
    accumulation-based resources (S3, ECR, Backup) to support Core forecasting.
- **[Planned] Topology Awareness:**
  - **Lineage Metadata:** Populate `ParentResourceID` for dependent resources
    (e.g., EBS Volumes attached to Instances, NAT Gateways attached to VPCs) to
    support "Blast Radius" visualization.
- **[Planned] Capability Discovery Enhancements:**
  - **Dual-Layer Discovery:** Service-level and resource-level capability
    introspection for richer client integration (#258).
- **[Planned] Multi-Region Router:** Single-port request routing for the Docker
  image to simplify client integration (#245).

---

## Strategic Guardrails (From CONTEXT.md)

1. **Statelessness:** No local databases or historical trend storage. Data
   "intelligence" (comparisons) belongs in FinFocus Core.
2. **Air-Gapped:** Zero runtime network calls. All estimates derived from
   build-time snapshots.
3. **Static Logic:** Recommendations are based on static mappings and SKU
   attributes, never on live monitoring or external telemetry.
