# Strategic Roadmap: finfocus-plugin-aws-public

## Mission Statement

To provide the most comprehensive, air-gapped cost and carbon estimation engine
for AWS, enabling continuous governance and pre-deployment planning without the
security overhead of cloud credentials.

---

## Immediate Focus [In Progress / Planned]

- **[In Progress] Bug Fixes:**
  - **Sparse OldState:** Handle sparse `OldState` properties for cost diff
    accuracy (#292) [M]
  - **ARN Region Extraction:** Extract region from ARN-format resource IDs in
    `GetActualCost` routing (#320) [M]
  - **Context Propagation:** Restore context propagation for child process
    lifecycle (#314) [M]
  - ~~**Zero-Cost PricingSpec:** Add zero-cost branch to `GetPricingSpec`
    switch (#319) [S]~~ ✅
  - **allowEmptyRegion:** Make `allowEmptyRegion` service-aware in
    `parseResourceFromRequest` (#324) [S]
  - **Logger Injection:** Replace global logger in `parsePositiveIntField`
    with injected logger (#323) [S]
  - **Zero-Cost PricingSpec:** Add zero-cost branch to `GetPricingSpec`
    switch (#319) [S]
- **[In Progress] Code Quality (PR #305 Follow-ups):**
  - ~~**Doc Fixes:** Fix doc/code mismatches in instance_specs, client.go,
    registry.go (#325) [S]~~ ✅
  - ~~**Doc Clarification:** Clarify `parsePositiveIntField` and
    `parseGoMapString` limitations (#317, #315) [S]~~ ✅
  - ~~**RWMutex Optimization:** Use `RWMutex` for child getters and reuse EBS
    estimator (#322) [S]~~ ✅
  - **Service Constants:** Use service constants in `ZeroCostServices` and
    `ZeroCostPulumiPatterns` maps (#321) [S]
  - **Lint Directives:** Replace global threshold increases with targeted
    `nolint` directives (#316) [S]
  - **Gosec Narrowing:** Narrow gosec suppression for tools/ to specific
    rules (#318) [S]
  - **golangci-lint Upgrade:** Upgrade from v2.5.0 to v2.8.0 (#291) [S]
- **[Planned] Build Infrastructure:**
  - **Region Mapping Consolidation:** Consolidate all region-to-tag mappings
    to use `regions.yaml` as single source of truth, eliminating hardcoded
    duplicates in shell scripts (#287) [M]
- **[Planned] Service Breadth Expansion:**
  - **Route53:** Hosted zones and basic query volume estimation.
  - **CloudFront:** Basic data transfer and request pricing (based on regional
    estimates).

---

## Future Vision [Researching / Planned]

- **[Planned] Service Breadth (New Services):**
  - **ASG Estimator:** Add estimator for
    `aws:autoscaling/group:Group` (#295) [L]
- **[Researching] Memory Optimization:** Implementing lazy-loading or
  memory-mapped access for embedded JSON files to reduce the runtime memory
  footprint without moving to an external database (#84) [L]
- **[Planned] Service Depth (Phase 2):**
  - **EBS Depth:** Adding IOPS and Throughput pricing for `gp3`, `io1`, and
    `io2`.
- **[Researching] Cross-Service Recommendations:** Static lookup logic to
  suggest move-to-managed alternatives (e.g., self-managed DB on EC2 -> RDS)
  based on Resource Tags.
- **[Planned] Additional Regions:** Expansion to include GovCloud
  (us-gov-west-1, us-gov-east-1) and specialized regions (Beijing/Ningxia,
  EU-North-1) as public pricing data parity allows. Infrastructure exists but
  regions.yaml catalog incomplete (#271, #272) [M]
- **[Planned] Forecasting Intelligence:**
  - **Growth Hints:** Implement logic to return `GrowthType` (Linear) for
    accumulation-based resources (S3, ECR, Backup) to support Core forecasting.
- **[Planned] Topology Awareness:**
  - **Lineage Metadata:** Populate `ParentResourceID` for dependent resources
    (e.g., EBS Volumes attached to Instances, NAT Gateways attached to VPCs) to
    support "Blast Radius" visualization.
- **[Planned] Capability Discovery Enhancements:**
  - **Dual-Layer Discovery:** Service-level and resource-level capability
    introspection for richer client integration (#258) [L]
- **[Planned] Testing & Quality:**
  - **E2E Integration Tests:** Pulumi YAML fixture-based end-to-end
    tests (#216) [L]
  - **Memory Profiling:** CI memory usage tracking for parsing (#182) [M]
  - **Pricing Metadata Validation:** Cross-service consistency checks
    (#181) [M]
  - **gRPCurl Integration Tests:** Pricing verification via gRPCurl (#174) [M]
  - **Pricing Client Refactor:** Better maintainability for initialization
    (#83) [M]
- **[Planned] Small Improvements:**
  - Test cleanup: #267, #266, #227, #185, #184, #144, #129 [S]
  - Perf: traceLogger optimization (#265) [S]
  - Refactor: nolint conditionals (#264), port dedup (#259) [S]
  - Docker: configurable metrics threshold (#260) [S]
  - CloudWatch validation warnings (#212) [S]

---

## Completed Milestones

### Q1 2026

- **Zero-Cost PricingSpec:** `GetPricingSpec` now returns `BillingMode: "zero_cost"`
  for VPC, Security Groups, Subnets, IAM, Launch Templates, and Launch
  Configurations instead of falling through to "unknown" (#319).
- **Sparse OldState Defaults Metadata:** Populate `Metadata` field with
  `defaults_applied` and `estimate_quality` on `GetProjectedCostResponse`,
  enabling cost diff engine to distinguish "real $0" from "defaulted $0"
  (#292).
- **ARN Region Extraction:** Extract region from ARN-format resource IDs in
  `GetActualCost` routing (#320).
- **Context Propagation:** Restored context propagation for child process
  lifecycle (#314).
- **allowEmptyRegion:** Made `allowEmptyRegion` service-aware in
  `parseResourceFromRequest` (#324).
- **Logger Injection:** Replaced global logger in `parsePositiveIntField`
  with injected logger (#323).
- **Doc Fixes:** Fixed doc/code mismatches in instance_specs, client.go,
  registry.go (#325).
- **Doc Clarification:** Clarified `parsePositiveIntField` and
  `parseGoMapString` limitations (#317, #315).
- **LaunchTemplate Fix:** LaunchTemplate no longer incorrectly priced as EC2
  instance (#294).
- **GetActualCost Zero-Cost:** `GetActualCost` now handles zero-cost resources
  (VPC, Subnet, SecurityGroup) without error (#293).
- **Linter Compliance:** Resolved all golangci-lint issues and protected
  `.golangci.yml` configuration (#289).
- **Multi-Region Router:** Single-port entry point that auto-discovers and
  delegates to region-specific child processes. Supports parallel fan-out
  for multi-region recommendations and automatic binary downloads from
  GitHub Releases (#245).
- **Router Binary Entrypoint:** Created `cmd/finfocus-plugin-aws-public-router`
  entry point with eager warm-up of discovered region binaries. Release
  pipeline builds router archive alongside region archives.
- **Service Type Caching:** `serviceResolver` with lazy initialization
  pattern to memoize `normalizeResourceType()` and `detectService()` results,
  reducing redundant calls across all RPC methods (#157).
- **IAM Zero-Cost Resources:** Added IAM users, roles, policies, groups, and
  instance profiles to zero-cost handling (#274).
- **us-west-1 Region Support:** Added N. California region to the supported
  region matrix (#273).
- **Carbon Metrics Advertisement:** `getSupportedMetrics` now accurately
  reflects carbon estimation availability per service (#257).
- **Zero-Cost Resource Handling:** Graceful handling for AWS resources
  with no direct cost (VPC, Security Groups, Subnets) - return $0 estimates
  instead of SKU errors (#237).
- **Multi-Region Docker:** Single Docker image containing all regional
  binaries with tini init and Prometheus metrics aggregation (#244).
- **CORS Configuration:** Expose CORS configuration via environment
  variables (#243).
- **Plugin Rename:** `pulumicost-plugin-aws-public` renamed to
  `finfocus-plugin-aws-public` (#239).
- **Metadata Enrichment:** Dev Mode Heuristics (#209), Resource Topology
  Linking (#208), and Storage Growth Heuristics (#207).
- **JSON Parsing Optimizations:** `go-json` integration and map
  pre-allocation for faster pricing data initialization (#228, #176).
- **Recommendations Enhancements:** Configurable `maxBatchSize` (#160)
  and optional strict validation mode (#156).

### Q4 2025

- **Actual Cost:** Runtime-based `GetActualCost` using Pulumi state
  metadata, with intelligent fallback to 730-hour monthly projections
  when usage is absent (#196).
- **ResourceID Pass-Through:** Pass through ResourceID in
  `GetRecommendations` responses (#198).
- **Per-Service Embedding:** Transition to per-service raw JSON embedding
  to manage binary size and initialization speed (#170, #171).
- **Cost Standards:** FOCUS 1.2 cost record format support with
  standardized pricing specifications.

### Foundation

- **Core Infrastructure:** gRPC `CostSourceService` implementation, regional
  build matrix (12 regions), and `zerolog` trace propagation.
- **Compute:** EC2 On-Demand cost estimation, Lambda (requests + GB-seconds,
  x86_64/arm64), and CCF-based Carbon Footprint (gCO2e) metrics.
- **Storage:** EBS (Basic Storage GB-month pricing), S3 (Storage by storage
  class).
- **Managed Services:** EKS Control Plane, DynamoDB (On-Demand/Provisioned
  with validation and hardening), ELB (ALB/NLB with LCU/NLCU support), RDS
  (instance + storage, multi-engine), and ElastiCache (Redis/Memcached/Valkey
  node pricing).
- **Networking:** NAT Gateway (hourly + data processing per GB), CloudWatch
  (Logs ingestion/storage with tiered pricing, custom metrics).
- **Optimization:** `GetRecommendations` batch processing for
  `target_resources` (up to 100 items), SDK mapping package integration for
  configurable recommendation rules.
- **Carbon Estimation (Comprehensive):** Full carbon footprint estimation
  suite covering EC2 (CPU/GPU), EBS (SSD/HDD), RDS (compute + storage),
  S3 (by storage class), Lambda (vCPU-equivalent), DynamoDB (storage-based),
  EKS (control plane), ElastiCache (EC2-equivalent mapping), embodied carbon,
  and GPU-specific power specs.

---

## Strategic Guardrails

1. **Statelessness:** No local databases or historical trend storage. Data
   "intelligence" (comparisons) belongs in FinFocus Core.
2. **Air-Gapped:** Zero runtime network calls. All estimates derived from
   build-time snapshots.
3. **Static Logic:** Recommendations are based on static mappings and SKU
   attributes, never on live monitoring or external telemetry.
