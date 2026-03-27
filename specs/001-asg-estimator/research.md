# Research: ASG Cost Estimator

**Feature**: 001-asg-estimator | **Date**: 2026-03-26

## R1: ASG Pricing Model

**Decision**: ASG cost = EC2 on-demand hourly rate × desired_capacity × 730
hours/month. No separate ASG pricing data needed.

**Rationale**: AWS Auto Scaling Groups have no direct charge. The cost is
entirely from the EC2 instances the ASG manages. The plugin already embeds
full EC2 pricing data, so the ASG estimator simply performs a lookup in the
existing EC2 price index and multiplies by instance count.

**Alternatives considered**:
- Fetch separate ASG pricing from AWS API: Rejected — ASGs have no separate
  pricing SKU in the AWS Price List API.
- Include EBS root volume cost per instance: Rejected — risks double-counting
  with separately tracked EBS volumes. The EC2 estimator already has
  optional root EBS support via tags, but for ASGs this should be
  opt-in via the individual EC2 resource estimates.

## R2: Instance Type Resolution Strategy

**Decision**: Priority-based resolution: `sku` → `instance_type` tag →
`launch_template.instance_type` → `launch_configuration.instance_type`.
Error if none found.

**Rationale**: In real Pulumi stacks, the ASG resource's properties are
flattened into the tags map. The instance type may appear in several forms:
- Direct `instance_type` tag (when Pulumi serializes the ASG's
  `mixedInstancesPolicy.launchTemplate.overrides[0].instanceType`)
- Nested `launch_template.instance_type` (when the launch template
  properties are flattened with dot notation)
- The `sku` field on ResourceDescriptor (when FinFocus Core pre-resolves
  the instance type)

No reasonable default exists for instance type (unlike capacity, where 1
is a safe default). A `t3.micro` default would massively underestimate
costs for production ASGs.

**Alternatives considered**:
- Default to `t3.micro` when unknown: Rejected — silently wrong is worse
  than explicit error.
- Require `sku` field only: Rejected — Pulumi state exports don't
  populate `sku` for ASG resources.

## R3: Capacity Extraction

**Decision**: `desired_capacity` → `desiredCapacity` (camelCase variant) →
`min_size` → `minSize` → default 1.

**Rationale**: Pulumi state exports use camelCase (`desiredCapacity`), while
manual/CLI usage may use snake_case (`desired_capacity`). The fallback to
`min_size` is reasonable because when `desired_capacity` is absent, `min_size`
represents the minimum guaranteed fleet size. Default of 1 is the safest
assumption when no capacity information is available — it represents the
smallest possible ASG.

**Alternatives considered**:
- Use `max_size` as fallback: Rejected — would overestimate costs.
- Error when no capacity: Rejected — unlike instance type, a capacity
  default of 1 is reasonable and doesn't silently mislead users.

## R4: Carbon Estimation Approach

**Decision**: Delegate to existing EC2 `Estimator.EstimateCarbonGrams()`,
multiply result by `desired_capacity`. Create an `ASGEstimator` struct
following the ElastiCache delegation pattern.

**Rationale**: ASG-managed instances are standard EC2 instances. The
ElastiCache carbon estimator already demonstrates this delegation pattern
(mapping `cache.m5.large` → `m5.large`, then delegating to EC2). The ASG
case is even simpler — the instance type is already in EC2 format, so no
type mapping is needed.

**Alternatives considered**:
- Inline carbon calculation in the estimator: Rejected — would duplicate
  the EC2 carbon logic and risk drift.
- Skip carbon for v1: Rejected — the delegation is trivial and carbon is
  a differentiating feature.

## R5: Service Normalization and ARN Handling

**Decision**: Add `serviceASG = "asg"` constant. Normalize
`aws:autoscaling/group:Group` → `"asg"` in `normalizeResourceType()`. Update
ARN parsing to distinguish ASG from LaunchConfiguration under the
`autoscaling` service.

**Rationale**: The existing ARN parser already handles the `autoscaling`
service but only distinguishes LaunchConfigurations. ASG ARNs have
`resourceType: "autoScalingGroup"`, which currently falls through to
returning the raw service name `"autoscaling"`. This needs to map to the
new `serviceASG` constant.

**Alternatives considered**:
- Reuse `"autoscaling"` as the service constant: Rejected — ambiguous with
  LaunchConfigurations which are also under the autoscaling service.

## R6: GetPricingSpec Response

**Decision**: Return a PricingSpec response showing the per-instance EC2
rate with `BillingMode: "on_demand"`, `RateUnit: "per-instance-hour"`,
and a `notes` field describing the multiplication by desired capacity.

**Rationale**: Follows the pattern of other estimators (EKS, ELB) that
return the base unit rate in PricingSpec and describe the total calculation
in notes. This gives consumers enough information to understand how the
cost was derived without reimplementing the calculation.

## R7: Classification and Growth Type

**Decision**: `GROWTH_TYPE_STATIC` with `AffectedByDevMode: true`.
Parent tag keys: `["vpc_id"]` with relationship `RelationshipWithin`.

**Rationale**: ASG costs are fixed based on desired capacity (not
accumulating like S3). Dev mode should apply the 160-hour reduction since
ASG instances can be scaled down in non-production environments. The VPC
parent relationship matches the ElastiCache and RDS patterns.
