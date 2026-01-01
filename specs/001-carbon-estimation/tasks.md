# Tasks: Comprehensive Carbon Estimation Expansion

**Input**: Design documents from `/specs/001-carbon-estimation/`  
**Prerequisites**: plan.md (completed), spec.md (8 user stories), research.md (complete), data-model.md (complete), contracts/ (gRPC contract defined), quickstart.md (complete)

**Tests**: Unit tests required per constitution (Testing Discipline II). Table-driven tests for all carbon estimators.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

\n## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

\n## Path Conventions

- **Single project**: `internal/carbon/`, `test/fixtures/`
- Tests: `internal/carbon/*_test.go`, `test/integration/*_test.go`
- Embedded data: `internal/carbon/data/*.csv`

---

\n###\n### Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure for carbon estimation feature

- [X] T001 Create embedded data directory structure at internal/carbon/data/
- [X] T002 [P] Create data directory placeholder in internal/carbon/data/
- [X] T003 [P] Create test fixtures directory at test/fixtures/carbon/
- [X] T004 Verify existing carbon package structure is compatible with new estimators

---

\n## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Create types.go with service configuration structs at internal/carbon/types.go
- [X] T006 Add constants for carbon calculation in internal/carbon/constants.go (AWSPUE, VCPUPer1792MB, etc.)
- [X] T007 [P] Create GPU specs embedded CSV at internal/carbon/data/gpu_specs.csv
- [X] T008 [P] Create storage specs embedded CSV at internal/carbon/data/storage_specs.csv
- [X] T009 Create gpu_specs.go with GPUSpec struct and parser at internal/carbon/gpu_specs.go
- [X] T010 Create storage_specs.go with StorageSpec struct and parser at internal/carbon/storage_specs.go
- [X] T011 Verify all embedded CSV files compile with //go:embed directive

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

\n## Phase 3: User Story 1 - GPU Instance Carbon Estimation (Priority: P1) 🎯 MVP

**Goal**: Add GPU power consumption support to EC2 carbon estimation for ML/AI workloads

**Independent Test**: Request carbon estimate for p4d.24xlarge instance and validate GPU power (8x A100 @ 400W each) is included in total carbon

## Tests for User Story 1

- [X] T012 [P] [US1] Write table-driven test for GPU power lookup in internal/carbon/gpu_specs_test.go
- [X] T013 [P] [US1] Write test for EC2 estimator with GPU power in internal/carbon/estimator_test.go

## Implementation for User Story 1

- [X] T014 [US1] Add GetGPUSpec function to gpu_specs.go with sync.Once initialization
- [X] T015 [US1] Add GPU power calculation to EC2 estimator in internal/carbon/estimator.go
- [X] T016 [US1] Update EstimateCarbonGrams to include GPU power in internal/carbon/estimator.go
- [X] T017 [US1] Add GPU power to billing_detail breakdown in internal/plugin/carbon.go

**Checkpoint**: GPU instance carbon estimation should be fully functional and testable independently

---

\n## Phase 4: User Story 2 - EBS Storage Carbon Estimation (Priority: P1)

**Goal**: Add carbon footprint estimation for EBS volumes using storage technology coefficients and replication factors

**Independent Test**: Request carbon estimate for 100GB gp3 SSD volume and validate estimate uses SSD coefficient (1.2 Wh/TB) with 2× replication

## Tests for User Story 2

- [X] T018 [P] [US2] Write table-driven test for storage spec lookup in internal/carbon/storage_specs_test.go
- [X] T019 [P] [US2] Write test for EBS estimator in internal/carbon/ebs_estimator_test.go

## Implementation for User Story 2

- [X] T020 [US2] Create EBSEstimator struct and NewEBSEstimator in internal/carbon/ebs_estimator.go
- [X] T021 [US2] Implement EstimateCarbonGrams for EBS in internal/carbon/ebs_estimator.go
- [X] T022 [US2] Update gRPC router to dispatch aws:ebs/volume to EBS estimator in internal/plugin/router.go
- [X] T023 [US2] Add carbon_footprint to billing_detail for EBS responses in internal/plugin/carbon.go

**Checkpoint**: EBS storage carbon estimation should be fully functional and testable independently

---

\n## Phase 5: User Story 3 - S3 Storage Carbon Estimation (Priority: P2)

**Goal**: Add carbon footprint estimation for S3 storage with different storage classes and replication factors

**Independent Test**: Request carbon estimate for 100GB S3 STANDARD storage and validate estimate uses SSD coefficient (1.2 Wh/TB) with 3× replication

## Tests for User Story 3

- [X] T024 [P] [US3] Write test for S3 estimator in internal/carbon/s3_estimator_test.go

## Implementation for User Story 3

- [X] T025 [US3] Create S3Estimator struct and NewS3Estimator in internal/carbon/s3_estimator.go
- [X] T026 [US3] Implement EstimateCarbonGrams for S3 in internal/carbon/s3_estimator.go
- [X] T027 [US3] Update gRPC router to dispatch aws:s3/bucket to S3 estimator in internal/plugin/router.go
- [X] T028 [US3] Add carbon_footprint to billing_detail for S3 responses in internal/plugin/carbon.go

**Checkpoint**: S3 storage carbon estimation should be fully functional and testable independently

---

\n## Phase 6: User Story 4 - Lambda Function Carbon Estimation (Priority: P2)

**Goal**: Add carbon footprint estimation for Lambda functions based on memory, duration, invocations, and architecture

**Independent Test**: Request carbon estimate for Lambda function (1792MB memory, 500ms duration, 1M invocations) and validate estimate reflects vCPU equivalent

## Tests for User Story 4

- [X] T029 [P] [US4] Write table-driven test for Lambda estimator in internal/carbon/lambda_estimator_test.go

## Implementation for User Story 4

- [X] T030 [US4] Create LambdaEstimator struct and NewLambdaEstimator in internal/carbon/lambda_estimator.go
- [X] T031 [US4] Implement EstimateCarbonGrams for Lambda in internal/carbon/lambda_estimator.go
- [X] T032 [US4] Add ARM64 efficiency factor (0.80) in internal/carbon/lambda_estimator.go
- [X] T033 [US4] Update gRPC router to dispatch aws:lambda/function to Lambda estimator in internal/plugin/router.go
- [X] T034 [US4] Add carbon_footprint to billing_detail for Lambda responses in internal/plugin/carbon.go

**Checkpoint**: Lambda function carbon estimation should be fully functional and testable independently

---

\n## Phase 7: User Story 5 - RDS Instance Carbon Estimation (Priority: P2)

**Goal**: Add composite carbon estimation for RDS instances (compute EC2-equivalent + storage EBS-equivalent) with Multi-AZ support

**Independent Test**: Request carbon estimate for RDS instance (db.m5.large, 100GB storage, Multi-AZ) and validate compute + storage components summed correctly

## Tests for User Story 5

- [X] T035 [P] [US5] Write table-driven test for RDS estimator in internal/carbon/rds_estimator_test.go

## Implementation for User Story 5

- [X] T036 [US5] Create RDSEstimator struct and NewRDSEstimator in internal/carbon/rds_estimator.go
- [X] T037 [US5] Implement EstimateCarbonGrams for RDS (compute + storage) in internal/carbon/rds_estimator.go
- [X] T038 [US5] Add Multi-AZ multiplier (2×) for compute and storage in internal/carbon/rds_estimator.go
- [X] T039 [US5] Update gRPC router to dispatch aws:rds/instance to RDS estimator in internal/plugin/router.go
- [X] T040 [US5] Add carbon_footprint to billing_detail for RDS responses in internal/plugin/carbon.go

**Checkpoint**: RDS instance carbon estimation should be fully functional and testable independently

---

\n## Phase 8: User Story 6 - DynamoDB Table Carbon Estimation (Priority: P2)

**Goal**: Add carbon footprint estimation for DynamoDB tables based on storage consumption (SSD, 3× replication)

**Independent Test**: Request carbon estimate for DynamoDB table (50GB storage) and validate estimate matches S3 Standard methodology

## Tests for User Story 6

- [X] T041 [P] [US6] Write table-driven test for DynamoDB estimator in internal/carbon/dynamodb_estimator_test.go

## Implementation for User Story 6

- [X] T042 [US6] Create DynamoDBEstimator struct and NewDynamoDBEstimator in internal/carbon/dynamodb_estimator.go
- [X] T043 [US6] Implement EstimateCarbonGrams for DynamoDB in internal/carbon/dynamodb_estimator.go
- [X] T044 [US6] Update gRPC router to dispatch aws:dynamodb/table to DynamoDB estimator in internal/plugin/router.go
- [X] T045 [US6] Add carbon_footprint to billing_detail for DynamoDB responses in internal/plugin/carbon.go

**Checkpoint**: DynamoDB table carbon estimation should be fully functional and testable independently

---

\n## Phase 9: User Story 7 - Embodied Carbon Estimation (Priority: P3)

**Goal**: Add embodied carbon (manufacturing) to EC2 carbon estimates, amortized over server lifespan

**Independent Test**: Request carbon estimate with embodied carbon enabled and validate response includes separate operational and embodied carbon values

## Tests for User Story 7

- [X] T046 [P] [US7] Write test for embodied carbon calculator in internal/carbon/embodied_carbon_test.go
- [X] T047 [P] [US7] Write integration test for EC2 estimator with embodied carbon in internal/carbon/estimator_test.go

## Implementation for User Story 7

- [X] T048 [US7] Create EmbodiedCarbonConfig struct in internal/carbon/types.go
- [X] T049 [US7] Create embodied_carbon.go with CalculateEmbodiedCarbonGrams function
- [X] T050 [US7] Add embodied carbon calculation to EC2 estimator in internal/carbon/estimator.go
- [X] T051 [US7] Add operational/embodied breakdown to billing_detail in internal/plugin/carbon.go
- [X] T052 [US7] Update Supports() to advertise embodied carbon capability in internal/plugin/supports.go

**Checkpoint**: Embodied carbon estimation should be fully functional and testable independently

---

\n## Phase 10: User Story 8 - Grid Factor Update Process (Priority: P3)

**Goal**: Create automated or semi-automated process to update regional grid emission factors annually

**Independent Test**: Run grid factor update tool and validate it fetches data from CCF repository and produces valid grid factor data

## Implementation for User Story 8

- [X] T053 [US8] Create grid factor update tool in tools/update-grid-factors/main.go
- [X] T054 [US8] Implement fetch from CCF repository in tools/update-grid-factors/main.go
- [X] T055 [US8] Add validation for grid factor ranges (0.0 to 2.0 metric tons CO2e/kWh) in tools/update-grid-factors/main.go
- [X] T056 [US8] Implement auto-update of GridEmissionFactors map in tools/update-grid-factors/main.go
- [X] T057 [US8] Add validation tests for grid factor updates in internal/carbon/grid_factors_test.go
- [X] T058 [US8] Document grid factor update process in docs/grid-factor-updates.md
- [X] T059 [US8] Add calendar reminder notes in docs/grid-factor-updates.md

**Checkpoint**: Grid factor update process should be documented and executable

---

\n## Phase 11: EKS Control Plane Carbon (Priority: P2 - Part of RDS/Lambda/DynamoDB phase)

**Goal**: Return zero carbon for EKS control plane and document worker node estimation

**Independent Test**: Request carbon estimate for EKS cluster and validate response returns zero carbon with documentation message

## Tests for User Story EKS

- [X] T060 [P] Write test for EKS estimator in internal/carbon/eks_estimator_test.go

### Implementation for EKS

- [X] T061 Create EKSEstimator struct and NewEKSEstimator in internal/carbon/eks_estimator.go
- [X] T062 Implement EstimateCarbonGrams to return 0 carbon and documentation in internal/carbon/eks_estimator.go
- [X] T063 Update gRPC router to dispatch aws:eks/cluster to EKS estimator in internal/plugin/router.go
- [X] T064 Update Supports() for EKS to exclude carbon metric in internal/plugin/supports.go

**Checkpoint**: EKS carbon estimation should be fully functional and testable independently

---

\n## Phase 12: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T065 Update Supports() method to advertise METRIC_KIND_CARBON_FOOTPRINT for all services in internal/plugin/supports.go
- [X] T066 [P] Create integration test for gRPC service with carbon estimation in test/integration/carbon_estimation_test.go
- [X] T067 [P] Create concurrent access test for all estimators (100+ goroutines) in test/integration/concurrent_access_test.go
- [X] T068 [P] Add performance benchmarks to ensure <100ms latency target in test/benchmark/carbon_estimation_bench_test.go
- [X] T069 [P] Update documentation in docs/carbon-estimation.md with usage examples
- [X] T070 [P] Update README.md with carbon estimation feature summary
- [X] T071 [P] Run make lint and fix all issues
- [X] T072 [P] Run make test and ensure all tests pass
- [X] T073 [P] Run markdownlint on all documentation files
- [X] T074 Verify binary size <250MB constraint after build

---

\n## Dependencies & Execution Order

\n###\n### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-11)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Phase 12)**: Depends on all desired user stories being complete

\n###\n### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 4 (P2)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 5 (P2)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 6 (P2)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 7 (P3)**: Can start after Foundational (Phase 2) - Depends on US1 (EC2 estimator) for embodied carbon integration
- **User Story 8 (P3)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **EKS**: Can start after Foundational (Phase 2) - No dependencies on other stories

\n###\n### Within Each User Story

- Tests MUST be written before implementation
- Models/embedded data before estimators
- Estimators before gRPC integration
- Story complete before moving to next priority

---

\n## Parallel Opportunities

\n###\n### Phase 1: Setup
- T001-T004: Can run in parallel (different directories)

\n###\n### Phase 2: Foundational
- T005-T006: Can run in parallel (different files)
- T007-T008: Can run in parallel (different CSV files)
- T009-T010: Can run in parallel (different Go files)

\n###\n### Phase 3: User Story 1 (GPU)
- T012-T013: Can run in parallel (different test files)
- T014-T015: Must be sequential (T014 before T015)
- T016-T017: Can run in parallel after T015-T016 complete

\n###\n### Phase 4: User Story 2 (EBS)
- T018-T019: Can run in parallel (different test files)
- T020-T023: Must be sequential (T020 before T021, T021 before T022, etc.)

\n###\n### Phase 5: User Story 3 (S3)
- T024-T025: Can run in parallel (test before implementation)
- T025-T028: Must be sequential (T025 before T026, etc.)

\n###\n### Phase 6: User Story 4 (Lambda)
- T029-T034: Can run in parallel (test before implementation)
- T030-T032: Can run in parallel (different estimator logic)
- T033-T034: Must be sequential (T032,T033 before T034)

\n###\n### Phase 7: User Story 5 (RDS)
- T035-T040: Can run in parallel (test before implementation)
- T036-T040: Must be sequential (T036 before T037, etc.)

\n###\n### Phase 8: User Story 6 (DynamoDB)
- T041-T045: Can run in parallel (test before implementation)
- T042-T045: Must be sequential (T042 before T043, etc.)

\n###\n### Phase 9: User Story 7 (Embodied Carbon)
- T046-T047: Can run in parallel (different test files)
- T048-T051: Must be sequential (T048 before T049, T049 before T050, etc.)

\n###\n### Phase 10: User Story 8 (Grid Factor Update)
- T053-T059: Can run in parallel (different implementation tasks)

\n###\n### Phase 11: EKS
- T060-T064: Can run in parallel (test before implementation)
- T061-T064: Must be sequential (T061 before T062, etc.)

\n###\n### Phase 12: Polish
- T065-T070: Can run in parallel (different documentation files)
- T066-T074: Can run in parallel (different test/fix tasks)

\n###\n### Multiple User Stories
Once Foundational phase completes, all user stories can be worked on in parallel by different team members:
- Developer A: User Story 1 (GPU) - P1 (MVP candidate)
- Developer B: User Story 2 (EBS) - P1 (MVP candidate)
- Developer C: User Stories 3-6 (S3, Lambda, RDS, DynamoDB) - P2
- Developer D: User Stories 7-8 (Embodied Carbon, Grid Factor Update) - P3
- Developer E: EKS (after Foundational) - P2

---

\n## Parallel Example: User Story 1 (GPU Carbon Estimation)

```bash
# Launch all tests for User Story 1 together:
Task: "Write table-driven test for GPU power lookup in internal/carbon/gpu_specs_test.go"
Task: "Write test for EC2 estimator with GPU power in internal/carbon/estimator_test.go"

# Once tests pass, launch GPU implementation tasks:
Task: "Add GetGPUSpec function to gpu_specs.go with sync.Once initialization"
Task: "Add GPU power calculation to EC2 estimator in internal/carbon/estimator.go"
Task: "Update EstimateCarbonGrams to include GPU power in internal/carbon/estimator.go"
Task: "Add GPU power to billing_detail breakdown in internal/plugin/carbon.go"
```

---

\n## Parallel Example: Foundational Phase

```bash
# Launch all embedded data creation tasks together:
Task: "Create GPU specs embedded CSV at internal/carbon/data/gpu_specs.csv"
Task: "Create storage specs embedded CSV at internal/carbon/data/storage_specs.csv"

# Launch all estimator foundation tasks together:
Task: "Create gpu_specs.go with GPUSpec struct and parser at internal/carbon/gpu_specs.go"
Task: "Create storage_specs.go with StorageSpec struct and parser at internal/carbon/storage_specs.go"
```

---

\n## Implementation Strategy

### MVP First (User Stories 1 + 2 Only)

**Complete Phase 1 (Setup) → Phase 2 (Foundational) → Phase 3 (US1) → Phase 4 (US2) → STOP and VALIDATE**

1. Complete Phase 1: Setup project structure
2. Complete Phase 2: Foundational infrastructure (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 - GPU Instance Carbon Estimation (P1)
4. Complete Phase 4: User Story 2 - EBS Storage Carbon Estimation (P1)
5. **STOP and VALIDATE**: Test GPU + EBS carbon estimation independently via gRPC
6. Deploy/Demo if ready (MVP delivers GPU + storage carbon for ML/AI workloads)

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 (GPU) → Test independently → Deploy/Demo (MVP core)
3. Add User Story 2 (EBS) → Test independently → Deploy/Demo (expands storage coverage)
4. Add User Stories 3-6 (S3, Lambda, RDS, DynamoDB) → Test each independently → Deploy/Demo
5. Add User Story 7 (Embodied Carbon) → Test independently → Deploy/Demo
6. Add User Story 8 (Grid Factor Update) + EKS → Test independently → Deploy/Demo
7. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (GPU) - P1 (MVP priority)
   - Developer B: User Story 2 (EBS) - P1 (MVP priority)
   - Developer C: User Story 3 (S3) - P2
   - Developer D: User Story 4 (Lambda) - P2
   - Developer E: User Story 5 (RDS) - P2
   - Developer F: User Story 6 (DynamoDB) - P2
3. Stories complete and integrate independently
4. Phase 12: Polish (all developers contribute)

---

\n## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing (TDD approach recommended)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Constitution compliance: All code must pass make lint and make test
- Performance: All estimators must complete in <100ms per call
- Thread safety: All gRPC handlers must support concurrent calls
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence

---

\n## Summary

**Total Task Count**: 74 tasks across 12 phases

**Task Count by User Story**:
- Setup (Phase 1): 4 tasks
- Foundational (Phase 2): 7 tasks
- US1 - GPU (Phase 3): 6 tasks (2 tests, 4 implementation)
- US2 - EBS (Phase 4): 6 tasks (2 tests, 4 implementation)
- US3 - S3 (Phase 5): 5 tasks (1 test, 4 implementation)
- US4 - Lambda (Phase 6): 6 tasks (1 test, 5 implementation)
- US5 - RDS (Phase 7): 6 tasks (1 test, 5 implementation)
- US6 - DynamoDB (Phase 8): 5 tasks (1 test, 4 implementation)
- US7 - Embodied Carbon (Phase 9): 7 tasks (2 tests, 5 implementation)
- US8 - Grid Factor Update (Phase 10): 7 tasks (1 test, 6 implementation)
- EKS (Phase 11): 5 tasks (1 test, 4 implementation)
- Polish (Phase 12): 10 tasks (all cross-cutting)

**Parallel Opportunities Identified**:
- Setup phase: 4 parallelizable tasks
- Foundational phase: 4 parallelizable tasks
- User stories: 1-2 parallelizable tasks per story (tests)
- Polish phase: 7 parallelizable tasks
- Total: 24 parallelizable tasks out of 74 (32.4%)

**Independent Test Criteria per Story**:
- US1 (GPU): Request p4d.24xlarge carbon estimate, validate includes 8x A100 GPU power
- US2 (EBS): Request 100GB gp3 carbon estimate, validate uses SSD 1.2 Wh/TB with 2× replication
- US3 (S3): Request 100GB S3 STANDARD carbon estimate, validate uses SSD 1.2 Wh/TB with 3× replication
- US4 (Lambda): Request Lambda (1792MB, 500ms, 1M invocations) carbon estimate, validate vCPU equivalent
- US5 (RDS): Request RDS (db.m5.large, 100GB, Multi-AZ) carbon estimate, validate compute + storage
- US6 (DynamoDB): Request DynamoDB (50GB) carbon estimate, validate matches S3 Standard methodology
- US7 (Embodied Carbon): Request carbon with embodied enabled, validate operational + embodied breakdown
- US8 (Grid Factor Update): Run update tool, validate fetches from CCF and produces valid data

**Suggested MVP Scope**: User Stories 1 (GPU) + 2 (EBS)
- Rationale: Both P1 priority, directly address ML/AI workload gap and storage gap
- Effort: 16 tasks (Setup 4 + Foundational 7 + US1 6 + US2 6) = 23 tasks
- Value: Delivers GPU + storage carbon for ML/AI workloads
- Testability: Can be tested independently via gRPC calls
