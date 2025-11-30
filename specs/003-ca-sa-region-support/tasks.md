---
description: "Task list template for feature implementation"
---

# Tasks: Canada and South America Region Support

**Input**: Design documents from `/specs/003-ca-sa-region-support/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: The examples below include test tasks. Tests are OPTIONAL - only include them if explicitly requested in the feature specification.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `src/`, `tests/` at repository root
- **Web app**: `backend/src/`, `frontend/src/`
- **Mobile**: `api/src/`, `ios/src/` or `android/src/`
- Paths shown below assume single project - adjust based on plan.md structure

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Create task documentation structure in specs/003-ca-sa-region-support/
- [x] T002 [P] Update .goreleaser.yaml with new build targets (cac1, sae1)
- [x] T003 [P] Update tools/generate-pricing/main.go to support new region arguments

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Generate placeholder pricing data in internal/pricing/data/aws_pricing_ca-central-1.json (via updated tool)
- [x] T005 Generate placeholder pricing data in internal/pricing/data/aws_pricing_sa-east-1.json (via updated tool)

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Canada Region Support (Priority: P1) 🎯 MVP

**Goal**: Support for ca-central-1 region with dedicated binary and pricing data

**Independent Test**: Build ca-central-1 binary and verify it runs and returns correct cost estimates

### Tests for User Story 1 (OPTIONAL - only if tests requested) ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T006 [P] [US1] Create integration test for ca-central-1 in internal/plugin/integration_test.go

### Implementation for User Story 1

- [x] T007 [US1] Create embed_cac1.go in internal/pricing/embed_cac1.go
- [x] T008 [US1] Verify build tags for ca-central-1
- [x] T009 [US1] Run local build for ca-central-1 to verify artifact creation

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - South America Region Support (Priority: P1)

**Goal**: Support for sa-east-1 region with dedicated binary and pricing data

**Independent Test**: Build sa-east-1 binary and verify it runs and returns correct cost estimates

### Tests for User Story 2 (OPTIONAL - only if tests requested) ⚠️

- [x] T010 [P] [US2] Create integration test for sa-east-1 in internal/plugin/integration_test.go

### Implementation for User Story 2

- [x] T011 [US2] Create embed_sae1.go in internal/pricing/embed_sae1.go
- [x] T012 [US2] Verify build tags for sa-east-1
- [x] T013 [US2] Run local build for sa-east-1 to verify artifact creation

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T014 [P] Update README.md with new supported regions
- [x] T015 [P] Verify logging requirements (zerolog) across new regions
- [x] T016 Run full test suite (unit + integration) for all regions
- [x] T017 Verify release artifacts generation via goreleaser check

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories

### Within Each User Story

- Tests (if included) MUST be written and FAIL before implementation
- Models before services
- Services before endpoints
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All tests for a user story marked [P] can run in parallel
- Models within a story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together (if tests requested):
Task: "Create integration test for ca-central-1 in internal/plugin/integration_test.go"

# Launch all implementation tasks for User Story 1 together:
Task: "Create embed_cac1.go in internal/pricing/embed_cac1.go"
Task: "Verify build tags for ca-central-1"
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
4. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
   - Developer B: User Story 2
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
