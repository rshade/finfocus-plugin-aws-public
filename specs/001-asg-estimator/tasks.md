# Tasks: ASG Cost Estimator

**Input**: Design documents from `/specs/001-asg-estimator/`
**Prerequisites**: plan.md (required), spec.md (required), research.md,
data-model.md, contracts/

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: No project initialization needed — existing Go module. This
phase verifies the development environment is ready.

- [x] T001 Verify development environment with `make develop` and
  `make build-default-region`

---

## Phase 2: Foundational (Service Registration)

**Purpose**: Register ASG as a recognized service type across all
normalization and detection paths. MUST complete before any estimator work.

- [x] T002 Add `serviceASG = "asg"` constant in
  `internal/plugin/constants.go` alongside existing service constants
  (after `serviceLaunchConfig`)
- [x] T003 Add ASG normalization in `normalizeResourceType()` in
  `internal/plugin/projected.go` to map
  `aws:autoscaling/group:Group` → `serviceASG`
- [x] T004 Add ASG detection in `detectService()` in
  `internal/plugin/projected.go` to map `"asg"` and `"autoscaling"`
  short-forms → `serviceASG`
- [x] T005 [P] Update autoscaling ARN case in `ToPulumiResourceType()` in
  `internal/plugin/arn.go` to return `serviceASG` for
  `resourceType: "autoScalingGroup"` (existing code only handles
  `launchConfiguration`)

**Checkpoint**: `normalizeResourceType("aws:autoscaling/group:Group")`
returns `"asg"` and `detectService("asg")` returns `serviceASG`

---

## Phase 3: User Story 1 — Basic ASG Cost Estimation (Priority: P1) MVP

**Goal**: ASG resources return cost estimates based on EC2 pricing ×
desired capacity

**Independent Test**: Send `GetProjectedCost` with
`resource_type: "aws:autoscaling/group:Group"`, `sku: "t3.medium"`,
`desired_capacity: "3"` and verify `cost_per_month` equals t3.medium
hourly rate × 3 × 730

### Implementation for User Story 1

- [x] T006 [P] [US1] [US2] [US3] Create `internal/plugin/asg_attrs.go` with
  `ExtractASGAttributes()` function that returns `ASGAttributes` struct
  containing: `InstanceType` (string), `DesiredCapacity` (int), `OS`
  (string). Implement instance type resolution priority (US2): `sku` →
  `instance_type` tag → `launch_template.instance_type` tag →
  `launch_configuration.instance_type` tag. Implement capacity resolution
  (US3): `desired_capacity` → `desiredCapacity` → `min_size` → `minSize` →
  default 1. Track defaults via `DefaultsTracker`. Return error when no
  instance type found.
- [x] T007 [P] [US1] [US2] [US3] Add unit tests for
  `ExtractASGAttributes()` in `internal/plugin/asg_attrs_test.go` with
  table-driven tests covering: sku priority over tags, launch_template
  fallback, launch_configuration fallback, no instance type error (US2),
  desired_capacity extraction, min_size fallback, camelCase variants
  (desiredCapacity, minSize), default capacity of 1, zero capacity,
  negative/non-numeric capacity (US3), operating_system tag passthrough,
  default OS Linux
- [x] T008 [US1] Add `estimateASG()` method on `AWSPublicPlugin` in
  `internal/plugin/projected.go`. Function signature:
  `func (p *AWSPublicPlugin) estimateASG(traceID string, resource *pbc.ResourceDescriptor, req *pbc.GetProjectedCostRequest) (*pbc.GetProjectedCostResponse, error)`.
  Call `ExtractASGAttributes()`, look up EC2 pricing with
  `p.pricing.EC2OnDemandPricePerHour()`, calculate
  `cost = hourlyRate × desiredCapacity × 730`, build `billing_detail`
  string: `"ASG: N× <type> On-demand <OS>, 730 hrs/month"`, return
  `PricingUnavailableError` when instance type not found or not in pricing
  data. Handle zero capacity returning $0 gracefully. Include
  `DefaultsTracker` metadata in response. Call `setGrowthHint()` on
  response to apply growth type metadata (FR-009). If
  `mixed_instances_policy` tags are detected, note in `billing_detail`
  that mixed instance policies are not yet supported.
- [x] T009 [US1] Add `case serviceASG:` to the `GetProjectedCost` switch
  in `internal/plugin/projected.go` routing to
  `p.estimateASG(traceID, resource, req)`
- [x] T010 [US1] Add unit tests for `estimateASG()` in
  `internal/plugin/projected_test.go` with table-driven tests covering:
  basic cost calculation (verify rate × count × 730), zero capacity
  returns $0, missing instance type returns error, unknown instance type
  returns PricingUnavailableError, billing_detail format verification,
  metadata defaults tracking, all resource type formats produce same
  result

**Checkpoint**: `GetProjectedCost` with ASG resource type returns correct
cost. Run `go test ./internal/plugin/... -run TestASG` — all pass.

---

## Phase 4: User Story 5 — Supports and Service Discovery (Priority: P2)

**Goal**: Plugin advertises ASG support through Supports RPC, handles ASG
in GetPricingSpec, GetActualCost, and classification

**Independent Test**: Call `Supports` with
`resource_type: "aws:autoscaling/group:Group"` and verify
`supported: true` with carbon footprint in `supported_metrics`

### Implementation for User Story 5

- [x] T011 [P] [US5] Add ASG entry to `serviceClassifications` map in
  `internal/plugin/classification.go` with key
  `"aws:autoscaling:autoScalingGroup"`, `GrowthType:
  GROWTH_TYPE_STATIC`, `AffectedByDevMode: true`, `ParentTagKeys:
  ["vpc_id"]`, `ParentType: "aws:ec2:vpc:Vpc"`, `Relationship:
  RelationshipWithin`
- [x] T012 [P] [US5] Add `case serviceASG:` to the `Supports` switch in
  `internal/plugin/supports.go` routing to return `supported: true` with
  `getSupportedMetrics(serviceASG)`. Add `serviceASG` to the
  carbon-capable services case in `getSupportedMetrics()`
- [x] T013 [P] [US5] Add `asgPricingSpec()` method and `case serviceASG:`
  to the `GetPricingSpec` switch in `internal/plugin/pricingspec.go`.
  Return `BillingMode: "on_demand"`, `RateUnit: "per-instance-hour"`,
  with notes describing the ASG cost formula
- [x] T014 [P] [US5] Add `case serviceASG:` to the
  `getProjectedForResource` switch in `internal/plugin/actual.go`
  routing to the ASG estimator
- [x] T015 [US5] Add unit tests for ASG service discovery in
  `internal/plugin/supports_test.go`: verify Supports returns true for
  `"aws:autoscaling/group:Group"`, `"asg"`, and `"autoscaling"` with
  carbon metric advertised. Add test in `internal/plugin/pricingspec_test.go`
  for ASG PricingSpec response format. Add test in
  `internal/plugin/classification_test.go` verifying ASG classification
  entry exists with correct growth type. Add test in
  `internal/plugin/actual_test.go` verifying `getProjectedForResource`
  routes ASG resources correctly through GetActualCost (dual-path coverage
  per CLAUDE.md convention)

**Checkpoint**: All 4 RPC paths (Supports, GetProjectedCost, GetPricingSpec,
GetActualCost) handle ASG resources consistently.

---

## Phase 5: User Story 4 — Carbon Footprint Estimation (Priority: P3)

**Goal**: ASG responses include carbon footprint metrics by delegating to
EC2 carbon estimator × instance count

**Independent Test**: Send `GetProjectedCost` for ASG with
`instance_type: "m5.large"`, `desired_capacity: "3"` and verify
`ImpactMetrics` contains `METRIC_KIND_CARBON_FOOTPRINT` with value
equal to 3× the m5.large single-instance carbon

### Implementation for User Story 4

- [x] T016 [P] [US4] Add `ASGConfig` struct to `internal/carbon/types.go`
  with fields: `InstanceType string`, `Region string`,
  `DesiredCapacity int`, `Utilization float64`, `Hours float64`
- [x] T017 [P] [US4] Create `internal/carbon/asg_estimator.go` with
  `ASGEstimator` struct and `EstimateCarbonGrams(config ASGConfig)
  (float64, bool)` method. Delegate to `NewEstimator()` for single-instance
  carbon, multiply by `DesiredCapacity`. Return `(0, false)` when EC2
  estimator returns false. Follow ElastiCache delegation pattern.
- [x] T018 [US4] Add carbon estimation call to `estimateASG()` in
  `internal/plugin/projected.go`. After cost calculation, create
  `ASGConfig`, call `ASGEstimator.EstimateCarbonGrams()`, and if
  successful, append `ImpactMetric` with
  `METRIC_KIND_CARBON_FOOTPRINT` to response
- [x] T019 [US4] Add unit tests for ASG carbon estimation in
  `internal/carbon/asg_estimator_test.go` with table-driven tests:
  single instance carbon matches EC2 estimator, multi-instance carbon
  scales linearly, unknown instance type returns false, zero capacity
  returns 0 carbon, utilization parameter is passed through correctly

**Checkpoint**: `GetProjectedCost` for ASG includes `ImpactMetrics` with
carbon footprint proportional to instance count.

---

## Phase 6: Polish and Cross-Cutting Concerns

**Purpose**: Validation, documentation, and quality gates

- [x] T020 Run `make test` to verify all existing and new tests pass
- [x] T021 Run `make lint` to verify no linting issues (use extended
  timeout)
- [x] T022 Run `go test -tags=region_use1 ./internal/plugin/... -run
  TestASG` to verify ASG estimation with real pricing data
- [x] T023 Update CLAUDE.md estimation logic section to document ASG
  estimator (resource_type, sku, tags, formula, billing_detail format)
- [x] T024 Update ROADMAP.md to move ASG Estimator (#295) from Future
  Vision to Completed Milestones Q1 2026

---

## Dependencies and Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — verify environment
- **Foundational (Phase 2)**: Depends on Phase 1 — registers ASG as a
  known service type. BLOCKS all user story phases.
- **US1 (Phase 3)**: Depends on Phase 2 — core estimation logic
- **US5 (Phase 4)**: Depends on Phase 2 — can run in parallel with US1
  (different files) but T014 depends on T008 (estimateASG must exist
  for actual.go routing)
- **US4 (Phase 5)**: Depends on Phase 3 (T008 — estimateASG must exist
  to add carbon call) and Phase 4 (T012 — carbon metric must be
  advertised in Supports)
- **Polish (Phase 6)**: Depends on all user stories complete

### User Story Dependencies

- **US1 (P1)**: Requires Phase 2 only — MVP
- **US5 (P2)**: Requires Phase 2 + T008 from US1 (for actual.go routing)
- **US4 (P3)**: Requires T008 (estimateASG) + T012 (carbon in Supports)

### Within Each Phase

- Tasks marked [P] can run in parallel (different files)
- T006 and T007 can run in parallel (implementation + tests in separate files)
- T011, T012, T013, T014 can ALL run in parallel (4 different files)
- T016 and T017 can run in parallel (types.go + estimator in separate files)

### Parallel Opportunities

```text
# Phase 2: All foundational tasks are sequential (same files: projected.go, constants.go)
# Except T005 (arn.go) can run in parallel with T002-T004

# Phase 3: Tag extraction and tests in parallel
Parallel: T006 (asg_attrs.go) + T007 (asg_attrs_test.go)
Sequential: T008 (estimateASG) → T009 (switch case) → T010 (tests)

# Phase 4: All 4 registration tasks in parallel (different files)
Parallel: T011 (classification.go) + T012 (supports.go) + T013 (pricingspec.go) + T014 (actual.go)
Sequential: T015 (tests after all registrations)

# Phase 5: Type + estimator in parallel, then integration
Parallel: T016 (types.go) + T017 (asg_estimator.go)
Sequential: T018 (integrate into projected.go) → T019 (tests)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Verify environment
2. Complete Phase 2: Register ASG service type (T002-T005)
3. Complete Phase 3: Core ASG estimation (T006-T010)
4. **STOP and VALIDATE**: Run `go test ./internal/plugin/... -run TestASG`
5. ASG resources now return cost estimates — MVP complete

### Incremental Delivery

1. Phase 2 → Foundation ready (ASG recognized as service)
2. Phase 3 (US1) → Core estimation works → MVP
3. Phase 4 (US5) → Full RPC coverage (Supports, PricingSpec, ActualCost)
4. Phase 5 (US4) → Carbon estimation added
5. Phase 6 → Polish, docs, validation

---

## Notes

- No new embed files or pricing data needed — ASG reuses EC2 pricing
- No binary size increase — this feature adds only Go source code
- The `estimateASG` function signature includes `req` parameter for future
  dev mode support (utilization percentage)
- All tag key lookups should check both snake_case and camelCase variants
- Follow the two-step normalization pattern (CLAUDE.md) in ALL code paths
