# Research Findings: Add support for additional US regions (us-west-1, us-gov-west-1, us-gov-east-1)

## GovCloud Pricing Differences

**Decision:** GovCloud regions (us-gov-west-1, us-gov-east-1) require separate pricing data that differs from commercial regions

**Rationale:** AWS GovCloud (US) is physically and logically isolated from standard AWS regions, with pricing that reflects the specialized infrastructure and compliance requirements. Pricing data must be region-specific and embedded at build time, following the existing pattern for commercial regions.

**Alternatives considered:**
- Using commercial pricing for GovCloud regions (rejected: inaccurate cost estimates)
- Fetching pricing at runtime (rejected: violates embedded data principle and performance requirements)
- Single pricing dataset for all regions (rejected: GovCloud isolation and pricing differences)

## Build Tag Strategy

**Decision:** Use build tags `region_usw1`, `region_govw1`, `region_gove1` following existing naming convention

**Rationale:** Maintains consistency with current regions (us-east-1 = use1, us-west-2 = usw2), enables selective compilation of region-specific binaries.

**Alternatives considered:**
- Different naming scheme (rejected: breaks consistency)
- No build tags (rejected: would embed all pricing data, violating memory constraints)

## Pricing Data Sources

**Decision:** Generate pricing data using existing tools/generate-pricing/main.go with region-specific API calls

**Rationale:** Leverages existing infrastructure for pricing data generation, ensures data accuracy and consistency with current regions.

**Alternatives considered:**
- Manual pricing data entry (rejected: error-prone and maintenance intensive)
- Third-party pricing APIs (rejected: may not be reliable or up-to-date)