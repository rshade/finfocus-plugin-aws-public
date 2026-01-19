# Tasks: Windows vs Linux Pricing Differentiation Integration Test

**Feature Name**: Windows vs Linux Pricing Differentiation Integration Test
**Implementation Strategy**: MVP first, implementing the core Windows vs Linux pricing comparison followed by tenancy and architecture differentiation.

## Phase 1: Setup

- [x] T001 Create integration test file `internal/plugin/integration_pricing_test.go` with build tag `//go:build integration`
- [x] T002 Import necessary dependencies (context, testing, pbc, grpc, etc.) in `internal/plugin/integration_pricing_test.go`
- [x] T003 Copy helper functions `startPluginBinary` and `waitForPort` from existing integration tests to `internal/plugin/integration_pricing_test.go`

## Phase 2: Foundational

- [x] T004 Implement test skeleton `TestIntegration_PricingDifferentiation` in `internal/plugin/integration_pricing_test.go`
- [x] T005 [P] Add logic to build `us-east-1` binary with `region_use1` tag in `internal/plugin/integration_pricing_test.go`
- [x] T006 Add logic to start plugin and establish gRPC connection in `internal/plugin/integration_pricing_test.go`

## Phase 3: User Story 1 (P1) - OS Differentiation

- [x] T007 [US1] Implement Linux baseline request for `t2.medium` in `internal/plugin/integration_pricing_test.go`
- [x] T008 [US1] Implement Windows request for `t2.medium` in `internal/plugin/integration_pricing_test.go`
- [x] T009 [US1] Add assertions to verify Windows price > Linux price in `internal/plugin/integration_pricing_test.go`
- [x] T010 [US1] Add assertions to verify billing detail contains "Windows" or "Linux" in `internal/plugin/integration_pricing_test.go`
- [x] T011 [US1] Verify test failure if pricing data is missing (SC-005) in `internal/plugin/integration_pricing_test.go`

## Phase 4: User Story 2 (P2) - Tenancy Differentiation

- [x] T012 [US2] Implement Shared tenancy request for `m5.large` (Linux) in `internal/plugin/integration_pricing_test.go`
- [x] T013 [US2] Implement Dedicated tenancy request for `m5.large` (Linux) in `internal/plugin/integration_pricing_test.go`
- [x] T014 [US2] Add assertions to verify Dedicated price > Shared price in `internal/plugin/integration_pricing_test.go`
- [x] T015 [US2] Add assertions to verify billing detail contains "Dedicated" in `internal/plugin/integration_pricing_test.go`

## Phase 5: User Story 3 (P3) - Architecture & Advanced Platforms

- [x] T016 [US3] Add CPU architecture differentiation test (x86_64 vs arm64) in `internal/plugin/integration_pricing_test.go`
- [x] T019 [P] [US1] Add RHEL and SUSE platform pricing tests to verify FR-003 in `internal/plugin/integration_pricing_test.go`

## Phase 6: Polish & Cross-cutting

- [x] T017 [P] Add ratio check (1.3x to 2.0x) for Windows/Linux pricing (SC-002) in `internal/plugin/integration_pricing_test.go`
- [x] T018 Run full integration test suite and verify all success criteria in `internal/plugin/integration_pricing_test.go`

## Dependencies

1. **Foundational (T004-T006)** must be completed before any User Story tasks.
2. **US1 (T007-T011)** should be completed before **US2 (T012-T015)** to establish the baseline.
3. **Polish (T017-T018)** can be completed after all User Stories.

## Parallel Execution Examples

- **Setup & Foundational**: T001, T002, T003 can be done sequentially; T005 can be prepared while T004 is being structured.
- **Story Implementation**: Tasks within US1 (T007-T011) are mostly sequential but can be developed as a single block.
- **Polish**: T017 and T019 are independent and can be developed in parallel if using separate test cases or subtests.

## Implementation Strategy

- **MVP**: Complete US1 (OS Differentiation) first. This delivers the highest value by validating the primary requirement.
- **Incremental**: Add Tenancy and Architecture differentiation once the baseline OS comparison is stable.
- **Verification**: Each user story should be validated by running the specific test case: `go test -tags=integration -run TestIntegration_PricingDifferentiation`.

## Implementation Notes

### Instance Type Selection for Windows vs Linux Test

AWS pricing varies significantly by instance type. The original spec suggested `t3.medium`, but investigation revealed that many instance types (r5, m5.large, c5, c6i.large) have **identical** Linux/Windows Shared tenancy prices. The test was updated to use `t2.medium` which consistently shows the Windows license premium (~1.39x ratio).

Instance types with confirmed Windows > Linux pricing differentiation:

- `t2.small`: 1.39x ratio
- `t2.medium`: 1.39x ratio
- `t3.micro`: 1.88x ratio
- `t3.small`: 1.88x ratio
- `m5.2xlarge`: 1.96x ratio
- `m6i.large`: 1.96x ratio (larger sizes)
