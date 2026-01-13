# Feature Specification: Core Protocol Intelligence

**Feature Branch**: `030-core-protocol-intelligence`
**Created**: 2026-01-04
**Status**: Draft
**Input**: Consolidate #207 (Growth Heuristics), #208 (Topology Linking), #209 (Dev Mode) into response enrichment feature

## Overview

Enrich cost estimation responses with metadata that enables FinFocus Core's advanced features: Cost Time Machine forecasting, Blast Radius topology visualization, and Dev Mode realistic estimates. This is a protocol enhancement that adds intelligence to existing cost estimates without changing pricing calculations.

## Clarifications

### Session 2026-01-04

- Q: For Dev Mode (160 hrs/month), how should Lambda and DynamoDB costs be calculated? → A: No reduction - treat as usage-based like S3 storage. These are pay-per-use services where "hours running" doesn't apply.
- Q: What should be the maximum acceptable response time for a single resource cost estimation request? → A: < 100ms - Fast response for interactive dev/test scenarios
- Q: For the BURST UsageProfile enum value, what behavior should it implement? → A: 730 hours (same as PRODUCTION) - no special handling
- Q: What logging level and details should be used when dev mode, growth hints, or lineage fields are added to responses? → A: INFO level with structured fields for UsageProfile, GrowthType, parent detection
- Q: How should the plugin detect and handle protocol field availability when the upstream finfocus-spec version varies? → A: Feature detection - check if protocol fields exist at runtime, omit if absent

### Consolidated GitHub Issues

| Issue | Title | Core Feature Enabled |
|-------|-------|---------------------|
| [#207](https://github.com/rshade/finfocus-plugin-aws-public/issues/207) | Storage Growth Heuristics | Cost Time Machine forecasting |
| [#208](https://github.com/rshade/finfocus-plugin-aws-public/issues/208) | Resource Topology Linking | Blast Radius visualization |
| [#209](https://github.com/rshade/finfocus-plugin-aws-public/issues/209) | Dev Mode Heuristics | Realistic dev/test estimates |

### External Dependency

**CRITICAL**: All three features require protocol changes in the upstream specification repository. Before implementation:

1. Define `GrowthType` enum and `growth_hint` field (#207)
2. Define `parent_resource_id` and `CostAllocationLineage` message (#208)
3. Define `UsageProfile` enum and request field (#209)

**Dependency**: rshade/finfocus-spec (version TBD with protocol extensions)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Dev Mode Cost Estimates (Priority: P1)

As a developer estimating costs for a dev/test environment, I want the plugin to use realistic dev assumptions (160 hours/month instead of 730) so that I get accurate estimates for environments that aren't running continuously.

**Why this priority**: Highest immediate user value - dev environments are significantly overestimated today (~4.5x too high), causing budget confusion and poor planning decisions.

**Independent Test**: Can be fully tested by sending a request with `UsageProfile=DEVELOPMENT` and verifying EC2 cost is ~22% of production estimate (160/730).

**Acceptance Scenarios**:

1. **Given** EC2 t3.medium request with `UsageProfile=DEVELOPMENT`, **When** cost estimation is called, **Then** cost is ~$6.66/month (not $30.37) and billing detail includes "(dev profile)"
2. **Given** request with no UsageProfile specified, **When** cost estimation is called, **Then** default 730 hours behavior unchanged (backward compatible)
3. **Given** S3 storage request with `UsageProfile=DEVELOPMENT`, **When** cost estimation is called, **Then** storage cost unchanged (storage is not time-based)

---

### User Story 2 - Growth Type Hints for Forecasting (Priority: P2)

As a platform engineer using FinFocus's forecasting feature, I want the plugin to indicate which resources naturally accumulate data over time so that Core can apply appropriate growth models to project future costs accurately.

**Why this priority**: Enables "Cost Time Machine" - a key differentiating feature. Without growth hints, Core cannot distinguish S3 (grows linearly) from EC2 (static cost).

**Independent Test**: Can be fully tested by calling cost estimation for S3 and verifying `growth_hint=GROWTH_TYPE_LINEAR`, then for EC2 and verifying `growth_hint=GROWTH_TYPE_STATIC`.

**Acceptance Scenarios**:

1. **Given** S3 bucket request, **When** cost estimation is called, **Then** response includes `growth_hint=GROWTH_TYPE_LINEAR`
2. **Given** DynamoDB table request, **When** cost estimation is called, **Then** response includes `growth_hint=GROWTH_TYPE_LINEAR`
3. **Given** EC2 instance request, **When** cost estimation is called, **Then** response includes `growth_hint=GROWTH_TYPE_STATIC`
4. **Given** EKS cluster request, **When** cost estimation is called, **Then** response includes `growth_hint=GROWTH_TYPE_STATIC`

---

### User Story 3 - Resource Topology Linking (Priority: P3)

As a platform engineer reviewing infrastructure changes, I want the plugin to identify parent/child relationships between resources so that Core can display an "Impact Tree" showing which resources are affected when a parent changes.

**Why this priority**: Enables "Blast Radius" visualization. Lower priority because it requires the upstream resource mapper to provide parent IDs in tags - plugin can only surface what it receives.

**Independent Test**: Can be fully tested by sending EBS volume request with `tags["instance_id"]="i-abc123"` and verifying response includes `parent_resource_id="i-abc123"`.

**Acceptance Scenarios**:

1. **Given** EBS volume with `tags["instance_id"]="i-abc123"`, **When** cost estimation is called, **Then** response includes `parent_resource_id="i-abc123"` and `lineage.relationship="attached_to"`
2. **Given** NAT Gateway with `tags["vpc_id"]="vpc-xyz"`, **When** cost estimation is called, **Then** response includes `parent_resource_id="vpc-xyz"` and `lineage.relationship="within"`
3. **Given** EBS volume without instance_id tag, **When** cost estimation is called, **Then** response omits parent_resource_id (no error)

---

### Edge Cases

- What happens when protocol fields are missing in older spec versions? Feature detection at runtime - omit new fields gracefully (no version checks)
- How does system handle invalid UsageProfile enum values? Treat as UNSPECIFIED, use production defaults
- What if parent tag contains malformed ID? Include as-is, Core validates
- How to handle resources with multiple parents (EBS in ASG)? Use primary parent only (instance_id takes precedence)
- BURST UsageProfile behavior: Same as PRODUCTION (730 hrs/month), no special cost adjustment

## Requirements *(mandatory)*

### Functional Requirements

**Dev Mode (#209)**:

- **FR-001**: System MUST reduce hourly-based costs to 160 hrs/month when `UsageProfile=DEVELOPMENT`
- **FR-002**: System MUST NOT modify usage-based costs (S3 storage, EBS storage, Lambda requests/compute, DynamoDB throughput, CloudWatch ingestion) for dev profile
- **FR-003**: System MUST include "(dev profile)" in billing detail when dev mode active
- **FR-004**: System MUST default to 730 hrs/month when UsageProfile is unspecified or BURST

**Growth Hints (#207)**:

- **FR-005**: System MUST return `GROWTH_TYPE_LINEAR` for S3 and DynamoDB
- **FR-006**: System MUST return `GROWTH_TYPE_STATIC` for EC2, EBS, EKS, Lambda, ELB, NAT Gateway, CloudWatch, ElastiCache, RDS
- **FR-007**: System MUST omit growth_hint (or return UNSPECIFIED) if protocol field unavailable

**Topology Linking (#208)**:

- **FR-008**: System MUST extract `parent_resource_id` from known tag keys (`instance_id`, `vpc_id`, `subnet_id`, `cluster_name`)
- **FR-009**: System MUST populate `lineage.relationship` with appropriate value (`attached_to`, `within`, `managed_by`)
- **FR-010**: System MUST gracefully omit lineage fields when parent tags are absent

### Non-Functional Requirements

**Protocol Compatibility**:

- **NFR-001**: System MUST use feature detection (runtime field checks) instead of version-specific logic for protocol fields
- **NFR-002**: System MUST gracefully omit new protocol fields when unavailable in upstream spec (no errors)

**Observability**:

- **NFR-003**: System MUST log at INFO level with structured fields when dev mode, growth hints, or lineage metadata is added to responses
- **NFR-004**: Structured log fields MUST include: `usage_profile`, `growth_type`, `parent_detected`, `parent_type` when applicable

### Key Entities

- **UsageProfile**: Enum indicating operational context (PRODUCTION, DEVELOPMENT, BURST)
- **GrowthType**: Enum indicating cost growth pattern (STATIC, LINEAR, EXPONENTIAL)
- **CostAllocationLineage**: Message containing parent_resource_id, parent_resource_type, relationship

### Service Classifications

| Service | GrowthType | Affected by Dev Mode | Parent Tag Keys |
|---------|------------|---------------------|-----------------|
| EC2 | STATIC | Yes (hours) | - |
| EBS | STATIC | No (storage) | `instance_id` |
| EKS | STATIC | Yes (hours) | - |
| S3 | LINEAR | No (storage) | - |
| Lambda | STATIC | No (usage-based) | - |
| DynamoDB | LINEAR | No (usage-based) | - |
| ELB | STATIC | Yes (hours) | `vpc_id` |
| NAT Gateway | STATIC | Yes (hours) | `vpc_id`, `subnet_id` |
| CloudWatch | STATIC | No (ingestion) | - |
| ElastiCache | STATIC | Yes (hours) | `vpc_id` |
| RDS | STATIC | Yes (hours) | `vpc_id` |

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Dev mode estimates are ~22% of production (160/730 ratio) for hourly-based services
- **SC-002**: 100% of implemented services return appropriate GrowthType enum value
- **SC-003**: Parent relationships detected for 100% of resources that include known parent tags
- **SC-004**: Zero breaking changes to existing cost calculations (backward compatible)
- **SC-005**: All new protocol fields gracefully omitted when upstream spec lacks definitions
- **SC-006**: Single resource cost estimation completes in < 100ms including dev mode, growth hints, and lineage extraction

## Assumptions

- Dev mode hours (160/month) assumes standard 8-hour workday, 5 days/week, 4 weeks/month
- Growth type classification is static per service (not configurable per resource)
- Parent tag extraction uses exact key matching (case-sensitive)
- Lineage relationship types are a fixed set: `attached_to`, `within`, `managed_by`

## Out of Scope

- **Growth rate calculation**: Future phase requiring historical data
- **Graph traversal**: Core handles multi-level dependency graphs
- **Cross-account relationships**: Single-account scope only
- **Dynamic AWS discovery**: Plugin relies on tags from upstream mapper, cannot query AWS

## Files Likely Affected

| File | Changes |
|------|---------|
| `go.mod` | Update finfocus-spec dependency |
| `internal/plugin/constants.go` | Add `hoursPerMonthDev = 160` |
| `internal/plugin/projected.go` | Add UsageProfile handling, GrowthType, lineage extraction |
| `internal/plugin/projected_test.go` | Tests for all three features |
| `CLAUDE.md` | Document growth classification, dev mode, topology relationships |

## Closes

- Closes #207
- Closes #208
- Closes #209
