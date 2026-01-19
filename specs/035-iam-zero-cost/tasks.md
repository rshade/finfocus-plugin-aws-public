# Tasks: IAM Zero-Cost Resource Handling

**Feature Branch**: `035-iam-zero-cost`
**Status**: Pending

## Phase 1: Setup
*Goal: Prepare environment for changes.*

- [X] T001 Verify project state and run existing tests to ensure clean baseline in `internal/plugin/`

## Phase 2: Foundational
*Goal: No foundational changes required for this feature.*

## Phase 3: User Story 1 - Estimate IAM Resources
*Goal: Correctly identify and estimate AWS IAM resources as zero cost.*
*Independent Test: Unit tests pass for normalization, support check, and cost estimation.*

### Tests
- [X] T002 [P] [US1] Create unit tests for `normalizeResourceType` covering `aws:iam/*` case-insensitivity in `internal/plugin/projected_test.go`
- [X] T003 [P] [US1] Create unit tests for `Supports` verifying IAM resource support in `internal/plugin/supports_test.go`
- [X] T004 [P] [US1] Create unit tests for `GetProjectedCost` verifying $0 estimate for IAM in `internal/plugin/projected_test.go`

### Implementation
- [X] T005 [P] [US1] Update `ZeroCostServices` map to include "iam" in `internal/plugin/constants.go`
- [X] T006 [US1] Implement `aws:iam/*` case-insensitive prefix normalization in `normalizeResourceType` in `internal/plugin/projected.go`
- [X] T007 [US1] Update `detectService` to map "iam" service type in `internal/plugin/projected.go`
- [X] T008 [US1] Update `Supports` method to handle "iam" service type in `internal/plugin/supports.go`
- [X] T009 [US1] Update `GetProjectedCost` to handle "iam" service type returning $0 cost and no carbon metrics in `internal/plugin/projected.go`

## Phase 4: Verification & Polish
*Goal: Ensure code quality and adherence to standards.*

- [X] T010 Run full test suite `make test` to ensure no regressions
- [X] T011 Run linter `make lint` and fix any issues

## Dependencies

- **US1**: Independent. T006, T007, T008, T009 can be implemented in parallel after T005, but logical flow suggests T005 -> T006 -> T007/T008 -> T009.
- **Tests**: T002-T004 can be written before implementation (TDD) or in parallel.

## Parallel Execution Opportunities

- **US1**: T002, T003, T004 (Tests) can be written in parallel.
- **US1**: T005 (Map Update) is independent.

## Implementation Strategy

1.  **MVP Scope**: Complete all tasks in Phase 3 (US1). This provides the full feature value.
2.  **Order**:
    -   Write tests (T002-T004).
    -   Update validation map (T005).
    -   Implement normalization (T006) and detection (T007).
    -   Implement support check (T008).
    -   Implement cost logic (T009).
    -   Verify (T010-T011).
