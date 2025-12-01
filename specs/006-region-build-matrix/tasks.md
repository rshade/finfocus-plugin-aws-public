# Tasks: Automated Build Matrix for AWS Regions

**Input**: Design documents from `/specs/006-region-build-matrix/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: No test tasks included - tests not requested in feature specification.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **CLI/plugin project**: `tools/`, `scripts/`, `internal/pricing/` at repository root
- Paths shown below follow plan.md structure

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure for generation tools

- [x] T001 Create tools/generate-embeds directory and basic Go module structure
- [x] T002 Create tools/generate-goreleaser directory and basic Go module structure
- [x] T003 Create initial regions.yaml configuration file in internal/pricing/regions.yaml
- [x] T004 [P] Add goccy/go-yaml dependency to both tools

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 Implement regions.yaml parsing logic in tools/generate-embeds/main.go
- [x] T006 Create embed file template in tools/generate-embeds/embed_template.go.tmpl
- [x] T007 Implement basic embed file generation logic in tools/generate-embeds/main.go
- [x] T008 Implement GoReleaser config generation logic in tools/generate-goreleaser/main.go
- [x] T009 Create basic verification script structure in scripts/verify-regions.sh

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Add New AWS Region Support (Priority: P1) 🎯 MVP

**Goal**: Enable developers to add new AWS regions by editing regions.yaml and running generation scripts

**Independent Test**: Add a new region to regions.yaml, run generation scripts, verify embed files and goreleaser config are created correctly

### Implementation for User Story 1

- [x] T010 [US1] Complete regions.yaml parsing with validation in tools/generate-embeds/main.go
- [x] T011 [US1] Implement embed file generation with build tags in tools/generate-embeds/main.go
- [x] T012 [US1] Add command-line interface to tools/generate-embeds/main.go
- [x] T013 [US1] Complete GoReleaser config generation with all build blocks in tools/generate-goreleaser/main.go
- [x] T014 [US1] Add command-line interface to tools/generate-goreleaser/main.go
- [x] T015 [US1] Update Makefile with generate-embeds and generate-goreleaser targets
- [x] T016 [US1] Test end-to-end region addition workflow manually

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Verify Region Configuration Consistency (Priority: P2)

**Goal**: Provide automated verification that all regions are properly configured and files exist

**Independent Test**: Run verification script and confirm it detects missing embed files or configuration issues

### Implementation for User Story 2

- [x] T017 [US2] Implement embed file existence checks in scripts/verify-regions.sh
- [x] T018 [US2] Implement pricing data file existence checks in scripts/verify-regions.sh
- [x] T019 [US2] Implement build tag consistency validation in scripts/verify-regions.sh
- [x] T020 [US2] Implement GoReleaser config sync validation in scripts/verify-regions.sh
- [x] T021 [US2] Add command-line options and error reporting to scripts/verify-regions.sh
- [x] T022 [US2] Update Makefile with verify-regions target

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Update CI/CD Pipeline (Priority: P3)

**Goal**: Integrate region matrix generation into CI/CD pipeline for automated builds

**Independent Test**: Update CI/CD workflow to run generation scripts before builds and verify all regions build correctly

### Implementation for User Story 3

- [x] T023 [US3] Update .github/workflows/test.yml to run generation scripts before builds
- [x] T024 [US3] Update .github/workflows/release.yml to use generated GoReleaser config
- [x] T025 [US3] Add generation script execution to build-region.sh
- [x] T026 [US3] Update release-region.sh to use generated configs
- [x] T027 [US3] Test CI/CD pipeline with region matrix generation

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T028 [P] Update README.md with region addition instructions
- [x] T029 Code cleanup and error handling improvements across all tools
- [x] T030 [P] Add comprehensive documentation in docs/region-management.md
- [x] T031 Validate quickstart.md instructions work end-to-end
- [x] T032 Performance optimization for large region sets

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
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Uses generation outputs from US1 but independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Integrates with US1 outputs but independently testable

### Within Each User Story

- Core generation logic before command-line interfaces
- Individual tool completion before Makefile integration
- Manual testing before CI/CD integration

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch embed generation and goreleaser generation in parallel:
Task: "Complete regions.yaml parsing with validation in tools/generate-embeds/main.go"
Task: "Complete GoReleaser config generation with all build blocks in tools/generate-goreleaser/main.go"

# Launch CLI implementations in parallel:
Task: "Add command-line interface to tools/generate-embeds/main.go"
Task: "Add command-line interface to tools/generate-goreleaser/main.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently by adding a region
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
   - Developer A: User Story 1 (region addition workflow)
   - Developer B: User Story 2 (verification system)
   - Developer C: User Story 3 (CI/CD integration)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence</content>
<parameter name="filePath">specs/006-region-build-matrix/tasks.md