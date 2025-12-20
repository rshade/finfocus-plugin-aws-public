---
description: "Task list for DynamoDB cost estimation implementation"
---

# Tasks: DynamoDB Cost Estimation

**Input**: Design documents from `/specs/016-dynamodb-cost/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/resource_descriptor.md

**Tests**: Tests are explicitly requested in the Testing Strategy of the feature specification.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Update the pricing generation tool to fetch DynamoDB data.

- [x] T001 Update `tools/generate-pricing/main.go` to fetch `AmazonDynamoDB` pricing from AWS API
- [x] T002 Verify `tools/generate-pricing/main.go` correctly filters for `Amazon DynamoDB PayPerRequest Throughput`, `Provisioned IOPS`, and `Database Storage`
- [x] T003 [P] Update `scripts/build-region.sh` if necessary to ensure new pricing data is included in regional builds

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T004 Add `dynamoDBPrice` struct to `internal/pricing/types.go`
- [x] T005 [P] Add DynamoDB methods to `PricingClient` interface in `internal/pricing/client.go`
- [x] T006 Add `dynamoDBPricing *dynamoDBPrice` field to `Client` struct in `internal/pricing/client.go`
- [x] T007 Implement DynamoDB pricing extraction logic in `newClientFromJSON` within `internal/pricing/client.go`
- [x] T008 [P] Implement `PricingClient` interface methods for DynamoDB in `internal/pricing/client.go`
- [x] T009 Update mock pricing client in `internal/plugin/plugin_test.go` to support new DynamoDB methods

**Checkpoint**: Foundation ready - pricing data is now accessible via the client.

---

## Phase 3: User Story 1 - On-Demand Cost Estimation (Priority: P1) 🎯 MVP

**Goal**: Accurately calculate projected monthly cost for DynamoDB tables in On-Demand mode.

**Independent Test**: Use `go test ./internal/plugin -v -run TestProjectedCost_DynamoDB_OnDemand` with a resource descriptor specifying `sku: "on-demand"`.

### Tests for User Story 1

- [x] T010 [P] [US1] Create unit tests for `estimateDynamoDB` with On-Demand scenarios in `internal/plugin/projected_test.go`
- [x] T011 [P] [US1] Create unit tests for On-Demand pricing lookup in `internal/pricing/client_test.go`

### Implementation for User Story 1

- [x] T012 [US1] Implement `estimateDynamoDB` function stub in `internal/plugin/projected.go`
- [x] T013 [US1] Update `GetProjectedCost` router in `internal/plugin/projected.go` to call `p.estimateDynamoDB` for `dynamodb` resources
- [x] T014 [US1] Update `Supports` logic in `internal/plugin/supports.go` to return `supported: true` for `dynamodb`
- [x] T015 [US1] Implement On-Demand cost calculation logic in `p.estimateDynamoDB` within `internal/plugin/projected.go`
- [x] T016 [US1] Add `BillingDetail` generation for On-Demand mode in `internal/plugin/projected.go`
- [x] T017 [US1] Add debug logging for On-Demand lookups in `internal/plugin/projected.go`

**Checkpoint**: On-Demand estimation is fully functional and testable independently.

---

## Phase 4: User Story 2 - Provisioned Capacity Cost Estimation (Priority: P2)

**Goal**: Accurately calculate projected monthly cost for DynamoDB tables in Provisioned mode.

**Independent Test**: Use `go test ./internal/plugin -v -run TestProjectedCost_DynamoDB_Provisioned` with a resource descriptor specifying `sku: "provisioned"`.

### Tests for User Story 2

- [x] T018 [P] [US2] Create unit tests for `estimateDynamoDB` with Provisioned scenarios in `internal/plugin/projected_test.go`
- [x] T019 [P] [US2] Create unit tests for Provisioned pricing lookup in `internal/pricing/client_test.go`

### Implementation for User Story 2

- [x] T020 [US2] Implement Provisioned cost calculation logic in `p.estimateDynamoDB` within `internal/plugin/projected.go`
- [x] T021 [US2] Add `BillingDetail` generation for Provisioned mode in `internal/plugin/projected.go`
- [x] T022 [US2] Add debug logging for Provisioned lookups in `internal/plugin/projected.go`

**Checkpoint**: Provisioned estimation is fully functional and testable independently.

---

## Phase 5: User Story 3 - Handling Missing Data (Priority: P3)

**Goal**: Gracefully handle missing usage tags and provide clear explanations in the billing detail.

**Independent Test**: Use `go test ./internal/plugin -v -run TestProjectedCost_DynamoDB_MissingTags` with a resource descriptor containing no usage tags.

### Tests for User Story 3

- [x] T023 [P] [US3] Create unit tests for missing/invalid usage tags in `internal/plugin/projected_test.go`

### Implementation for User Story 3

- [x] T024 [US3] Implement default values (0) for missing tags in `p.estimateDynamoDB` within `internal/plugin/projected.go`
- [x] T025 [US3] Implement default SKU ("on-demand") logic in `p.estimateDynamoDB` within `internal/plugin/projected.go`
- [x] T026 [US3] Update `BillingDetail` to list missing/zero inputs when applicable in `internal/plugin/projected.go`

**Checkpoint**: The system handles incomplete inputs gracefully with clear user feedback.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, cleanup, and documentation.

- [x] T027 [P] Run `make generate-pricing` to ensure all 12 regional data files are updated
- [x] T028 Run `make build-all-regions` to verify compilation for all regions
- [x] T029 [P] Run all tests with `make test`
- [x] T030 [P] Final code cleanup and refactoring in `internal/plugin/projected.go`
- [x] T031 Update `CHANGELOG.md` with DynamoDB support
- [x] T032 Verify that regional binaries are < 10MB per Constitution IV (Note: Dependency floor is ~13MB; JSON reduced to 5.5MB)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Can start immediately.
- **Foundational (Phase 2)**: Depends on Phase 1 completion (pricing tool must be ready to generate data for testing).
- **User Story 1 (Phase 3)**: Depends on Foundational completion.
- **User Story 2 & 3 (Phases 4 & 5)**: Depend on Phase 3 (since they build upon the `estimateDynamoDB` structure).
- **Polish (Final Phase)**: Depends on all user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: Foundation.
- **User Story 2 (P2)**: Extends US1 to support a different SKU.
- **User Story 3 (P3)**: Adds robustness to both US1 and US2.

### Parallel Opportunities

- T003, T005, T008, T009 can run in parallel within their phases.
- Testing tasks (T010, T011, T018, T019, T023) can run in parallel before implementation.
- Polish tasks (T027, T029) can run in parallel.

---

## Parallel Example: User Story 1

```bash
# Prepare all tests for User Story 1:
go test ./internal/plugin -v -run TestProjectedCost_DynamoDB_OnDemand  # Should fail initially
go test ./internal/pricing -v -run TestPricingLookup_DynamoDB_OnDemand # Should fail initially
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 & 2 to get the data into the system.
2. Complete Phase 3 (US1) to support basic On-Demand estimation.
3. Validate with a simple resource descriptor for `us-east-1`.

### Incremental Delivery

1. Foundation ready.
2. On-Demand support added (MVP).
3. Provisioned support added.
4. Robustness (missing data) added.
5. Full regional verification.
