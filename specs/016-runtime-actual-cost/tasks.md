# Tasks: Runtime-Based Actual Cost Estimation

**Input**: Design documents from `/specs/016-runtime-actual-cost/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/
**Tests**: Included per Constitution II Testing Discipline requirements

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single Go project**: `internal/plugin/` at repository root
- **Test files**: Co-located with source files (`*_test.go`)
- **Test fixtures**: `test/fixtures/actual-cost/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Define constants and types shared by all user stories

- [ ] T001 Add Pulumi metadata tag constants (TagPulumiCreated, TagPulumiModified, TagPulumiExternal) in internal/plugin/actual.go
- [ ] T002 Add ConfidenceLevel type and constants (ConfidenceHigh, ConfidenceMedium, ConfidenceLow) in internal/plugin/actual.go
- [ ] T003 Add TimestampResolution struct type in internal/plugin/actual.go
- [ ] T004 [P] Create test fixtures directory at test/fixtures/actual-cost/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core helper functions that MUST be complete before user story work

**⚠️ CRITICAL**: All three user stories depend on these helper functions

- [ ] T005 Implement extractPulumiCreated() function to parse RFC3339 from tags in internal/plugin/actual.go
- [ ] T006 [P] Implement isImportedResource() function to check pulumi:external=true in internal/plugin/actual.go
- [ ] T007 Implement formatSourceWithConfidence() function for semantic source encoding in internal/plugin/actual.go
- [ ] T008 Add table-driven unit tests for extractPulumiCreated() in internal/plugin/actual_test.go
- [ ] T009 [P] Add table-driven unit tests for isImportedResource() in internal/plugin/actual_test.go
- [ ] T010 [P] Add table-driven unit tests for formatSourceWithConfidence() in internal/plugin/actual_test.go

**Checkpoint**: Foundation ready - helper functions tested and working

---

## Phase 3: User Story 1 - Auto-Calculate Cost from Creation Time (Priority: P1) 🎯 MVP

**Goal**: Enable automatic runtime detection from `pulumi:created` timestamps when explicit timestamps are not provided

**Independent Test**: Provide a resource with `pulumi:created` in tags and verify cost is calculated from that timestamp

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T011 [P] [US1] Add unit tests for resolveTimestamps() priority logic in internal/plugin/actual_test.go
- [ ] T012 [P] [US1] Add unit test for runtime hours calculation edge case (zero duration returns zero cost) in internal/plugin/actual_test.go

### Implementation for User Story 1

- [ ] T013 [US1] Implement resolveTimestamps() with priority: explicit > pulumi:created > error in internal/plugin/actual.go
- [ ] T014 [US1] Implement determineConfidence() to map resolution source to confidence level in internal/plugin/actual.go
- [ ] T015 [US1] Modify GetActualCost() handler to call resolveTimestamps() before validation in internal/plugin/plugin.go
- [ ] T016 [US1] Update GetActualCost() to use resolved timestamps for cost calculation in internal/plugin/plugin.go
- [ ] T017 [US1] Add logging for timestamp resolution source (explicit vs pulumi:created) in internal/plugin/plugin.go

### Test Fixtures for User Story 1

- [ ] T018 [P] [US1] Create with-created.json test fixture with valid pulumi:created in test/fixtures/actual-cost/
- [ ] T019 [P] [US1] Create explicit-override.json test fixture with both explicit times and pulumi:created in test/fixtures/actual-cost/

**Checkpoint**: Resources with `pulumi:created` automatically use that time for cost calculation

---

## Phase 4: User Story 2 - Identify Imported Resources with Lower Confidence (Priority: P2)

**Goal**: Flag imported resources with MEDIUM confidence indicator so users know estimates may be less accurate

**Independent Test**: Provide a resource with `pulumi:external=true` and verify response includes MEDIUM confidence

### Tests for User Story 2

- [ ] T020 [P] [US2] Add unit test for determineConfidence() with imported=true scenarios in internal/plugin/actual_test.go
- [ ] T021 [P] [US2] Add unit test for confidence encoding in source field output in internal/plugin/actual_test.go

### Implementation for User Story 2

- [ ] T022 [US2] Update resolveTimestamps() to track IsImported flag from pulumi:external tag in internal/plugin/actual.go
- [ ] T023 [US2] Update determineConfidence() to return MEDIUM when IsImported=true with pulumi:created source in internal/plugin/actual.go
- [ ] T024 [US2] Update GetActualCost() to include confidence and explanatory notes in response source field per FR-006 in internal/plugin/plugin.go
- [ ] T025 [US2] Add specific "imported resource" note when pulumi:external=true detected in internal/plugin/plugin.go

### Test Fixtures for User Story 2

- [ ] T026 [P] [US2] Create with-external.json test fixture with pulumi:external=true in test/fixtures/actual-cost/

**Checkpoint**: Imported resources correctly show MEDIUM confidence indicator

---

## Phase 5: User Story 3 - Prioritize Runtime from Request Metadata (Priority: P2)

**Goal**: Allow explicit request timestamps to override pulumi:created for specific time window queries

**Independent Test**: Provide both `pulumi:created` and explicit timestamps, verify explicit timestamps take precedence

### Tests for User Story 3

- [ ] T027 [P] [US3] Add unit test for resolveTimestamps() explicit override scenario in internal/plugin/actual_test.go
- [ ] T028 [P] [US3] Add unit test for partial explicit (start only, end only) scenarios in internal/plugin/actual_test.go

### Implementation for User Story 3

- [ ] T029 [US3] Add docstring to resolveTimestamps() documenting priority order (explicit > pulumi:created > error) in internal/plugin/actual.go
- [ ] T030 [US3] Handle mixed source case (explicit start, pulumi:created end default) in internal/plugin/actual.go
- [ ] T031 [US3] Update source field to indicate "explicit" when user-provided timestamps used in internal/plugin/plugin.go

### Edge Case Handling for User Story 3

- [ ] T032 [US3] Handle invalid RFC3339 in pulumi:created by falling back to explicit requirement in internal/plugin/actual.go
- [ ] T033 [US3] Handle end time before start time with zero cost and explanation in internal/plugin/plugin.go
- [ ] T034 [US3] Ensure pulumi:modified is NOT used as fallback (explicit behavior) in internal/plugin/actual.go

**Checkpoint**: Explicit timestamps correctly override pulumi:created timestamps

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Integration verification, documentation, and final validation

- [ ] T035 Run `make lint` and fix any linting issues
- [ ] T036 Run `make test` and verify all unit tests pass
- [ ] T037 [P] Run integration test with grpcurl commands from quickstart.md
- [ ] T038 [P] Verify backward compatibility: existing callers without pulumi metadata still work
- [ ] T039 Update CLAUDE.md if new patterns or conventions emerged
- [ ] T040 Run `npx markdownlint` on all spec markdown files

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User stories can proceed sequentially in priority order (P1 → P2)
  - US2 and US3 are both P2 but can be done in order listed
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - Core MVP functionality
- **User Story 2 (P2)**: Depends on US1 confidence infrastructure - Adds imported resource handling
- **User Story 3 (P2)**: Depends on US1 timestamp resolution - Refines priority semantics

### Within Each User Story

- Tests SHOULD be written and FAIL before implementation (TDD)
- Helper functions before handler modifications
- Core implementation before edge case handling
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks can run sequentially (same file)
- T006, T007 can run in parallel with T005 (different functions)
- Test fixture creation (T018, T019, T026) can run in parallel
- Test writing (T011, T012) can run in parallel (different test tables)

---

## Parallel Example: Foundational Phase

```bash
# After T005 completes, these can run in parallel:
Task: T006 - Implement isImportedResource() in actual.go
Task: T007 - Implement formatSourceWithConfidence() in actual.go

# Then all tests can run in parallel:
Task: T008 - Unit tests for extractPulumiCreated()
Task: T009 - Unit tests for isImportedResource()
Task: T010 - Unit tests for formatSourceWithConfidence()
```

## Parallel Example: Test Fixtures

```bash
# All fixture creation can run in parallel:
Task: T018 - Create with-created.json
Task: T019 - Create explicit-override.json
Task: T026 - Create with-external.json
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T004)
2. Complete Phase 2: Foundational (T005-T010)
3. Complete Phase 3: User Story 1 (T011-T019)
4. **STOP and VALIDATE**: Test with grpcurl using with-created.json fixture
5. Deploy/demo if ready - core functionality working

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → **MVP!** (auto-calculate from creation time)
3. Add User Story 2 → Test independently → **Enhanced!** (imported resource confidence)
4. Add User Story 3 → Test independently → **Complete!** (explicit override support)
5. Each story adds value without breaking previous stories

### Estimated Scope

- **MVP (US1 only)**: ~19 tasks (T001-T019)
- **Full feature**: 40 tasks (T001-T040)
- **Parallel potential**: ~15 tasks can be parallelized

---

## Notes

- All code changes isolated to `internal/plugin/` package
- No proto changes required (semantic encoding in existing source field)
- Backward compatible: existing callers unaffected
- Constitution compliance: Table-driven tests, zerolog logging, thread-safe functions
- Verify with `make lint && make test` after each phase
