# Tasks: Fallback GetActualCost Implementation

**Input**: Design documents from `/specs/004-actual-cost-fallback/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md

**Tests**: Include test tasks as spec requires 100% coverage (SC-004).

**Organization**: Tasks grouped by user story for independent implementation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story (US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go plugin project**: `internal/plugin/`, `internal/pricing/`
- Tests alongside implementation files with `_test.go` suffix

---

## Phase 1: Setup

**Purpose**: Verify existing code structure and prepare for implementation

- [x] T001 Review existing internal/plugin/plugin.go GetActualCost method
- [x] T002 Review internal/plugin/projected.go for reusable helpers
- [x] T003 [P] Verify hoursPerMonth constant is exported or accessible

---

## Phase 2: Foundational (Helpers)

**Purpose**: Create shared helper functions before user story implementation

**CRITICAL**: These helpers are used by all user stories

- [x] T004 Create internal/plugin/actual.go with package declaration
- [x] T005 Implement calculateRuntimeHours helper in internal/plugin/actual.go
- [x] T006 Implement getProjectedForResource helper in internal/plugin/actual.go
- [x] T007 [P] Create internal/plugin/actual_test.go with package declaration

**Checkpoint**: Helpers ready - user story implementation can begin

---

## Phase 3: User Story 1 - Calculate Actual Cost (Priority: P1) MVP

**Goal**: Calculate actual cost using fallback formula for EC2 and EBS

**Independent Test**: Call GetActualCost with resource and 24-hour range,
verify non-zero cost returned with correct formula application

### Tests for User Story 1

- [x] T008 [P] [US1] Test calculateRuntimeHours in internal/plugin/actual_test.go
- [x] T009 [P] [US1] Test GetActualCost EC2 calculation in internal/plugin/actual_test.go
- [x] T010 [P] [US1] Test GetActualCost EBS calculation in internal/plugin/actual_test.go

### Implementation for User Story 1

- [x] T011 [US1] Implement GetActualCost in internal/plugin/plugin.go
- [x] T012 [US1] Add request validation (nil checks, region match)
- [x] T013 [US1] Parse timestamps using AsTime() in GetActualCost
- [x] T014 [US1] Calculate runtime hours and apply formula
- [x] T015 [US1] Format billing_detail with fallback explanation

**Checkpoint**: EC2/EBS actual cost calculation works independently

---

## Phase 4: User Story 2 - Handle Invalid Time Ranges (Priority: P2)

**Goal**: Return appropriate errors for invalid time ranges

**Independent Test**: Call GetActualCost with start > end, verify error returned

### Tests for User Story 2

- [x] T016 [P] [US2] Test invalid range (start > end) in internal/plugin/actual_test.go
- [x] T017 [P] [US2] Test nil timestamps in internal/plugin/actual_test.go
- [x] T018 [P] [US2] Test zero duration (start = end) in internal/plugin/actual_test.go

### Implementation for User Story 2

- [x] T019 [US2] Add nil timestamp validation in GetActualCost
- [x] T020 [US2] Add start > end validation with error in GetActualCost
- [x] T021 [US2] Handle zero duration returning $0.00 in GetActualCost

**Checkpoint**: Invalid time ranges handled with proper errors

---

## Phase 5: User Story 3 - Support Stub Services (Priority: P3)

**Goal**: Return $0 with explanation for S3, Lambda, RDS, DynamoDB

**Independent Test**: Call GetActualCost for S3, verify $0 with explanation

### Tests for User Story 3

- [x] T022 [P] [US3] Test S3 stub response in internal/plugin/actual_test.go
- [x] T023 [P] [US3] Test Lambda stub response in internal/plugin/actual_test.go
- [x] T024 [P] [US3] Test RDS/DynamoDB stub responses in internal/plugin/actual_test.go

### Implementation for User Story 3

- [x] T025 [US3] Handle stub services in GetActualCost routing
- [x] T026 [US3] Return $0 with appropriate billing_detail for stubs

**Checkpoint**: All stub services return consistent $0 responses

---

## Phase 6: Polish & Validation

**Purpose**: Final validation and documentation

- [x] T027 [P] Run make lint and fix any issues
- [x] T028 [P] Run make test and verify 100% pass rate
- [x] T029 Add benchmark test in internal/plugin/actual_test.go using
  testing.B to verify SC-003 latency < 10ms
- [x] T030 Update CLAUDE.md Recent Changes section

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - review existing code
- **Foundational (Phase 2)**: Depends on Setup - creates shared helpers
- **User Stories (Phase 3-5)**: All depend on Foundational completion
- **Polish (Phase 6)**: Depends on all user stories

### User Story Dependencies

- **US1 (P1)**: After Foundational - core calculation, no story deps
- **US2 (P2)**: After Foundational - error handling, independent of US1
- **US3 (P3)**: After Foundational - stub services, independent of US1/US2

### Within Each User Story

- Tests written first, verify they fail
- Validation before calculation
- Core implementation before formatting
- Commit after each checkpoint

### Parallel Opportunities

- T008, T009, T010 can run in parallel (different test cases)
- T016, T017, T018 can run in parallel (different test cases)
- T022, T023, T024 can run in parallel (different test cases)
- T027, T028 can run in parallel (lint vs test)

---

## Parallel Example: User Story 1 Tests

```bash
# Launch all US1 tests together:
Task: "Test calculateRuntimeHours in internal/plugin/actual_test.go"
Task: "Test GetActualCost EC2 calculation in internal/plugin/actual_test.go"
Task: "Test GetActualCost EBS calculation in internal/plugin/actual_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (review existing code)
2. Complete Phase 2: Foundational (create helpers)
3. Complete Phase 3: User Story 1 (EC2/EBS calculation)
4. **STOP and VALIDATE**: Test with grpcurl
5. Continue to US2/US3 if time permits

### Incremental Delivery

1. Setup + Foundational → Helpers ready
2. Add US1 → EC2/EBS actual cost works → MVP!
3. Add US2 → Error handling complete
4. Add US3 → Stub services consistent
5. Polish → Ready for PR

### File Summary

| File                           | Action | User Stories  |
| ------------------------------ | ------ | ------------- |
| internal/plugin/actual.go      | CREATE | All           |
| internal/plugin/actual_test.go | CREATE | All           |
| internal/plugin/plugin.go      | MODIFY | US1, US2, US3 |

---

## Notes

- [P] tasks = different test cases or files, no dependencies
- [Story] label maps task to specific user story
- Each user story independently testable via grpcurl
- Reuse existing estimateEC2, estimateEBS, estimateStub helpers
- Formula: `cost = monthly_cost × (runtime_hours / 730)`
- Commit after each checkpoint
