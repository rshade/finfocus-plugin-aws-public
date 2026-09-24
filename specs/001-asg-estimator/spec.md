# Feature Specification: ASG Cost Estimator

**Feature Branch**: `001-asg-estimator`
**Created**: 2026-03-26
**Status**: Draft
**Input**: GitHub Issue #295 — feat: Add ASG estimator for aws:autoscaling/group:Group

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Basic ASG Cost Estimation (Priority: P1)

A FinFocus user has an AWS stack containing Auto Scaling Groups. When they run
cost estimation, the plugin should return a meaningful cost for each ASG based
on the desired number of instances and the instance type configured in the
associated launch template or launch configuration. Currently, ASGs return
"no cost data available," leaving a significant gap in total infrastructure
cost visibility — especially for EKS-managed node groups that are backed by
ASGs.

**Why this priority**: ASGs are where real EC2 compute costs live. Without this
estimator, users undercount their infrastructure costs by the entire compute
spend of their auto-scaled workloads.

**Independent Test**: Can be fully tested by sending a `GetProjectedCost` RPC
with `resource_type: "aws:autoscaling/group:Group"`, an instance type in
tags, and verifying the response contains a non-zero `cost_per_month` equal
to (EC2 hourly rate × desired capacity × 730 hours).

**Acceptance Scenarios**:

1. **Given** a ResourceDescriptor with `resource_type:
   "aws:autoscaling/group:Group"` and tags `instance_type: "t3.medium"`,
   `desired_capacity: "3"`, **When** `GetProjectedCost` is called, **Then**
   the response returns `cost_per_month` equal to the t3.medium hourly rate ×
   3 × 730, with `billing_detail` describing the assumptions.
2. **Given** a ResourceDescriptor with `resource_type: "asg"` and `sku:
   "m5.large"` and no capacity tags, **When** `GetProjectedCost` is called,
   **Then** the response defaults `desired_capacity` to 1 and returns the
   cost for a single m5.large instance, with metadata indicating the default
   was applied.
3. **Given** a ResourceDescriptor with `resource_type:
   "aws:autoscaling/group:Group"` and a region that does not match the
   plugin binary's embedded region, **When** `Supports` is called, **Then**
   the response returns `supported: false` with reason "Region not supported
   by this binary."

---

### User Story 2 — Tag-Based Instance Type Resolution (Priority: P2)

When Pulumi state is used as input, the ASG resource's tags may contain
instance type information in several forms: directly as `instance_type`, or
nested within `launch_template` or `launch_configuration` properties
(serialized as tag map entries). The estimator should resolve the instance
type from these sources using a priority-based lookup.

**Why this priority**: Real-world Pulumi stacks express instance types through
launch template references, not directly on the ASG. Without this resolution
logic, the estimator would fail on most real stacks.

**Independent Test**: Can be tested by sending `GetProjectedCost` with various
tag configurations (direct `instance_type`, `launch_template.instance_type`,
`sku` field) and verifying the correct instance type is resolved in each case.

**Acceptance Scenarios**:

1. **Given** tags containing `instance_type: "c5.xlarge"`, **When**
   `GetProjectedCost` is called, **Then** the instance type "c5.xlarge" is
   used for pricing lookup.
2. **Given** tags containing `launch_template.instance_type: "r5.2xlarge"` but
   no direct `instance_type` tag, **When** `GetProjectedCost` is called,
   **Then** the instance type "r5.2xlarge" is used for pricing lookup.
3. **Given** `sku: "t3.small"` on the ResourceDescriptor and no instance type
   tags, **When** `GetProjectedCost` is called, **Then** "t3.small" is used
   for pricing lookup.
4. **Given** no instance type in `sku` or tags, **When** `GetProjectedCost`
   is called, **Then** the response returns an appropriate error indicating
   that instance type could not be determined.

---

### User Story 3 — Capacity Tag Extraction with Fallback (Priority: P2)

The estimator should extract the number of instances from `desired_capacity`
(preferred), falling back to `min_size` if `desired_capacity` is absent. This
accommodates both Pulumi state exports (which may include `desiredCapacity`
or `desired_capacity`) and manual ResourceDescriptor construction.

**Why this priority**: Accurate instance count is critical for cost accuracy.
The fallback chain ensures reasonable estimates even with incomplete data.

**Independent Test**: Can be tested by varying capacity tags and verifying
the correct multiplier is applied to the per-instance cost.

**Acceptance Scenarios**:

1. **Given** tags `desired_capacity: "5"`, **When** `GetProjectedCost` is
   called, **Then** the cost is calculated for 5 instances.
2. **Given** tags `min_size: "2"` with no `desired_capacity`, **When**
   `GetProjectedCost` is called, **Then** the cost is calculated for 2
   instances with metadata indicating `desired_capacity` was defaulted from
   `min_size`.
3. **Given** no capacity tags at all, **When** `GetProjectedCost` is called,
   **Then** the cost is calculated for 1 instance with metadata indicating
   the default was applied.

---

### User Story 4 — Carbon Footprint Estimation (Priority: P3)

The ASG estimator should provide carbon footprint estimates by delegating
to the existing EC2 carbon estimator, multiplied by the number of instances.
This follows the same delegation pattern used by ElastiCache.

**Why this priority**: Carbon estimation is a differentiating feature of
FinFocus. Since ASGs manage EC2 instances, the carbon calculation is
straightforward — delegate to EC2 carbon per instance × count.

**Independent Test**: Can be tested by verifying that `ImpactMetrics` in the
response contains a `METRIC_KIND_CARBON_FOOTPRINT` entry with a value equal
to (single-instance carbon × desired capacity).

**Acceptance Scenarios**:

1. **Given** a ResourceDescriptor for an ASG with `instance_type: "m5.large"`
   and `desired_capacity: "3"` in a region with known grid emission factors,
   **When** `GetProjectedCost` is called, **Then** the response includes
   `ImpactMetrics` with carbon footprint equal to 3× the m5.large carbon
   estimate.
2. **Given** an ASG in a region without grid emission data, **When**
   `GetProjectedCost` is called, **Then** the response omits carbon metrics
   gracefully (no error).

---

### User Story 5 — Supports and Service Discovery (Priority: P2)

The plugin should correctly advertise support for ASG resources through the
`Supports` RPC, including both Pulumi-format
(`aws:autoscaling/group:Group`) and short-form (`asg`, `autoscaling`)
resource types.

**Why this priority**: Without proper `Supports` registration, FinFocus Core
will not route ASG resources to this plugin.

**Independent Test**: Can be tested by calling `Supports` with various ASG
resource type formats and verifying `supported: true` with appropriate
metrics advertisement.

**Acceptance Scenarios**:

1. **Given** `resource_type: "aws:autoscaling/group:Group"` with a matching
   region, **When** `Supports` is called, **Then** the response returns
   `supported: true` with carbon footprint in `supported_metrics`.
2. **Given** `resource_type: "asg"` with a matching region, **When**
   `Supports` is called, **Then** the response returns `supported: true`.

---

### Edge Cases

- What happens when `desired_capacity` is 0? The estimator returns $0 cost
  with billing detail noting zero instances.
- What happens when the instance type is not found in embedded pricing data?
  The estimator returns a `PricingUnavailableError` with a clear message.
- What happens when `desired_capacity` is negative or non-numeric? The
  estimator uses the validation helper pattern to log a warning and default
  to 1.
- What happens when both `sku` and `instance_type` tag are present but
  differ? The `sku` field takes priority (consistent with EC2 estimator
  behavior).
- What happens when `mixed_instances_policy` tags are present? For v1, the
  estimator uses the primary instance type only and notes in `billing_detail`
  that mixed instance policies are not yet supported.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST estimate ASG costs by multiplying per-instance EC2
  on-demand pricing by the desired instance count.
- **FR-002**: System MUST resolve instance type from ResourceDescriptor using
  priority: `sku` field → `instance_type` tag → `launch_template.instance_type`
  tag → `launch_configuration.instance_type` tag.
- **FR-003**: System MUST resolve instance count from tags using priority:
  `desired_capacity`/`desiredCapacity` → `min_size`/`minSize` → default of 1.
- **FR-004**: System MUST support Pulumi-format resource type
  `aws:autoscaling/group:Group` and short-form types `asg` and `autoscaling`.
- **FR-005**: System MUST return `PricingUnavailableError` when instance type
  cannot be determined from any source.
- **FR-006**: System MUST track all applied defaults (capacity, instance type
  source) in response `Metadata` using the `DefaultsTracker` pattern.
- **FR-007**: System MUST include carbon footprint estimation in
  `ImpactMetrics` by delegating to the EC2 carbon estimator, scaled by
  instance count.
- **FR-008**: System MUST advertise `METRIC_KIND_CARBON_FOOTPRINT` in
  `Supports` response for ASG resources.
- **FR-009**: System MUST classify ASG with `GROWTH_TYPE_STATIC` growth hint
  and `AffectedByDevMode: true` in service classifications.
- **FR-010**: System MUST handle zero `desired_capacity` gracefully, returning
  $0 cost without error.
- **FR-011**: System MUST include `billing_detail` describing assumptions:
  instance type, count, on-demand pricing, 730 hours/month.
- **FR-012**: System MUST handle ASG resources in `GetActualCost`,
  `GetPricingSpec`, and `Supports` RPCs consistently (two-step normalization
  pattern).

### Key Entities

- **Auto Scaling Group**: A resource that manages a fleet of EC2 instances.
  Key attributes: desired capacity, min/max size, instance type (via launch
  template/configuration reference). Has no direct AWS charge — cost is
  represented by the aggregate of managed instances.
- **Launch Template / Launch Configuration**: Configuration-only resources
  (already handled as zero-cost) that define the instance type, AMI, and
  other parameters for instances launched by an ASG.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: ASG resources return accurate cost estimates matching the
  formula: EC2 on-demand hourly rate × desired capacity × 730, with less
  than 1% deviation from manual calculation.
- **SC-002**: All supported resource type formats
  (`aws:autoscaling/group:Group`, `asg`, `autoscaling`) produce identical
  cost estimates for the same configuration.
- **SC-003**: Users see ASG costs reflected in total infrastructure estimates
  instead of "no cost data available."
- **SC-004**: Carbon footprint estimates for ASGs are proportional to instance
  count and consistent with standalone EC2 carbon estimates for the same
  instance type.
- **SC-005**: Default tracking metadata enables downstream consumers to
  distinguish "real configuration" from "estimated defaults" for cost diff
  accuracy.
- **SC-006**: Plugin initialization and ASG cost estimation complete within
  existing performance budgets (startup < 1s, RPC < 100ms).

## Assumptions

- **A-001**: ASG costs are estimated using on-demand pricing only. Spot
  instances, mixed instance policies, and capacity reservations are out of
  scope for v1.
- **A-002**: The ASG estimator does not need separate pricing data from AWS.
  It reuses the existing EC2 pricing data already embedded in the binary.
- **A-003**: When no instance type can be resolved, the estimator returns an
  error rather than guessing a default instance type — there is no
  reasonable universal default.
- **A-004**: `desiredCapacity` represents the steady-state instance count for
  cost estimation purposes, even though actual ASG scaling may vary.
- **A-005**: The `operating_system` tag (if present) is passed through to
  EC2 pricing lookup; otherwise defaults to Linux (consistent with EC2
  estimator).
- **A-006**: Root EBS volume costs are NOT included in ASG estimates to avoid
  double-counting — users should estimate EBS volumes separately if needed.

## Scope Boundaries

### In Scope

- ASG cost estimation via EC2 pricing delegation
- Instance type resolution from multiple tag sources
- Capacity extraction with fallback chain
- Carbon footprint estimation
- Service registration (Supports, normalization, classification)
- DefaultsTracker metadata for applied defaults
- All RPC paths (GetProjectedCost, GetActualCost, GetPricingSpec, Supports)

### Out of Scope

- Spot instance pricing or mixed instance policies
- Auto-scaling behavior simulation (scaling policies, scheduled actions)
- Correlation with CloudWatch metrics for actual utilization
- EBS volume costs for ASG-managed instances (separate resource)
- Network costs (data transfer, NAT Gateway) for ASG instances
- GetRecommendations RPC path (ASG-specific recommendations are out of scope
  for v1; batch processing logic does not require ASG awareness)
