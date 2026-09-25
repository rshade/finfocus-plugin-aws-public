# Strategic Roadmap: finfocus-plugin-aws-public

## Mission Statement

To provide the most comprehensive, air-gapped cost and carbon estimation engine
for AWS, enabling continuous governance and pre-deployment planning without the
security overhead of cloud credentials.

---

## Immediate Focus [In Progress / Planned]

- **[Planned] Pricing Correctness:**
  - [ ] #389 NAT Gateway pricing not found: AWS moved NAT Gateway products
    from the VPC offer to AmazonEC2; v0.1.9 likely affected [M]
- **[Planned] Terraform Support:**
  - [ ] #388 Accept Terraform-state attribute names and optional SKUs so all
    23 mapped Terraform types price (9 currently fail with
    `resource.sku is required`) [L]
- **[In Progress] Build Infrastructure:**
  - [ ] #287 Consolidate region mappings to `regions.yaml` as single source
    of truth, eliminating hardcoded duplicates in shell scripts [M]
- **[Ready] Router Hardening:**
  - [ ] #351 Re-enable strict checksum verification for region binary
    downloads [S]
    - Unblocked: v0.1.9 `checksums.txt` is clean (66 entries, no `./`
      prefix).
  - [ ] #396 Make runtime region-binary downloads visible (startup warning,
    documented `FINFOCUS_PLUGIN_OFFLINE`) and plan offline-by-default;
    depends on #351 [M]
- **[Planned] Service Breadth Expansion:**
  - **Route53:** Hosted zones and basic query volume estimation.
  - **CloudFront:** Basic data transfer and request pricing (based on regional
    estimates).

---

## Future Vision [Researching / Planned]

- **[Planned] Service Breadth (Phase 2):**
  - [ ] #393 ECS Fargate cost estimator for `aws:ecs/service:Service` [L]
  - [ ] #394 Amazon EFS cost estimator [M]
- **[Planned] Recommendations & Planning:**
  - [ ] #395 Static ElastiCache, Lambda arm64, and NAT Gateway endpoint
    recommendations [M]
  - [ ] #392 Opt-in cross-region cost and carbon comparison via
    region-migration recommendations [L]
- **[Planned] Pricing Data Trust:**
  - [ ] #390 Expose Price List `publicationDate` as a pricing freshness
    stamp [M]
  - [ ] #391 Pricing drift report comparing each release's pricing data with
    the previous release [L]
- **[Cross-repo] SDK:** backfill inferred capabilities when a
  `PluginInfoProvider` omits them (finfocus-spec#504); this plugin's explicit
  capability list stays as a workaround until the SDK bump.
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

### 2026-Q3

- [x] #385 `plugin`: Terraform types resolve to Pulumi tokens (#382). Closed 2026-09-24. [M]
- [x] #295 `plugin`: ASG cost estimator for autoscaling groups. Closed 2026-09-24. [L]
- [x] `lint`: golangci-lint v2.13.2 upgrade and goconst fixes (PR #377). [S]

### 2026-Q1

- [x] #347 `pricing`: bare metal EC2 instances no longer return $0. Closed 2026-03-26. [S]
- [x] #344 `router`: logger plugin_name aligned with region binaries. Closed 2026-03-24. [S]
- [x] #343 `release`: exact artifact count validation in release-router.sh. Closed 2026-03-24. [S]
- [x] #321 `plugin`: service constants in zero-cost resource maps. Closed 2026-03-19. [S]
- [x] #319 `plugin`: zero-cost branch added to GetPricingSpec. Closed 2026-03-19. [S]
- [x] #318 `lint`: gosec suppression for tools/ narrowed to rules. Closed 2026-03-19. [S]
- [x] #316 `lint`: targeted nolint replaces global threshold increases. Closed 2026-03-19. [S]
- [x] #292 `plugin`: defaults metadata for sparse OldState cost diffs. Closed 2026-03-19. [M]
- [x] #291 `lint`: golangci-lint upgraded from v2.5.0 to v2.8.0. Closed 2026-03-19. [S]
- [x] #324 `plugin`: allowEmptyRegion made service-aware. Closed 2026-03-18. [S]
- [x] #323 `plugin`: injected logger in parsePositiveIntField. Closed 2026-03-18. [S]
- [x] #322 `router`: RWMutex child getters, reused EBS estimator. Closed 2026-03-18. [S]
- [x] #320 `router`: region extracted from ARN resource IDs. Closed 2026-03-18. [M]
- [x] #317 `plugin`: documented parsePositiveIntField zero rejection. Closed 2026-03-18. [S]
- [x] #315 `plugin`: documented parseGoMapString space limitation. Closed 2026-03-18. [S]
- [x] #325 `docs`: fixed doc/code mismatches across packages. Closed 2026-03-17. [S]
- [x] #314 `router`: restored context propagation for child processes. Closed 2026-03-17. [M]
- [x] #294 `plugin`: LaunchTemplate no longer priced as EC2. Closed 2026-03-16. [S]
- [x] #293 `plugin`: GetActualCost handles zero-cost resources. Closed 2026-03-16. [S]
- [x] #289 `lint`: resolved all findings, protected .golangci.yml. Closed 2026-03-16. [M]
- [x] #245 `router`: single-port multi-region router with fan-out. Closed 2026-03-02. [L]
- [x] `router`: router binary entrypoint with eager region warm-up. [M]
- [x] #157 `plugin`: memoized service type resolution. Closed 2026-01-24. [S]
- [x] #274 `plugin`: IAM resources handled as zero-cost. Closed 2026-01-19. [S]
- [x] #273 `region`: us-west-1 (N. California) region support. Closed 2026-01-19. [M]
- [x] #257 `plugin`: carbon metrics advertised per service. Closed 2026-01-18. [S]
- [x] #237 `plugin`: zero-cost VPC, Security Group, Subnet handling. Closed 2026-01-18. [M]
- [x] #244 `docker`: multi-region Docker image with all regions. Closed 2026-01-16. [L]
- [x] #243 `config`: CORS configuration via environment variables. Closed 2026-01-14. [S]
- [x] #239 `repo`: renamed to finfocus-plugin-aws-public. Closed 2026-01-13. [M]
- [x] #209 `plugin`: dev mode usage-profile heuristics. Closed 2026-01-12. [M]
- [x] #208 `plugin`: resource topology lineage linking. Closed 2026-01-12. [M]
- [x] #207 `plugin`: storage growth-type heuristics. Closed 2026-01-12. [S]
- [x] #228 `pricing`: go-json parsing for faster initialization. Closed 2026-01-04. [M]
- [x] #176 `pricing`: pre-allocated pricing index map capacity. Closed 2026-01-04. [S]
- [x] #160 `recommendations`: configurable maxBatchSize. Closed 2026-01-03. [S]
- [x] #156 `recommendations`: optional strict validation mode. Closed 2026-01-03. [S]

### 2025-Q4

- [x] #196 `plugin`: runtime-based GetActualCost with 730-hour fallback. Closed 2025-12-31. [L]
- [x] #198 `recommendations`: ResourceID pass-through in responses. Closed 2025-12-26. [S]
- [x] #171 `pricing`: per-service raw JSON embedding. Closed 2025-12-21. [L]
- [x] #170 `pricing`: per-service embedding refactor (companion issue). Closed 2025-12-21. [M]
- [x] `plugin`: FOCUS 1.2 cost records and standardized pricing specs. [M]

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
