# Project Context: finfocus-plugin-aws-public

## Core Architectural Identity

The **finfocus-plugin-aws-public** is a stateless, air-gapped gRPC projection
engine. It serves as a specialized provider for the FinFocus ecosystem,
transforming resource descriptors into cost and carbon estimates. Its primary
design goal is to provide "good-enough" on-demand estimates without requiring
AWS credentials or network access at runtime.

## Technical Boundaries (Hard No's)

1. **No Runtime Network Dependencies:** The plugin MUST NOT make any outbound
   network calls (AWS APIs, Pricing APIs, etc.) during execution. All data
   required for estimation must be embedded in the binary at build time.
2. **No Credential Management:** This plugin does not handle AWS IAM roles,
   access keys, or STS tokens. It is intentionally decoupled from the user's AWS
   account security boundary.
3. **No Dynamic Resource Discovery:** The plugin does not "verify" if a resource
   exists in AWS. It purely calculates costs based on the `ResourceDescriptor`
   provided via gRPC.
4. **No Persistent State:** The plugin is completely ephemeral. It does not
   maintain a database, cache (beyond in-memory initialization), or local
   filesystem state.
5. **No Real-time Pricing:** It does not reflect Spot instance fluctuations,
   Savings Plans, or Reserved Instance discounts. It provides static On-Demand
   estimates based on the version of data embedded at build time.

## Data Source of Truth

- **Financial Data:** AWS Public Price List API. This data is fetched during the
  build process by `tools/generate-pricing`, filtered for On-Demand terms, and
  embedded via `//go:embed`.
- **Carbon Data:** Cloud Carbon Footprint (CCF) methodology. Power coefficients
  are sourced from the `cloud-carbon-coefficients` repository, stored in
  `ccf_instance_specs.csv`, and embedded at build time.
- **Regionality:** The source of truth is partitioned by AWS region. Each
  binary is compiled with a specific region's data subset using Go build tags
  (e.g., `region_use1`).

## Interaction Model

- **Protocol:** gRPC (implementing `finfocus.v1.CostSourceService`).
- **Lifecycle:** Orchestrated as a subprocess by FinFocus Core.
- **Discovery:** Announces its listening port via `stdout` (format:
  `PORT=XXXXX`).
- **Telemetry:** Structured JSON logging (via `zerolog`) sent exclusively to
  `stderr` to avoid corrupting the port discovery channel.
- **Input:** Receives `ResourceDescriptor` objects containing provider, type,
  SKU, region, and tags.
