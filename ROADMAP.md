# Strategic Roadmap: pulumicost-plugin-aws-public

## Mission Statement

To provide the most comprehensive, air-gapped cost and carbon estimation engine
for AWS, enabling continuous governance and pre-deployment planning without the
security overhead of cloud credentials.

---

## Past Milestones [Done]

- **Core Infrastructure:** gRPC `CostSourceService` implementation, regional
  build matrix (9+ regions), and `zerolog` trace propagation.
- **Compute:** EC2 On-Demand cost estimation and CCF-based Carbon Footprint
  (gCO2e) metrics.
- **Storage:** EBS (Basic Storage), S3 (Standard Storage), and basic Lambda
  estimation.
- **Managed Services:** EKS Control Plane, DynamoDB (On-Demand/Provisioned), and
  ELB (ALB/NLB with LCU/NLCU support).
- **Optimization:** `GetRecommendations` batch processing for
  `target_resources` (up to 100 items).
- **Architecture:** Transition to per-service raw JSON embedding to manage
  binary size and initialization speed.

---

## Immediate Focus [In Progress / Planned]

- **[In Progress] RDS Implementation:** Move RDS from stub to full support
  (Instance types, Multi-AZ flags, and Storage).
- **[Planned] Documentation Alignment:** Update `README.md` and `CLAUDE.md` to
  reflect that S3, EKS, DynamoDB, and ELB are fully implemented and no longer
  stubs.
- **[Planned] Refined "Actual Cost" Logic:** Enhance `GetActualCost` to
  intelligently prioritize usage hours from request metadata, defaulting to
  730-hour monthly projections only when usage is absent.
- **[Planned] Service Breadth Expansion:**
  - **ElastiCache:** Node type and engine-based pricing.
  - **Route53:** Hosted zones and basic query volume estimation.
  - **CloudFront:** Basic data transfer and request pricing (based on regional
    estimates).

---

## Future Vision [Researching / Planned]

- **[Researching] Memory Optimization:** Implementing lazy-loading or
  memory-mapped access for embedded JSON files to reduce the runtime memory
  footprint without moving to an external database.
- **[Planned] Service Depth (Phase 2):**
  - **EBS Depth:** Adding IOPS and Throughput pricing for `gp3`, `io1`, and
    `io2`.
  - **EC2 Depth:** Integrating GPU-specific carbon coefficients as data becomes
    available in CCF.
- **[Researching] Cross-Service Recommendations:** Static lookup logic to
  suggest move-to-managed alternatives (e.g., self-managed DB on EC2 -> RDS)
  based on Resource Tags.
- **[Planned] Additional Regions:** Expansion to GovCloud (US-West/East) and
  specialized regions (e.g., Beijing/Ningxia) as public pricing data parity
  allows.
- **[Planned] Forecasting Intelligence:**
  - **Growth Hints:** Implement logic to return `GrowthType` (Linear) for
    accumulation-based resources (S3, ECR, Backup) to support Core forecasting.
- **[Planned] Topology Awareness:**
  - **Lineage Metadata:** Populate `ParentResourceID` for dependent resources
    (e.g., EBS Volumes attached to Instances, NAT Gateways attached to VPCs) to
    support "Blast Radius" visualization.

---

## Strategic Guardrails (From CONTEXT.md)

1. **Statelessness:** No local databases or historical trend storage. Data
   "intelligence" (comparisons) belongs in PulumiCost Core.
2. **Air-Gapped:** Zero runtime network calls. All estimates derived from
   build-time snapshots.
3. **Static Logic:** Recommendations are based on static mappings and SKU
   attributes, never on live monitoring or external telemetry.
