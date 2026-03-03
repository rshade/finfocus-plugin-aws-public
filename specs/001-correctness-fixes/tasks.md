# Tasks: Correctness Fixes Batch

**Input**: Design documents from `/specs/001-correctness-fixes/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md

**Scope**: 3 active issues (#294, #293, #289). Issue #292 deferred
pending rshade/finfocus-spec#381.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 = Zero-Cost Resources, US3 = Lint Compliance

## Phase 1: Setup

**Purpose**: Ensure lint configuration is tracked in version control

- [x] T001 Verify `.golangci.yml` is committed and tracked in git (FR-004)

**Checkpoint**: `.golangci.yml` appears in `git ls-files` output

---

## Phase 2: Foundational (Zero-Cost Registry Extension)

**Purpose**: Extend the zero-cost resource maps that both projected
and actual cost paths depend on

- [x] T002 Add `launchtemplate` and `launchconfiguration` to `ZeroCostServices` map and add `ec2/launchtemplate` and `ec2/launchconfiguration` to `ZeroCostPulumiPatterns` map in `internal/plugin/constants.go` (FR-001, research R3)

**Checkpoint**: `IsZeroCostService("launchtemplate")` returns true;
`normalizeResourceType("aws:ec2/launchTemplate:LaunchTemplate")`
resolves to a zero-cost service

---

## Phase 3: User Story 1 - Zero-Cost Resources Return $0 (P1)

**Goal**: LaunchTemplate/Config return $0 via GetProjectedCost; VPC,
Subnet, SecurityGroup, and IAM return $0 via GetActualCost without
"tags missing sku" errors

**Independent Test**: `go test -v ./internal/plugin/... -run TestZeroCost`
passes; no error log entries for zero-cost ARNs

### Tests for User Story 1

- [x] T003 [P] [US1] Add table-driven test for LaunchTemplate and LaunchConfiguration GetProjectedCost returning $0 in `internal/plugin/projected_test.go` - verify cost_per_month=0, billing_detail contains "zero-cost" or "no direct cost"
- [x] T004 [P] [US1] Add table-driven test for VPC, Subnet, SecurityGroup, and IAM ARN GetActualCost returning empty results (no error) in `internal/plugin/actual_test.go` - verify no "tags missing sku" error, response is empty `[]*pbc.ActualCostResult{}`
- [x] T005 [P] [US1] Add edge case test: LaunchTemplate with `instanceType` tag still returns $0 in `internal/plugin/projected_test.go`

### Implementation for User Story 1

- [x] T006 [US1] Add `isZeroCostResource()` check to `ValidateActualCostRequest()` in `internal/plugin/validation.go` - mirror the pattern from `ValidateProjectedCostRequest()` (lines 60-86): check after ARN parsing but before SKU extraction, skip SDK validation for zero-cost resources (research R2)
- [x] T007 [US1] Add zero-cost check to `parseResourceFromARN()` in `internal/plugin/validation.go` - return valid `ResourceDescriptor` with empty SKU when ARN resource type maps to a zero-cost service (research R2)
- [x] T008 [US1] Add zero-cost routing in `GetActualCost()` in `internal/plugin/actual.go` - after validation, detect zero-cost resources via `serviceResolver` and return empty `[]*pbc.ActualCostResult{}` instead of calling `getProjectedForResource()` (FR-002, FR-003)
- [x] T009 [US1] Run `make test` to validate US1 changes pass with zero failures

**Checkpoint**: All zero-cost resource types return $0 through both
projected and actual cost paths with zero error log entries (SC-001)

---

## Phase 4: User Story 3 - Lint Compliance (P2)

**Goal**: `make lint` reports zero issues; all high-priority (8) and
medium-priority (73) lint violations fixed

**Independent Test**: `make lint` exits with code 0 and zero issues

### High Priority - Code Correctness (8 issues)

- [x] T010 [US3] Fix errcheck: handle `proto.Clone()` error returns in `internal/plugin/recommendations.go` - wrap with error check or assign to `_` with justification comment (FR-005)
- [x] T011 [P] [US3] Fix errorlint: replace `err == io.EOF` with `errors.Is(err, io.EOF)` in `internal/carbon/gpu_specs.go` (FR-006)
- [x] T012 [P] [US3] Fix errorlint: replace `err == io.EOF` with `errors.Is(err, io.EOF)` in `internal/carbon/instance_specs.go` (FR-006)
- [x] T013 [P] [US3] Fix errorlint: replace `err == io.EOF` with `errors.Is(err, io.EOF)` in `internal/carbon/storage_specs.go` (FR-006)
- [x] T014 [P] [US3] Fix errorlint: replace `err == io.EOF` with `errors.Is(err, io.EOF)` in `tools/generate-carbon-data/main.go` (FR-006)
- [x] T015 [US3] Fix wastedassign: remove unused assignment in `internal/plugin/projected.go` (line ~1607)
- [x] T016 [US3] Fix nilnil: address simultaneous nil error and invalid value return in affected function

### Medium Priority - Code Quality (73 issues)

- [x] T017 [US3] Fix protogetter: replace all direct proto field accesses with getter methods across `internal/plugin/projected.go`, `internal/plugin/enrichment.go`, `internal/plugin/plugin.go`, and other affected files (~50 violations) (FR-007)
- [x] T018 [P] [US3] Fix revive: resolve naming stuttering (`CarbonEstimator` -> `Estimator`, `PricingClient` -> `Client`), unused parameters, and builtin shadowing (`min`/`max` variable names) across `internal/carbon/` and `internal/pricing/`
- [x] T019 [P] [US3] Fix usestdlibvars: replace literal `"GET"` with `http.MethodGet` in `cmd/metrics-aggregator/`
- [x] T020 [P] [US3] Fix noctx: add context to HTTP requests missing `context.Context` parameter
- [x] T021 [P] [US3] Fix whitespace, nonamedreturns, and godoclint issues across affected files
- [x] T022 [US3] Add justified exclusions to `.golangci.yml` for intentional patterns: `gochecknoglobals` for config maps in `internal/`, `forbidigo` for `tools/` directory, `mnd` for common numeric patterns

### Validation

- [x] T023 [US3] Run `make lint` and verify zero issues reported (FR-008)
- [x] T024 [US3] Run `make test` and verify zero failures (FR-009)

**Checkpoint**: `make lint` and `make test` both pass with zero
issues/failures (SC-002, SC-003)

---

## Phase 5: Polish and Cross-Cutting Concerns

**Purpose**: Final validation and documentation

- [x] T025 Run `make lint` and `make test` together as final validation
- [x] T026 Run `npx markdownlint` on all modified markdown files
- [x] T027 Verify no regressions: `go test -tags=region_use1 ./internal/pricing/...` (pre-existing pricing data gaps confirmed not related to correctness fixes)

---

## Dependencies and Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - start immediately
- **Foundational (Phase 2)**: Depends on Phase 1
- **US1 (Phase 3)**: Depends on Phase 2 (needs zero-cost maps updated)
- **US3 (Phase 4)**: Depends on Phase 1 only (lint fixes are
  independent of zero-cost logic, but should run after US1 to avoid
  merge conflicts in shared files like projected.go)
- **Polish (Phase 5)**: Depends on Phase 3 and Phase 4

### User Story Dependencies

- **US1 (P1)**: Can start after Phase 2 - no dependencies on US3
- **US3 (P2)**: Can start after Phase 1 - recommend after US1 to
  avoid conflicts in `projected.go` and `validation.go`

### Within Each User Story

- Tests (T003-T005) can run in parallel
- Implementation (T006-T008) should be sequential: validation.go
  before actual.go
- Lint high-priority fixes (T011-T014) can run in parallel
- Lint medium-priority fixes (T018-T021) can run in parallel

### Parallel Opportunities

```text
Phase 2:  T002 (constants.go)
Phase 3:  T003 ─┐
          T004 ─┼─ parallel tests
          T005 ─┘
          T006 → T007 → T008 → T009 (sequential implementation)
Phase 4:  T011 ─┐
          T012 ─┤
          T013 ─┼─ parallel errorlint fixes
          T014 ─┘
          T018 ─┐
          T019 ─┤
          T020 ─┼─ parallel medium-priority fixes
          T021 ─┘
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001)
2. Complete Phase 2: Foundational (T002)
3. Complete Phase 3: User Story 1 (T003-T009)
4. **STOP and VALIDATE**: Zero-cost resources return $0, no errors
5. This alone fixes #294 and #293

### Full Delivery

1. Complete MVP (above)
2. Complete Phase 4: User Story 3 (T010-T024)
3. Complete Phase 5: Polish (T025-T027)
4. This additionally fixes #289 and unblocks #291

---

## Notes

- US2 (#292 sparse properties) is DEFERRED pending
  rshade/finfocus-spec#381
- T017 (protogetter) is the largest single task (~50 mechanical
  replacements) - consider splitting by file if needed
- T018 (revive naming) may require updating test files that reference
  renamed types
- `make lint` has extended timeout (>5 minutes) per CLAUDE.md
