# Tasks: Carbon Emission Estimation

**Input**: Design documents from `/specs/015-carbon-estimation/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Tests are included as per Constitution II (Testing Discipline) requirement.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md structure (existing Go gRPC plugin):

- `internal/carbon/` - NEW carbon estimation module
- `internal/plugin/` - MODIFY existing plugin code
- `data/` - NEW embedded data files
- `tools/generate-ccf-data/` - NEW data generation tool

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, dependency updates, and data acquisition

- [x] T001 Update go.mod to use finfocus-spec v0.4.10 via `go get github.com/rshade/finfocus-spec@v0.4.10`
- [x] T002 Run `go mod tidy` to update go.sum
- [x] T003 Create carbon package directory structure at internal/carbon/
- [x] T004 [P] Download CCF instance data CSV to data/ccf_instance_specs.csv from cloud-carbon-coefficients repo
- [x] T005 [P] Add CCF attribution to README.md (Attribution section) per Apache 2.0 license requirements - no NOTICE file exists in this project

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core carbon estimation infrastructure that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T006 [P] Create constants file internal/carbon/constants.go with AWS_PUE (1.135), DefaultUtilization (0.50), HoursPerMonth (730.0)
- [x] T007 [P] Create grid emission factors map internal/carbon/grid_factors.go with 12 AWS regions plus default fallback
- [x] T008 Create InstanceSpec struct and embed CSV in internal/carbon/instance_specs.go with go:embed directive
- [x] T009 Implement CSV parsing logic with sync.Once in internal/carbon/instance_specs.go
- [x] T010 Implement GetInstanceSpec(instanceType string) lookup function in internal/carbon/instance_specs.go
- [x] T011 [P] Create unit tests for instance spec lookup in internal/carbon/instance_specs_test.go
- [x] T012 Create CarbonEstimator interface and Estimator struct in internal/carbon/estimator.go
- [x] T013 Implement CalculateCarbonGrams function with CCF formula in internal/carbon/estimator.go
- [x] T014 Implement EstimateCarbonGrams method combining lookup + calculation in internal/carbon/estimator.go
- [x] T015 [P] Create unit tests for carbon formula in internal/carbon/estimator_test.go (table-driven with known values)
- [x] T016 Create getUtilization helper function with priority logic in internal/carbon/utilization.go

**Checkpoint**: Foundation ready - carbon estimation module is complete and tested in isolation

---

## Phase 3: User Story 1 - View Carbon Footprint for EC2 Instance (Priority: P1) 🎯 MVP

**Goal**: Return carbon footprint in gCO2e alongside financial cost for EC2 instances

**Independent Test**: Request cost estimate for t3.micro in us-east-1, verify response includes ImpactMetrics with METRIC_KIND_CARBON_FOOTPRINT

### Tests for User Story 1

- [x] T017 [P] [US1] Add carbon metric assertions to existing EC2 tests in internal/plugin/projected_test.go
- [x] T018 [P] [US1] Create test for carbon = 0 when instance type unknown in internal/plugin/projected_test.go
- [x] T019 [P] [US1] Create test verifying region affects carbon value (eu-north-1 vs us-east-1) in internal/plugin/projected_test.go

### Implementation for User Story 1

- [x] T020 [US1] Add CarbonEstimator field to AWSPublicPlugin struct in internal/plugin/plugin.go
- [x] T021 [US1] Initialize CarbonEstimator in NewAWSPublicPlugin constructor in internal/plugin/plugin.go
- [x] T022 [US1] Modify estimateEC2 to call carbon estimator after financial cost in internal/plugin/projected.go
- [x] T023 [US1] Add ImpactMetrics to GetProjectedCostResponse in estimateEC2 when carbon calculation succeeds in internal/plugin/projected.go
- [x] T024 [US1] Handle unknown instance types gracefully (return carbon=0, log warning) in internal/plugin/projected.go
- [x] T025 [US1] Add zerolog logging for carbon calculations in internal/plugin/projected.go

**Checkpoint**: EC2 cost estimates now include carbon footprint. US1 is fully functional and testable.

---

## Phase 4: User Story 2 - Discovery of Carbon Estimation Capabilities (Priority: P2)

**Goal**: Advertise carbon footprint capability via Supports() method's supported_metrics field

**Independent Test**: Call Supports() for EC2 resource, verify supported_metrics includes METRIC_KIND_CARBON_FOOTPRINT

### Tests for User Story 2

- [x] T026 [P] [US2] Add test for supported_metrics containing METRIC_KIND_CARBON_FOOTPRINT for EC2 in internal/plugin/supports_test.go
- [x] T027 [P] [US2] Add test for supported_metrics NOT containing carbon for unsupported types (DynamoDB) in internal/plugin/supports_test.go

### Implementation for User Story 2

- [x] T028 [US2] Modify Supports() to return SupportedMetrics field for EC2 resources in internal/plugin/supports.go
- [x] T029 [US2] Add helper function to determine supported metrics by resource type in internal/plugin/supports.go
- [x] T030 [US2] Ensure unsupported resource types return empty supported_metrics in internal/plugin/supports.go

**Checkpoint**: FinFocus core can now discover which plugins support carbon estimation. US2 is fully functional.

---

## Phase 5: User Story 3 - Custom Utilization Override (Priority: P3)

**Goal**: Allow users to specify custom CPU utilization percentage to improve carbon estimate accuracy

**Independent Test**: Send request with utilization_percentage=0.8, verify carbon estimate is higher than default

### Tests for User Story 3

- [x] T031 [P] [US3] Add test for request-level utilization_percentage override in internal/plugin/projected_test.go
- [x] T032 [P] [US3] Add test for per-resource utilization_percentage override in internal/plugin/projected_test.go
- [x] T033 [P] [US3] Add test for utilization priority (per-resource > request > default) in internal/plugin/projected_test.go
- [x] T034 [P] [US3] Add test for utilization clamping to 0.0-1.0 range in internal/carbon/utilization_test.go

### Implementation for User Story 3

- [x] T035 [US3] Update estimateEC2 to extract utilization from GetProjectedCostRequest in internal/plugin/projected.go
- [x] T036 [US3] Update estimateEC2 to extract per-resource utilization from ResourceDescriptor in internal/plugin/projected.go
- [x] T037 [US3] Implement utilization priority logic (per-resource > request > default) in internal/plugin/projected.go
- [x] T038 [US3] Add utilization clamping to valid range (0.0-1.0) in internal/carbon/utilization.go

**Checkpoint**: Users can now customize utilization for more accurate estimates. US3 is fully functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, documentation, and cleanup

- [x] T039 [P] Run `make lint` and fix any linting issues across all new files
- [x] T040 [P] Run `make test` and ensure all tests pass
- [x] T041 [P] Verify single-region build compiles: `go build -tags region_use1 ./cmd/finfocus-plugin-aws-public`
- [x] T042 [P] Update CLAUDE.md with carbon estimation documentation
- [x] T043 Validate quickstart.md scenarios work with actual build
- [x] T044 [P] Add edge case test for GPU instance types (should still return financial cost) in internal/plugin/projected_test.go
- [x] T045 [P] Document GPU limitation in CLAUDE.md under "Cost Estimation Scope" table (GPU power consumption not included in v1)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - US1 (P1) can start immediately after Foundational
  - US2 (P2) can start after Foundational (independent of US1)
  - US3 (P3) can start after Foundational (independent of US1/US2)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Depends only on Foundational - Core MVP
- **User Story 2 (P2)**: Depends only on Foundational - Can run parallel to US1
- **User Story 3 (P3)**: Depends only on Foundational - Can run parallel to US1/US2

### Within Each User Story

- Tests MUST be written first and FAIL before implementation
- Implementation tasks in order: struct changes → method changes → logging
- Story complete before moving to next priority

### Parallel Opportunities

**Phase 1 (Setup)**:

- T004 and T005 can run in parallel

**Phase 2 (Foundational)**:

- T006, T007 can run in parallel (different files)
- T011, T015 can run in parallel (test files)

**Phase 3-5 (User Stories)**:

- All user stories can run in parallel after Foundational
- Test tasks within each story can run in parallel

---

## Parallel Example: Phase 2 Foundational

```bash
# Launch parallel constant/data files:
Task: "Create constants file internal/carbon/constants.go"
Task: "Create grid emission factors map internal/carbon/grid_factors.go"

# After instance_specs.go complete, launch parallel tests:
Task: "Create unit tests for instance spec lookup in internal/carbon/instance_specs_test.go"
Task: "Create unit tests for carbon formula in internal/carbon/estimator_test.go"
```

## Parallel Example: User Stories

```bash
# After Foundational phase completes, all three user stories can start in parallel:
# Developer A: User Story 1 (T017-T025)
# Developer B: User Story 2 (T026-T030)
# Developer C: User Story 3 (T031-T038)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T005)
2. Complete Phase 2: Foundational (T006-T016)
3. Complete Phase 3: User Story 1 (T017-T025)
4. **STOP and VALIDATE**: Run `make test`, verify carbon metrics in EC2 response
5. Deploy/demo if ready - Feature delivers value at this point

### Incremental Delivery

1. Setup + Foundational → Carbon module tested in isolation
2. Add User Story 1 → EC2 returns carbon metrics (MVP!)
3. Add User Story 2 → Core engine can discover carbon capability
4. Add User Story 3 → Users can customize utilization
5. Each story adds value without breaking previous stories

### Recommended Order (Single Developer)

1. T001-T005 (Setup)
2. T006-T016 (Foundational)
3. T017-T025 (US1 - MVP)
4. T026-T030 (US2)
5. T031-T038 (US3)
6. T039-T044 (Polish)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Total: 45 tasks (5 Setup + 11 Foundational + 9 US1 + 5 US2 + 8 US3 + 7 Polish)
