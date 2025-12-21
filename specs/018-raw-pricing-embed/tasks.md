# Tasks: Embed Raw AWS Pricing JSON Per Service

**Input**: Design documents from `/specs/018-raw-pricing-embed/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md

**Tests**: Tests are included as this is a refactor with regression risk (v0.0.10/v0.0.11 bugs).

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Go module**: `internal/pricing/`, `tools/generate-pricing/`
- Paths assume repository root

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Clean up old combined pricing approach and prepare for per-service files

- [X] T001 Remove old combined pricing file pattern from `.gitignore` and add per-service pattern in `.gitignore`
- [X] T002 Delete old combined pricing data files from `internal/pricing/data/aws_pricing_*.json`
- [X] T003 [P] Update Makefile build targets to document per-service file generation in `Makefile`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core changes that MUST be complete before user stories can be validated

**CRITICAL**: All embed files depend on the generation tool producing per-service files

### Rewrite Generation Tool

- [X] T004 Refactor `tools/generate-pricing/main.go` to output per-service JSON files instead of combined blob
- [X] T005 Add fail-fast behavior if any service fetch fails in `tools/generate-pricing/main.go`
- [X] T006 Update file naming convention to `{service}_{region}.json` in `tools/generate-pricing/main.go`

### Rewrite Embed Files (All 12 Regions)

- [X] T007 [P] Rewrite `internal/pricing/embed_use1.go` with 7 per-service embed directives
- [X] T008 [P] Rewrite `internal/pricing/embed_usw2.go` with 7 per-service embed directives
- [X] T009 [P] Rewrite `internal/pricing/embed_euw1.go` with 7 per-service embed directives
- [X] T010 [P] Rewrite `internal/pricing/embed_apse1.go` with 7 per-service embed directives
- [X] T011 [P] Rewrite `internal/pricing/embed_apse2.go` with 7 per-service embed directives
- [X] T012 [P] Rewrite `internal/pricing/embed_apne1.go` with 7 per-service embed directives
- [X] T013 [P] Rewrite `internal/pricing/embed_aps1.go` with 7 per-service embed directives
- [X] T014 [P] Rewrite `internal/pricing/embed_cac1.go` with 7 per-service embed directives
- [X] T015 [P] Rewrite `internal/pricing/embed_sae1.go` with 7 per-service embed directives
- [X] T016 [P] Rewrite `internal/pricing/embed_govw1.go` with 7 per-service embed directives
- [X] T017 [P] Rewrite `internal/pricing/embed_gove1.go` with 7 per-service embed directives
- [X] T018 [P] Rewrite `internal/pricing/embed_usw1.go` with 7 per-service embed directives
- [X] T019 [P] Rewrite `internal/pricing/embed_fallback.go` with 7 per-service fallback variables

### Update Client Initialization

- [ ] T020 Modify `internal/pricing/client.go` `init()` to parse each service file independently
- [ ] T021 Add helper functions for per-service parsing in `internal/pricing/client.go`

**Checkpoint**: Foundation ready - generate pricing data and verify build compiles

---

## Phase 3: User Story 1 - Accurate Pricing Data Retrieval (Priority: P1)

**Goal**: Plugin returns accurate pricing for all supported AWS services matching real AWS pricing

**Independent Test**: Query EC2 instance pricing (t3.micro in us-east-1) and verify price matches AWS API

### Tests for User Story 1

- [ ] T022 [P] [US1] Add integration test for EC2 pricing accuracy in `internal/plugin/integration_verify_pricing_test.go`
- [ ] T023 [P] [US1] Add unit test for client parsing of raw EC2 JSON in `internal/pricing/client_test.go`
- [ ] T024 [P] [US1] Add unit test for client parsing of raw ELB JSON in `internal/pricing/client_test.go`

### Implementation for User Story 1

- [ ] T025 [US1] Generate pricing data for us-east-1 using updated generation tool
- [ ] T026 [US1] Build us-east-1 binary with `-tags=region_use1` and verify no compilation errors
- [ ] T027 [US1] Run existing integration tests to verify backward compatibility

**Checkpoint**: EC2/EBS/ELB pricing returns accurate values from raw embedded data

---

## Phase 4: User Story 2 - Service-Specific Pricing Isolation (Priority: P1)

**Goal**: Each AWS service's pricing data is stored in separate files for debugging

**Independent Test**: Examine generated pricing files and verify each contains only its service's data

### Tests for User Story 2

- [ ] T028 [P] [US2] Add unit test verifying ec2 file contains only EC2 products in `internal/pricing/client_test.go`
- [ ] T029 [P] [US2] Add unit test verifying elb file contains only ELB products in `internal/pricing/client_test.go`

### Implementation for User Story 2

- [ ] T030 [US2] Verify generation tool preserves `offerCode` from AWS response in `tools/generate-pricing/main.go`
- [ ] T031 [US2] Add metadata validation in `internal/pricing/client.go` init() to verify offerCode matches expected service

**Checkpoint**: Each service file contains only its service's products with correct offerCode

---

## Phase 5: User Story 3 - Preserved AWS Metadata (Priority: P2)

**Goal**: Embedded pricing data retains AWS version and publication metadata

**Independent Test**: Parse embedded pricing file and extract version/publicationDate fields

### Tests for User Story 3

- [ ] T032 [P] [US3] Add unit test verifying AWS metadata preservation (version, publicationDate) in `internal/pricing/client_test.go`

### Implementation for User Story 3

- [ ] T033 [US3] Ensure generation tool does not modify any AWS metadata fields in `tools/generate-pricing/main.go`
- [ ] T034 [US3] Add optional logging of embedded pricing metadata during client init in `internal/pricing/client.go`

**Checkpoint**: AWS metadata (version, publicationDate) accessible from embedded data

---

## Phase 6: User Story 4 - Consistent Build Verification (Priority: P2)

**Goal**: Per-service threshold tests prevent release of binaries with broken pricing

**Independent Test**: Run unit tests with build tags and verify they fail when data is below thresholds

### Tests for User Story 4

- [ ] T035 [P] [US4] Add per-service size threshold test for EC2 (>100MB) in `internal/pricing/embed_test.go`
- [ ] T036 [P] [US4] Add per-service size threshold test for RDS (>10MB) in `internal/pricing/embed_test.go`
- [ ] T037 [P] [US4] Add per-service size threshold test for EKS (>2MB) in `internal/pricing/embed_test.go`
- [ ] T038 [P] [US4] Add per-service size threshold test for Lambda (>1MB) in `internal/pricing/embed_test.go`
- [ ] T039 [P] [US4] Add per-service size threshold test for S3 (>500KB) in `internal/pricing/embed_test.go`
- [ ] T040 [P] [US4] Add per-service size threshold test for DynamoDB (>400KB) in `internal/pricing/embed_test.go`
- [ ] T041 [P] [US4] Add per-service size threshold test for ELB (>400KB) in `internal/pricing/embed_test.go`

### Implementation for User Story 4

- [ ] T042 [US4] Remove old combined-data threshold tests in `internal/pricing/embed_test.go`
- [ ] T043 [US4] Document threshold values in CLAUDE.md for future reference

**Checkpoint**: Per-service threshold tests pass for all 7 services

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Multi-region validation and documentation

- [ ] T044 [P] Generate pricing data for all 12 regions using `tools/generate-pricing`
- [ ] T045 [P] Build binaries for us-west-2 and eu-west-1 to verify multi-region support
- [ ] T046 Run `make lint` and fix any linting issues
- [ ] T047 Run `make test` and verify all tests pass
- [ ] T048 Update quickstart.md with new file structure in `specs/018-raw-pricing-embed/quickstart.md`
- [ ] T049 Update CLAUDE.md with per-service file naming convention and thresholds
- [ ] T050 Add binary size check to CI workflow (200MB warning, 240MB critical) per constitution requirement in `.github/workflows/`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - US1 and US2 can proceed in parallel (both P1 priority)
  - US3 and US4 can proceed after US1/US2 or in parallel (P2 priority)
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - Core pricing accuracy
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - Service isolation (parallel with US1)
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Metadata preservation
- **User Story 4 (P2)**: Can start after Foundational (Phase 2) - Build verification

### Within Foundational Phase

Critical path: T004 → T005 → T006 (generation tool) must complete before T007-T019 (embed files)
T007-T019 (embed files) can all run in parallel
T020-T021 (client init) must complete after embed files

### Parallel Opportunities

- T007-T019: All 13 embed file rewrites can run in parallel
- T022-T024: All US1 tests can run in parallel
- T028-T029: All US2 tests can run in parallel
- T035-T041: All US4 threshold tests can run in parallel
- T044-T045: Multi-region builds can run in parallel

---

## Parallel Example: Embed File Rewrites

```bash
# Launch all embed file rewrites together (after T004-T006 complete):
Task: "Rewrite internal/pricing/embed_use1.go with 7 per-service embed directives"
Task: "Rewrite internal/pricing/embed_usw2.go with 7 per-service embed directives"
Task: "Rewrite internal/pricing/embed_euw1.go with 7 per-service embed directives"
# ... (all 13 files in parallel)
```

---

## Parallel Example: Per-Service Threshold Tests

```bash
# Launch all threshold tests together:
Task: "Add per-service size threshold test for EC2 (>100MB) in internal/pricing/embed_test.go"
Task: "Add per-service size threshold test for RDS (>10MB) in internal/pricing/embed_test.go"
Task: "Add per-service size threshold test for EKS (>2MB) in internal/pricing/embed_test.go"
# ... (all 7 services in parallel)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1 (Accurate Pricing)
4. **STOP and VALIDATE**: Run integration tests, verify t3.micro pricing
5. Deploy if ready - core functionality works

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Core pricing works (MVP!)
3. Add User Story 2 → Test independently → Service isolation for debugging
4. Add User Story 3 → Test independently → Metadata available for troubleshooting
5. Add User Story 4 → Test independently → Build verification prevents regressions

### Risk Mitigation

This refactor specifically addresses v0.0.10/v0.0.11 pricing bugs:

- US1 validates pricing accuracy (prevents $0 returns)
- US2 enables debugging (prevents silent data corruption)
- US4 adds threshold tests (prevents future regressions)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- **CRITICAL**: Run `make lint` and `make test` after completing each phase
