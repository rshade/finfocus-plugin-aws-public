# Tasks: EKS Cluster Cost Estimation

**Input**: Design documents from `/specs/010-eks-cost-estimation/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Included as requested in testing strategy from spec.md

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Single Go project: internal/, tools/, cmd/ at repository root

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization for EKS feature

- [ ] T001 Extend PricingClient interface with EKSClusterPricePerHour method in internal/pricing/client.go
- [ ] T002 Add eksPrice struct to internal/pricing/types.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core pricing infrastructure that MUST be complete before EKS estimation can be implemented

**⚠️ CRITICAL**: No EKS work can begin until this phase is complete

- [ ] T003 Update Client struct to include eksPricing field in internal/pricing/client.go
- [ ] T004 Implement EKSClusterPricePerHour method in internal/pricing/client.go
- [ ] T005 Update pricing initialization to filter AmazonEKS service data in internal/pricing/client.go
- [ ] T006 Extend generate-pricing tool to support AmazonEKS service in tools/generate-pricing/main.go

**Checkpoint**: Foundation ready - EKS estimation implementation can now begin

---

## Phase 3: User Story 1 - Estimate EKS Cluster Costs (Priority: P1) 🎯 MVP

**Goal**: Enable accurate EKS cluster cost estimates based on cluster count, distinguishing between standard and extended support

**Independent Test**: Call GetProjectedCost API with EKS resource descriptor and verify returned monthly cost calculation matches expected values ($73.00 standard, $365.00 extended)

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T007 [P] [US1] Unit test for estimateEKS function in internal/plugin/projected_test.go
- [ ] T008 [P] [US1] Unit test for EKSClusterPricePerHour lookup in internal/pricing/client_test.go
- [ ] T009 [P] [US1] Integration test for EKS cost calculation in internal/plugin/integration_test.go

### Implementation for User Story 1

- [ ] T010 [US1] Add "eks" to supported resource types in internal/plugin/supports.go
- [ ] T011 [US1] Implement estimateEKS function in internal/plugin/projected.go
- [ ] T012 [US1] Add "eks" case to GetProjectedCost router in internal/plugin/projected.go
- [ ] T013 [US1] Update mock pricing client with EKSClusterPricePerHour in internal/plugin/plugin_test.go
- [ ] T014 [US1] Generate EKS pricing data for all regions using tools/generate-pricing/main.go

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: Polish & Cross-Cutting Concerns

**Purpose**: Final improvements and validation

- [ ] T015 [P] Run make test to verify all EKS tests pass
- [ ] T016 [P] Run make lint to check EKS code quality
- [ ] T017 Update CHANGELOG.md with EKS feature
- [ ] T018 Run quickstart.md validation steps

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS User Story 1
- **User Stories (Phase 3)**: Depends on Foundational phase completion
- **Polish (Final Phase)**: Depends on User Story 1 being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Interface/types before implementation
- Core estimation before router
- Tests pass before polish

### Parallel Opportunities

- All Setup tasks can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- All tests for User Story 1 can run in parallel
- Different phases can be worked on sequentially in priority order

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "Unit test for estimateEKS function in internal/plugin/projected_test.go"
Task: "Unit test for EKSClusterPricePerHour lookup in internal/pricing/client_test.go"
Task: "Integration test for EKS cost calculation in internal/plugin/integration_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks story)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Each story adds value without breaking previous functionality

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
- Avoid: vague tasks, same file conflicts, cross-story dependencies</content>
<parameter name="filePath">/mnt/c/GitHub/go/src/github.com/rshade/finfocus-plugin-aws-public/specs/010-eks-cost-estimation/tasks.md