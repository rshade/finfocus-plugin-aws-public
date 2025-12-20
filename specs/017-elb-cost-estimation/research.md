# Research: ELB Pricing Data Structure

## Research Tasks

1. **Verify AWSELB pricing schema**: Confirm how fixed rates and LCU/NLCU rates are labeled in the Pricing API.
2. **Tag extraction pattern**: Review how other services (like Lambda or EBS) extract tags in `projected.go`.

## Findings

### 1. Pricing Schema (AWSELB)
- **ServiceCode**: `AWSELB`
- **ALB (Application Load Balancer)**:
  - Fixed Hourly: `productFamily: "Load Balancer"`, `group: "ELB:Application"`, `usageType: "LoadBalancerUsage"`, `operation: "LoadBalancer"`
  - LCU: `productFamily: "Load Balancer"`, `group: "ELB:Application"`, `usageType: "LCUUsage"`, `operation: "LoadBalancer"`
- **NLB (Network Load Balancer)**:
  - Fixed Hourly: `productFamily: "Load Balancer"`, `group: "ELB:Network"`, `usageType: "LoadBalancerUsage"`, `operation: "LoadBalancer"`
  - NLCU: `productFamily: "Load Balancer"`, `group: "ELB:Network"`, `usageType: "NLCUUsage"`, `operation: "LoadBalancer"`

### 2. Tag Extraction Pattern
- Existing services use `resource.Tags` map.
- `EBS` example: `strconv.Atoi(resource.Tags["size"])`.
- `Lambda` example: `strconv.Atoi(resource.Tags["memory"])`.
- ELB will follow this pattern for `lcu_per_hour`, `nlcu_per_hour`, and `capacity_units`.

## Decisions

- **Decision**: Use `group` attribute ("ELB:Application" vs "ELB:Network") to distinguish ALB/NLB in the pricing parser.
- **Rationale**: This is the most consistent way to filter these products in the AWSELB offer file.
- **Alternatives Considered**: Filtering by `usageType` substring, but `group` is more explicit.

- **Decision**: Default to ALB if SKU is missing.
- **Rationale**: Per user clarification, ALB is the primary use case.

## Verification
- Checked `tools/generate-pricing/main.go` for existing service codes. `AWSELB` needs to be added to the list of fetched services.