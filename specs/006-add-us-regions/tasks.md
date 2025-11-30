# Tasks: Add support for additional US regions (us-west-1, us-gov-west-1, us-gov-east-1)

**Input**: Design documents from `/specs/006-add-us-regions/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Included as requested in feature specification (integration tests for each region)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go project**: `cmd/`, `internal/`, `scripts/`, `.goreleaser.yaml` at repository root

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Update tools and scripts to support new regions

- [ ] T001 Update pricing generator to support us-west-1, us-gov-west-1, us-gov-east-1 regions in tools/generate-pricing/main.go
- [ ] T002 [P] Add region mappings for new regions in scripts/region-tag.sh
- [ ] T003 [P] Update build scripts for new regions in scripts/build-region.sh

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core build configuration that MUST be complete before ANY region can be implemented

**⚠️ CRITICAL**: No region work can begin until this phase is complete

- [ ] T004 Add build tags region_usw1, region_govw1, region_gove1 to relevant Go files
- [ ] T005 Update .goreleaser.yaml with new region configurations for us-west-1, us-gov-west-1, us-gov-east-1
- [ ] T006 Verify build system works with new region configurations

**Checkpoint**: Foundation ready - region implementation can now begin in parallel

---

## Phase 3: User Story 1 - Get pricing for us-west-1 region (Priority: P1) 🎯 MVP

**Goal**: Enable accurate cost estimation for AWS resources in us-west-1 (N. California) region

**Independent Test**: Request projected costs for resources in us-west-1 and verify pricing data is returned without errors

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T007 [P] [US1] Integration test for us-west-1 region in internal/plugin/integration_usw1_test.go

### Implementation for User Story 1

- [ ] T008 [US1] Generate pricing data for us-west-1 region using tools/generate-pricing/
- [ ] T009 [US1] Create embed_usw1.go with us-west-1 pricing data in internal/pricing/embed_usw1.go
- [ ] T010 [US1] Update pricing client to support us-west-1 region in internal/pricing/client.go

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Get pricing for us-gov-west-1 region (Priority: P1)

**Goal**: Enable accurate cost estimation for AWS resources in us-gov-west-1 (AWS GovCloud US-West) region

**Independent Test**: Request projected costs for resources in us-gov-west-1 and verify GovCloud-specific pricing data is returned

### Tests for User Story 2 ⚠️

- [ ] T011 [P] [US2] Integration test for us-gov-west-1 region in internal/plugin/integration_govw1_test.go

### Implementation for User Story 2

- [ ] T012 [US2] Generate pricing data for us-gov-west-1 region using tools/generate-pricing/
- [ ] T013 [US2] Create embed_govw1.go with us-gov-west-1 pricing data in internal/pricing/embed_govw1.go
- [ ] T014 [US2] Update pricing client to support us-gov-west-1 region in internal/pricing/client.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Get pricing for us-gov-east-1 region (Priority: P1)

**Goal**: Enable accurate cost estimation for AWS resources in us-gov-east-1 (AWS GovCloud US-East) region

**Independent Test**: Request projected costs for resources in us-gov-east-1 and verify GovCloud-specific pricing data is returned

### Tests for User Story 3 ⚠️

- [ ] T015 [P] [US3] Integration test for us-gov-east-1 region in internal/plugin/integration_gove1_test.go

### Implementation for User Story 3

- [ ] T016 [US3] Generate pricing data for us-gov-east-1 region using tools/generate-pricing/
- [ ] T017 [US3] Create embed_gove1.go with us-gov-east-1 pricing data in internal/pricing/embed_gove1.go
- [ ] T018 [US3] Update pricing client to support us-gov-east-1 region in internal/pricing/client.go

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final improvements and validation

- [ ] T019 [P] Update README.md with new supported regions
- [ ] T020 Run make test to ensure all tests pass
- [ ] T021 Run make build-region for all new regions to verify builds
- [ ] T022 Update CHANGELOG.md with new region support
- [ ] T023 Validate quickstart.md instructions work correctly

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User stories can proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 3 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Pricing data generation before embed file creation
- Embed file before client updates
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All tests for a user story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch test for User Story 1:
Task: "Integration test for us-west-1 region in internal/plugin/integration_usw1_test.go"

# Launch implementation for User Story 1:
Task: "Generate pricing data for us-west-1 region using tools/generate-pricing/"
Task: "Create embed_usw1.go with us-west-1 pricing data in internal/pricing/embed_usw1.go"
Task: "Update pricing client to support us-west-1 region in internal/pricing/client.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (us-west-1)
   - Developer B: User Story 2 (us-gov-west-1)
   - Developer C: User Story 3 (us-gov-east-1)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence