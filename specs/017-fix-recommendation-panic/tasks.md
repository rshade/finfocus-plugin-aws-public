# Tasks: Bug Fix and Documentation Sprint - Dec 2025

**Input**: Design documents from `/specs/017-fix-recommendation-panic/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md

**Tests**: Tests are included for all bug fixes to ensure 100% verification of SC-002.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Verify project structure and branch environment in specs/017-fix-recommendation-panic/plan.md
- [x] T002 [P] Configure development environment and ensure `make develop` passes

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

- [x] T003 [P] Implement `SetLogger` helper in `internal/carbon/instance_specs.go` to allow external logger injection
- [x] T004 Update `AWSPublicPlugin` initialization to inject logger into carbon package in `internal/plugin/plugin.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - System Stability & Robustness (Priority: P1) 🎯 MVP

**Goal**: Address critical panics and validation errors to ensure plugin stability.

**Independent Test**: Run `GetRecommendations` with malformed resources and verify no panic; Verify S3 cost estimation with global ARNs.

### Tests for User Story 1

- [x] T005 [P] [US1] Create regression test for recommendation panic in `internal/plugin/recommendations_panic_test.go`
- [x] T006 [P] [US1] Add test for S3 global service region fallback in `internal/plugin/supports_test.go`
- [x] T007 [P] [US1] Add integration test for comprehensive cost validation in `internal/plugin/integration_test.go`

### Implementation for User Story 1

- [x] T008 [US1] Implement nil check for `rec.Impact` in `internal/plugin/recommendations.go`
- [x] T009 [US1] Implement structured JSON error logging for CSV failures in `internal/carbon/instance_specs.go` (ensure plugin prefix propagation)
- [x] T010 [US1] Update `Supports` to handle empty regions for S3 global services in `internal/plugin/supports.go`
- [x] T011 [US1] Restore strict validation for all required parameters in `internal/plugin/projected.go` (within `GetProjectedCost`)
- [x] T012 [US1] Fix region fallback logic for S3 resources in `internal/plugin/validation.go`

**Checkpoint**: At this point, all critical bugs (SC-002) should be resolved and verified.

---

## Phase 4: User Story 2 - Documentation Clarity (Priority: P2)

**Goal**: Improve codebase maintainability and user understanding through better documentation.

**Independent Test**: Verify GoDoc renders correctly and warning logs appear when using deprecated env vars.

### Implementation for User Story 2

- [x] T013 [P] [US2] Consolidate duplicate docstrings for `GetUtilization` in `internal/carbon/utilization.go`
- [x] T014 [P] [US2] Add GoDoc explaining correlation ID population (`ResourceId`/`Name`) in `internal/plugin/recommendations.go`
- [x] T015 [US2] Implement deprecation warning for `PORT` env var in `cmd/finfocus-plugin-aws-public/main.go`
- [x] T016 [P] [US2] Document internal platform-to-OS mapping logic in `internal/plugin/ec2_attrs.go`

**Checkpoint**: Documentation sprint completed (SC-003).

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Final style fixes and standalone documentation.

- [x] T017 [P] Ensure trailing newlines are present in all Go files within `internal/carbon/`
- [x] T018 [P] Create `TROUBLESHOOTING.md` in repository root with common error scenarios
- [x] T019 Run final `make lint` and `make test` to verify all sprint goals
- [x] T020 [P] Run `quickstart.md` validation steps to ensure user-facing fixes work as expected
- [x] T021 Run performance benchmarks and verify no regression (SC-004) in `internal/plugin/...`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS US1.
- **User Stories (Phase 3+)**: All depend on Foundational phase completion.
- **Polish (Final Phase)**: Depends on all user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: Foundation ready - No dependencies on other stories.
- **User Story 2 (P2)**: Independent documentation tasks.

### Parallel Opportunities

- T005, T006, T007 (Tests) can run in parallel.
- T013, T014, T016 (Docs) can run in parallel.
- T017, T018 (Polish) can run in parallel.

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 & 2.
2. Complete Phase 3 (US1) - Addresses the high-priority panic (#123).
3. **STOP and VALIDATE**: Verify zero panics in batch processing.

### Incremental Delivery

1. Foundation ready (Phase 2).
2. Bug fixes complete (Phase 3).
3. Documentation updates complete (Phase 4).
4. Final polish and troubleshooting guide (Phase 5).

---

## Notes

- All logging MUST use `zerolog` for structured JSON.
- `PORT` deprecation warning MUST be at `WARN` level.
- Trailing newlines fix (#145) should be applied to all `.go` files in `internal/carbon/`.