# Tasks: Lambda Cost Estimation

**Feature Branch**: `014-lambda-cost-estimation`
**Spec**: [specs/014-lambda-cost-estimation/spec.md](./spec.md)

## Implementation Strategy

We will implement this feature in three main phases, following the priority of
user stories.

1. **Foundation**: Establish the pricing data structures and client interfaces
   required to support Lambda.
2. **Core Estimation (US1)**: Implement the cost calculation logic, ensuring
   accuracy for standard scenarios.
3. **Robustness (US2 & US3)**: Add graceful handling for missing data and
   ensure full regional support.

## Dependencies

1. **Phase 1: Setup** (No dependencies)
2. **Phase 2: Foundation** (Depends on Setup)
3. **Phase 3: User Story 1** (Depends on Foundation)
4. **Phase 4: User Story 2** (Depends on US1 logic)
5. **Phase 5: User Story 3** (Depends on Foundation data structures)
6. **Phase 6: Polish** (Depends on all stories)

---

## Phase 1: Setup

**Goal**: Initialize the feature workspace.

- [x] T001 Create feature branch `014-lambda-cost-estimation` (already done by
  automation)

---

## Phase 2: Foundational

**Goal**: Update pricing types and client interface to support Lambda data
structures.
**Blocking**: Must be completed before any estimation logic can be written.

- [x] T002 Define `lambdaPrice` struct in `internal/pricing/types.go`
- [x] T003 [P] Extend `PricingClient` interface with `LambdaPricePerRequest`
  and `LambdaPricePerGBSecond` in `internal/pricing/client.go`
- [x] T004 Add `lambdaPricing` field to `Client` struct in
  `internal/pricing/client.go`
- [x] T005 Implement `LambdaPricePerRequest` and `LambdaPricePerGBSecond`
  methods in `internal/pricing/client.go`

---

## Phase 3: User Story 1 - Accurate Cost Estimation

**Goal**: Enable accurate monthly cost estimates for Lambda functions based on
memory and usage tags.
**Priority**: P1
**Independent Test**: Verify cost calculation for a known configuration (e.g.,
512MB, 1M requests, 200ms) matches manual calculation.

- [x] T006 [US1] Create unit tests for `estimateLambda` calculation logic in
  `internal/plugin/projected_test.go`
- [x] T007 [US1] Implement `estimateLambda` private method in
  `internal/plugin/projected.go`
- [x] T008 [US1] Update `GetProjectedCost` router in
  `internal/plugin/projected.go` to handle `aws:lambda:function`
- [x] T009 [US1] Update `Supports` method in `internal/plugin/supports.go` to
  return true for `aws:lambda:function`
- [x] T010 [US1] Update `generate-pricing` tool to fetch Lambda pricing data
  in `tools/generate-pricing/main.go`
- [x] T011 [US1] Regenerate embedded pricing data for all regions using
  `make generate-pricing` (Verified for us-east-1, sa-east-1)

---

## Phase 4: User Story 2 - Graceful Default Handling

**Goal**: Ensure the system handles missing or invalid usage tags without
crashing or misleading the user.
**Priority**: P2
**Independent Test**: Verify that a resource with missing tags returns $0 cost
and a descriptive message.

- [x] T012 [P] [US2] Add test cases for missing/invalid tags in
  `internal/plugin/projected_test.go`
- [x] T013 [P] [US2] Add test cases for invalid SKU (memory parsing fallback)
  in `internal/plugin/projected_test.go`
- [x] T014 [US2] Implement default value logic (0 requests, 100ms duration) in
  `estimateLambda` in `internal/plugin/projected.go`
- [x] T015 [US2] Implement billing detail string generation for missing data
  scenarios in `internal/plugin/projected.go`

---

## Phase 5: User Story 3 - Regional Support

**Goal**: Verify and ensure Lambda pricing is available and correct across all
supported regions.
**Priority**: P2
**Independent Test**: Confirm pricing data loads correctly for a non-US region
(e.g., `sa-east-1`).

- [x] T016 [US3] Verify `generate-pricing` tool correctly fetches regional
  pricing for non-US regions in `tools/generate-pricing/main.go`
- [x] T017 [US3] Add integration test for multi-region pricing availability in
  `internal/plugin/integration_test.go`
- [x] T018 [US3] Implement "Region not supported" error handling in
  `estimateLambda` (return $0 cost) in `internal/plugin/projected.go`

---

## Phase 6: Polish & Cross-Cutting

**Goal**: Final cleanup, documentation, and rigorous verification.

- [x] T019 Update `README.md` or `docs/` to document supported Lambda tags
  (`requests_per_month`, `avg_duration_ms`)
- [x] T020 Run full test suite `make test` to ensure no regressions
- [x] T021 Run `make lint` and fix any new linting issues
- [x] T022 Manual verification using `specs/014-lambda-cost-estimation/quickstart.md`
