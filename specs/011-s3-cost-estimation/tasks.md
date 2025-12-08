# Tasks: S3 Storage Cost Estimation

**Input**: Design documents from `/specs/011-s3-cost-estimation/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Included as requested in feature specification testing strategy.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Single Go plugin project: internal/plugin/, internal/pricing/, cmd/, tools/

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization for S3 feature

- [ ] T001 Extend generate-pricing tool to fetch S3 pricing data in tools/generate-pricing/main.go
- [ ] T002 [P] Add S3Price struct to internal/pricing/types.go
- [ ] T003 [P] Extend PricingClient interface with S3PricePerGBMonth in internal/pricing/client.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core pricing infrastructure that MUST be complete before S3 cost estimation can be implemented

**⚠️ CRITICAL**: No S3 work can begin until this phase is complete

- [ ] T004 Implement S3PricePerGBMonth method in internal/pricing/client.go
- [ ] T005 Add s3Index map to Client struct in internal/pricing/client.go
- [ ] T006 Update pricing initialization to index S3 pricing data in internal/pricing/client.go
- [ ] T007 [P] Update mock pricing client in internal/plugin/plugin_test.go
- [ ] T008 [P] Update mock pricing client in internal/plugin/projected_test.go

**Checkpoint**: Foundation ready - S3 cost estimation implementation can now begin

---

## Phase 3: User Story 1 - Estimate S3 Storage Costs (Priority: P1) 🎯 MVP

**Goal**: Enable accurate projected monthly cost calculation for S3 storage based on storage class and bucket size

**Independent Test**: Call GetProjectedCost with S3 ResourceDescriptor and verify correct unit_price, cost_per_month, and billing_detail returned

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T009 [P] [US1] Unit test for estimateS3 function in internal/plugin/projected_test.go
- [ ] T010 [P] [US1] Unit test for S3PricePerGBMonth lookup in internal/pricing/client_test.go
- [ ] T011 [P] [US1] Table-driven tests for all storage classes in internal/plugin/projected_test.go
- [ ] T012 [P] [US1] Integration test with embedded pricing data in internal/plugin/integration_test.go

### Implementation for User Story 1

- [ ] T013 [US1] Implement estimateS3 function in internal/plugin/projected.go
- [ ] T014 [US1] Update GetProjectedCost router to call estimateS3 for s3 resource type in internal/plugin/projected.go
- [ ] T015 [US1] Update Supports method to return supported=true for s3 in internal/plugin/supports.go
- [ ] T016 [US1] Add debug logging for S3 pricing lookups in internal/plugin/projected.go
- [ ] T017 [US1] Handle unknown storage classes with $0 cost and explanatory billing detail in internal/plugin/projected.go

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: Polish & Cross-Cutting Concerns

**Purpose**: Improvements and validation across the S3 implementation

- [ ] T018 [P] Run make lint and fix any issues
- [ ] T019 [P] Run make test and ensure all tests pass
- [ ] T020 [P] Test with real embedded pricing data for all regions
- [ ] T021 [P] Validate performance requirements (<100ms RPC, <50ms lookup)
- [ ] T022 [P] Test concurrent RPC calls for thread safety
- [ ] T023 [P] Update CLAUDE.md with new S3 implementation patterns
- [ ] T024 [P] Run quickstart.md validation with grpcurl

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS S3 implementation
- **User Story 1 (Phase 3)**: Depends on Foundational phase completion
- **Polish (Phase 4)**: Depends on User Story 1 being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Pricing infrastructure before estimation logic
- Core implementation before logging and error handling
- Story complete before moving to polish

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- All tests for User Story 1 marked [P] can run in parallel
- Different polish tasks can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "Unit test for estimateS3 function in internal/plugin/projected_test.go"
Task: "Unit test for S3PricePerGBMonth lookup in internal/pricing/client_test.go"
Task: "Table-driven tests for all storage classes in internal/plugin/projected_test.go"
Task: "Integration test with embedded pricing data in internal/plugin/integration_test.go"

# Launch implementation tasks sequentially:
Task: "Implement estimateS3 function in internal/plugin/projected.go"
Task: "Update GetProjectedCost router to call estimateS3 for s3 resource type in internal/plugin/projected.go"
Task: "Update Supports method to return supported=true for s3 in internal/plugin/supports.go"
Task: "Add debug logging for S3 pricing lookups in internal/plugin/projected.go"
Task: "Handle unknown storage classes with $0 cost and explanatory billing detail in internal/plugin/projected.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks S3 implementation)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently with grpcurl
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Each addition adds value without breaking previous functionality

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 implementation
   - Developer B: User Story 1 tests
3. Story complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [US1] label maps task to User Story 1 for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies