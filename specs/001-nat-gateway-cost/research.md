# Research: VPC NAT Gateway Pricing Structure

## Decision: Pricing Source
**Decision**: Use `AmazonVPC` service code from AWS Pricing API.
**Rationale**: NAT Gateway pricing is published under the `AmazonVPC` service family, distinct from EC2 or ELB.

## Decision: Pricing Attributes
**Decision**: Filter products by `productFamily: "NAT Gateway"` and use `usagetype` to distinguish rates.
**Rationale**:
- **Hourly Rate**: `usagetype` containing `NatGateway-Hours`. Unit: `Hrs`.
- **Data Processing Rate**: `usagetype` containing `NatGateway-Bytes`. Unit: `Quantity`.
**Verification**: Confirmed via AWS Pricing API documentation and existing community terraform providers/pricing tools.

## Alternatives Considered
- **Sourcing from EC2 pricing**: Rejected because NAT Gateways are VPC resources, not EC2 compute instances.
- **Hardcoding values**: Rejected as it violates the project's principle of using real regional data fetched at build time.

## Dependency Assessment
- **generate-pricing tool**: Needs update to include `AmazonVPC`.
- **embed_* files**: Need update to include the new `vpc` service files.
- **PricingClient**: Interface needs two new methods for NAT Gateway.
