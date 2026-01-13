# Tasks: Asia Pacific Region Support

**Input**: Design documents from `/specs/002-ap-region-support/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/region-mappings.md

**Tests**: Tests are NOT explicitly requested in this specification. Task list focuses on implementation and manual validation.

**Organization**: Tasks are grouped by user story (AP region) to enable independent implementation and testing of each region binary.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1=Singapore, US2=Sydney, US3=Tokyo, US4=Mumbai)
- Include exact file paths in descriptions

## Path Conventions

Repository structure (single Go project):
- Source code: `internal/`, `cmd/`, `tools/`
- Pricing data: `internal/pricing/data/`
- Build config: `.goreleaser.yaml`
- Documentation: `README.md`, `CLAUDE.md`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Generate pricing data for all AP regions and update build infrastructure

**⚠️ CRITICAL**: These tasks must complete before ANY region-specific binary can be built

- [ ] T001 Generate pricing data for ap-southeast-1 in internal/pricing/data/aws_pricing_ap-southeast-1.json using tools/generate-pricing with --dummy flag
- [ ] T002 [P] Generate pricing data for ap-southeast-2 in internal/pricing/data/aws_pricing_ap-southeast-2.json using tools/generate-pricing with --dummy flag
- [ ] T003 [P] Generate pricing data for ap-northeast-1 in internal/pricing/data/aws_pricing_ap-northeast-1.json using tools/generate-pricing with --dummy flag
- [ ] T004 [P] Generate pricing data for ap-south-1 in internal/pricing/data/aws_pricing_ap-south-1.json using tools/generate-pricing with --dummy flag
- [ ] T005 Update fallback embed file internal/pricing/embed_fallback.go to exclude new AP region tags (!region_apse1 && !region_apse2 && !region_apne1 && !region_aps1)
- [ ] T006 Update tools/generate-pricing/main.go to include ap-southeast-1, ap-southeast-2, ap-northeast-1, ap-south-1 in supported regions list
- [ ] T007 Update .goreleaser.yaml before hook to generate pricing data for all 7 regions (existing 3 + new 4 AP regions)

**Checkpoint**: Pricing data files exist for all 4 AP regions, infrastructure ready for region-specific builds

---

## Phase 2: User Story 1 - Singapore Region (ap-southeast-1) (Priority: P1) 🎯 MVP

**Goal**: Enable cost estimation for AWS resources in Singapore (ap-southeast-1) with region-specific pricing

**Independent Test**: Build ap-southeast-1 binary, start it, use grpcurl to request cost for t3.micro in ap-southeast-1, verify Singapore pricing returned and wrong-region requests rejected

### Implementation for User Story 1

- [ ] T008 [US1] Create embed file internal/pricing/embed_apse1.go with build tag region_apse1 and go:embed for data/aws_pricing_ap-southeast-1.json
- [ ] T009 [US1] Add GoReleaser build configuration to .goreleaser.yaml for ap-southeast-1 binary (id: ap-southeast-1, binary: finfocus-plugin-aws-public-ap-southeast-1, tags: region_apse1, platforms: linux/darwin/windows, arch: amd64/arm64)
- [ ] T010 [US1] Extend internal/plugin/supports_test.go with table-driven test cases for ap-southeast-1 region validation (matching and mismatching regions)
- [ ] T011 [US1] Extend internal/plugin/projected_test.go with test cases for ap-southeast-1 EC2 and EBS pricing calculations
- [ ] T012 [US1] Extend internal/pricing/client_test.go with test cases for ap-southeast-1 pricing data loading and lookup
- [ ] T013 [US1] Build ap-southeast-1 binary with go build -tags region_apse1 and verify pricing data embedded correctly
- [ ] T014 [US1] Manual validation: Start ap-southeast-1 binary, test with grpcurl for Supports(), GetProjectedCost() with t3.micro and gp3 volume, verify Singapore pricing and region rejection

**Checkpoint**: Singapore binary builds successfully, returns correct pricing, rejects non-Singapore regions, all tests pass

---

## Phase 3: User Story 2 - Sydney Region (ap-southeast-2) (Priority: P2)

**Goal**: Enable cost estimation for AWS resources in Sydney (ap-southeast-2) with region-specific pricing

**Independent Test**: Build ap-southeast-2 binary, verify Sydney pricing for m5.large and io2 volume, verify region rejection works

### Implementation for User Story 2

- [ ] T015 [P] [US2] Create embed file internal/pricing/embed_apse2.go with build tag region_apse2 and go:embed for data/aws_pricing_ap-southeast-2.json
- [ ] T016 [US2] Add GoReleaser build configuration to .goreleaser.yaml for ap-southeast-2 binary (id: ap-southeast-2, binary: finfocus-plugin-aws-public-ap-southeast-2, tags: region_apse2, platforms: linux/darwin/windows, arch: amd64/arm64)
- [ ] T017 [P] [US2] Extend internal/plugin/supports_test.go with test cases for ap-southeast-2 region validation
- [ ] T018 [P] [US2] Extend internal/plugin/projected_test.go with test cases for ap-southeast-2 EC2 and EBS pricing calculations
- [ ] T019 [P] [US2] Extend internal/pricing/client_test.go with test cases for ap-southeast-2 pricing data loading
- [ ] T020 [US2] Build ap-southeast-2 binary with go build -tags region_apse2 and verify pricing data embedded
- [ ] T021 [US2] Manual validation: Start ap-southeast-2 binary, test with grpcurl for m5.large and io2 volume pricing, verify Sydney rates

**Checkpoint**: Sydney binary works independently, returns correct pricing, all tests pass

---

## Phase 4: User Story 3 - Tokyo Region (ap-northeast-1) (Priority: P2)

**Goal**: Enable cost estimation for AWS resources in Tokyo (ap-northeast-1) with region-specific pricing

**Independent Test**: Build ap-northeast-1 binary, verify Tokyo pricing for t3.medium, verify multi-region scenarios work correctly

### Implementation for User Story 3

- [ ] T022 [P] [US3] Create embed file internal/pricing/embed_apne1.go with build tag region_apne1 and go:embed for data/aws_pricing_ap-northeast-1.json
- [ ] T023 [US3] Add GoReleaser build configuration to .goreleaser.yaml for ap-northeast-1 binary (id: ap-northeast-1, binary: finfocus-plugin-aws-public-ap-northeast-1, tags: region_apne1, platforms: linux/darwin/windows, arch: amd64/arm64)
- [ ] T024 [P] [US3] Extend internal/plugin/supports_test.go with test cases for ap-northeast-1 region validation
- [ ] T025 [P] [US3] Extend internal/plugin/projected_test.go with test cases for ap-northeast-1 EC2 and EBS pricing calculations
- [ ] T026 [P] [US3] Extend internal/pricing/client_test.go with test cases for ap-northeast-1 pricing data loading
- [ ] T027 [US3] Build ap-northeast-1 binary with go build -tags region_apne1 and verify pricing data embedded
- [ ] T028 [US3] Manual validation: Start ap-northeast-1 binary, test with grpcurl for t3.medium pricing, verify Tokyo rates and region rejection

**Checkpoint**: Tokyo binary works independently, returns correct pricing, all tests pass

---

## Phase 5: User Story 4 - Mumbai Region (ap-south-1) (Priority: P3)

**Goal**: Enable cost estimation for AWS resources in Mumbai (ap-south-1) with region-specific pricing

**Independent Test**: Build ap-south-1 binary, verify Mumbai pricing for c5.xlarge and gp2 volume

### Implementation for User Story 4

- [ ] T029 [P] [US4] Create embed file internal/pricing/embed_aps1.go with build tag region_aps1 and go:embed for data/aws_pricing_ap-south-1.json
- [ ] T030 [US4] Add GoReleaser build configuration to .goreleaser.yaml for ap-south-1 binary (id: ap-south-1, binary: finfocus-plugin-aws-public-ap-south-1, tags: region_aps1, platforms: linux/darwin/windows, arch: amd64/arm64)
- [ ] T031 [P] [US4] Extend internal/plugin/supports_test.go with test cases for ap-south-1 region validation
- [ ] T032 [P] [US4] Extend internal/plugin/projected_test.go with test cases for ap-south-1 EC2 and EBS pricing calculations
- [ ] T033 [P] [US4] Extend internal/pricing/client_test.go with test cases for ap-south-1 pricing data loading
- [ ] T034 [US4] Build ap-south-1 binary with go build -tags region_aps1 and verify pricing data embedded
- [ ] T035 [US4] Manual validation: Start ap-south-1 binary, test with grpcurl for c5.xlarge and gp2 volume pricing, verify Mumbai rates

**Checkpoint**: Mumbai binary works independently, returns correct pricing, all tests pass

---

## Phase 6: Integration & Multi-Region Build

**Purpose**: Verify all 4 AP regions build together and validate success criteria

- [ ] T036 Run make test to verify all unit tests pass for all regions
- [ ] T037 Run make lint to ensure code quality across all new files
- [ ] T038 Build all 4 AP region binaries using goreleaser build --snapshot --clean --id ap-southeast-1,ap-southeast-2,ap-northeast-1,ap-south-1
- [ ] T039 Verify each binary is under 20MB using ls -lh dist/ (Success Criteria SC-002)
- [ ] T040 Run concurrent gRPC call test (10+ parallel requests) against one AP binary to verify thread safety (Success Criteria SC-006)
- [ ] T041 Verify region mismatch error response time is under 100ms using benchmark test (Success Criteria SC-005)
- [ ] T042 Test that cost estimates for t3.micro differ across AP regions, confirming region-specific pricing (Success Criteria SC-003)
- [ ] T043 Verify each AP binary correctly rejects requests for other regions 100% of the time (Success Criteria SC-008)

**Checkpoint**: All 4 AP region binaries build successfully, all success criteria validated

---

## Phase 7: Polish & Documentation

**Purpose**: Complete documentation and prepare for release

- [ ] T044 [P] Update README.md with Asia Pacific region support section listing all 4 new regions with city names
- [ ] T045 [P] Update README.md with build examples for AP region binaries
- [ ] T046 [P] Add AP region examples to README.md's quick start section
- [ ] T047 Run markdownlint on README.md and fix any issues
- [ ] T048 [P] Update CLAUDE.md if any new patterns emerged during implementation (build tags, pricing data generation, testing patterns)
- [ ] T049 Verify all quickstart.md commands work correctly for AP regions
- [ ] T050 Create git commit with conventional commit message describing AP region support addition

**Checkpoint**: Documentation complete, ready for PR

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
  - Tasks T001-T004 (pricing data generation) can run in parallel
  - Tasks T005-T007 (infrastructure updates) must complete before region builds
- **User Stories (Phases 2-5)**: Each depends on Setup (Phase 1) completion
  - User stories are INDEPENDENT - can proceed in parallel or sequentially by priority
  - Singapore (P1) → Sydney (P2) → Tokyo (P2) → Mumbai (P3)
- **Integration (Phase 6)**: Depends on all 4 user stories completing
- **Polish (Phase 7)**: Depends on Integration phase

### User Story Dependencies

- **User Story 1 - Singapore (P1)**: Depends only on Phase 1 (Setup) - MVP region
- **User Story 2 - Sydney (P2)**: Depends only on Phase 1 (Setup) - INDEPENDENT of US1
- **User Story 3 - Tokyo (P2)**: Depends only on Phase 1 (Setup) - INDEPENDENT of US1, US2
- **User Story 4 - Mumbai (P3)**: Depends only on Phase 1 (Setup) - INDEPENDENT of US1, US2, US3

### Within Each User Story

- Embed file creation (T008, T015, T022, T029) before GoReleaser config
- GoReleaser config before build
- Tests can be written in parallel with embed/config creation
- Manual validation after successful build

### Parallel Opportunities

**Phase 1 (Setup)**:
- Tasks T001-T004: All pricing data generation in parallel
- Task T005: Fallback embed update (parallel with T001-T004)
- Task T006: Generate-pricing tool update (parallel with T001-T004)

**User Stories (After Setup Complete)**:
- All 4 user stories (Phases 2-5) can proceed in parallel with separate developers
- Within each story:
  - T010+T011+T012 (tests) can run in parallel for US1
  - T017+T018+T019 (tests) can run in parallel for US2
  - T024+T025+T026 (tests) can run in parallel for US3
  - T031+T032+T033 (tests) can run in parallel for US4

**Phase 7 (Polish)**:
- Tasks T044-T046 (README updates) can run in parallel
- Task T048 (CLAUDE.md) can run in parallel with README tasks

---

## Parallel Example: User Story 1 (Singapore)

```bash
# After Phase 1 completes, launch all Singapore tasks in parallel:

# Parallel batch 1: Embed + Config + Tests
Task T008: "Create internal/pricing/embed_apse1.go"
Task T009: "Add ap-southeast-1 build config to .goreleaser.yaml"
Task T010: "Extend supports_test.go for ap-southeast-1"
Task T011: "Extend projected_test.go for ap-southeast-1"
Task T012: "Extend client_test.go for ap-southeast-1"

# Then sequential: Build and validate
Task T013: "Build ap-southeast-1 binary"
Task T014: "Manual validation with grpcurl"
```

---

## Parallel Example: All User Stories (After Setup)

```bash
# With 4 developers after Phase 1 completes:

Developer A (US1 - Singapore):
  Tasks T008-T014

Developer B (US2 - Sydney):
  Tasks T015-T021

Developer C (US3 - Tokyo):
  Tasks T022-T028

Developer D (US4 - Mumbai):
  Tasks T029-T035

# All developers work in parallel, no conflicts
```

---

## Implementation Strategy

### MVP First (Singapore Only)

1. Complete Phase 1: Setup (T001-T007)
2. Complete Phase 2: Singapore (T008-T014)
3. **STOP and VALIDATE**: Test Singapore binary independently
4. Verify all acceptance scenarios for US1
5. Deploy/demo if ready (1 region is better than 0!)

### Incremental Delivery

1. Setup → Foundation ready (T001-T007)
2. Add Singapore → Test independently → Deploy (MVP with 1 AP region)
3. Add Sydney → Test independently → Deploy (2 AP regions)
4. Add Tokyo → Test independently → Deploy (3 AP regions)
5. Add Mumbai → Test independently → Deploy (4 AP regions complete)
6. Each region adds value without breaking previous regions

### Parallel Team Strategy

With 4+ developers:

1. Team completes Phase 1 (Setup) together
2. Once Setup is done (T007 complete):
   - Developer A: Singapore (US1)
   - Developer B: Sydney (US2)
   - Developer C: Tokyo (US3)
   - Developer D: Mumbai (US4)
3. Each developer works independently on their region
4. Integration phase validates all regions work together

---

## Success Criteria Validation

Map tasks to spec.md success criteria:

- **SC-001** (Build all 4 binaries): Validated by T038
- **SC-002** (Binary size < 20MB): Validated by T039
- **SC-003** (Pricing differs across regions): Validated by T042
- **SC-004** (All tests pass): Validated by T036
- **SC-005** (Region mismatch < 100ms): Validated by T041
- **SC-006** (100+ concurrent calls): Validated by T040
- **SC-007** (Build time < 2 min): Monitored during T038
- **SC-008** (Region rejection 100%): Validated by T043

---

## Notes

- [P] tasks = different files, no dependencies between them
- [US1/US2/US3/US4] labels map tasks to specific regions for traceability
- Each region (user story) is independently completable and testable
- Stop at any checkpoint to validate region independently
- Commit after each user story phase or logical group
- No tests explicitly requested in spec, relying on manual validation + existing test extension
- All regions follow identical pattern (embed file → GoReleaser config → tests → build → validate)
- Build tags ensure exactly one pricing file embedded per binary
- Manual grpcurl testing critical for validating gRPC protocol compliance
