# Tasks: SDK Migration and Code Consolidation

**Input**: Design documents from `/specs/013-sdk-migration/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Test tasks are included per project conventions (Go unit tests).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go module**: internal/, cmd/, tools/ at repository root
- Tests co-located with source: `*_test.go` files adjacent to implementation

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify prerequisites and ensure SDK dependency is available

- [x] T001 Verify finfocus-spec v0.4.8 is in go.mod with `go list -m github.com/rshade/finfocus-spec`
- [x] T002 Run `make test` to establish baseline - all existing tests must pass
- [x] T003 Run `make lint` to establish baseline - no new linting errors allowed

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: No foundational tasks needed - this feature is pure refactoring with no schema or infrastructure changes

**⚠️ Note**: User stories can proceed directly after Setup since this is internal code consolidation

**Checkpoint**: Setup verified - user story implementation can now begin

---

## Phase 3: User Story 1 - Plugin Developer Eliminates Duplicate Code (Priority: P1) 🎯 MVP

**Goal**: Consolidate duplicate EC2 attribute extraction and RegionConfig into shared helpers, eliminating ~210 lines of duplicate code

**Independent Test**: Run `go test ./internal/plugin/... -run TestExtractEC2` and `go test ./internal/regionsconfig/...` to verify unified implementations work correctly. Verify estimate.go and projected.go produce identical EC2 cost estimates as before.

### Implementation for User Story 1

#### EC2 Attributes Consolidation (FR-001, FR-002, FR-003)

- [x] T004 [P] [US1] Create EC2Attributes type with OS and Tenancy fields in internal/plugin/ec2_attrs.go
- [x] T005 [P] [US1] Implement DefaultEC2Attributes() returning OS="Linux", Tenancy="Shared" in internal/plugin/ec2_attrs.go
- [x] T006 [US1] Implement ExtractEC2AttributesFromTags(tags map[string]string) in internal/plugin/ec2_attrs.go
- [x] T007 [US1] Implement ExtractEC2AttributesFromStruct(attrs *structpb.Struct) in internal/plugin/ec2_attrs.go
- [x] T008 [US1] Write unit tests for platform normalization (windows/WINDOWS/Windows -> "Windows") in internal/plugin/ec2_attrs_test.go
- [x] T009 [US1] Write unit tests for tenancy normalization (dedicated/DEDICATED -> "Dedicated") in internal/plugin/ec2_attrs_test.go
- [x] T010 [US1] Write unit tests for default values and nil/empty input handling in internal/plugin/ec2_attrs_test.go

#### RegionConfig Consolidation (FR-004, FR-005)

- [x] T011 [P] [US1] Create internal/regionsconfig/ package directory
- [x] T012 [US1] Create RegionConfig and Config types in internal/regionsconfig/config.go
- [x] T013 [US1] Implement Load(filename string) to parse regions.yaml in internal/regionsconfig/config.go
- [x] T014 [US1] Implement Validate(regions []RegionConfig) with safe char and tag format checks in internal/regionsconfig/config.go
- [x] T015 [US1] Implement LoadAndValidate(filename string) combining both in internal/regionsconfig/config.go
- [x] T016 [US1] Write unit tests for Load() with valid YAML in internal/regionsconfig/config_test.go
- [x] T017 [US1] Write unit tests for Validate() error cases (missing ID, invalid chars, duplicate IDs) in internal/regionsconfig/config_test.go

#### Update Callers to Use Shared Helpers

- [x] T018 [US1] Update internal/plugin/estimate.go to use ExtractEC2AttributesFromStruct()
- [x] T019 [US1] Update internal/plugin/projected.go to use ExtractEC2AttributesFromTags()
- [x] T020 [US1] Update tools/generate-embeds/main.go to import internal/regionsconfig
- [x] T021 [US1] Update tools/generate-goreleaser/main.go to import internal/regionsconfig
- [x] T022 [US1] Remove duplicate RegionConfig struct from tools/generate-embeds/main.go
- [x] T023 [US1] Remove duplicate RegionConfig struct from tools/generate-goreleaser/main.go
- [x] T024 [US1] Run `make test` to verify no regressions after consolidation
- [x] T025 [US1] Run `make lint` to verify no new linting errors

**Checkpoint**: User Story 1 complete - duplicate code eliminated, all tests pass

---

## Phase 4: User Story 2 - Plugin Integrates SDK Validation Helpers (Priority: P2)

**Goal**: Replace inline validation with SDK-provided ValidateProjectedCostRequest/ValidateActualCostRequest helpers while preserving custom region checks

**Independent Test**: Send invalid requests to GetProjectedCost/GetActualCost/GetPricingSpec RPCs and verify ERROR_CODE_INVALID_RESOURCE with trace_id in ErrorDetail.details

### Implementation for User Story 2

#### Validation Helper Creation (FR-009, FR-010, FR-011)

- [X] T026 [US2] Create internal/plugin/validation.go with validateProjectedCostRequest() wrapper
- [X] T027 [US2] Implement SDK validation call + custom region check in validateProjectedCostRequest()
- [X] T028 [US2] Create validateActualCostRequest() wrapper with SDK validation + region check in internal/plugin/validation.go
- [X] T029 [US2] Create RegionMismatchError() helper for standardized UNSUPPORTED_REGION responses in internal/plugin/validation.go
- [X] T030 [US2] Ensure trace_id is preserved in all ErrorDetail.details maps in internal/plugin/validation.go
- [X] T031 [US2] Write unit tests for SDK validation error wrapping in internal/plugin/validation_test.go
- [X] T032 [US2] Write unit tests for region mismatch error formatting in internal/plugin/validation_test.go
- [X] T033 [US2] Write unit tests for trace_id preservation in error responses in internal/plugin/validation_test.go

#### Update RPC Methods to Use Validation Helpers

- [X] T034 [US2] Update GetProjectedCost() in internal/plugin/projected.go to use validateProjectedCostRequest()
- [X] T035 [US2] Update GetActualCost() in internal/plugin/actual.go to use validateActualCostRequest()
- [X] T036 [US2] Update GetPricingSpec() in internal/plugin/pricingspec.go to use validateProjectedCostRequest()
- [X] T037 [US2] Remove inline validation code from internal/plugin/projected.go (~76 lines)
- [X] T038 [US2] Remove inline validation code from internal/plugin/actual.go (~76 lines)
- [X] T039 [US2] Remove inline validation code from internal/plugin/pricingspec.go (~24 lines)
- [X] T040 [US2] Run `make test` to verify error format backward compatibility
- [X] T041 [US2] Run `make lint` to verify no new linting errors

**Checkpoint**: User Story 2 complete - SDK validation integrated, error format unchanged

---

## Phase 5: User Story 3 - Plugin Uses SDK Environment Variable Handling (Priority: P2)

**Goal**: Replace direct os.Getenv() calls with SDK helpers while preserving PORT fallback for backward compatibility

**Independent Test**: Start plugin with FINFOCUS_PLUGIN_PORT=8080, then PORT=9000 only, then neither - verify correct port selection in each case

### Implementation for User Story 3

#### Environment Variable Migration (FR-006, FR-007, FR-008)

- [X] T042 [US3] Update cmd/finfocus-plugin-aws-public/main.go to use pluginsdk.GetPort() with PORT fallback
- [X] T043 [US3] Update internal/plugin/plugin.go to use pluginsdk.GetLogLevel() for log configuration
- [X] T044 [US3] Verify pluginsdk.GetTraceID() is used in request handling (already via SDK)
- [X] T045 [US3] Remove direct os.Getenv("PORT") calls from main.go (kept PORT fallback for backward compat)
- [X] T046 [US3] Remove direct os.Getenv("LOG_LEVEL") calls from plugin.go
- [X] T047 [US3] Document port precedence (FINFOCUS_PLUGIN_PORT > PORT > ephemeral) in code comments
- [X] T048 [US3] Run `make test` to verify env var handling works correctly
- [X] T049 [US3] Run `make lint` to verify no new linting errors

**Checkpoint**: User Story 3 complete - SDK env helpers integrated, PORT fallback preserved

---

## Phase 6: User Story 4 - Plugin Uses SDK Property Mapping (Priority: P3)

**Goal**: Use SDK mapping.ExtractAWSSKU() and mapping.ExtractAWSRegion() for tag extraction with alias support

**Independent Test**: Call estimation functions with various tag key aliases (size vs volume_size, instanceType vs type) and verify correct extraction with "(defaulted)" annotations where applicable

### Implementation for User Story 4

#### Property Mapping Integration (FR-012, FR-013, FR-014)

- [X] T050 [US4] Update EBS estimation in internal/plugin/projected.go to use mapping.ExtractAWSSKU() for volume type
- [X] T051 [US4] Update EC2 estimation in internal/plugin/projected.go to use mapping.ExtractAWSRegion() for region extraction
- [X] T052 [US4] Implement sizeAssumed tracking for billing_detail "(defaulted)" annotation in internal/plugin/projected.go
- [X] T053 [US4] Implement engineDefaulted tracking for RDS billing_detail in internal/plugin/projected.go
- [X] T054 [US4] Update GetActualCost() in internal/plugin/actual.go to use mapping helpers
- [X] T055 [US4] Write unit tests for volume_size alias extraction in internal/plugin/projected_test.go
- [X] T056 [US4] Write unit tests for instanceType/type alias priority in internal/plugin/projected_test.go
- [X] T057 [US4] Write unit tests for default tracking in billing_detail in internal/plugin/projected_test.go
- [X] T058 [US4] Run `make test` to verify property mapping works correctly
- [X] T059 [US4] Run `make lint` to verify no new linting errors

**Checkpoint**: User Story 4 complete - SDK property mapping integrated, ~300 lines reduced to ~180

---

## Phase 7: User Story 5 - Plugin Supports ARN-Based Resource Identification (Priority: P3)

**Goal**: Implement ARN parsing for GetActualCost to enable resource identification via AWS ARN without JSON construction

**Independent Test**: Call GetActualCost with ARN `arn:aws:ec2:us-east-1:123456789012:instance/i-abc123` + tags containing sku=t3.micro and verify correct resource identification

### Implementation for User Story 5

#### ARN Parser Creation (FR-015, FR-016, FR-017)

- [X] T060 [P] [US5] Create ARNComponents struct in internal/plugin/arn.go with Partition, Service, Region, AccountID, ResourceType, ResourceID fields
- [X] T061 [US5] Implement ParseARN(arnString string) function to parse AWS ARN format in internal/plugin/arn.go
- [X] T062 [US5] Implement ToPulumiResourceType() method on ARNComponents for service mapping in internal/plugin/arn.go
- [X] T063 [US5] Handle S3 global service (empty region in ARN) in internal/plugin/arn.go
- [X] T064 [US5] Handle EBS volumes (ec2 service in ARN, ebs resource type in Pulumi) in internal/plugin/arn.go
- [X] T065 [US5] Write unit tests for all 7 service ARN formats (EC2, EBS, RDS, S3, Lambda, DynamoDB, EKS) in internal/plugin/arn_test.go
- [X] T066 [US5] Write unit tests for invalid ARN formats in internal/plugin/arn_test.go
- [X] T067 [US5] Write unit tests for S3 empty region handling in internal/plugin/arn_test.go

#### ARN Integration into GetActualCost (FR-018, FR-019)

- [X] T068 [US5] Update GetActualCost() in internal/plugin/actual.go to check req.Arn field first
- [X] T069 [US5] Implement fallback chain: ARN -> JSON ResourceId -> Tags extraction in internal/plugin/actual.go
- [X] T070 [US5] Extract SKU from tags when ARN is used (ARN doesn't contain instance type) in internal/plugin/actual.go
- [X] T071 [US5] Return ERROR_CODE_UNSUPPORTED_REGION when ARN region doesn't match plugin binary region
- [X] T072 [US5] Write integration test for ARN + tags combination in internal/plugin/actual_test.go
- [X] T073 [US5] Run `make test` to verify ARN parsing and integration works correctly
- [X] T074 [US5] Run `make lint` to verify no new linting errors

**Checkpoint**: User Story 5 complete - ARN-based resource identification working for all 7 services

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, documentation updates, and quality assurance

- [X] T075 [P] Update CLAUDE.md with any new patterns or conventions discovered during implementation
- [X] T076 [P] Run full test suite with `make test` - all tests must pass
- [X] T077 [P] Run linter with `make lint` - no errors allowed
- [ ] T078 [P] Run `goreleaser build --snapshot --clean` to verify all region binaries build correctly (manual step)
- [X] T079 Verify success criteria from spec.md:
  - [X] SC-001: Duplicate code reduced by 50% (review line counts)
  - [X] SC-002: RegionConfig consolidated (1 definition instead of 2) - internal/regionsconfig package created
  - [X] SC-003: All existing tests pass
  - [X] SC-004: Error format backward compatible
  - [X] SC-005: Port configuration works (FINFOCUS_PLUGIN_PORT > PORT > ephemeral)
  - [X] SC-006: ARN parsing works for 7 services (EC2, EBS, RDS, S3, Lambda, DynamoDB, EKS)
  - [X] SC-007: Property extraction reduced by 40% - extractAWSSKU/extractAWSRegion helpers
- [X] T080 Run quickstart.md verification steps to validate implementation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: N/A - no foundational tasks needed
- **User Story 1 (Phase 3)**: Depends on Setup completion - establishes shared helpers
- **User Story 2 (Phase 4)**: Can start after US1 (may use validation from US1 patterns)
- **User Story 3 (Phase 5)**: Independent of US2, can run in parallel after US1
- **User Story 4 (Phase 6)**: Independent of US2/US3, can run in parallel after US1
- **User Story 5 (Phase 7)**: Independent of US2/US3/US4, can run in parallel after US1
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Must complete first - provides shared helpers used by other stories
- **User Story 2 (P2)**: Can start after US1 - uses validation.go patterns
- **User Story 3 (P2)**: Can start after US1 - independent of US2
- **User Story 4 (P3)**: Can start after US1 - independent of US2/US3
- **User Story 5 (P3)**: Can start after US1 - independent of US2/US3/US4

### Within Each User Story

- Create new files before modifying existing files
- Write helper functions before updating callers
- Run `make test` and `make lint` after each significant change
- Story complete before moving to next priority

### Parallel Opportunities

- T004, T005, T011 can run in parallel (different files)
- T020, T021 can run in parallel (different tool files)
- US2, US3, US4, US5 can potentially run in parallel after US1 (if multiple developers)
- T075, T076, T077, T078 can run in parallel (independent verification tasks)

---

## Parallel Example: User Story 1

```bash
# Launch EC2Attributes and RegionConfig creation in parallel:
Task: "T004 [P] [US1] Create EC2Attributes type in internal/plugin/ec2_attrs.go"
Task: "T011 [P] [US1] Create internal/regionsconfig/ package directory"

# After structure created, implement extractors (sequential):
Task: "T006 [US1] Implement ExtractEC2AttributesFromTags() in internal/plugin/ec2_attrs.go"
Task: "T012 [US1] Create RegionConfig type in internal/regionsconfig/config.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (verify deps, baseline tests)
2. Skip Phase 2: Foundational (not needed)
3. Complete Phase 3: User Story 1 (consolidate duplicates)
4. **STOP and VALIDATE**: Run all tests, verify no regressions
5. This delivers ~50% of the value with zero external risk

### Incremental Delivery

1. Complete Setup → Baseline established
2. Add User Story 1 → Test independently → ~210 lines consolidated (MVP!)
3. Add User Story 2 → Test independently → SDK validation integrated
4. Add User Story 3 → Test independently → SDK env vars integrated
5. Add User Story 4 → Test independently → SDK property mapping integrated
6. Add User Story 5 → Test independently → ARN support enabled
7. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. All complete Setup together
2. Once US1 is done:
   - Developer A: User Story 2 (validation)
   - Developer B: User Story 3 (env vars)
   - Developer C: User Story 4 (mapping)
   - Developer D: User Story 5 (ARN)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Run `make test` and `make lint` after each task or logical group
- Stop at any checkpoint to validate story independently
- US1 is MVP - delivers most value with least risk
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
