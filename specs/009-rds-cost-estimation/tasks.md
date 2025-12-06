# Tasks: RDS Instance Cost Estimation

**Input**: Design documents from `/specs/009-rds-cost-estimation/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md

**Tests**: Included per spec requirement (SC-005: 100% unit tests pass)

**Organization**: Tasks grouped by user story for independent implementation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: User story (US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md structure:

- `internal/pricing/` - Pricing client and data types
- `internal/plugin/` - Plugin implementation
- `tools/generate-pricing/` - Pricing data generation tool

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Extend existing project structure for RDS support

- [ ] T001 Add RDS price types to internal/pricing/types.go
- [ ] T002 [P] Add RDS interface methods to PricingClient in internal/pricing/client.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core RDS pricing infrastructure that ALL user stories depend on

**CRITICAL**: No user story work can begin until this phase is complete

- [ ] T003 Add rdsInstanceIndex and rdsStorageIndex maps to Client struct in internal/pricing/client.go
- [ ] T004 Implement RDS instance parsing in init() function in internal/pricing/client.go
- [ ] T005 Implement RDS storage parsing in init() function in internal/pricing/client.go (after T004)
- [ ] T006 Implement RDSOnDemandPricePerHour() lookup method in internal/pricing/client.go
- [ ] T007 Implement RDSStoragePricePerGBMonth() lookup method in internal/pricing/client.go
- [ ] T008 Update tools/generate-pricing/main.go to support AmazonRDS service code

**Checkpoint**: Foundation ready - RDS pricing lookups functional

---

## Phase 3: User Story 1 - Basic RDS Instance Cost Query (Priority: P1)

**Goal**: Return accurate cost estimates for RDS instances based on type and engine

**Independent Test**: Query db.t3.medium MySQL, verify non-zero hourly rate and monthly cost

### Tests for User Story 1

- [ ] T009 [P] [US1] Add unit test for RDSOnDemandPricePerHour() in internal/pricing/client_test.go
- [ ] T010 [P] [US1] Add unit test for estimateRDS() with MySQL in internal/plugin/projected_test.go

### Implementation for User Story 1

- [ ] T011 [US1] Create estimateRDS() function skeleton in internal/plugin/projected.go
- [ ] T012 [US1] Implement instance type extraction from resource.Sku in internal/plugin/projected.go
- [ ] T013 [US1] Implement engine extraction with mysql default in internal/plugin/projected.go
- [ ] T014 [US1] Implement engine normalization map in internal/plugin/projected.go
- [ ] T015 [US1] Implement hourly rate lookup via pricing client in internal/plugin/projected.go
- [ ] T016 [US1] Implement monthly cost calculation (rate × 730) in internal/plugin/projected.go
- [ ] T017 [US1] Implement billing_detail message for instance-only in internal/plugin/projected.go
- [ ] T018 [US1] Update router switch case for "rds" in internal/plugin/projected.go
- [ ] T019 [US1] Handle unknown instance type ($0 with explanation) in internal/plugin/projected.go

**Checkpoint**: User Story 1 complete - RDS instance cost queries return accurate results

---

## Phase 4: User Story 4 - Supports Query for RDS (Priority: P1)

**Goal**: Supports() returns supported=true for RDS without "Limited support" caveat

**Independent Test**: Call Supports() with resource_type "rds", verify supported=true

### Tests for User Story 4

- [ ] T020 [P] [US4] Add unit test for Supports() with RDS in internal/plugin/supports_test.go

### Implementation for User Story 4

- [ ] T021 [US4] Move "rds" from stub case to fully-supported in internal/plugin/supports.go
- [ ] T022 [US4] Update Supports() reason message for RDS in internal/plugin/supports.go

**Checkpoint**: User Story 4 complete - plugin correctly reports RDS support

---

## Phase 5: User Story 2 - RDS Storage Cost Estimation (Priority: P2)

**Goal**: Include storage costs (per GB-month) in RDS estimates

**Independent Test**: Query with storage_type and storage_size, verify storage added to total

### Tests for User Story 2

- [ ] T023 [P] [US2] Add unit test for RDSStoragePricePerGBMonth() in internal/pricing/client_test.go
- [ ] T024 [P] [US2] Add unit test for estimateRDS() with storage in internal/plugin/projected_test.go

### Implementation for User Story 2

- [ ] T025 [US2] Extract storage_type from tags with gp2 default in internal/plugin/projected.go
- [ ] T026 [US2] Extract storage_size from tags with 20GB default in internal/plugin/projected.go
- [ ] T027 [US2] Implement storage rate lookup via pricing client in internal/plugin/projected.go
- [ ] T028 [US2] Combine instance + storage costs in cost_per_month in internal/plugin/projected.go
- [ ] T029 [US2] Update billing_detail to include storage info in internal/plugin/projected.go
- [ ] T030 [US2] Handle invalid storage_size (default to 20GB) in internal/plugin/projected.go
- [ ] T031 [US2] Handle unknown storage_type (default to gp2) in internal/plugin/projected.go

**Checkpoint**: User Story 2 complete - RDS estimates include storage costs

---

## Phase 6: User Story 3 - Multi-Engine Support (Priority: P3)

**Goal**: Support all major database engines with engine-specific pricing

**Independent Test**: Query each engine type, verify engine-specific pricing returned

### Tests for User Story 3

- [ ] T032 [P] [US3] Add table-driven test for all engines in internal/plugin/projected_test.go
- [ ] T033 [P] [US3] Add test for unknown engine defaulting to MySQL in internal/plugin/projected_test.go

### Implementation for User Story 3

- [ ] T034 [US3] Add PostgreSQL to engine normalization map in internal/plugin/projected.go
- [ ] T035 [US3] Add MariaDB to engine normalization map in internal/plugin/projected.go
- [ ] T036 [US3] Add Oracle SE2 to engine normalization map in internal/plugin/projected.go
- [ ] T037 [US3] Add SQL Server Express to engine normalization map in internal/plugin/projected.go
- [ ] T038 [US3] Handle unknown engine (default to mysql with note) in internal/plugin/projected.go
- [ ] T039 [US3] Update billing_detail to show actual engine used in internal/plugin/projected.go

**Checkpoint**: User Story 3 complete - all engines supported with accurate pricing

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements affecting multiple user stories

- [ ] T040 [P] Add debug logging for RDS pricing lookups in internal/plugin/projected.go
- [ ] T041 [P] Add performance warning logging (>50ms) in internal/pricing/client.go
- [ ] T042 Run make lint and fix any issues
- [ ] T043 Run make test and verify all tests pass
- [ ] T044 Validate against quickstart.md scenarios manually with grpcurl
- [ ] T045 Update mock pricing client for test coverage in internal/plugin/plugin_test.go
- [ ] T046 Build and validate RDS pricing in all 9 regional binaries (goreleaser build --snapshot)
- [ ] T047 Spot-check db.t3.medium MySQL price against AWS Calculator for us-east-1, eu-west-1
- [ ] T048 [P] Add concurrent access test for RDS pricing lookups in internal/pricing/client_test.go

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Stories (Phases 3-6)**: All depend on Foundational completion
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational - No dependencies on other stories
- **US4 (P1)**: Can start after Foundational - No dependencies on other stories
- **US2 (P2)**: Depends on US1 (uses estimateRDS() function created in US1)
- **US3 (P3)**: Depends on US1 (extends engine normalization from US1)

### Within Each User Story

- Tests written first (TDD approach)
- Core implementation before edge cases
- Commit after each logical group

### Parallel Opportunities

**Phase 1 (Setup)**:

- T001 and T002 can run in parallel (different areas of types.go and client.go interface)

**Phase 2 (Foundational)**:

- T004 then T005 sequentially (same init() function to avoid conflicts)
- T006 and T007 can run in parallel (different lookup methods)

**Phase 3 (US1) + Phase 4 (US4)**:

- US1 and US4 can proceed in parallel (different files: projected.go vs supports.go)
- T009 and T010 can run in parallel (different test files)

**Phase 5-6 (US2, US3)**:

- US2 and US3 tests can run in parallel
- After US1, implementation of US2 and US3 can partially overlap

---

## Parallel Example: Foundational Phase

```bash
# Launch instance and storage parsing together:
Task: "Implement RDS instance parsing in init() function in internal/pricing/client.go"
Task: "Implement RDS storage parsing in init() function in internal/pricing/client.go"

# Launch lookup methods together:
Task: "Implement RDSOnDemandPricePerHour() lookup method in internal/pricing/client.go"
Task: "Implement RDSStoragePricePerGBMonth() lookup method in internal/pricing/client.go"
```

## Parallel Example: User Story 1 & 4

```bash
# US1 and US4 can proceed in parallel after Foundational:
# Developer A (US1):
Task: "Create estimateRDS() function skeleton in internal/plugin/projected.go"
Task: "Implement instance type extraction from resource.Sku in internal/plugin/projected.go"

# Developer B (US4):
Task: "Move 'rds' from stub case to fully-supported in internal/plugin/supports.go"
Task: "Update Supports() reason message for RDS in internal/plugin/supports.go"
```

---

## Implementation Strategy

### MVP First (User Stories 1 & 4 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1 (instance cost)
4. Complete Phase 4: User Story 4 (Supports)
5. **STOP and VALIDATE**: Test RDS instance queries work with grpcurl
6. Deploy/demo if ready - basic RDS support functional

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 + US4 → Test independently → MVP ready
3. Add US2 → Storage costs included → Enhanced MVP
4. Add US3 → All engines supported → Full feature
5. Polish → Production ready

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to user story
- Each user story independently testable after Foundational
- Verify tests fail before implementing
- Commit after each task or logical group
- Follow existing EC2/EBS patterns for consistency
