---

description: "Task list for feature implementation"
---

# Tasks: Core Protocol Intelligence

**Input**: Design documents from `/specs/030-core-protocol-intelligence/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/
**Tests**: Unit tests are included for all features. Integration tests for gRPC methods.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `internal/`, `test/`, `cmd/` at repository root
- Paths shown below follow Go plugin structure from plan.md

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure for metadata enrichment

- [ ] T001 Create `internal/plugin/classification.go` file with ServiceClassification struct definition
- [ ] T002 Create `internal/plugin/enrichment.go` file with placeholder package declaration

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T003 Add dev mode constants to `internal/plugin/constants.go` (hoursPerMonthProd=730, hoursPerMonthDev=160)
- [ ] T004 [P] Add relationship type constants to `internal/plugin/constants.go` (relationshipAttachedTo, relationshipWithin, relationshipManagedBy)
- [ ] T005 [P] Implement `serviceClassifications` map in `internal/plugin/classification.go` with all 10 AWS services
- [ ] T006 [P] Implement `hasUsageProfile()` function in `internal/plugin/enrichment.go` (feature detection)
- [ ] T007 [P] Implement `hasGrowthHint()` function in `internal/plugin/enrichment.go` (feature detection)
- [ ] T008 [P] Implement `hasLineage()` function in `internal/plugin/enrichment.go` (feature detection)

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 2 - Growth Type Hints (Priority: P2) ✅ READY NOW

**Goal**: Populate growth_type field in GetProjectedCostResponse to enable Cost Time Machine forecasting

**Independent Test**: Can be tested by calling cost estimation for S3 (LINEAR) and EC2 (STATIC) and verifying growth_hint values

### Implementation for User Story 2

- [ ] T009 [P] Implement `setGrowthHint()` function in `internal/plugin/enrichment.go` with feature detection guard
- [ ] T010 [P] Integrate `setGrowthHint()` call in `internal/plugin/projected.go` estimateEC2Cost function (after cost calculation, before return)
- [ ] T011 [P] Integrate `setGrowthHint()` call in `internal/plugin/projected.go` estimateEBSCost function (after cost calculation, before return)
- [ ] T012 [P] Integrate `setGrowthHint()` call in `internal/plugin/projected.go` estimateEKSCost function (after cost calculation, before return)
- [ ] T013 [P] Integrate `setGrowthHint()` call in `internal/plugin/projected.go` estimateS3Cost function (after cost calculation, before return)
- [ ] T014 [P] Integrate `setGrowthHint()` call in `internal/plugin/projected.go` estimateLambdaCost function (after cost calculation, before return)
- [ ] T015 [P] Integrate `setGrowthHint()` call in `internal/plugin/projected.go` estimateDynamoDBCost function (after cost calculation, before return)
- [ ] T016 [P] Integrate `setGrowthHint()` call in `internal/plugin/projected.go` estimateELBCost function (after cost calculation, before return)
- [ ] T017 [P] Integrate `setGrowthHint()` call in `internal/plugin/projected.go` estimateNATGatewayCost function (after cost calculation, before return)
- [ ] T018 [P] Integrate `setGrowthHint()` call in `internal/plugin/projected.go` estimateCloudWatchCost function (after cost calculation, before return)
- [ ] T019 [P] Integrate `setGrowthHint()` call in `internal/plugin/projected.go` estimateElastiCacheCost function (after cost calculation, before return)
- [ ] T020 [P] Integrate `setGrowthHint()` call in `internal/plugin/projected.go` estimateRDSCost function (after cost calculation, before return)

**Checkpoint**: Growth Type Hints (P2) fully functional and independently testable

---

## Phase 4: User Story 1 - Dev Mode (Priority: P1) ⚠️ BLOCKED ON PROTO

**Goal**: Apply UsageProfile-based cost reduction (160 vs 730 hours) for dev/test environments

**Independent Test**: Can be tested by sending request with UsageProfile=DEVELOPMENT and verifying EC2 cost is ~22% of production

**⚠️ PRE-REQUISITE**: Must create PR to finfocus-spec to add UsageProfile enum before implementing this phase

### Implementation for User Story 1

- [ ] T021 [P] Create PR to rshade/finfocus-spec for UsageProfile enum - reference issue #209 (specs/030-core-protocol-intelligence/contracts/usage-profile-proto.md)
- [ ] T022 [P] Implement `applyDevMode()` function in `internal/plugin/enrichment.go` with context parameter for logging
- [ ] T023 [P] Add zerolog logging to `applyDevMode()` function (INFO level with usage_profile, resource_type fields)
- [ ] T024 [P] Integrate `applyDevMode()` call in `internal/plugin/projected.go` estimateEC2Cost function (before setGrowthHint)
- [ ] T025 [P] Integrate `applyDevMode()` call in `internal/plugin/projected.go` estimateEKSCost function (before setGrowthHint)
- [ ] T026 [P] Integrate `applyDevMode()` call in `internal/plugin/projected.go` estimateELBCost function (before setGrowthHint)
- [ ] T027 [P] Integrate `applyDevMode()` call in `internal/plugin/projected.go` estimateNATGatewayCost function (before setGrowthHint)
- [ ] T028 [P] Integrate `applyDevMode()` call in `internal/plugin/projected.go` estimateElastiCacheCost function (before setGrowthHint)
- [ ] T029 [P] Integrate `applyDevMode()` call in `internal/plugin/projected.go` estimateRDSCost function (before setGrowthHint)

**Checkpoint**: Dev Mode (P1) fully functional and independently testable

---

## Phase 5: User Story 3 - Resource Topology Linking (Priority: P3) ⚠️ BLOCKED ON PROTO

**Goal**: Extract parent_resource_id from tags to enable Blast Radius topology visualization

**Independent Test**: Can be tested by sending EBS volume request with instance_id tag and verifying response includes parent_resource_id

**⚠️ PRE-REQUISITE**: Must create PR to finfocus-spec to add CostAllocationLineage message before implementing this phase

### Implementation for User Story 3

- [ ] T030 [P]Create PR to rshade/finfocus-spec for CostAllocationLineage message - reference issue #208 (specs030-core-protocol-intelligence/contracts/cost-allocation-lineage-proto.md)
- [ ] T031 [P] Implement `extractLineage()` function in `internal/plugin/enrichment.go` with context parameter for logging
- [ ] T032 [P] Add zerolog logging to `extractLineage()` function (INFO level with parent_detected, parent_type, relationship fields)
- [ ] T033 [P] Integrate `extractLineage()` call in `internal/plugin/projected.go` estimateEBSCost function (after applyDevMode, after setGrowthHint)
- [ ] T034 [P] Integrate `extractLineage()` call in `internal/plugin/projected.go` estimateELBCost function (after applyDevMode, after setGrowthHint)
- [ ] T035 [P] Integrate `extractLineage()` call in `internal/plugin/projected.go` estimateNATGatewayCost function (after applyDevMode, after setGrowthHint)
- [ ] T036 [P] Integrate `extractLineage()` call in `internal/plugin/projected.go` estimateElastiCacheCost function (after applyDevMode, after setGrowthHint)
- [ ] T037 [P] Integrate `extractLineage()` call in `internal/plugin/projected.go` estimateRDSCost function (after applyDevMode, after setGrowthHint)

**Checkpoint**: Resource Topology Linking (P3) fully functional and independently testable

---

## Phase 6: Testing & Validation

**Purpose**: Comprehensive test coverage for all three features

### Unit Tests

- [ ] T038 [P] Add `TestServiceClassifications()` to `internal/plugin/classification_test.go` (verify all 10 services)
- [ ] T039 [P] Add `TestHasUsageProfile()` to `internal/plugin/enrichment_test.go` (verify feature detection)
- [ ] T040 [P] Add `TestHasGrowthHint()` to `internal/plugin/enrichment_test.go` (verify feature detection)
- [ ] T041 [P] Add `TestHasLineage()` to `internal/plugin/enrichment_test.go` (verify feature detection)
- [ ] T042 [P] Add `TestSetGrowthHint()` to `internal/plugin/enrichment_test.go` (table-driven, all 11 services)
- [ ] T043 [P] Add `TestApplyDevMode()` to `internal/plugin/enrichment_test.go` (table-driven, time-based vs usage-based)
- [ ] T044 [P] Add `TestExtractLineage()` to `internal/plugin/enrichment_test.go` (table-driven, parent tag priority)
- [ ] T045 [P] Add `TestFeatureDetectionGracefulDegradation()` to `internal/plugin/enrichment_test.go` (verify old spec versions handled)
- [ ] T046 [P] Add edge case tests to `internal/plugin/enrichment_test.go` (unknown services, empty tags, invalid enum values)

### Integration Tests

- [ ] T047 [P] Add `TestGrowthTypeHintsIntegration()` to `test/integration/metadata_enrichment_test.go` (verify growth_type in gRPC response)
- [ ] T048 [P] Add `TestDevModeIntegration()` to `test/integration/metadata_enrichment_test.go` (verify cost reduction and billing detail)
- [ ] T049 [P] Add `TestLineageIntegration()` to `test/integration/metadata_enrichment_test.go` (verify parent extraction in gRPC response)
- [ ] T050 [P] Add `TestAllMetadataFeaturesIntegration()` to `test/integration/metadata_enrichment_test.go` (verify all three features together)

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation updates and final validation

- [ ] T051 Update `CLAUDE.md` with growth classification, dev mode, and topology relationship patterns
- [ ] T052 Update `go.mod` with finfocus-spec dependency after proto merges
- [ ] T053 Run `make lint` and fix all issues
- [ ] T054 Run `make test` and verify all tests pass
- [ ] T055 Run `go test -race ./internal/plugin/...` to verify thread safety (no data races)
- [ ] T056 Verify < 100ms performance target for GetProjectedCost RPC (benchmark tests if needed)
- [ ] T057 Run `make build-region REGION=us-east-1` and verify binary compiles
- [ ] T058 Check binary size is < 250MB (ls -lh finfocus-plugin-aws-public-us-east-1)
- [ ] T059 Verify backward compatibility (existing cost calculations unchanged for UNSPECIFIED UsageProfile)
- [ ] T060 Create comprehensive integration test validation per quickstart.md Verification Commands

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User Story 2 (P2) can start immediately after Foundational - READY NOW
  - User Story 1 (P1) requires proto PR to merge - BLOCKED
  - User Story 3 (P3) requires proto PR to merge - BLOCKED
- **Testing (Phase 6)**: Depends on all implemented user stories
- **Polish (Phase 7)**: Depends on Testing completion

### User Story Dependencies

- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 1 (P1)**: Can start after proto PR merge - No dependencies on US2 or US3 (independently testable)
- **User Story 3 (P3)**: Can start after proto PR merge - No dependencies on US1 or US2 (independently testable)

### Within Each User Story

- Implementation before integration tests
- Unit tests before feature integration
- Feature detection functions before enrichment functions
- All enrichment functions (applyDevMode, setGrowthHint, extractLineage) in priority order
- Story complete before moving to next priority

### Parallel Opportunities

- **Setup Phase (Phase 1)**: T001 and T002 can run in parallel (different files)
- **Foundational Phase (Phase 2)**: T003, T004, T005, T006, T007, T008 can all run in parallel (different files)
- **User Story 2 Implementation (Phase 3)**: T009-T020 can all run in parallel (different functions in projected.go)
- **User Story 1 Implementation (Phase 4)**: T021 (proto PR) must complete first, then T022-T029 can run in parallel
- **User Story 3 Implementation (Phase 5)**: T030 (proto PR) must complete first, then T031-T037 can run in parallel
- **Testing Phase (Phase 6)**: T038-T046 (unit tests) can run in parallel, T047-T050 (integration tests) can run in parallel
- **Polish Phase (Phase 7)**: T051-T054 can run in parallel, T055-T060 can run in parallel

---

## Parallel Example: User Story 2 (Growth Type Hints)

```bash
# Launch all enrichment function integrations together (can run in parallel):
Task: "Integrate setGrowthHint() in estimateEC2Cost"
Task: "Integrate setGrowthHint() in estimateEBSCost"
Task: "Integrate setGrowthHint() in estimateEKSCost"
Task: "Integrate setGrowthHint() in estimateS3Cost"
Task: "Integrate setGrowthHint() in estimateLambdaCost"
Task: "Integrate setGrowthHint() in estimateDynamoDBCost"
Task: "Integrate setGrowthHint() in estimateELBCost"
Task: "Integrate setGrowthHint() in estimateNATGatewayCost"
Task: "Integrate setGrowthHint() in estimateCloudWatchCost"
Task: "Integrate setGrowthHint() in estimateElastiCacheCost"
Task: "Integrate setGrowthHint() in estimateRDSCost"
```

---

## Implementation Strategy

### MVP First (User Story 2 - Growth Type Hints Only)

1. Complete Phase 1: Setup (T001-T002)
2. Complete Phase 2: Foundational (T003-T008) - CRITICAL - blocks all stories
3. Complete Phase 3: User Story 2 (T009-T020) - READY NOW
4. **STOP and VALIDATE**: Test Growth Type Hints independently
5. Deploy/demo if ready
6. Create proto PRs for User Story 1 (T021) and User Story 3 (T030) in parallel

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 2 → Test independently → Deploy/Demo (MVP!)
3. Create proto PRs → Merge when ready
4. Add User Story 1 → Test independently → Deploy/Demo
5. Add User Story 3 → Test independently → Deploy/Demo
6. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. **Team completes Setup + Foundational together** (T001-T008)
2. **Once Foundational is done:**
   - **Developer A**: User Story 2 (P2) - Growth Type Hints (T009-T020) - READY NOW
   - **Developer B**: Create proto PR for User Story 1 (P1) - UsageProfile (T021)
   - **Developer C**: Create proto PR for User Story 3 (P3) - CostAllocationLineage (T030)
3. **After proto PRs merge:**
   - **Developer B**: User Story 1 implementation (T022-T029)
   - **Developer C**: User Story 3 implementation (T031-T037)
4. **All developers collaborate on testing** (T038-T060)

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability (US1, US2, US3)
- Each user story should be independently completable and testable
- Tests are included for all features (unit + integration)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
- Proto PRs (T021, T030) are BLOCKED dependencies for US1 and US3
- User Story 2 (P2) is READY NOW - can implement immediately after Foundational phase completes
- User Story 1 (P1) and User Story 3 (P3) are BLOCKED on proto PRs
- Zero breaking changes to existing cost calculations (SC-004) - enrichment is separate
- Feature detection ensures backward compatibility with older spec versions (NFR-001)

---

## Summary

- **Total Tasks**: 60
- **Setup Tasks**: 2 (T001-T002)
- **Foundational Tasks**: 6 (T003-T008)
- **User Story 1 (P1 - Dev Mode)**: 9 tasks (T021-T029) - BLOCKED on proto PR
- **User Story 2 (P2 - Growth Hints)**: 12 tasks (T009-T020) - READY NOW
- **User Story 3 (P3 - Topology)**: 8 tasks (T030-T037) - BLOCKED on proto PR
- **Testing Tasks**: 13 (T038-T050)
- **Polish Tasks**: 10 (T051-T060)

**Parallel Opportunities**: 42 tasks marked [P] can run in parallel

**Independent Test Criteria**:
- **US1 (P1)**: Verify 160/730 = 21.9% cost reduction for time-based services with UsageProfile=DEVELOPMENT
- **US2 (P2)**: Verify growth_type field populated correctly (LINEAR for S3/DynamoDB, STATIC for others)
- **US3 (P3)**: Verify parent_resource_id populated from tags with correct relationship mapping

**Suggested MVP Scope**:
- **Phase 1 (Setup) + Phase 2 (Foundational) + Phase 3 (User Story 2 - Growth Hints)**
- Ready to implement now after Foundational phase completes
- Delivers immediate value: Cost Time Machine forecasting capability
- Can demo independently before proto PRs merge

**Format Validation**: ✅ ALL tasks follow checklist format (checkbox, ID, [P] marker, [Story] label, file paths included)
