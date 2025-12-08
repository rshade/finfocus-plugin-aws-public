# Tasks: Lambda Function Cost Estimation

**Input**: Design documents from `/specs/001-lambda-cost-estimation/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Unit tests and integration tests are included as they are required for this plugin feature to ensure correctness of cost calculations.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Plugin project**: `internal/` for core logic, `cmd/` for entry points, `tools/` for build tools
- All paths are relative to repository root

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure validation

- [X] T001 Verify existing plugin structure matches plan.md specifications
- [X] T002 Confirm Go 1.25.4 and required dependencies are available
- [X] T003 Validate existing build and test infrastructure works

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before Lambda cost estimation can be implemented

**⚠️ CRITICAL**: No Lambda implementation work can begin until pricing client extensions are complete

- [X] T004 Extend PricingClient interface with Lambda methods in internal/pricing/client.go
- [X] T005 Add lambdaPrice struct to internal/pricing/types.go
- [X] T006 Update Client struct to include lambdaPricing field in internal/pricing/client.go
- [X] T007 Implement Lambda pricing initialization logic in internal/pricing/client.go
- [X] T008 Add LambdaPricePerRequest() method implementation in internal/pricing/client.go
- [X] T009 Add LambdaPricePerGBSecond() method implementation in internal/pricing/client.go

**Checkpoint**: Pricing client ready - Lambda cost estimation implementation can now begin

---

## Phase 3: User Story 1 - Accurate Lambda Cost Estimates (Priority: P1) 🎯 MVP

**Goal**: Implement Lambda function cost estimation based on request volume and compute duration

**Independent Test**: Can be fully tested by providing Lambda resource descriptors with memory, request count, and duration tags, and verifying that the returned cost matches expected calculations based on AWS pricing data.

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T010 [P] [US1] Unit tests for estimateLambda function in internal/plugin/projected_test.go
- [X] T011 [P] [US1] Integration tests for Lambda cost calculation in internal/plugin/projected_test.go
- [X] T012 [P] [US1] Tests for Lambda pricing client methods in internal/pricing/client_test.go

### Implementation for User Story 1

- [X] T013 [US1] Create estimateLambda function in internal/plugin/projected.go
- [X] T014 [US1] Implement memory size extraction from resource.Sku in internal/plugin/projected.go
- [X] T015 [US1] Implement request count extraction from tags in internal/plugin/projected.go
- [X] T016 [US1] Implement duration extraction from tags in internal/plugin/projected.go
- [X] T017 [US1] Implement GB-seconds calculation logic in internal/plugin/projected.go
- [X] T018 [US1] Implement total cost calculation (requests + duration) in internal/plugin/projected.go
- [X] T019 [US1] Add Lambda case to projected cost router in internal/plugin/projected.go
- [X] T020 [US1] Update Supports method for Lambda in internal/plugin/supports.go
- [X] T021 [US1] Add input validation and error handling in internal/plugin/projected.go
- [X] T022 [US1] Add structured logging for Lambda operations in internal/plugin/projected.go

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect the Lambda implementation and overall plugin quality

- [X] T023 [P] Update generate-pricing tool for Lambda support in tools/generate-pricing/main.go
- [X] T024 Run comprehensive unit tests for Lambda functionality
- [X] T025 Run integration tests with mock pricing client
- [X] T026 Validate cost calculation accuracy against AWS pricing formulas
- [X] T027 Test concurrent access to Lambda pricing lookups
- [X] T032 Add performance benchmark for 100 concurrent requests per second target
- [X] T033 Add memory usage benchmark for 100MB limit under 1000 concurrent requests
- [X] T034 Benchmark Lambda cost estimation throughput (target: 100 req/sec)
- [X] T035 Benchmark Lambda pricing client memory usage (target: <100MB for 1000 concurrent requests)
- [X] T036 Profile Lambda cost calculation performance (target: <100ms per request)
- [X] T028 Run make lint and make test to ensure code quality
- [X] T029 Test Lambda support across all 9 regional binaries
- [X] T030 Update CHANGELOG.md with Lambda cost estimation feature
- [ ] T031 Validate quickstart.md implementation steps work correctly

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS Lambda implementation
- **User Story 1 (Phase 3)**: Depends on Foundational phase completion
- **Polish (Phase 4)**: Depends on User Story 1 being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Pricing client extensions before Lambda estimator
- Core calculation logic before router updates
- Basic functionality before validation and logging
- Story complete before moving to polish phase

### Parallel Opportunities

- All Setup tasks can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- All tests for User Story 1 marked [P] can run in parallel
- Different implementation tasks within User Story 1 can run in parallel where dependencies allow

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "Unit tests for estimateLambda function in internal/plugin/projected_test.go"
Task: "Integration tests for Lambda cost calculation in internal/plugin/projected_test.go"
Task: "Tests for Lambda pricing client methods in internal/pricing/client_test.go"

# Launch foundational pricing tasks together:
Task: "Extend PricingClient interface with Lambda methods in internal/pricing/client.go"
Task: "Add lambdaPrice struct to internal/pricing/types.go"
Task: "Update Client struct to include lambdaPricing field in internal/pricing/client.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup validation
2. Complete Phase 2: Foundational pricing client extensions (CRITICAL - blocks all Lambda work)
3. Complete Phase 3: User Story 1 Lambda cost estimation
4. **STOP and VALIDATE**: Test Lambda cost estimation independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Pricing infrastructure ready
2. Add User Story 1 → Test Lambda cost estimation independently → Deploy/Demo (MVP!)
3. Add Polish phase → Enhanced testing and validation → Production ready

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together (pricing client work)
2. Once Foundational is done:
   - Developer A: Lambda estimator implementation (T013-T022)
   - Developer B: Test implementation and validation (T010-T012, T024-T030)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [US1] label maps task to User Story 1 for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate functionality independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence</content>
<parameter name="filePath">/mnt/c/GitHub/go/src/github.com/rshade/pulumicost-plugin-aws-public/specs/001-lambda-cost-estimation/tasks.md