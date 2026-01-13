# Tasks: E2E Test Support and Validation

**Input**: Design documents from `/specs/001-e2e-test-support/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md

**Tests**: Unit tests included per constitution requirement (Testing Discipline)

**Organization**: Tasks grouped by user story for independent implementation

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Go plugin**: `internal/plugin/`, `cmd/finfocus-plugin-aws-public/`
- **Tests**: Co-located as `*_test.go` files per Go convention

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare test mode infrastructure shared by all user stories

- [X] T001 Create testmode.go with IsTestMode() and ValidateTestModeEnv()
  functions in internal/plugin/testmode.go
- [X] T002 [P] Create expected.go with ExpectedCostRange struct and
  ExpectedCostRanges map in internal/plugin/expected.go
- [X] T003 Add testMode field to AWSPublicPlugin struct in
  internal/plugin/plugin.go
- [X] T004 Update NewAWSPublicPlugin() to initialize testMode from env var in
  internal/plugin/plugin.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core test mode detection that MUST work before any user story

**CRITICAL**: All user stories depend on test mode detection working correctly

- [X] T005 Add startup validation of FINFOCUS_TEST_MODE in
  cmd/finfocus-plugin-aws-public/main.go
- [X] T006 Log test mode status at plugin startup in
  cmd/finfocus-plugin-aws-public/main.go
- [X] T007 [P] Create testmode_test.go with unit tests for IsTestMode() and
  ValidateTestModeEnv() in internal/plugin/testmode_test.go
- [X] T008 Verify existing tests pass with test mode disabled via make test

**Checkpoint**: Test mode detection works; plugin starts correctly with/without
FINFOCUS_TEST_MODE set

---

## Phase 3: User Story 1 - Validate Projected Costs (Priority: P1) MVP

**Goal**: E2E tests can validate projected cost calculations against expected
ranges for t3.micro EC2 and gp2 EBS in us-east-1

**Independent Test**: Request projected costs for t3.micro EC2 and verify
response is within 1% of $7.592/month

### Tests for User Story 1

- [X] T009 [P] [US1] Create expected_test.go with tests for GetExpectedRange()
  lookup in internal/plugin/expected_test.go
- [X] T010 [P] [US1] Add test for IsWithinTolerance() function in
  internal/plugin/expected_test.go

### Implementation for User Story 1

- [X] T011 [US1] Add GetExpectedRange() lookup function in
  internal/plugin/expected.go
- [X] T012 [US1] Add IsWithinTolerance() validation function in
  internal/plugin/expected.go
- [X] T013 [US1] Add EC2 t3.micro us-east-1 expected cost range ($0.0104/hr,
  $7.592/mo, 1% tolerance) in internal/plugin/expected.go
- [X] T014 [US1] Add EBS gp2 us-east-1 expected cost range ($0.10/GB-mo,
  $0.80/8GB, 5% tolerance) in internal/plugin/expected.go
- [X] T015 [US1] Add ReferenceDate field with "2025-12-01" to expected ranges in
  internal/plugin/expected.go

**Checkpoint**: GetProjectedCost returns values matching expected ranges; E2E
tests can validate accuracy

---

## Phase 4: User Story 2 - Validate Actual Cost Fallback (Priority: P2)

**Goal**: E2E tests can validate actual cost fallback calculations with runtime
proration formula: `projected_monthly × (runtime_hours / 730)`

**Independent Test**: Request actual cost for 30-minute runtime and verify
result is ~$0.0052 for t3.micro EC2

### Tests for User Story 2

- [X] T016 [P] [US2] Add test for actual cost proration formula validation in
  internal/plugin/expected_test.go

### Implementation for User Story 2

- [X] T017 [US2] Add CalculateExpectedActualCost() helper function in
  internal/plugin/expected.go
- [X] T018 [US2] Add 30-minute runtime expected cost ($0.0052 for EC2,
  $0.00055 for EBS) to expected ranges in internal/plugin/expected.go
- [X] T019 [US2] Add test case validating GetActualCost matches proration
  formula in internal/plugin/actual_test.go

**Checkpoint**: GetActualCost fallback returns correct prorated values

---

## Phase 5: User Story 3 - Enhanced Test Diagnostics (Priority: P3)

**Goal**: When test mode enabled, logs include additional context for debugging
(trace ID, resource details, calculation breakdown)

**Independent Test**: Enable test mode, make cost request, verify debug logs
include calculation breakdown

### Tests for User Story 3

- [X] T020 [P] [US3] Add test for enhanced logging when testMode=true in
  internal/plugin/projected_test.go
- [X] T021 [P] [US3] Add test for no enhanced logging when testMode=false in
  internal/plugin/projected_test.go

### Implementation for User Story 3

- [X] T022 [US3] Add conditional debug logging for request details in
  GetProjectedCost in internal/plugin/projected.go
- [X] T023 [US3] Add conditional debug logging for calculation result in
  GetProjectedCost in internal/plugin/projected.go
- [X] T024 [US3] Add conditional debug logging for request details in
  GetActualCost in internal/plugin/plugin.go
- [X] T025 [US3] Add conditional debug logging for calculation result in
  GetActualCost in internal/plugin/plugin.go
- [X] T026 [US3] Verify zero overhead when test mode disabled (no string
  formatting unless log emitted)

**Checkpoint**: Debug logs with calculation details appear only when test mode
enabled

---

## Phase 6: User Story 4 - Access Expected Cost Ranges (Priority: P4)

**Goal**: E2E test framework can query expected cost ranges to set appropriate
validation tolerances

**Independent Test**: Call GetExpectedRange("ec2", "t3.micro", "us-east-1") and
verify response includes min, max, expected, and tolerance

### Tests for User Story 4

- [X] T027 [P] [US4] Add test for GetExpectedRange with valid resource in
  internal/plugin/expected_test.go
- [X] T028 [P] [US4] Add test for GetExpectedRange with unsupported resource in
  internal/plugin/expected_test.go

### Implementation for User Story 4

- [X] T029 [US4] Add Min() and Max() methods to ExpectedCostRange struct in
  internal/plugin/expected.go
- [X] T030 [US4] Document expected cost ranges in quickstart.md with reference
  date in specs/001-e2e-test-support/quickstart.md

**Checkpoint**: Expected cost ranges are queryable and documented

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and documentation

- [X] T031 Run make lint and fix any issues
- [X] T032 Run make test and verify all tests pass
- [X] T033 [P] Add test mode documentation to README.md
- [X] T034 [P] Update CLAUDE.md with 001-e2e-test-support feature summary
- [X] T035 Run quickstart.md validation scenarios manually
- [X] T036 Verify backward compatibility: existing tests pass without
  FINFOCUS_TEST_MODE set

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational completion
  - Stories can proceed in priority order (P1 → P2 → P3 → P4)
  - Each story is independently testable after completion
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Phase 2 - No dependencies on other stories
- **User Story 2 (P2)**: Uses expected ranges from US1 but independently testable
- **User Story 3 (P3)**: Uses testMode from Phase 2 - independently testable
- **User Story 4 (P4)**: Uses expected ranges from US1 but independently testable

### Within Each User Story

- Tests written first and verified to fail
- Data structures before functions
- Core implementation before integration
- Verify tests pass after implementation

### Parallel Opportunities

- T001/T002 can run in parallel (different files)
- T007 can run with T005/T006 (test file vs main.go)
- T009/T010 can run in parallel
- T020/T021 can run in parallel
- T027/T028 can run in parallel
- T033/T034 can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "T009 [P] [US1] Create expected_test.go with tests for GetExpectedRange()"
Task: "T010 [P] [US1] Add test for IsWithinTolerance() function"

# Then implement (sequential):
Task: "T011 [US1] Add GetExpectedRange() lookup function"
Task: "T012 [US1] Add IsWithinTolerance() validation function"
Task: "T013 [US1] Add EC2 t3.micro expected cost range"
Task: "T014 [US1] Add EBS gp2 expected cost range"
Task: "T015 [US1] Add ReferenceDate field"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T004)
2. Complete Phase 2: Foundational (T005-T008)
3. Complete Phase 3: User Story 1 (T009-T015)
4. **STOP and VALIDATE**: Verify expected cost ranges work
5. Deploy as v0.0.5 with E2E test support

### Incremental Delivery

1. Setup + Foundational → Test mode detection works
2. Add User Story 1 → Expected cost ranges available (MVP)
3. Add User Story 2 → Actual cost validation supported
4. Add User Story 3 → Enhanced debugging available
5. Add User Story 4 → Full expected range query API

### Full Implementation

For complete E2E test support:

- Complete all phases in order
- Each checkpoint validates story independence
- Final polish ensures production readiness

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- Tests use Go testing package with table-driven tests
- All tests must pass `make test` before commit
- Commit after each completed phase
- Stop at any checkpoint to validate story independently
