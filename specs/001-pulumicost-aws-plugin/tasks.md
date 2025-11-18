# Tasks: PulumiCost AWS Public Plugin

**Input**: Design documents from `/specs/001-pulumicost-aws-plugin/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Unit tests are included for critical pricing logic and gRPC handlers per constitution testing requirements.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Repository follows standard Go project layout
- `cmd/pulumicost-plugin-aws-public/` for entrypoint
- `internal/plugin/` for gRPC service implementation
- `internal/pricing/` for embedded pricing data
- `internal/config/` for configuration
- `tools/generate-pricing/` for build-time tooling

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Initialize Go module with `go mod init github.com/rshade/pulumicost-plugin-aws-public`
- [x] T002 Create directory structure: cmd/, internal/plugin, internal/pricing, internal/config, tools/generate-pricing, data/
- [x] T003 [P] Add proto dependencies: `go get github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1`
- [x] T004 [P] Add pluginsdk dependency: `go get github.com/rshade/pulumicost-core/pkg/pluginsdk`
- [x] T005 [P] Add gRPC dependencies: `go get google.golang.org/grpc google.golang.org/protobuf`
- [x] T006 [P] Create Makefile with lint, test, and build targets
- [x] T007 [P] Create .gitignore with data/*.json (generated pricing files)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Pricing Client Foundation

- [x] T008 [P] Define pricingData, ec2OnDemandPrice, ebsVolumePrice types in internal/pricing/types.go
- [x] T009 [P] Define PricingClient interface in internal/pricing/client.go
- [x] T010 Implement Client struct with sync.Once initialization in internal/pricing/client.go
- [x] T011 Implement EC2OnDemandPricePerHour lookup method in internal/pricing/client.go
- [x] T012 Implement EBSPricePerGBMonth lookup method in internal/pricing/client.go
- [x] T013 [P] Create embed_fallback.go with dummy pricing data (no build tags) in internal/pricing/embed_fallback.go
- [x] T014 [P] Write unit tests for pricing client parsing and lookups in internal/pricing/client_test.go (covers edge case: corrupted pricing data)

### Build-Time Pricing Tool

- [x] T015 Implement pricing generation CLI tool with --dummy flag in tools/generate-pricing/main.go
- [x] T016 Generate dummy pricing data for us-east-1, us-west-2, eu-west-1: `go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data --dummy`
- [x] T017 [P] Create embed_use1.go with build tag region_use1 embedding data/aws_pricing_us-east-1.json in internal/pricing/embed_use1.go
- [x] T018 [P] Create embed_usw2.go with build tag region_usw2 embedding data/aws_pricing_us-west-2.json in internal/pricing/embed_usw2.go
- [x] T019 [P] Create embed_euw1.go with build tag region_euw1 embedding data/aws_pricing_eu-west-1.json in internal/pricing/embed_euw1.go

### Plugin Infrastructure

- [ ] T020 Define AWSPublicPlugin struct implementing CostSourceService in internal/plugin/plugin.go
- [ ] T021 Implement NewAWSPublicPlugin constructor in internal/plugin/plugin.go
- [ ] T022 [P] Create mockPricingClient for tests in internal/plugin/plugin_test.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 2 - gRPC Service Lifecycle Management (Priority: P1) 🎯 MVP Foundation

**Goal**: Enable plugin to start, announce PORT, serve gRPC, and shutdown gracefully

**Independent Test**: Start plugin subprocess, capture PORT from stdout, connect gRPC client, verify graceful shutdown

**Why First**: US2 is foundational for US1. Without lifecycle, we can't test cost estimation. Lifecycle is simpler than cost logic.

### Implementation for User Story 2

- [ ] T023 [P] [US2] Implement Name() RPC returning "aws-public" in internal/plugin/name.go
- [ ] T024 [P] [US2] Write unit test for Name() RPC in internal/plugin/name_test.go
- [ ] T025 [US2] Implement main.go entrypoint: initialize pricing client, create plugin, call pluginsdk.Serve() in cmd/pulumicost-plugin-aws-public/main.go
- [ ] T026 [US2] Add stderr logging for initialization and region in cmd/pulumicost-plugin-aws-public/main.go
- [ ] T027 [US2] Test manual plugin startup: `go run ./cmd/pulumicost-plugin-aws-public` verifies PORT announcement
- [ ] T028 [US2] Test Name() RPC with grpcurl: `grpcurl -plaintext localhost:<port> pulumicost.v1.CostSourceService/Name`
- [ ] T029 [US2] Test graceful shutdown with Ctrl+C

**Checkpoint**: Plugin starts, announces PORT, serves gRPC Name() RPC, shuts down gracefully

---

## Phase 4: User Story 3 - Resource Support Detection (Priority: P2)

**Goal**: Enable core to detect plugin capabilities via Supports() RPC

**Independent Test**: Call Supports() with various ResourceDescriptors, validate supported/reason responses

**Why Next**: Supports() is simpler than GetProjectedCost() but completes the gRPC contract for discovery

### Implementation for User Story 3

- [ ] T030 [US3] Implement Supports() RPC with provider/region/resource_type checks in internal/plugin/supports.go
- [ ] T031 [P] [US3] Write table-driven unit tests for Supports() covering EC2, EBS, S3 stubs, wrong region, unknown type in internal/plugin/supports_test.go (covers edge case: unknown resource_type values)
- [ ] T032 [US3] Test Supports() for EC2 in correct region with grpcurl
- [ ] T033 [US3] Test Supports() for wrong region with grpcurl, verify supported=false
- [ ] T034 [US3] Test Supports() for stub service (S3) with grpcurl, verify "Limited support" reason

**Checkpoint**: Supports() RPC works for all resource types and region checks

---

## Phase 5: User Story 1 - Basic Cost Estimation via gRPC (Priority: P1) 🎯 MVP Core

**Goal**: Provide accurate cost estimates for EC2 and EBS resources via GetProjectedCost RPC

**Independent Test**: Call GetProjectedCost with EC2/EBS ResourceDescriptors, validate cost_per_month calculations

**Why Last of P1**: This is the core value but depends on lifecycle (US2) being complete

### Implementation for User Story 1

- [ ] T035 [US1] Implement GetProjectedCost() RPC with validation and routing in internal/plugin/projected.go (covers edge case: unknown instance types return $0 with "not found" billing_detail)
- [ ] T036 [US1] Implement estimateEC2() helper: lookup pricing, calculate 730 hrs/month cost in internal/plugin/projected.go
- [ ] T037 [US1] Implement estimateEBS() helper: extract size from tags, default to 8GB, calculate cost in internal/plugin/projected.go
- [ ] T038 [US1] Add region mismatch error handling with ERROR_CODE_UNSUPPORTED_REGION in internal/plugin/projected.go
- [ ] T038b [US1] Attach ErrorDetail proto message with details map {"pluginRegion": "<region>", "requiredRegion": "<resource.region>"} to UNSUPPORTED_REGION gRPC errors in internal/plugin/projected.go
- [ ] T039 [US1] Add missing required field validation with ERROR_CODE_INVALID_RESOURCE in internal/plugin/projected.go
- [ ] T040 [P] [US1] Write unit test for GetProjectedCost EC2 with t3.micro in internal/plugin/projected_test.go
- [ ] T041 [P] [US1] Write unit test for GetProjectedCost EBS with gp3 100GB in internal/plugin/projected_test.go
- [ ] T042 [P] [US1] Write unit test for GetProjectedCost EBS defaulting to 8GB in internal/plugin/projected_test.go (covers edge case: EBS without size in tags)
- [ ] T043 [P] [US1] Write unit test for region mismatch error in internal/plugin/projected_test.go
- [ ] T044 [P] [US1] Write unit test for missing sku error in internal/plugin/projected_test.go (covers edge case: ResourceDescriptor lacks required fields)
- [ ] T045 [US1] Test GetProjectedCost for EC2 t3.micro with grpcurl, verify cost_per_month ≈ 7.592
- [ ] T046 [US1] Test GetProjectedCost for EBS gp3 100GB with grpcurl, verify cost_per_month = 8.0
- [ ] T047 [US1] Test GetProjectedCost for wrong region with grpcurl, verify FailedPrecondition error
- [ ] T048 [US1] Run all unit tests: `go test ./internal/plugin -v` and verify all pass

**Checkpoint**: MVP COMPLETE - EC2 and EBS cost estimation works end-to-end via gRPC

---

## Phase 6: User Story 5 - Stub Support for Additional AWS Services (Priority: P3)

**Goal**: Acknowledge S3, Lambda, RDS, DynamoDB resources with $0 estimates

**Independent Test**: Call GetProjectedCost for stub services, validate $0 cost with explanatory billing_detail

### Implementation for User Story 5

- [ ] T049 [US5] Implement estimateStub() helper returning $0 with service-specific billing_detail in internal/plugin/projected.go
- [ ] T050 [US5] Add S3, Lambda, RDS, DynamoDB cases to GetProjectedCost routing in internal/plugin/projected.go
- [ ] T051 [P] [US5] Write unit test for S3 stub returning $0 in internal/plugin/projected_test.go
- [ ] T052 [P] [US5] Write unit test for Lambda stub returning $0 in internal/plugin/projected_test.go
- [ ] T053 [US5] Test GetProjectedCost for S3 with grpcurl, verify cost_per_month=0 and billing_detail

**Checkpoint**: All 6 AWS service types (EC2, EBS, S3, Lambda, RDS, DynamoDB) handled

---

## Phase 7: User Story 4 - Region-Specific Cost Estimation (Priority: P2)

**Goal**: Enable region-specific binary builds with embedded pricing data

**Independent Test**: Build region-specific binaries, verify each embeds only its region's data

### Implementation for User Story 4

- [ ] T054 [US4] Create .goreleaser.yaml with 3 builds (us-east-1, us-west-2, eu-west-1) using build tags
- [ ] T055 [US4] Add before hook to .goreleaser.yaml: `go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data --dummy`
- [ ] T056 [US4] Test GoReleaser snapshot build: `goreleaser build --snapshot --clean`
- [ ] T057 [US4] Verify us-east-1 binary: `./dist/pulumicost-plugin-aws-public-us-east-1_linux_amd64_v1/pulumicost-plugin-aws-public-us-east-1` announces PORT and logs region
- [ ] T058 [US4] Verify us-west-2 binary embeds us-west-2 pricing
- [ ] T059 [US4] Test us-east-1 binary with us-west-2 resource, verify ERROR_CODE_UNSUPPORTED_REGION error

**Checkpoint**: Region-specific binaries build and enforce region boundaries

---

## Phase 8: User Story 6 - Transparent Cost Breakdown via Pricing Spec (Priority: P2)

**Goal**: Optionally provide detailed pricing specifications via GetPricingSpec() RPC

**Independent Test**: Call GetPricingSpec for EC2/EBS, validate rate_per_unit and billing_mode

**Note**: This user story is optional and can be deferred to v2 if needed

### Implementation for User Story 6

- [ ] T060 [US6] Implement GetPricingSpec() RPC for EC2 with billing_mode="per_hour" in internal/plugin/pricing_spec.go
- [ ] T061 [US6] Implement GetPricingSpec() RPC for EBS with billing_mode="per_gb_month" in internal/plugin/pricing_spec.go
- [ ] T062 [US6] Implement GetPricingSpec() for stub services returning rate_per_unit=0 in internal/plugin/pricing_spec.go
- [ ] T063 [P] [US6] Write unit test for GetPricingSpec EC2 in internal/plugin/pricing_spec_test.go
- [ ] T064 [P] [US6] Write unit test for GetPricingSpec EBS in internal/plugin/pricing_spec_test.go
- [ ] T065 [US6] Test GetPricingSpec for t3.micro with grpcurl, verify billing_mode and rate_per_unit

**Checkpoint**: GetPricingSpec() provides transparency for all resource types

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T066 [P] Create README.md documenting gRPC protocol, usage, ResourceDescriptor format, integration with core
- [ ] T067 [P] Create RELEASING.md with pre-release checklist, tag process, GoReleaser usage
- [ ] T068 [P] Update CLAUDE.md with implementation notes and gRPC architecture details
- [ ] T069 Run `make lint` and fix any linting issues
- [ ] T070 Run `make test` and verify all tests pass
- [ ] T071 [P] Add performance logging for pricing lookups >50ms to stderr
- [ ] T072 Validate quickstart.md examples work end-to-end
- [ ] T073 Create example testdata/ files with sample ResourceDescriptors for manual testing
- [ ] T074 Document edge cases and error handling in README.md

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 2 (Phase 3 - P1)**: Depends on Foundational (Phase 2) - Required for all other stories
- **User Story 3 (Phase 4 - P2)**: Depends on Foundational + US2 lifecycle - Can start after US2
- **User Story 1 (Phase 5 - P1)**: Depends on Foundational + US2 lifecycle - MVP core value
- **User Story 5 (Phase 6 - P3)**: Depends on US1 GetProjectedCost foundation - Extends US1
- **User Story 4 (Phase 7 - P2)**: Depends on US1+US5 being complete - Region-specific builds
- **User Story 6 (Phase 8 - P2)**: Optional - Can be deferred or done in parallel with US4/US5
- **Polish (Phase 9)**: Depends on desired user stories being complete

### User Story Dependencies

- **User Story 2 (P1 - Lifecycle)**: BLOCKS all other stories - Foundation for gRPC testing
- **User Story 1 (P1 - EC2/EBS)**: Depends on US2 - Core cost estimation logic
- **User Story 3 (P2 - Supports)**: Depends on US2 - Independent capability detection
- **User Story 4 (P2 - Regions)**: Depends on US1 - Region-specific binary distribution
- **User Story 5 (P3 - Stubs)**: Depends on US1 - Extends cost estimation to more services
- **User Story 6 (P2 - PricingSpec)**: Optional - Depends on US1 - Transparency enhancement

### Within Each User Story

- Tests before or parallel with implementation (unit tests can guide development)
- Infrastructure (types, interfaces) before business logic
- Business logic before integration/manual testing
- Story complete and validated before moving to next priority

### Parallel Opportunities

- **Phase 1 (Setup)**: T003-T007 can all run in parallel (different dependencies)
- **Phase 2 (Foundational)**:
  - T008-T009 (types/interface) in parallel
  - T014, T017-T019 (tests and embed files) in parallel after T010-T013
- **Phase 3 (US2)**: T023-T024 in parallel (Name RPC + test)
- **Phase 4 (US3)**: T031 (tests) can start with T030 if using TDD
- **Phase 5 (US1)**: T040-T044 (all unit tests) in parallel after T035-T039
- **Phase 6 (US5)**: T051-T052 (stub tests) in parallel
- **Phase 8 (US6)**: T063-T064 (PricingSpec tests) in parallel
- **Phase 9 (Polish)**: T066-T068, T071, T073 all in parallel (different files)

**After Foundational Phase**: US3 (Supports) and US1+US2 can be worked on by different developers in parallel since they touch different RPC methods

---

## Parallel Example: User Story 1 (Core Cost Estimation)

```bash
# Launch all unit tests for User Story 1 together:
Task T040: "Write unit test for GetProjectedCost EC2 with t3.micro in internal/plugin/projected_test.go"
Task T041: "Write unit test for GetProjectedCost EBS with gp3 100GB in internal/plugin/projected_test.go"
Task T042: "Write unit test for GetProjectedCost EBS defaulting to 8GB in internal/plugin/projected_test.go"
Task T043: "Write unit test for region mismatch error in internal/plugin/projected_test.go"
Task T044: "Write unit test for missing sku error in internal/plugin/projected_test.go"
```

---

## Implementation Strategy

### MVP First (Minimum Viable Product)

**Goal**: Get EC2 and EBS cost estimation working end-to-end

1. Complete Phase 1: Setup (T001-T007)
2. Complete Phase 2: Foundational (T008-T022) - CRITICAL
3. Complete Phase 3: User Story 2 - Lifecycle (T023-T029) - REQUIRED for testing
4. Complete Phase 5: User Story 1 - Core Cost Estimation (T035-T048) - CORE VALUE
5. **STOP and VALIDATE**: Test EC2 and EBS cost estimation end-to-end with grpcurl
6. Optionally add Phase 4: User Story 3 - Supports() for better capability detection
7. Deploy/demo if ready

**MVP Scope**: US2 (lifecycle) + US1 (EC2/EBS cost estimation) = ~36 tasks (T001-T022 setup/foundational + T023-T029 US2 + T035-T048 US1)

### Incremental Delivery

1. **Foundation** (Phases 1-2): Setup + Pricing infrastructure → Can't test yet
2. **Add Lifecycle** (Phase 3 - US2): Plugin starts and serves → Can test Name() RPC
3. **Add Cost Estimation** (Phase 5 - US1): EC2/EBS costs work → MVP COMPLETE! ✅
4. **Add Supports Detection** (Phase 4 - US3): Better capability routing → Enhanced
5. **Add Stub Services** (Phase 6 - US5): S3/Lambda/RDS/DynamoDB acknowledged → More complete
6. **Add Region Builds** (Phase 7 - US4): Multi-region binaries → Production ready
7. **Add Transparency** (Phase 8 - US6 - Optional): GetPricingSpec() → Trust building

Each increment adds value and is independently testable.

### Parallel Team Strategy

With 2-3 developers after Foundational phase:

1. **All**: Complete Setup + Foundational together (Phases 1-2)
2. **Once Foundational done**:
   - **Developer A**: User Story 2 (Lifecycle) - MUST finish first
3. **After US2 complete**:
   - **Developer A**: User Story 1 (Cost Estimation)
   - **Developer B**: User Story 3 (Supports) - Can work in parallel
4. **After US1+US3 complete**:
   - **Developer A**: User Story 5 (Stubs)
   - **Developer B**: User Story 4 (Region builds)
   - **Developer C**: User Story 6 (PricingSpec) - Optional

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing (for TDD approach)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Constitution requires: linting passes, tests pass, no stdout except PORT
- Thread safety critical: pricing client uses sync.Once, all RPC handlers must be stateless
- Proto-defined types only: Never create custom JSON envelopes

### Edge Case Coverage

Tasks explicitly cover the following edge cases from spec.md:

1. ✅ **ResourceDescriptor lacks required fields** → T044 (missing sku error test)
2. ✅ **Unknown resource_type values** → T031 (Supports test for unknown type)
3. ✅ **Corrupted pricing data** → T014 (pricing client parsing tests)
4. ✅ **EBS without size in tags** → T042 (defaulting to 8GB test)

Edge cases handled implicitly by design (validate during implementation):

5. **High concurrent RPC volumes** → Design ensures thread safety (sync.Once in T010, stateless handlers), can add load test post-MVP if needed
6. **Unknown instance types not in pricing data** → Pricing lookup returns (0, false) triggering $0 cost with "not found" billing_detail (T040-T041 pattern)
7. **GetProjectedCost called for unsupported resource** → Returns error per validation in T035-T039, Supports() prevents this scenario
8. **Context cancellation during RPC** → Handled by gRPC runtime and pluginsdk.Serve() (T025), inherent to gRPC design
