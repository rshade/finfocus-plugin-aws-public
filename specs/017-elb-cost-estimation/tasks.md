# Tasks: Implement Elastic Load Balancing (ALB/NLB) cost estimation

**Input**: Design documents from `/specs/017-elb-cost-estimation/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md

**Tests**: Tests are included as per project standards (Go unit/integration tests).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Update pricing generation tools to include ELB data.

- [x] T001 Update `tools/generate-pricing/main.go` to include `AWSELB` in the list of service codes
- [x] T002 Generate updated pricing data files using `make generate-pricing`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core pricing data structures and extraction logic.

- [x] T003 [P] Add `elbPrice` struct to `internal/pricing/types.go`
- [x] T004 [P] Add ALB/NLB hourly and capacity unit methods to `PricingClient` interface in `internal/pricing/client.go`
- [x] T005 Update pricing parser in `internal/pricing/client.go` to extract ELB rates using the `group` attribute
- [x] T006 Update `internal/pricing/client_test.go` to verify ELB pricing extraction and lookup

**Checkpoint**: Foundation ready - ELB pricing data can now be retrieved by the plugin.

---

## Phase 3: User Story 1 - ALB Cost Estimation (Priority: P1) 🎯 MVP

**Goal**: Estimate ALB costs (fixed + LCU) from resource SKU and tags.

**Independent Test**: `go test ./internal/plugin -v -run TestAWSPublicPlugin_GetProjectedCost/elb_alb`

### Implementation for User Story 1

- [x] T007 [P] [US1] Add "elb" and "alb" to supported resource types in `internal/plugin/supports.go`
- [x] T008 [US1] Implement `estimateELB` with ALB calculation logic and router case in `internal/plugin/projected.go`
- [x] T009 [US1] Create `internal/plugin/elb_test.go` and add unit tests for ALB cost calculation
- [x] T010 [US1] Add ALB integration test cases in `internal/plugin/projected_test.go`

**Checkpoint**: ALB estimation is functional and testable independently.

---

## Phase 4: User Story 2 - NLB Cost Estimation (Priority: P1)

**Goal**: Estimate NLB costs (fixed + NLCU) from resource SKU and tags.

**Independent Test**: `go test ./internal/plugin -v -run TestAWSPublicPlugin_GetProjectedCost/elb_nlb`

### Implementation for User Story 2

- [x] T011 [P] [US2] Add "nlb" to supported resource types in `internal/plugin/supports.go`
- [x] T012 [US2] Extend `estimateELB` calculation logic in `internal/plugin/projected.go` to support NLB SKU and NLCU/Generic capacity tags
- [x] T013 [US2] Add NLB unit test cases in `internal/plugin/elb_test.go`
- [x] T014 [US2] Add NLB integration test cases in `internal/plugin/projected_test.go`

**Checkpoint**: NLB estimation is functional and testable independently.

---

## Phase 5: User Story 3 - Regional Pricing Support (Priority: P2)

**Goal**: Ensure ELB pricing works correctly across all target regions.

**Independent Test**: `make build-all-regions` and manual verification of binary sizes/contents.

### Implementation for User Story 3

- [x] T015 [US3] Verify `AWSELB` data is correctly embedded in regional binaries via `make build-all-regions`
- [x] T016 [US3] Add a regional integration test case (e.g., us-west-2) in `internal/plugin/integration_test.go`

**Checkpoint**: Regional pricing support is verified.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Quality assurance and final documentation.

- [x] T017 [P] Run `make lint` and resolve any static analysis findings
- [x] T018 [P] Update `CLAUDE.md` with any new ELB-specific patterns if applicable
- [x] T019 Validate final implementation against all scenarios in `specs/017-elb-cost-estimation/quickstart.md` and verify RPC latency is < 100ms (SC-004)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Must fetch pricing data first.
- **Foundational (Phase 2)**: Depends on Phase 1 completion.
- **User Stories (Phase 3+)**: Depend on Foundational (Phase 2). US1 and US2 are both P1 and can be done in parallel or sequence.

### Parallel Opportunities

- T003 and T004 (Pricing types/interfaces)
- T007 and T011 (Supports entries)
- T009 and T013 (Unit tests for different types)
- T017 and T018 (Polish tasks)

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Setup and Foundational.
2. Complete User Story 1 (ALB).
3. Validate ALB estimation works.

### Incremental Delivery

1. Foundation ready.
2. ALB support added.
3. NLB support added.
4. Regional verification.
