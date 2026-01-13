# Tasks: Support target_resources Scope

**Feature Branch**: `014-target-resources-scope`
**Status**: Complete

## Phase 1: Setup
**Goal**: Prepare environment for batch processing implementation.

- [x] T001 Verify `finfocus-spec` v0.4.9 dependency and proto definitions in `internal/plugin/plugin.go`
- [x] T002 Define `ProcessingContext` and `BatchStats` structs in `internal/plugin/recommendations.go`
- [x] T003 [P] Implement `matchesFilter` helper function in `internal/plugin/recommendations.go`

## Phase 2: Foundational (User Story 1 - Batch Resource Analysis)
**Goal**: Enable core batch processing for multiple resources in a single request.
**Independent Test**: `TestGetRecommendations_Batch` passes with mixed resource list.

- [x] T004 [US1] Create `TestGetRecommendations_Batch` in `internal/plugin/plugin_test.go` with 5 mixed resources
- [x] T005 [US1] Implement batch size validation (max 100) in `GetRecommendations` in `internal/plugin/recommendations.go`
- [x] T006 [US1] Refactor `GetRecommendations` to normalize input into `ProcessingContext` in `internal/plugin/recommendations.go`
- [x] T007 [US1] Implement iteration loop over `ProcessingContext.Scope` calling generators in `internal/plugin/recommendations.go`
- [x] T008 [US1] Implement `ResourceId`/`Arn`/`Name` correlation logic in `ResourceRecommendationInfo` population in `internal/plugin/recommendations.go`

## Phase 3: User Story 2 - Filtered Batch Analysis
**Goal**: Apply filtering criteria to batch resources.
**Independent Test**: `TestGetRecommendations_FilteredBatch` passes, returning only matching subset.

- [x] T009 [US2] Create `TestGetRecommendations_FilteredBatch` in `internal/plugin/plugin_test.go` (mixed regions, filter by one)
- [x] T010 [US2] Integrate `matchesFilter` check into the batch iteration loop in `internal/plugin/recommendations.go`
- [x] T011 [US2] Implement provider check (`resource.provider == "aws"`) in loop in `internal/plugin/recommendations.go`

## Phase 4: User Story 3 - Legacy Single-Resource Fallback
**Goal**: Ensure backward compatibility for clients sending only `Filter.Sku`.
**Independent Test**: `TestGetRecommendations_Legacy` passes with empty `target_resources`.

- [x] T012 [US3] Create `TestGetRecommendations_Legacy` in `internal/plugin/plugin_test.go` (empty target, valid filter)
- [x] T013 [US3] Verify normalization logic handles empty `TargetResources` by constructing single-item scope from `Filter` in `internal/plugin/recommendations.go`

## Phase 5: Polish & Cross-Cutting Concerns
**Goal**: Finalize logging and performance.

- [x] T014 Implement summary logging (resources processed, matched, total savings) in `internal/plugin/recommendations.go`
- [x] T015 Run `go mod tidy` and verify all tests pass
- [x] T016 Verify no individual resource logs appear at INFO level (only WARN/ERROR)

## Dependencies
1. T001-T003 (Setup) MUST complete before T006.
2. T006 (Normalization) is prerequisite for T007 and T013.
3. T007 (Iteration) is prerequisite for T010 (Filtering).
4. T003 (matchesFilter) is prerequisite for T010.

## Implementation Strategy
- **MVP**: Complete Phase 1 & 2 to prove batch capability.
- **Incremental**: Add filtering (Phase 3) and legacy safety (Phase 4) sequentially.
- **Observability**: Add logging (Phase 5) last to avoid noise during dev.

## Parallel Execution Examples
- T004 (Tests) and T005 (Validation) can be implemented in parallel.
- T003 (Helper) and T002 (Structs) are independent.
