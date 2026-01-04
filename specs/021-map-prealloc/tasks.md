# Tasks: Pre-allocate Map Capacity for Pricing Indexes

**Input**: Design documents from `/specs/021-map-prealloc/`
**Prerequisites**: plan.md (required), spec.md (required), research.md

**Tests**: No new test tasks - existing `BenchmarkNewClient` benchmarks validate this optimization.

**Organization**: Tasks organized for linear execution since this is a single-file change with benchmark validation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2)
- Include exact file paths in descriptions

## Path Conventions

- **Project type**: Single Go project
- **Modified file**: `internal/pricing/client.go`
- **Validation**: `internal/pricing/client_test.go` (existing benchmarks)

---

## Phase 1: Setup (Baseline Capture)

**Purpose**: Capture benchmark baseline before any code changes

- [ ] T001 Checkout `main` branch and ensure clean working directory
- [ ] T002 Run baseline benchmarks: `go test -tags=region_use1 -bench=BenchmarkNewClient -benchmem -count=10 ./internal/pricing/... > baseline.txt`
- [ ] T003 Record baseline metrics (ns/op, B/op, allocs/op) for comparison

**Checkpoint**: Baseline captured in `baseline.txt` - ready for implementation

---

## Phase 2: Implementation (Core Changes)

**Purpose**: Modify `client.go` with pre-allocated map capacities

**⚠️ Note**: Both user stories are satisfied by the same code change - map pre-allocation reduces both initialization time (US1) and memory churn (US2).

- [ ] T004 [US1] [US2] Add explanatory comment block before map initializations in `internal/pricing/client.go` (line ~192)
- [ ] T005 [US1] [US2] Modify `c.ec2Index = make(map[string]ec2Price)` to `make(map[string]ec2Price, 100000)` with inline comment in `internal/pricing/client.go`
- [ ] T006 [US1] [US2] Modify `c.ebsIndex = make(map[string]ebsPrice)` to `make(map[string]ebsPrice, 50)` with inline comment in `internal/pricing/client.go`
- [ ] T007 [US1] [US2] Modify `c.s3Index = make(map[string]s3Price)` to `make(map[string]s3Price, 100)` with inline comment in `internal/pricing/client.go`
- [ ] T008 [US1] [US2] Modify `c.rdsInstanceIndex = make(map[string]rdsInstancePrice)` to `make(map[string]rdsInstancePrice, 5000)` with inline comment in `internal/pricing/client.go`
- [ ] T009 [US1] [US2] Modify `c.rdsStorageIndex = make(map[string]rdsStoragePrice)` to `make(map[string]rdsStoragePrice, 100)` with inline comment in `internal/pricing/client.go`
- [ ] T010 [US1] [US2] Modify `c.elasticacheIndex = make(map[string]elasticacheInstancePrice)` to `make(map[string]elasticacheInstancePrice, 1000)` with inline comment in `internal/pricing/client.go`

**Checkpoint**: All 6 map initializations updated with capacity hints and documentation

---

## Phase 3: Validation (User Story 1 - Faster Initialization)

**Goal**: Verify initialization time (ns/op) has not regressed

**Independent Test**: Compare ns/op between baseline and feature branch

- [ ] T011 [US1] Run `make lint` to verify code style compliance
- [ ] T012 [US1] Run `make test` to verify all existing tests pass
- [ ] T013 [US1] Run feature benchmarks: `go test -tags=region_use1 -bench=BenchmarkNewClient -benchmem -count=10 ./internal/pricing/... > after.txt`
- [ ] T014 [US1] Compare timing: `benchstat baseline.txt after.txt` and verify ns/op is within 5% of baseline (SC-002)

**Checkpoint**: User Story 1 validated - no initialization time regression

---

## Phase 4: Validation (User Story 2 - Reduced Memory Churn)

**Goal**: Verify allocations (allocs/op) are reduced by at least 10%

**Independent Test**: Compare allocs/op between baseline and feature branch

- [ ] T015 [US2] Verify allocs/op decreased by ≥10% in benchstat output (SC-001)
- [ ] T016 [US2] Verify B/op increase is ≤10% in benchstat output (SC-005)
- [ ] T017 [US2] Document benchmark comparison results for PR description

**Checkpoint**: User Story 2 validated - memory allocations reduced

---

## Phase 5: Polish & Documentation

**Purpose**: Final cleanup and PR preparation

- [ ] T018 [P] Run `go build -tags=region_use1 ./cmd/pulumicost-plugin-aws-public` to verify build succeeds
- [ ] T019 [P] Verify markdown files are linted: `npx markdownlint specs/021-map-prealloc/*.md`
- [ ] T020 Prepare PR description with benchmark comparison table from benchstat output

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies - capture baseline first
- **Phase 2 (Implementation)**: Depends on baseline capture (Phase 1)
- **Phase 3 (US1 Validation)**: Depends on implementation (Phase 2)
- **Phase 4 (US2 Validation)**: Depends on implementation (Phase 2) - can run in parallel with Phase 3
- **Phase 5 (Polish)**: Depends on all validation passing

### Task Dependencies

```text
T001 → T002 → T003 (baseline capture sequence)
           ↓
T004 → T005 → T006 → T007 → T008 → T009 → T010 (implementation sequence)
           ↓
     ┌─────┴─────┐
     ↓           ↓
T011→T012→T013→T014   T015→T016→T017 (validation - can partially parallel)
     ↓           ↓
     └─────┬─────┘
           ↓
    T018 ∥ T019 → T020 (polish - T018/T019 parallel)
```

### Parallel Opportunities

- **Phase 5**: T018 and T019 can run in parallel (different tools, no file conflicts)
- **Cross-phase**: Once implementation is done, US1 and US2 validation can proceed in parallel since they analyze the same benchmark output file

---

## Parallel Example: Validation Phase

```bash
# After T013 completes (benchmark run), these validations can run in parallel:
# US1 check: Verify timing
grep "ns/op" benchstat_output.txt

# US2 check: Verify allocations
grep "allocs/op" benchstat_output.txt
```

---

## Implementation Strategy

### MVP First (Single PR)

1. Complete Phase 1: Capture baseline
2. Complete Phase 2: Implement all 6 map capacity changes
3. Complete Phase 3: Validate US1 (timing)
4. Complete Phase 4: Validate US2 (allocations)
5. Complete Phase 5: Polish and submit PR

### Validation Checklist

Before submitting PR, verify:

- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] ns/op within 5% of baseline (or improved)
- [ ] allocs/op reduced by ≥10%
- [ ] B/op increase ≤10%
- [ ] Benchmark comparison table in PR description

---

## Notes

- All implementation tasks (T004-T010) modify the same file (`internal/pricing/client.go`) so they cannot run in parallel
- No new test code required - existing `BenchmarkNewClient` validates both user stories
- The 6 capacity values are from research.md decision table
- Both user stories share implementation but have distinct validation criteria
- This is a low-risk change - easy to revert if unexpected issues arise
