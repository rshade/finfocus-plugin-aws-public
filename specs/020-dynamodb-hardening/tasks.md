# Tasks: DynamoDB Hardening Bundle

**Input**: Design documents from `/specs/020-dynamodb-hardening/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, quickstart.md

**Tests**: Tests are explicitly requested in the spec (FR-014, FR-015, FR-016). Both unit and integration tests are included.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Single project (Go)**: `internal/plugin/`, `internal/pricing/` at repository root
- No new files except `integration_dynamodb_test.go`

---

## Phase 1: Setup (No new infrastructure needed)

**Purpose**: Verify existing codebase and understand current implementation

- [X] T001 Review current `estimateDynamoDB()` implementation in internal/plugin/projected.go:474-593
- [X] T002 [P] Review DynamoDB pricing lookup methods in internal/pricing/client.go:1337-1453
- [X] T003 [P] Review existing integration test pattern in internal/plugin/integration_test.go

**Checkpoint**: Understanding of current implementation complete

---

## Phase 2: Foundational (Validation Helpers)

**Purpose**: Create reusable validation helpers that ALL user stories depend on

**⚠️ CRITICAL**: User stories 1 and 2 depend on these helpers

- [X] T004 Add `validateNonNegativeInt64()` helper method in internal/plugin/projected.go
- [X] T005 Add `validateNonNegativeFloat64()` helper method in internal/plugin/projected.go

**Checkpoint**: Validation helpers ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Transparent Cost Estimation Feedback (Priority: P1) 🎯 MVP

**Goal**: Emit warning logs and update billing_detail when pricing data is missing

**Independent Test**: Request DynamoDB cost estimate with fallback embed, verify warnings and billing_detail

**Related Issues**: #147

### Tests for User Story 1

- [X] T006 [P] [US1] Add unit test `TestEstimateDynamoDB_MissingStoragePricing` in internal/plugin/projected_test.go
- [X] T007 [P] [US1] Add unit test `TestEstimateDynamoDB_MissingProvisionedPricing` in internal/plugin/projected_test.go
- [X] T008 [P] [US1] Add unit test `TestEstimateDynamoDB_MissingOnDemandPricing` in internal/plugin/projected_test.go

### Implementation for User Story 1

- [X] T009 [US1] Modify storage pricing lookup to check `ok` return value in internal/plugin/projected.go:494-508
- [X] T010 [US1] Add warning log when storage pricing is unavailable in internal/plugin/projected.go
- [X] T011 [US1] Modify provisioned RCU/WCU pricing lookups to check `ok` return values in internal/plugin/projected.go:522-524
- [X] T012 [US1] Add warning logs when provisioned RCU/WCU pricing is unavailable in internal/plugin/projected.go
- [X] T013 [US1] Modify on-demand read/write pricing lookups to check `ok` return values in internal/plugin/projected.go:567-568
- [X] T014 [US1] Add warning logs when on-demand read/write pricing is unavailable in internal/plugin/projected.go
- [X] T015 [US1] Track unavailable components in slice variable in internal/plugin/projected.go
- [X] T016 [US1] Append "(pricing unavailable: ...)" to billing_detail when components missing in internal/plugin/projected.go

**Checkpoint**: User Story 1 complete - warnings emitted for missing pricing

---

## Phase 4: User Story 2 - Invalid Input Detection and Recovery (Priority: P2)

**Goal**: Validate all numeric tag inputs and log warnings for negative/invalid values

**Independent Test**: Submit DynamoDB cost request with negative tags, verify warnings and default to 0

**Related Issues**: #151

### Tests for User Story 2

- [X] T017 [P] [US2] Add table-driven unit test `TestValidateNonNegativeInt64` in internal/plugin/projected_test.go
- [X] T018 [P] [US2] Add table-driven unit test `TestValidateNonNegativeFloat64` in internal/plugin/projected_test.go
- [X] T019 [P] [US2] Add unit test `TestEstimateDynamoDB_NegativeCapacityUnits` in internal/plugin/projected_test.go
- [X] T020 [P] [US2] Add unit test `TestEstimateDynamoDB_InvalidTagValues` in internal/plugin/projected_test.go

### Implementation for User Story 2

- [X] T021 [US2] Replace storage_gb parsing with validateNonNegativeFloat64() in internal/plugin/projected.go:487-491
- [X] T022 [US2] Replace read_capacity_units parsing with validateNonNegativeInt64() in internal/plugin/projected.go:514-515
- [X] T023 [US2] Replace write_capacity_units parsing with validateNonNegativeInt64() in internal/plugin/projected.go:517-518
- [X] T024 [US2] Replace read_requests_per_month parsing with validateNonNegativeInt64() in internal/plugin/projected.go:559-560
- [X] T025 [US2] Replace write_requests_per_month parsing with validateNonNegativeInt64() in internal/plugin/projected.go:562-563

**Checkpoint**: User Story 2 complete - validation warnings emitted for invalid inputs

---

## Phase 5: User Story 3 - Accurate Pricing Extraction (Priority: P3)

**Goal**: Verify product family matching returns correct prices for all 5 DynamoDB components

**Independent Test**: Run pricing extraction tests against us-east-1 embedded data

**Related Issues**: #149

### Tests for User Story 3

- [X] T026 [P] [US3] Add `TestDynamoDBPricingExtraction_Storage` in internal/pricing/client_test.go (requires region_use1 tag)
- [X] T027 [P] [US3] Add `TestDynamoDBPricingExtraction_ProvisionedRCU` in internal/pricing/client_test.go
- [X] T028 [P] [US3] Add `TestDynamoDBPricingExtraction_ProvisionedWCU` in internal/pricing/client_test.go
- [X] T029 [P] [US3] Add `TestDynamoDBPricingExtraction_OnDemandRead` in internal/pricing/client_test.go
- [X] T030 [P] [US3] Add `TestDynamoDBPricingExtraction_OnDemandWrite` in internal/pricing/client_test.go

### Implementation for User Story 3

- [X] T031 [US3] Review parseDynamoDBPricing() product family filters in internal/pricing/client.go:779
- [X] T032 [US3] Verify provisioned vs on-demand pricing distinction in internal/pricing/client.go
- [X] T033 [US3] Add ambiguous match detection logging (FR-013) if multiple matches found in internal/pricing/client.go

**Checkpoint**: User Story 3 complete - pricing extraction verified

---

## Phase 6: User Story 4 - End-to-End Integration Tests (Priority: P4)

**Goal**: Create integration tests that validate the complete gRPC pipeline for DynamoDB

**Independent Test**: Run `go test -tags=integration ./internal/plugin/... -run DynamoDB`

**Related Issues**: #152

### Implementation for User Story 4

- [X] T034 [US4] Create new integration test file internal/plugin/integration_dynamodb_test.go with build tag
- [X] T035 [US4] Add `TestIntegration_DynamoDB_Provisioned` test case in internal/plugin/integration_dynamodb_test.go
- [X] T036 [US4] Add `TestIntegration_DynamoDB_OnDemand` test case in internal/plugin/integration_dynamodb_test.go
- [X] T037 [US4] Add test helper to build binary with region_use1 tag in internal/plugin/integration_dynamodb_test.go
- [X] T038 [US4] Add gRPC client setup and PORT detection logic in internal/plugin/integration_dynamodb_test.go
- [X] T039 [US4] Verify non-zero cost_per_month in integration test assertions

**Checkpoint**: User Story 4 complete - integration tests pass

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and cleanup

- [X] T040 Run `make lint` and fix any warnings
- [X] T041 Run `make test` with region_use1 tag and verify all tests pass
- [X] T042 [P] Run `go test -tags=integration,region_use1 ./internal/plugin/...` for integration tests
- [X] T043 Update CLAUDE.md if any new patterns or conventions emerged
- [X] T044 Verify all success criteria from spec.md are met (SC-001 through SC-007)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS User Stories 1 and 2
- **User Story 1 (Phase 3)**: Depends on Foundational (validation helpers)
- **User Story 2 (Phase 4)**: Depends on Foundational (validation helpers)
- **User Story 3 (Phase 5)**: Can start after Setup (no dependency on validation helpers)
- **User Story 4 (Phase 6)**: Can start after User Stories 1-3 (tests the complete implementation)
- **Polish (Phase 7)**: Depends on all user stories complete

### User Story Dependencies

- **User Story 1 (P1)**: Depends on T004, T005 (validation helpers)
- **User Story 2 (P2)**: Depends on T004, T005 (validation helpers)
- **User Story 3 (P3)**: No dependencies on other stories - can run in parallel with US1/US2
- **User Story 4 (P4)**: Depends on US1, US2, US3 being complete (tests the full implementation)

### Within Each User Story

- Tests (T006-T008, T017-T020, T026-T030) can run in parallel
- Implementation tasks generally sequential within each story
- All tasks within a story can be committed together

### Parallel Opportunities

**After Foundational phase:**

- US1 and US2 can run in parallel (both use validation helpers)
- US3 can run in parallel with US1/US2 (no shared code)

**Within User Story 3:**

- All pricing extraction tests (T026-T030) can run in parallel

---

## Parallel Example: User Story 3

```bash
# Launch all pricing extraction tests in parallel:
Task: "Add TestDynamoDBPricingExtraction_Storage in internal/pricing/client_test.go"
Task: "Add TestDynamoDBPricingExtraction_ProvisionedRCU in internal/pricing/client_test.go"
Task: "Add TestDynamoDBPricingExtraction_ProvisionedWCU in internal/pricing/client_test.go"
Task: "Add TestDynamoDBPricingExtraction_OnDemandRead in internal/pricing/client_test.go"
Task: "Add TestDynamoDBPricingExtraction_OnDemandWrite in internal/pricing/client_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (review existing code)
2. Complete Phase 2: Foundational (add validation helpers)
3. Complete Phase 3: User Story 1 (pricing warnings)
4. **STOP and VALIDATE**: Test warning emission with fallback embed
5. Can ship fix for #147 alone if urgent

### Incremental Delivery

1. Setup + Foundational → Helpers ready
2. Add User Story 1 → Test pricing warnings → Addresses #147
3. Add User Story 2 → Test validation warnings → Addresses #151
4. Add User Story 3 → Test pricing extraction → Addresses #149
5. Add User Story 4 → Integration tests → Addresses #152
6. Each story adds robustness without breaking previous functionality

### Recommended Approach

Given that this is a hardening bundle, the recommended approach is:

1. Complete all foundational work (T001-T005)
2. Implement US1 and US2 together (both modify estimateDynamoDB())
3. Add US3 tests to verify pricing extraction
4. Add US4 integration tests last
5. Single PR that closes #147, #149, #151, #152

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific GitHub issue
- Tests use `//go:build region_use1` or `//go:build integration` tags
- Validation helpers are foundational - must complete before US1/US2
- US3 pricing tests are independent of implementation changes
- Run `make test` after each phase to catch regressions early
