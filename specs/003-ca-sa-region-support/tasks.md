# Tasks: Canada and South America Region Support

**Input**: Design documents from `/specs/003-ca-sa-region-support/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Tests are included as this project has constitution requirements for testing discipline.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: Go plugin with `internal/`, `tools/`, `data/` at repository root
- Paths based on plan.md structure

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Generate pricing data for new regions

- [x] T001 Update pricing generator to support new regions in tools/generate-pricing/main.go
- [x] T002 Generate pricing data for ca-central-1 in internal/pricing/data/aws_pricing_ca-central-1.json
- [x] T003 [P] Generate pricing data for sa-east-1 in internal/pricing/data/aws_pricing_sa-east-1.json

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create embed files and update build configuration

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Create embed file for ca-central-1 in internal/pricing/embed_cac1.go
- [x] T005 [P] Create embed file for sa-east-1 in internal/pricing/embed_sae1.go
- [x] T006 Update fallback embed file to exclude new region tags in internal/pricing/embed_fallback.go
- [x] T007 Add ca-central-1 build target to .goreleaser.yaml
- [x] T008 [P] Add sa-east-1 build target to .goreleaser.yaml
- [x] T009 Update GoReleaser before hook to include new regions in .goreleaser.yaml

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Canada Region Cost Estimation (Priority: P1)

**Goal**: Provide accurate EC2 and EBS cost estimates for ca-central-1 resources

**Independent Test**: Build ca-central-1 binary and verify it returns correct pricing for t3.micro EC2 and gp3 EBS

### Tests for User Story 1

- [x] T010 [P] [US1] Add ca-central-1 EC2 test cases to pricing client tests in internal/pricing/client_test.go
- [x] T011 [P] [US1] Add ca-central-1 EBS test cases to pricing client tests in internal/pricing/client_test.go
- [x] T012 [P] [US1] Add ca-central-1 plugin test cases in internal/plugin/plugin_test.go

### Implementation for User Story 1

- [x] T013 [US1] Build and verify ca-central-1 binary with `go build -tags region_cac1`
- [x] T014 [US1] Verify binary size is under 20MB for ca-central-1
- [x] T015 [US1] Test ca-central-1 gRPC service responds correctly with grpcurl

**Checkpoint**: User Story 1 complete - ca-central-1 binary fully functional

---

## Phase 4: User Story 2 - South America Region Cost Estimation (Priority: P1)

**Goal**: Provide accurate EC2 and EBS cost estimates for sa-east-1 resources

**Independent Test**: Build sa-east-1 binary and verify it returns correct pricing for m5.large EC2 and io1 EBS

### Tests for User Story 2

- [x] T016 [P] [US2] Add sa-east-1 EC2 test cases to pricing client tests in internal/pricing/client_test.go
- [x] T017 [P] [US2] Add sa-east-1 EBS test cases to pricing client tests in internal/pricing/client_test.go
- [x] T018 [P] [US2] Add sa-east-1 plugin test cases in internal/plugin/plugin_test.go

### Implementation for User Story 2

- [x] T019 [US2] Build and verify sa-east-1 binary with `go build -tags region_sae1`
- [x] T020 [US2] Verify binary size is under 20MB for sa-east-1
- [x] T021 [US2] Test sa-east-1 gRPC service responds correctly with grpcurl

**Checkpoint**: User Story 2 complete - sa-east-1 binary fully functional

---

## Phase 5: User Story 3 - Region Mismatch Rejection (Priority: P2)

**Goal**: Correctly reject requests for resources in non-matching regions with ERROR_CODE_UNSUPPORTED_REGION

**Independent Test**: Send ca-central-1 resource request to sa-east-1 binary and verify proper error response

### Tests for User Story 3

- [x] T022 [P] [US3] Add region mismatch test for ca-central-1 binary rejecting other regions in internal/plugin/supports_test.go
- [x] T023 [P] [US3] Add region mismatch test for sa-east-1 binary rejecting other regions in internal/plugin/supports_test.go
- [x] T024 [P] [US3] Add cross-region validation tests in internal/plugin/projected_test.go

### Implementation for User Story 3

- [x] T025 [US3] Verify region mismatch latency is under 100ms
- [x] T026 [US3] Verify ERROR_CODE_UNSUPPORTED_REGION includes correct error details

**Checkpoint**: User Story 3 complete - region mismatch handling verified

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validation and documentation

- [x] T027 Run make lint and fix any issues
- [x] T028 Run make test and verify all tests pass
- [x] T029 [P] Build all region binaries with goreleaser build --snapshot --clean
- [x] T030 Verify concurrent RPC handling with stress test
- [x] T031 [P] Update CLAUDE.md with 003-ca-sa-region-support completion notes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - US1 and US2 can proceed in parallel (both P1)
  - US3 requires binaries from US1/US2 for testing
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational - No dependencies on other stories
- **User Story 3 (P2)**: Can start after US1 and US2 binaries exist (for cross-region testing)

### Within Each User Story

- Tests written first
- Build verification follows tests
- Binary size and service validation complete the story

### Parallel Opportunities

- T002, T003: Generate pricing data in parallel
- T004, T005: Create embed files in parallel
- T007, T008: Add build targets in parallel
- T010, T011, T012: All US1 tests in parallel
- T016, T017, T018: All US2 tests in parallel
- T022, T023, T024: All US3 tests in parallel
- US1 and US2 can be worked on in parallel (both P1 priority)

---

## Parallel Example: User Story 1 + User Story 2

```bash
# US1 and US2 are both P1 priority - work in parallel:

# Developer A (US1):
Task: "Add ca-central-1 EC2 test cases in internal/pricing/client_test.go"
Task: "Add ca-central-1 EBS test cases in internal/pricing/client_test.go"
Task: "Add ca-central-1 plugin test cases in internal/plugin/plugin_test.go"
# Then:
Task: "Build and verify ca-central-1 binary"

# Developer B (US2):
Task: "Add sa-east-1 EC2 test cases in internal/pricing/client_test.go"
Task: "Add sa-east-1 EBS test cases in internal/pricing/client_test.go"
Task: "Add sa-east-1 plugin test cases in internal/plugin/plugin_test.go"
# Then:
Task: "Build and verify sa-east-1 binary"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup (pricing data generation)
2. Complete Phase 2: Foundational (embed files, GoReleaser config)
3. Complete Phase 3: User Story 1 (ca-central-1)
4. Complete Phase 4: User Story 2 (sa-east-1)
5. **STOP and VALIDATE**: Both binaries build and serve correct pricing
6. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test ca-central-1 independently → Verify
3. Add User Story 2 → Test sa-east-1 independently → Verify
4. Add User Story 3 → Test region mismatch → Complete
5. Polish phase for final validation

### Single Developer Strategy

1. Complete Setup + Foundational (T001-T009)
2. Complete US1 (T010-T015) - ca-central-1 fully working
3. Complete US2 (T016-T021) - sa-east-1 fully working
4. Complete US3 (T022-T026) - region validation confirmed
5. Polish (T027-T031) - all tests pass, docs updated

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- US1 and US2 are both P1 priority and can be implemented in parallel
- US3 depends on binaries from US1/US2 for cross-region testing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Total: 31 tasks across 6 phases
