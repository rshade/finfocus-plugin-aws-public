# Tasks: Zerolog Structured Logging with Trace Propagation

**Input**: Design documents from `/specs/005-zerolog-logging/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Tests included per constitution (Testing Discipline section) and plan.md.

**Organization**: Tasks grouped by user story for independent implementation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story (US1, US2, US3, US4)
- Exact file paths included in descriptions

## Path Conventions

- **Go plugin**: `cmd/`, `internal/` at repository root
- Per plan.md: `cmd/pulumicost-plugin-aws-public/`, `internal/plugin/`,
  `internal/pricing/`

---

## Phase 1: Setup

**Purpose**: Add dependencies and logger infrastructure

- [x] T001 Add zerolog v1.34.0+ and google/uuid dependencies in go.mod
- [x] T002 Run `go mod tidy` to resolve dependencies
- [x] T003 Verify SDK logging utilities available from pulumicost-spec v0.3.0+

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core logger infrastructure that all handlers depend on

**CRITICAL**: No handler instrumentation can begin until this phase is complete

- [x] T004 Add logger field to AWSPublicPlugin struct in internal/plugin/plugin.go
- [x] T005 Update NewAWSPublicPlugin constructor to accept logger parameter in
  internal/plugin/plugin.go
- [x] T006 Add getTraceID helper method with metadata extraction and UUID fallback
  in internal/plugin/plugin.go (see research.md U1 Remediation pattern)
- [x] T007 Add logError helper method for consistent error logging in
  internal/plugin/plugin.go
- [x] T008 Initialize logger in cmd/pulumicost-plugin-aws-public/main.go using
  NewPluginLogger with LOG_LEVEL env var
- [x] T009 Pass logger to NewAWSPublicPlugin in
  cmd/pulumicost-plugin-aws-public/main.go
- [x] T010 Implement getTraceID helper with manual metadata extraction workaround
  (ServeConfig does not support interceptors - see research.md U1 Remediation,
  tracked in rshade/pulumicost-core#188)

**Checkpoint**: Logger infrastructure ready - handler instrumentation can begin

---

## Phase 3: User Story 1 - End-to-End Request Tracing (P1)

**Goal**: Enable trace_id propagation through all handlers for request
correlation

**Independent Test**: Send gRPC request with trace_id in metadata, verify
trace_id appears in all log entries

### Implementation for User Story 1

- [x] T011 [US1] Add trace_id extraction at start of GetProjectedCost in
  internal/plugin/projected.go
- [x] T012 [US1] Add trace_id extraction at start of Supports in
  internal/plugin/supports.go
- [x] T013 [US1] Add trace_id extraction at start of GetActualCost in
  internal/plugin/actual.go
- [x] T014 [US1] Add trace_id to all log entries in GetProjectedCost handler
- [x] T015 [US1] Add trace_id to all log entries in Supports handler
- [x] T016 [US1] Add trace_id to all log entries in GetActualCost handler

### Tests for User Story 1

- [x] T017 [P] [US1] Add test for trace_id propagation with provided trace_id
  in internal/plugin/plugin_test.go
- [x] T018 [P] [US1] Add test for UUID generation when trace_id missing in
  internal/plugin/plugin_test.go
- [x] T019 [P] [US1] Add test for concurrent requests with different trace_ids
  in internal/plugin/plugin_test.go

**Checkpoint**: US1 complete - trace_id appears in all logs for any request

---

## Phase 4: User Story 2 - Structured Operation Logging (P2)

**Goal**: Log all operations with consistent SDK field names for dashboards

**Independent Test**: Examine log output and confirm all entries use SDK-defined
field constants (operation, resource_type, cost_monthly, duration_ms)

### Implementation for User Story 2

- [x] T020 [US2] Add operation timing and info logging to GetProjectedCost in
  internal/plugin/projected.go
- [x] T021 [US2] Add operation timing and info logging to Supports in
  internal/plugin/supports.go
- [x] T022 [US2] Add operation timing and info logging to GetActualCost in
  internal/plugin/actual.go
- [x] T023 [US2] Add resource_type, aws_service, aws_region fields to
  GetProjectedCost logs
- [x] T024 [US2] Add resource_type, aws_region, supported fields to Supports
  logs
- [x] T025 [US2] Add resource_type, cost_monthly, duration_ms fields to
  GetActualCost logs
- [x] T026 [US2] Add error logging with error_code field to all handlers using
  logError helper

### Tests for User Story 2

- [x] T027 [P] [US2] Add test validating GetProjectedCost logs contain required
  fields in internal/plugin/projected_test.go
- [x] T028 [P] [US2] Add test validating Supports logs contain required fields
  in internal/plugin/supports_test.go
- [x] T029 [P] [US2] Add test validating error logs contain error_code field
  in internal/plugin/plugin_test.go

**Checkpoint**: US2 complete - all logs use SDK field constants consistently

---

## Phase 5: User Story 3 - Plugin Startup Logging (P3)

**Goal**: Log plugin startup with version and region for deployment verification

**Independent Test**: Start plugin and verify startup log contains plugin_name,
plugin_version, aws_region

### Implementation for User Story 3

- [x] T030 [US3] Add startup info log after logger creation in
  cmd/pulumicost-plugin-aws-public/main.go
- [x] T031 [US3] Include plugin_name, plugin_version, aws_region in startup log
- [x] T032 [US3] Add error logging for initialization failures (pricing client,
  etc.) in cmd/pulumicost-plugin-aws-public/main.go

### Tests for User Story 3

- [x] T033 [P] [US3] Add test validating startup log format in
  internal/plugin/plugin_test.go

**Checkpoint**: US3 complete - startup logs show version and region

---

## Phase 6: User Story 4 - Cost Calculation Debugging (P3)

**Goal**: Add debug-level logs for SKU resolution and pricing decisions

**Independent Test**: Set LOG_LEVEL=debug, process EC2/EBS request, verify logs
show instance_type/storage_type lookup and pricing details

### Implementation for User Story 4

- [x] T034 [US4] Add debug logging for EC2 instance type lookup in
  internal/plugin/projected.go
- [x] T035 [US4] Add debug logging for EBS volume type lookup in
  internal/plugin/projected.go
- [x] T036 [US4] Add debug logging for pricing lookup results (unit_price) in
  internal/plugin/projected.go
- [x] T037 [US4] Add debug logging for SKU not found scenarios with attempted
  SKU value

### Tests for User Story 4

- [x] T038 [P] [US4] Add test validating debug logs contain instance_type for
  EC2 in internal/plugin/projected_test.go
- [x] T039 [P] [US4] Add test validating debug logs contain storage_type for
  EBS in internal/plugin/projected_test.go

**Checkpoint**: US4 complete - debug logs show SKU resolution details

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Benchmarks, validation, documentation

- [x] T040 Add benchmark for logging overhead in internal/plugin/plugin_test.go
- [x] T041 Verify benchmark shows <1ms overhead per SC-005
- [x] T042 Run `make lint` and fix any issues
- [x] T043 Run `make test` and verify all tests pass
- [x] T044 Validate log output against contracts/log-schema.json format
- [x] T045 Update CLAUDE.md with any new patterns discovered

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational completion
  - Can proceed in parallel or sequentially by priority
- **Polish (Phase 7)**: Depends on all user stories complete

### User Story Dependencies

- **US1 (P1)**: After Foundational - no dependencies on other stories
- **US2 (P2)**: After Foundational - builds on trace_id from US1 but
  independently testable
- **US3 (P3)**: After Foundational - no dependencies on other stories
- **US4 (P3)**: After Foundational - uses same log infrastructure, independent

### Within Each User Story

- Implementation tasks before tests (tests verify implementation)
- Core functionality before edge cases
- Handler changes before test validation

### Parallel Opportunities

**Foundational phase**:

```text
T004, T005, T006, T007 can be done together (same file, but logical sequence)
T008, T009, T010 can be done after plugin.go changes
```

**User Story 1**:

```text
T011, T012, T013 can run in parallel (different handler files)
T017, T018, T019 tests can run in parallel
```

**User Story 2**:

```text
T020, T021, T022 can run in parallel (different handler files)
T027, T028, T029 tests can run in parallel
```

**User Stories 3 & 4**:

```text
US3 and US4 can run in parallel (different concerns)
```

---

## Parallel Example: User Story 1

```bash
# After Foundational complete, launch trace_id extraction in parallel:
Task: "T011 [US1] Add trace_id extraction to GetProjectedCost"
Task: "T012 [US1] Add trace_id extraction to Supports"
Task: "T013 [US1] Add trace_id extraction to GetActualCost"

# Then launch tests in parallel:
Task: "T017 [P] [US1] Test trace_id propagation with provided trace_id"
Task: "T018 [P] [US1] Test UUID generation when trace_id missing"
Task: "T019 [P] [US1] Test concurrent requests with different trace_ids"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T010)
3. Complete Phase 3: User Story 1 (T011-T019)
4. **STOP and VALIDATE**: Test trace_id propagation
5. This gives basic observability - can deploy

### Incremental Delivery

1. Setup + Foundational → Logger infrastructure ready
2. Add US1 → trace_id correlation works → Deploy (MVP!)
3. Add US2 → structured field logging → Deploy
4. Add US3 → startup logging → Deploy
5. Add US4 → debug logging → Deploy
6. Each story adds value without breaking previous

### Recommended Approach for Solo Developer

1. T001-T010: Setup + Foundational (~30 min)
2. T011-T019: US1 trace_id propagation (~45 min)
3. T020-T029: US2 structured logging (~45 min)
4. T030-T033: US3 startup logging (~15 min)
5. T034-T039: US4 debug logging (~30 min)
6. T040-T045: Polish (~20 min)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- Each user story independently testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story
- LOG_LEVEL env var controls debug output
- All logs to stderr (stdout reserved for PORT)
