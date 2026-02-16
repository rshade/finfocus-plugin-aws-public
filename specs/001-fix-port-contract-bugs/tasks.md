# Tasks: Port Binding & Contract Violation Fixes

**Input**: Design documents from `/specs/001-fix-port-contract-bugs/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, quickstart.md

**Tests**: Tests are included to verify contract compliance and routing fixes.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Create project structure per implementation plan (Done by setup)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T002 [P] Define `PricingUnavailableError` type in `internal/plugin/errors.go`
- [x] T003 [P] Implement unit tests for `PricingUnavailableError` in `internal/plugin/errors_test.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Explicit Port Selection (Priority: P1) 🎯 MVP

**Goal**: Allow plugin to be started on a specific port via `--port` CLI flag.

**Independent Test**: Run `./finfocus-plugin-aws-public-us-east-1 --port 9090` and verify `PORT=9090` in logs.

### Implementation for User Story 1

- [x] T004 [US1] Add `"flag"` to imports in `cmd/finfocus-plugin-aws-public/main.go`
- [x] T005 [US1] Insert `flag.Parse()` and `pluginsdk.ParsePortFlag()` at the start of `run()` in `cmd/finfocus-plugin-aws-public/main.go`
- [x] T006 [US1] Update port selection logic in `run()` to prioritize `ParsePortFlag()` over environment variables in `cmd/finfocus-plugin-aws-public/main.go`

**Checkpoint**: User Story 1 should be fully functional and testable independently.

---

## Phase 4: User Story 2 - Fail-Open Contract Compliance (Priority: P1)

**Goal**: Return empty results in `GetActualCost` when pricing is missing, enabling Core fallback.

**Independent Test**: Request `GetActualCost` for an unknown instance type and verify empty `Results`.

### Implementation for User Story 2

- [x] T007 [US2] Update estimators (EC2, EBS, EKS, S3, Lambda, ELB, NATGW, ElastiCache) in `internal/plugin/projected.go` to return `PricingUnavailableError` when lookup fails
- [x] T008 [US2] Update `GetProjectedCost` in `internal/plugin/projected.go` to catch `PricingUnavailableError` and return $0 with `BillingDetail` (backward compat)
- [x] T009 [US2] Update `GetActualCost` in `internal/plugin/plugin.go` to catch `PricingUnavailableError` and return empty `Results` list
- [x] T010 [P] [US2] Add regression tests for $0 backward compatibility in `internal/plugin/projected_test.go`
- [x] T011 [P] [US2] Add regression tests for empty results fallback in `internal/plugin/actual_test.go`

**Checkpoint**: User Story 2 should be fully functional and testable independently.

---

## Phase 5: User Story 3 - Full Resource Routing for Actual Cost (Priority: P2)

**Goal**: Use real estimators for S3, Lambda, RDS, and DynamoDB in Actual Cost mode.

**Independent Test**: Run Actual Cost for an S3 bucket and verify non-zero results.

### Implementation for User Story 3

- [x] T012 [US3] Update `getProjectedForResource` in `internal/plugin/actual.go` to route S3, Lambda, RDS, and DynamoDB to their real estimators
- [x] T013 [US3] Update `getProjectedForResource` in `internal/plugin/actual.go` to route VPC, SG, Subnet, and IAM to a zero-cost resource handler
- [x] T014 [P] [US3] Add integration-style tests for S3/Lambda Actual Cost estimation in `internal/plugin/actual_test.go`
- [x] T018 [P] [US2] Add test case for batch requests with mixed pricing availability (some found, some missing) in `internal/plugin/actual_test.go`

**Checkpoint**: User Story 3 should be fully functional and testable independently.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and cleanup

- [x] T015 Run `make test` to ensure all unit and integration tests pass
- [x] T016 Run `make lint` to ensure code style compliance
- [x] T017 Validate all scenarios in `quickstart.md` manually

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Complete.
- **Foundational (Phase 2)**: MUST complete before US2 and US3.
- **User Story 1 (P1)**: Independent of US2/US3. Can start after Phase 1.
- **User Story 2 (P1)**: Depends on Phase 2.
- **User Story 3 (P2)**: Depends on Phase 2. Should ideally follow US2 to ensure correct error handling for the new routes.

### Parallel Opportunities

- US1 can be implemented in parallel with Phase 2, US2, and US3 as it touches different files (`main.go`).
- Unit tests for US2 and US3 can be written in parallel with their implementations.

---

## Implementation Strategy

### MVP First (User Story 1 & 2)

1. Complete Phase 2 (Foundational).
2. Complete US1 (Port Binding) and US2 (Contract Compliance).
3. **VALIDATE**: Plugin respects `--port` and returns empty results for unknown pricing.

### Incremental Delivery

1. Foundation ready.
2. US1 added → Port control functional.
3. US2 added → Fallback contract functional.
4. US3 added → Actual cost reporting more complete.
