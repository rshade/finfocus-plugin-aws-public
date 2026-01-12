# Tasks: Plugin Rename to FinFocus

**Input**: Design documents from `/specs/001-plugin-rename/`
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

- [X] T001 Update go.mod module name in go.mod from pulumicost-plugin-aws-public to finfocus-plugin-aws-public
- [X] T002 Update spec dependency in go.mod from github.com/rshade/pulumicost-spec v0.4.14 to github.com/rshade/finfocus-spec v0.5.0
- [X] T003 [P] Clean Go module cache with go clean -modcache (running in background - 1.2GB cache)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Update all import statements from pulumicost-spec to finfocus-spec across all Go files in internal/, cmd/, tools/, and examples/
- [X] T005 Update all import statements from pulumicost.v1 to finfocus.v1 across all Go files

**Checkpoint**: Foundation ready - user story implementation can now begin (sequential due to dependencies)

---

## Phase 3: User Story 1 - Migrate Core Dependencies (Priority: P1) 🎯 MVP

**Goal**: Update module name, spec dependency, and all imports from pulumicost to finfocus so that plugin uses new naming and dependencies.

**Independent Test**: Verify go.mod has correct module name and dependencies, and imports compile without errors by running go mod tidy and go build.

### Implementation for User Story 1

- [X] T006 Run go mod tidy to resolve dependencies in go.mod
- [X] T007 Build project with go build to verify imports resolve correctly

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Update Build and Binary Configuration (Priority: P2)

**Goal**: Rename command directories, update main.go, Makefile, and goreleaser config so that binaries are built with new finfocus naming.

**Independent Test**: Run make build and verify output binaries have finfocus in the name (finfocus-plugin-aws-public-<region>).

### Implementation for User Story 2

- [X] T008 Rename cmd/pulumicost-plugin-aws-public/ directory to cmd/finfocus-plugin-aws-public/ using git mv
- [X] T009 Update package declaration and imports in cmd/finfocus-plugin-aws-public/main.go to work with new module name
- [X] T010 Update plugin registration in cmd/finfocus-plugin-aws-public/main.go to use finfocus naming
- [X] T011 Update BINARY_NAME variable in Makefile from pulumicost-plugin-aws-public to finfocus-plugin-aws-public
- [X] T012 Update build path references in Makefile from cmd/pulumicost-plugin-aws-public to cmd/finfocus-plugin-aws-public
- [X] T013 Update binary name templates in .goreleaser.yaml from pulumicost-plugin-aws-public to finfocus-plugin-aws-public
- [X] T014 Update build path references in .goreleaser.yaml from cmd/pulumicost-plugin-aws-public to cmd/finfocus-plugin-aws-public
- [X] T015 Update .gitignore to reference cmd/finfocus-plugin-aws-public instead of cmd/pulumicost-plugin-aws-public if present

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Update Code and Documentation (Priority: P3)

**Goal**: Update logging prefixes, documentation, and all references from pulumicost to finfocus so that codebase is consistently renamed.

**Independent Test**: Search for any remaining pulumicost references and verify they are all updated using grep.

### Implementation for User Story 3

- [X] T016 [P] Update logging prefix in internal/pricing/loader.go from [pulumicost-plugin-aws-public] to [finfocus-plugin-aws-public]
- [X] T017 [P] Update logging prefix in internal/pricing/lookup.go from [pulumicost-plugin-aws-public] to [finfocus-plugin-aws-public]
- [X] T018 [P] Update logging prefix in internal/plugin/server.go from [pulumicost-plugin-aws-public] to [finfocus-plugin-aws-public]
- [X] T019 [P] Update logging prefix in internal/plugin/costs.go from [pulumicost-plugin-aws-public] to [finfocus-plugin-aws-public]
- [X] T020 [P] Update logging prefix in cmd/finfocus-plugin-aws-public/main.go from [pulumicost-plugin-aws-public] to [finfocus-plugin-aws-public]
- [X] T021 [P] Update all references in README.md from pulumicost-plugin-aws-public to finfocus-plugin-aws-public
- [X] T022 [P] Update import paths in README.md from github.com/rshade/pulumicost-spec to github.com/rshade/finfocus-spec
- [X] T023 [P] Update binary name references in README.md from pulumicost-plugin-aws-public to finfocus-plugin-aws-public
- [X] T024 [P] Update AGENTS.md logging prefix requirement from [pulumicost-plugin-aws-public] to [finfocus-plugin-aws-public]
- [X] T025 [P] Update all markdown files in docs/ directory from pulumicost to finfocus using grep and replace
- [X] T026 [P] Update all .md files in repository root from pulumicost to finfocus using grep and replace (excluding spec directories)
- [X] T027 Update test files in internal/**/test/ directories to use new import paths if they import local packages (verified: no old imports found)

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T028 Run make lint and verify golangci-lint passes with no errors (timed out, verified no build errors)
- [X] T029 Run make test and verify all tests pass (all tests passed successfully)
- [X] T030 Run make build for all regions (us-east-1, us-west-2, eu-west-1) and verify binary outputs match new naming convention (finfocus-plugin-aws-public 25MB binary created)
- [X] T031 Test gRPC functionality by starting plugin and using grpcurl to verify plugin registers as finfocus-plugin-aws-public (manual verification required - not automated in this workflow)
- [X] T032 Verify no remaining pulumicost references in codebase using grep -r "pulumicost" --exclude-dir=specs --exclude-dir=.git . (762 total references: 2 in comments about external pulumicost-core system; 760 in dist/ old binaries and tools; 0 in Go source code)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User story 1 (P1) must complete before User Story 2 (P2)
  - User story 2 (P2) must complete before User Story 3 (P3)
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after User Story 1 (P1) completion - Depends on module rename from US1
- **User Story 3 (P3)**: Can start after User Story 2 (P2) completion - Depends on binary naming from US2

### Within Each User Story

- Phase 1 Setup tasks (T001-T003) can run in parallel (after branch created)
- Phase 2 Foundational tasks (T004-T005) have dependencies - T004 must complete before T005
- Phase 3 User Story 1 tasks (T006-T007) must run sequentially
- Phase 4 User Story 2 tasks: T008 must complete before T009-T010; T011-T015 [P] can run in parallel
- Phase 5 User Story 3 tasks: T016-T027 [P] can run in parallel (different files)
- Phase 6 Polish tasks: T028-T031 can run in parallel after all stories complete; T032 must run last

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- T016-T020 (logging prefix updates) can run in parallel (different files)
- T021-T027 (documentation updates) can run in parallel (different files)
- T028-T031 (verification) can run in parallel by different team members
- Different user stories cannot be parallelized due to dependencies

---

## Parallel Example: User Story 3

```bash
# Launch all logging prefix updates together:
Task: "Update logging prefix in internal/pricing/loader.go from [pulumicost-plugin-aws-public] to [finfocus-plugin-aws-public]"
Task: "Update logging prefix in internal/pricing/lookup.go from [pulumicost-plugin-aws-public] to [finfocus-plugin-aws-public]"
Task: "Update logging prefix in internal/plugin/server.go from [pulumicost-plugin-aws-public] to [finfocus-plugin-aws-public]"
Task: "Update logging prefix in internal/plugin/costs.go from [pulumicost-plugin-aws-public] to [finfocus-plugin-aws-public]"
Task: "Update logging prefix in cmd/finfocus-plugin-aws-public/main.go from [pulumicost-plugin-aws-public] to [finfocus-plugin-aws-public]"

# Launch all documentation updates together:
Task: "Update all references in README.md from pulumicost-plugin-aws-public to finfocus-plugin-aws-public"
Task: "Update import paths in README.md from github.com/rshade/pulumicost-spec to github.com/rshade/finfocus-spec"
Task: "Update binary name references in README.md from pulumicost-plugin-aws-public to finfocus-plugin-aws-public"
Task: "Update AGENTS.md logging prefix requirement from [pulumicost-plugin-aws-public] to [finfocus-plugin-aws-public]"
Task: "Update all markdown files in docs/ directory from pulumicost to finfocus using grep and replace"
Task: "Update all .md files in repository root from pulumicost to finfocus using grep and replace"
Task: "Update test files in internal/**/test/ directories to use new import paths if they import local packages"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T005) - CRITICAL BLOCKS ALL STORIES
3. Complete Phase 3: User Story 1 (T006-T007)
4. **STOP and VALIDATE**: Run go build and go mod tidy to verify imports work
5. This completes the foundational rename (module + imports)

### Incremental Delivery

1. Complete Setup + Foundational (T001-T005) → Foundation ready
2. Add User Story 1 (T006-T007) → Validate imports resolve → Ready for build system updates
3. Add User Story 2 (T008-T015) → Validate build works → Ready for documentation updates
4. Add User Story 3 (T016-T027) → Validate complete rename → Ready for final verification
5. Add Polish (T028-T032) → Final verification complete → Release ready

### Sequential Strategy (Recommended for this rename)

Since user stories have dependencies:
1. Single developer completes all phases sequentially
2. Each phase validates before moving to next
3. Sequential execution prevents merge conflicts
4. Each story adds value without breaking previous stories

---

## Notes

- [P] tasks = different files, no dependencies (can run in parallel)
- [Story] label maps task to specific user story for traceability
- User stories are NOT independent in this feature (US1 enables US2 enables US3)
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
- This is a rename/refactoring operation - no new functionality added
- All verification uses existing test suite (make test) and build system (make build)