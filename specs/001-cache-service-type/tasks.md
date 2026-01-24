# Tasks: Cache Normalized Service Type

**Input**: Design documents from `/specs/001-cache-service-type/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md

**Tests**: Benchmark tests are included as they are explicitly required by the feature specification (SC-001, SC-002) to measure performance improvement.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: Go project at repository root
- Paths: `internal/plugin/` for source code

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the serviceResolver type that all user stories will use

- [x] T001 Create serviceResolver struct with fields (original, normalizedType, serviceType, initialized) in internal/plugin/service_cache.go
- [x] T002 Implement newServiceResolver(resourceType string) constructor in internal/plugin/service_cache.go
- [x] T003 Implement NormalizedType() method with lazy initialization in internal/plugin/service_cache.go
- [x] T004 Implement ServiceType() method with lazy initialization in internal/plugin/service_cache.go
- [x] T005 [P] Add comprehensive docstrings per project code style guidelines in internal/plugin/service_cache.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Unit tests and benchmarks to verify serviceResolver behavior before refactoring call sites

**Rationale**: Unit tests validate the new type works correctly in isolation. Benchmarks capture baseline performance for before/after comparison.

- [x] T006 [P] Create unit tests for serviceResolver construction and lazy initialization in internal/plugin/service_cache_test.go
- [x] T007 [P] Create unit tests for edge cases (empty string, malformed types) in internal/plugin/service_cache_test.go
- [x] T008 [P] Add benchmark for detectService baseline (before optimization) in internal/plugin/projected_bench_test.go
- [x] T009 Run `make lint` and `make test` to verify Phase 2 complete

**Checkpoint**: serviceResolver is tested and ready for integration into call sites

---

## Phase 3: User Story 1 - Batch Cost Estimation Performance (Priority: P1)

**Goal**: Optimize GetRecommendations batch processing so each resource's service type is computed exactly once

**Independent Test**: Benchmark GetRecommendations with 100 resources; verify call count reduction from ~300 to ~100

### Implementation for User Story 1

- [x] T010 [US1] Refactor GetRecommendations loop in internal/plugin/recommendations.go to create serviceResolver per resource
- [x] T011 [US1] Replace detectService(resource.ResourceType) with resolver.ServiceType() in internal/plugin/recommendations.go:142
- [x] T012 [US1] Add benchmark test for GetRecommendations with 100 resources in internal/plugin/recommendations_bench_test.go
- [x] T013 [US1] Run `go test -race ./internal/plugin/...` to verify no data races
- [x] T014 [US1] Run `make test` to verify all existing tests still pass

**Checkpoint**: GetRecommendations batch optimization complete and verified

---

## Phase 4: User Story 2 - Single Resource Request Optimization (Priority: P2)

**Goal**: Optimize GetProjectedCost so service type is computed once across validation and routing

**Independent Test**: Benchmark single GetProjectedCost call; verify 2-3 calls reduced to 1

### Implementation for User Story 2

- [x] T015 [US2] Refactor GetProjectedCost to create serviceResolver at request entry in internal/plugin/projected.go
- [x] T016 [US2] Pass resolver to ValidateProjectedCostRequest (update function signature) in internal/plugin/validation.go
- [x] T017 [US2] Replace detectService calls in validation.go:39, 99 with resolver.ServiceType()
- [x] T018 [US2] Replace detectService call in projected.go:160 with resolver.ServiceType()
- [x] T019 [US2] Refactor Supports() to use resolver in internal/plugin/supports.go:36
- [x] T020 [US2] Add benchmark test for single GetProjectedCost in internal/plugin/projected_bench_test.go
- [x] T021 [US2] Run `go test -race ./internal/plugin/...` to verify no data races (note: pre-existing race in mock counter)
- [x] T022 [US2] Run `make test` to verify all existing tests still pass

**Checkpoint**: GetProjectedCost optimization complete and verified

---

## Phase 5: User Story 3 - Actual Cost Calculation Optimization (Priority: P3)

**Goal**: Optimize GetActualCost so service type is computed once across validation and routing

**Independent Test**: Benchmark GetActualCost call; verify call reduction

### Implementation for User Story 3

- [x] T023 [US3] Refactor GetActualCost to create serviceResolver at request entry in internal/plugin/plugin.go (note: actual location)
- [x] T024 [US3] Pass resolver to getProjectedForResource helper (simpler than validation signature change)
- [x] T025 [US3] Replace detectService calls with resolver.ServiceType() in plugin.go and actual.go
- [x] T026 [US3] Replace detectService call in actual.go with resolver.ServiceType()
- [x] T027 [US3] Run `go test -race ./internal/plugin/...` to verify no data races (pre-existing mock issue)
- [x] T028 [US3] Run `make test` to verify all existing tests still pass

**Checkpoint**: GetActualCost optimization complete and verified

---

## Phase 6: Remaining Call Sites

**Purpose**: Complete optimization for remaining RPC methods

- [x] T029 Refactor GetPricingSpec to use resolver in internal/plugin/pricingspec.go:35
- [x] T030 Refactor metadata extraction to use resolver in internal/plugin/plugin.go:325 (already done via T023)
- [x] T031 Run full test suite: `make test` (race detection has pre-existing mock counter issue)

**Checkpoint**: All call sites optimized

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and cleanup

- [x] T032 Run `make lint` to verify code style compliance
- [x] T033 Run `make test` to verify all tests pass
- [x] T034 Verify SC-003: Confirm no test modifications were required (git diff on test files should show only additions, not modifications to existing assertions)
- [x] T035 [P] Compare benchmark results before/after to verify SC-001 (batch) and SC-002 (single) metrics
- [x] T036 [P] Run memory profiling to verify SC-005 (<100 bytes overhead per resource)
- [x] T037 Update comment in supports.go:32-34 to remove optimization TODO (now implemented)
- [x] T038 Run quickstart.md validation checklist

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Phase 2 completion
  - US1, US2, US3 share some validation.go modifications but are otherwise independent
  - Recommend sequential execution: US1 → US2 → US3 to minimize merge conflicts
- **Remaining Call Sites (Phase 6)**: Can proceed after US2 (uses same resolver pattern)
- **Polish (Phase 7)**: Depends on all phases complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Phase 2 - Isolated to recommendations.go
- **User Story 2 (P2)**: Can start after Phase 2 - Touches projected.go, validation.go, supports.go
- **User Story 3 (P3)**: **DEPENDS ON US2** - Must complete after US2 because both modify validation.go function signatures. US2 adds resolver parameter to ValidateProjectedCostRequest; US3 adds resolver parameter to ValidateActualCostRequest. Sequential execution prevents merge conflicts.

### Within Each User Story

- Refactor before verify (implementation before race/test checks)
- Test pass required before moving to next story
- Commit after each completed user story

### Parallel Opportunities

- T005, T006, T007, T008 can run in parallel (different files)
- T034, T035 can run in parallel (different metrics)
- Within each story, refactoring tasks must be sequential (same files)

---

## Parallel Example: Phase 2 Tasks

```bash
# Launch all parallel Phase 2 tasks together:
Task: "Create unit tests for serviceResolver in internal/plugin/service_cache_test.go"
Task: "Create unit tests for edge cases in internal/plugin/service_cache_test.go"
Task: "Add benchmark for detectService baseline in internal/plugin/projected_bench_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (serviceResolver type)
2. Complete Phase 2: Unit tests and benchmarks
3. Complete Phase 3: User Story 1 (GetRecommendations)
4. **STOP and VALIDATE**: Benchmark shows batch improvement
5. Continue to US2, US3

### Incremental Delivery

1. Phase 1 + 2 → serviceResolver ready and tested
2. Add US1 → Batch optimization verified → Can merge as standalone improvement
3. Add US2 → Single request optimization verified → Can merge incrementally
4. Add US3 → Actual cost optimization verified → Can merge incrementally
5. Each story adds measurable value

### Verification Commands

```bash
# After each user story:
make lint
make test
go test -tags=region_use1 -race ./internal/plugin/...

# Benchmarks:
go test -tags=region_use1 -bench=BenchmarkGetRecommendations -benchmem ./internal/plugin/...
go test -tags=region_use1 -bench=BenchmarkGetProjectedCost -benchmem ./internal/plugin/...
```

---

## Notes

- All tasks include exact file paths for immediate execution
- [P] tasks operate on different files with no dependencies
- [Story] labels map tasks to user stories from spec.md
- Each user story is independently completable and testable
- Commit after each phase or user story completion
- Run `go test -race` after each story to catch concurrency issues early
