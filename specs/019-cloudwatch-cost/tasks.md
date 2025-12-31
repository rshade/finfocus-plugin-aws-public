# Tasks: CloudWatch Cost Estimation

**Input**: Design documents from `/specs/019-cloudwatch-cost/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Unit tests included per constitution requirement (Testing Discipline).

**Organization**: Tasks are grouped by user story to enable independent implementation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify project structure and add CloudWatch to pricing generator

- [x] T001 Verify Go 1.25+ and dependencies are installed
- [x] T002 Add "AmazonCloudWatch" to serviceConfig in tools/generate-pricing/main.go
- [x] T003 Generate CloudWatch pricing data for us-east-1 via `go run ./tools/generate-pricing --regions us-east-1 --out-dir ./internal/pricing/data`
- [x] T004 Verify cloudwatch_us-east-1.json was created in internal/pricing/data/
- [x] T004a Measure baseline binary size before CloudWatch implementation (for SC-003 validation)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Normalize pricing error messages across ALL services before CloudWatch implementation

**⚠️ CRITICAL**: This refactor ensures consistent soft failure behavior across the codebase

### Create Constants

- [x] T005 Create internal/plugin/constants.go with PricingNotFoundTemplate and PricingUnavailableTemplate constants
- [x] T006 [P] Create internal/plugin/constants_test.go with tests verifying template format

### Update Existing Services to Use Constants

- [x] T007 [P] Update EC2 in internal/plugin/projected.go to use PricingNotFoundTemplate (line ~251)
- [x] T008 [P] Update EBS in internal/plugin/projected.go to use PricingNotFoundTemplate (line ~354)
- [x] T009 [P] Update S3 in internal/plugin/projected.go to use PricingNotFoundTemplate (line ~421)
- [x] T010 [P] Update RDS in internal/plugin/projected.go to use PricingNotFoundTemplate (line ~754)
- [x] T011 [P] Update EKS in internal/plugin/projected.go to use PricingUnavailableTemplate (line ~891)
- [x] T012 [P] Update Lambda in internal/plugin/projected.go to use PricingUnavailableTemplate (line ~985)
- [x] T013 [P] Update NAT Gateway in internal/plugin/projected.go to use PricingUnavailableTemplate (line ~1071)
- [x] T014 [P] Update ELB in internal/plugin/projected.go to add explicit message using PricingUnavailableTemplate (line ~636)
- [x] T015 [P] Update DynamoDB in internal/plugin/projected.go to add explicit message when pricing lookup fails

### Verification

- [x] T016 Run `make test` to verify all existing tests still pass after refactor
- [x] T017 Run `make lint` to verify code style compliance

**Checkpoint**: Foundation ready - all services now use standardized error messages

---

## Phase 3: User Story 1 - Log Costs (Priority: P1) 🎯 MVP

**Goal**: Estimate CloudWatch Logs costs (ingestion + storage)

**Independent Test**: Provide ResourceDescriptor with `sku: "logs"`, `log_ingestion_gb: 100`, `log_storage_gb: 500` and verify cost matches manual calculation

### Pricing Infrastructure for Logs

- [x] T018 [P] [US1] Add cloudWatchPrice type to internal/pricing/types.go with LogsIngestionRate, LogsStorageRate fields
- [x] T019 [P] [US1] Add tierRate type to internal/pricing/types.go for tiered pricing support
- [x] T020 [US1] Add rawCloudWatchJSON variable to internal/pricing/embed_use1.go with //go:embed directive
- [x] T021 [US1] Add parseCloudWatchPricing() method to internal/pricing/client.go
- [x] T022 [US1] Add CloudWatchLogsIngestionPrice() method to internal/pricing/client.go
- [x] T023 [US1] Add CloudWatchLogsStoragePrice() method to internal/pricing/client.go
- [x] T024 [US1] Add CloudWatch parsing to init() goroutine pool in internal/pricing/client.go

### Plugin Implementation for Logs

- [x] T025 [US1] Add "cloudwatch" and "logs" cases to detectService() in internal/plugin/projected.go
- [x] T026 [US1] Add "cloudwatch" case to Supports() switch in internal/plugin/supports.go
- [x] T027 [US1] Add estimateCloudWatch() function stub in internal/plugin/projected.go
- [x] T027a [US1] Add calculateTieredCost() helper function in internal/plugin/projected.go (needed for log ingestion tiers)
- [x] T028 [US1] Implement log ingestion cost calculation with tiered pricing in estimateCloudWatch()
- [x] T029 [US1] Implement log storage cost calculation (flat rate) in estimateCloudWatch()
- [x] T030 [US1] Add cloudwatch case to GetProjectedCost switch in internal/plugin/projected.go

### Tests for User Story 1

- [x] T031 [P] [US1] Add TestCloudWatchLogsIngestion unit test in internal/plugin/projected_test.go
- [x] T032 [P] [US1] Add TestCloudWatchLogsStorage unit test in internal/plugin/projected_test.go
- [x] T033 [P] [US1] Add TestCloudWatchLogsCombined unit test in internal/plugin/projected_test.go
- [x] T034 [US1] Add TestSupportsCloudWatch unit test in internal/plugin/supports_test.go

**Checkpoint**: User Story 1 complete - Log cost estimation works independently

---

## Phase 4: User Story 2 - Custom Metrics (Priority: P2)

**Goal**: Estimate CloudWatch Custom Metrics costs with tiered pricing

**Independent Test**: Provide ResourceDescriptor with `sku: "metrics"`, `custom_metrics: 50` and verify cost matches $0.30 * 50 = $15.00

### Pricing Infrastructure for Metrics

- [x] T035 [US2] Add MetricsTiers field to cloudWatchPrice type in internal/pricing/types.go
- [x] T036 [US2] Add CloudWatchMetricsPrice() method to internal/pricing/client.go
- [x] T037 [US2] Extend parseCloudWatchPricing() to extract metrics pricing tiers

### Plugin Implementation for Metrics

- [x] T038 [US2] Add "metrics" case to detectService() in internal/plugin/projected.go
- [x] T039 [US2] Verify calculateTieredCost() helper works for metrics tiers (reuses T027a from US1)
- [x] T040 [US2] Implement metrics cost calculation in estimateCloudWatch() using tiered pricing
- [x] T041 [US2] Handle sku: "metrics" to return metrics-only cost

### Tests for User Story 2

- [x] T042 [P] [US2] Add TestCloudWatchMetricsFirstTier unit test (50 metrics @ $0.30)
- [x] T043 [P] [US2] Add TestCloudWatchMetricsSecondTier unit test (15,000 metrics with tier transition)
- [x] T044 [P] [US2] Add TestCloudWatchMetricsHighVolume unit test (100,000+ metrics)
- [x] T045 [US2] Add TestCalculateTieredCost unit test for the helper function

**Checkpoint**: User Story 2 complete - Metrics cost estimation works independently

---

## Phase 5: User Story 3 - Combined Usage (Priority: P3)

**Goal**: Estimate total CloudWatch cost combining logs and metrics

**Independent Test**: Provide ResourceDescriptor with all tags and verify total equals sum of components

### Plugin Implementation for Combined

- [x] T046 [US3] Handle sku: "combined" or empty to return sum of logs + metrics costs
- [x] T047 [US3] Update BillingDetail format for combined estimation
- [x] T048 [US3] Handle missing tags gracefully (default to 0, not error)

### Tests for User Story 3

- [x] T049 [P] [US3] Add TestCloudWatchCombined unit test with all tags
- [x] T050 [P] [US3] Add TestCloudWatchMissingTags unit test (verify $0 not error)
- [x] T051 [P] [US3] Add TestCloudWatchInvalidTags unit test (non-numeric values)

**Checkpoint**: User Story 3 complete - Combined estimation works

---

## Phase 6: Regional Support & Embedding

**Purpose**: Extend CloudWatch support to all 12 regions

### Embed Directives (all parallelizable)

- [x] T052 [P] Add CloudWatch embed to internal/pricing/embed_usw2.go (us-west-2)
- [x] T053 [P] Add CloudWatch embed to internal/pricing/embed_euw1.go (eu-west-1)
- [x] T054 [P] Add CloudWatch embed to internal/pricing/embed_apse1.go (ap-southeast-1)
- [x] T055 [P] Add CloudWatch embed to internal/pricing/embed_apse2.go (ap-southeast-2)
- [x] T056 [P] Add CloudWatch embed to internal/pricing/embed_apne1.go (ap-northeast-1)
- [x] T057 [P] Add CloudWatch embed to internal/pricing/embed_aps1.go (ap-south-1)
- [x] T058 [P] Add CloudWatch embed to internal/pricing/embed_cac1.go (ca-central-1)
- [x] T059 [P] Add CloudWatch embed to internal/pricing/embed_sae1.go (sa-east-1)
- [x] T060 [P] Add CloudWatch embed to internal/pricing/embed_usw1.go (us-west-1)
- [x] T061 [P] Add CloudWatch embed to internal/pricing/embed_govw1.go (us-gov-west-1)
- [x] T062 [P] Add CloudWatch embed to internal/pricing/embed_gove1.go (us-gov-east-1)
- [x] T063 [P] Add CloudWatch embed to internal/pricing/embed_fallback.go (test fallback)

### Regional Pricing Generation

- [x] T064 Generate CloudWatch pricing for all regions via generate-pricing tool
- [x] T065 Verify all cloudwatch_*.json files created (12 regions)

**Checkpoint**: All regions have CloudWatch pricing embedded

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, validation, and cleanup

### Documentation

- [x] T066 Update CLAUDE.md with soft failure pattern documentation
- [x] T067 Update CLAUDE.md "Cost Estimation Scope" table to include CloudWatch
- [x] T068 Update CLAUDE.md "Service Support" section to mark CloudWatch as implemented
- [x] T069 [P] Lint all markdown files with `npx markdownlint-cli specs/019-cloudwatch-cost/*.md`

### Final Validation

- [x] T070 Run `make lint` - verify zero errors
- [x] T071 Run `make test` - verify all tests pass
- [x] T072 Build us-east-1 binary with `make build-region REGION=us-east-1`
- [x] T072a Measure binary size and compare to baseline (SC-003: must be < 5MB increase) - CloudWatch adds ~109KB
- [ ] T073 Manual test: Query CloudWatch logs cost via grpcurl
- [ ] T074 Manual test: Query CloudWatch metrics cost via grpcurl
- [ ] T075 Manual test: Query CloudWatch combined cost via grpcurl

### PR Preparation

- [x] T076 Generate PR_MESSAGE.md with implementation summary
- [x] T077 Validate PR_MESSAGE.md with markdownlint and commitlint

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - US1 (Logs) → US2 (Metrics) → US3 (Combined) in sequence
  - US2 depends on tiered pricing infrastructure from US1
  - US3 depends on both logs and metrics implementation
- **Regional (Phase 6)**: Depends on US1 completion (embed pattern established)
- **Polish (Phase 7)**: Depends on all user stories complete

### User Story Dependencies

- **User Story 1 (P1)**: After Foundational - establishes pricing infrastructure
- **User Story 2 (P2)**: After US1 - reuses pricing types and calculateTieredCost
- **User Story 3 (P3)**: After US2 - combines both calculation paths

### Parallel Opportunities

Within Phase 2 (Foundational):

- T007-T015 can all run in parallel (updating different service estimators)

Within Phase 3 (US1):

- T018-T019 (types) can run in parallel
- T031-T034 (tests) can run in parallel after implementation

Within Phase 4 (US2):

- T042-T045 (tests) can run in parallel after implementation

Within Phase 6 (Regional):

- T052-T063 (embed files) can all run in parallel

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (normalize error messages)
3. Complete Phase 3: User Story 1 (Logs)
4. **STOP and VALIDATE**: Test log estimation independently
5. Can demo/ship logs-only support

### Full Feature Delivery

1. Setup → Foundational → US1 (Logs) → US2 (Metrics) → US3 (Combined)
2. Regional support (Phase 6)
3. Polish and PR preparation (Phase 7)

### Estimated Task Count

| Phase | Task Count | Parallelizable |
|-------|------------|----------------|
| Setup | 5 | 0 |
| Foundational | 13 | 9 |
| US1 (Logs) | 18 | 4 |
| US2 (Metrics) | 11 | 4 |
| US3 (Combined) | 6 | 3 |
| Regional | 14 | 12 |
| Polish | 13 | 1 |
| **Total** | **80** | **33** |

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- Each user story should be independently testable after completion
- Commit after each logical group of tasks
- The foundational refactor (T005-T017) pays down technical debt across ALL services
- CloudWatch is marked as NON-CRITICAL service (like S3, Lambda) - parsing errors don't fail startup
