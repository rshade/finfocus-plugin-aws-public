# Tasks: ElastiCache Cost Estimation

**Input**: Design documents from `/specs/001-elasticache/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, quickstart.md

**Tests**: Unit tests and integration tests are included per existing codebase patterns.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: Go plugin at repository root
- All source in `internal/` and `tools/`
- Tests colocated with source files (`*_test.go`)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Pricing data generation and embed infrastructure for ElastiCache

- [x] T001 Add "AmazonElastiCache" to serviceConfig map in tools/generate-pricing/main.go
- [x] T002 Add elasticache product family filter ("Cache Instance") in tools/generate-pricing/main.go (N/A - filtering done in parser per constitution; generator fetches raw data)
- [x] T003 Run `go run ./tools/generate-pricing --regions us-east-1 --out-dir ./data` to generate pricing data
- [x] T004 Verify elasticache_us-east-1.json file is created and contains Cache Instance products
- [x] T004a [P] Run `go run ./tools/generate-pricing --regions us-west-2,eu-west-1 --out-dir ./data` to verify multi-region support (FR-013)
- [x] T004b Verify elasticache JSON files are under 10MB per region per SC-006 (`ls -lh data/elasticache_*.json`) - All ~310KB

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Pricing types, embed files, and client infrastructure that MUST be complete before estimator work

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 Add elasticacheInstancePrice struct to internal/pricing/types.go (N/A - using existing index pattern)
- [x] T006 [P] Add rawElastiCacheJSON embed directive to internal/pricing/embed_use1.go
- [x] T007 [P] Add rawElastiCacheJSON embed directive to internal/pricing/embed_usw2.go
- [x] T008 [P] Add rawElastiCacheJSON embed directive to internal/pricing/embed_euw1.go
- [x] T009 [P] Add rawElastiCacheJSON embed directive to internal/pricing/embed_apse1.go
- [x] T010 [P] Add rawElastiCacheJSON embed directive to internal/pricing/embed_apse2.go
- [x] T011 [P] Add rawElastiCacheJSON embed directive to internal/pricing/embed_apne1.go
- [x] T012 [P] Add rawElastiCacheJSON embed directive to internal/pricing/embed_aps1.go
- [x] T013 [P] Add rawElastiCacheJSON embed directive to internal/pricing/embed_cac1.go
- [x] T014 [P] Add rawElastiCacheJSON embed directive to internal/pricing/embed_sae1.go
- [x] T015 [P] Add rawElastiCacheJSON embed directive to internal/pricing/embed_govw1.go
- [x] T016 [P] Add rawElastiCacheJSON embed directive to internal/pricing/embed_gove1.go
- [x] T017 [P] Add rawElastiCacheJSON embed directive to internal/pricing/embed_usw1.go
- [x] T018 Add rawElastiCacheJSON fallback test data to internal/pricing/embed_fallback.go
- [x] T019 Add elasticacheIndex field to Client struct in internal/pricing/client.go
- [x] T020 Add elasticacheEngineNormalization map to internal/pricing/client.go
- [x] T021 Implement parseElastiCachePricing() method in internal/pricing/client.go
- [x] T022 Add parseElastiCachePricing to parallel init goroutine in internal/pricing/client.go
- [x] T023 Implement ElastiCacheOnDemandPricePerHour() lookup method in internal/pricing/client.go
- [x] T024 Verify build compiles with `go build -tags region_use1 ./cmd/pulumicost-plugin-aws-public`

**Checkpoint**: Foundation ready - ElastiCache pricing data is embedded and parseable

---

## Phase 3: User Story 1 - Basic ElastiCache Cost Estimation (Priority: P1)

**Goal**: Return cost estimates for ElastiCache clusters with instance type and engine

**Independent Test**: Submit ElastiCache resource descriptor, verify non-zero monthly cost returned

### Implementation for User Story 1

- [x] T025 [US1] Add "elasticache" case to detectService() switch in internal/plugin/projected.go
- [x] T026 [US1] Add Pulumi resource type normalization for aws:elasticache/* in internal/plugin/projected.go
- [x] T027 [US1] Implement estimateElastiCache() function in internal/plugin/projected.go
- [x] T028 [US1] Add "elasticache" case to GetProjectedCost dispatcher in internal/plugin/projected.go
- [x] T029 [US1] Add unit test for basic Redis estimation in internal/plugin/projected_test.go
- [x] T030 [US1] Add unit test for Pulumi Cluster format in internal/plugin/projected_test.go
- [x] T031 [US1] Add unit test for Pulumi ReplicationGroup format in internal/plugin/projected_test.go

**Checkpoint**: Basic ElastiCache estimation works for Redis with single node

---

## Phase 4: User Story 2 - Multi-Engine Support (Priority: P1)

**Goal**: Support Redis, Memcached, and Valkey engines with correct pricing

**Independent Test**: Submit requests with each engine type, verify non-zero costs for all

### Implementation for User Story 2

- [x] T032 [US2] Add engine tag extraction in estimateElastiCache() in internal/plugin/projected.go
- [x] T033 [US2] Add engine normalization call (lowercase to title case) in internal/plugin/projected.go
- [x] T034 [US2] Add unit test for Memcached engine in internal/plugin/projected_test.go
- [x] T035 [US2] Add unit test for Valkey engine in internal/plugin/projected_test.go
- [x] T036 [US2] Add unit test for case-insensitive engine handling in internal/plugin/projected_test.go

**Checkpoint**: All three engines return valid pricing

---

## Phase 5: User Story 3 - Multi-Node Cluster Estimation (Priority: P2)

**Goal**: Calculate costs based on node count from tags

**Independent Test**: Submit requests with different node counts, verify cost scales proportionally

### Implementation for User Story 3

- [x] T037 [US3] Add node count extraction from num_cache_clusters tag in internal/plugin/projected.go
- [x] T038 [US3] Add node count extraction from num_nodes tag in internal/plugin/projected.go
- [x] T039 [US3] Add node count extraction from nodes tag in internal/plugin/projected.go (N/A - using num_cache_nodes instead)
- [x] T040 [US3] Update cost calculation to multiply by node count in internal/plugin/projected.go
- [x] T041 [US3] Add unit test for 3-node cluster pricing in internal/plugin/projected_test.go
- [x] T042 [US3] Add unit test for num_cache_clusters tag parsing in internal/plugin/projected_test.go

**Checkpoint**: Multi-node clusters calculate correctly

---

## Phase 6: User Story 4 - Sensible Defaults (Priority: P2)

**Goal**: Apply and document default values for missing parameters

**Independent Test**: Submit minimal resource descriptor, verify defaults applied with billing detail notes

### Implementation for User Story 4

- [x] T043 [US4] Add default engine (Redis) when not specified in internal/plugin/projected.go
- [x] T044 [US4] Add default node count (1) when not specified in internal/plugin/projected.go
- [x] T045 [US4] Add "engine defaulted to redis" to billing detail in internal/plugin/projected.go (covered in test)
- [x] T046 [US4] Add "node count defaulted to 1" to billing detail in internal/plugin/projected.go (covered in test)
- [x] T047 [US4] Add unit test for default engine application in internal/plugin/projected_test.go
- [x] T048 [US4] Add unit test for billing detail includes default notes in internal/plugin/projected_test.go (covered in T047)

**Checkpoint**: Defaults work and are documented in billing detail

---

## Phase 7: User Story 5 - Support Check (Priority: P3)

**Goal**: Supports() returns true for ElastiCache resource types

**Independent Test**: Call Supports() with ElastiCache types, verify supported: true

### Implementation for User Story 5

- [x] T049 [US5] Add "elasticache" to supported services switch in internal/plugin/supports.go
- [x] T050 [US5] Add unit test for Supports() with elasticache type in internal/plugin/supports_test.go
- [x] T051 [US5] Add unit test for Supports() with Pulumi Cluster format in internal/plugin/supports_test.go
- [x] T052 [US5] Add unit test for Supports() with Pulumi ReplicationGroup format in internal/plugin/supports_test.go

**Checkpoint**: Supports() correctly identifies ElastiCache resources

---

## Phase 8: Error Handling & Edge Cases

**Purpose**: Handle error cases per research.md D9 decisions

- [x] T053 Add graceful $0 return for unknown instance type in internal/plugin/projected.go
- [x] T054 Add ERROR_CODE_INVALID_RESOURCE for missing instance type in internal/plugin/projected.go
- [x] T055 Add unit test for unknown instance type returns $0 in internal/plugin/projected_test.go
- [x] T056 Add unit test for missing instance type returns error in internal/plugin/projected_test.go
- [x] T057 Add unit test for invalid node count returns error in internal/plugin/projected_test.go (returns error, not default)

**Checkpoint**: All error cases handled gracefully

---

## Phase 9: Integration Tests

**Purpose**: End-to-end validation with real binary

- [ ] T058 [P] Create integration_elasticache_test.go in internal/plugin/ (Deferred - comprehensive unit tests provide sufficient coverage)
- [ ] T059 Add integration test for basic Redis estimation in internal/plugin/integration_elasticache_test.go (Deferred)
- [ ] T060 Add integration test for multi-engine support in internal/plugin/integration_elasticache_test.go (Deferred)
- [ ] T061 Add integration test for multi-node cluster in internal/plugin/integration_elasticache_test.go (Deferred)
- [ ] T062 Run `go test -tags=integration,region_use1 ./internal/plugin/... -run ElastiCache` to validate (Deferred)

**Checkpoint**: Integration tests pass with real embedded pricing data

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and cleanup

- [x] T063 Run `make lint` and fix any linting issues
- [x] T064 Run `make test` and ensure all tests pass
- [x] T065 [P] Build with `go build -tags region_use1 ./cmd/pulumicost-plugin-aws-public` and verify binary size
- [ ] T066 Validate quickstart.md examples with grpcurl against running plugin (Deferred)
- [x] T067 Update CLAUDE.md with ElastiCache in supported services list

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phases 3-7)**: All depend on Foundational phase completion
  - US1 (P1) must complete before US2-US4 can start (provides estimator shell)
  - US2-US4 can run in parallel after US1
  - US5 can run in parallel with any story (different file)
- **Error Handling (Phase 8)**: Depends on US1 estimator being in place
- **Integration Tests (Phase 9)**: Depends on all user stories complete
- **Polish (Phase 10)**: Depends on all tests passing

### User Story Dependencies

```text
Phase 2 (Foundational)
        │
        ▼
    ┌───────┐
    │  US1  │  (Basic estimation - must be first)
    └───┬───┘
        │
    ┌───┴───┬───────┐
    ▼       ▼       ▼
┌──────┐ ┌──────┐ ┌──────┐
│  US2 │ │  US3 │ │  US4 │  (Can run in parallel)
└──────┘ └──────┘ └──────┘
        │
        ▼
    ┌───────┐
    │  US5  │  (Can run anytime after Phase 2)
    └───────┘
```

### Within Each User Story

- Implementation tasks before tests (but tests can be written first if TDD preferred)
- Core logic before edge cases
- Unit tests before integration tests

### Parallel Opportunities

- All embed file tasks (T006-T017) can run in parallel
- US2, US3, US4 can run in parallel after US1 completes
- US5 can run in parallel with any other story
- All integration tests (T058-T061) can run in parallel

---

## Parallel Example: Foundational Phase

```bash
# Launch all embed file updates together:
Task: "Add rawElastiCacheJSON embed directive to internal/pricing/embed_use1.go"
Task: "Add rawElastiCacheJSON embed directive to internal/pricing/embed_usw2.go"
Task: "Add rawElastiCacheJSON embed directive to internal/pricing/embed_euw1.go"
# ... (all 12 embed files in parallel)
```

## Parallel Example: After US1 Completes

```bash
# Launch US2, US3, US4 together:
Task: "[US2] Add engine tag extraction in estimateElastiCache()"
Task: "[US3] Add node count extraction from num_cache_clusters tag"
Task: "[US4] Add default engine (Redis) when not specified"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (pricing generation)
2. Complete Phase 2: Foundational (embed files, parser, types)
3. Complete Phase 3: User Story 1 (basic estimation)
4. **STOP and VALIDATE**: Test with grpcurl using quickstart.md examples
5. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Pricing infrastructure ready
2. Add User Story 1 → Basic Redis estimation works (MVP!)
3. Add User Story 2 → All engines supported
4. Add User Story 3 → Multi-node clusters work
5. Add User Story 4 → Sensible defaults applied
6. Add User Story 5 → Supports() works
7. Error handling + Integration tests → Production ready

### Single Developer Strategy

Execute in order: Phase 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10

Each phase is a logical commit point.

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Total: 69 tasks across 10 phases
