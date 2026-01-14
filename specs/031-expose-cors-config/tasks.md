# Tasks: Expose CORS configuration via environment variables

**Feature Branch**: `031-expose-cors-config`
**Status**: Completed

## Implementation Strategy

We will implement the CORS configuration support in phases, starting with a robust parsing foundation, then incrementally adding features per user story, and validating with integration tests.

- **MVP Scope**: User Story 1 (Basic CORS with Allowed Origins)
- **Approach**: TDD-lite (Unit tests for parser first, then implementation, then integration verification)
- **Parallelism**: Configuration parsing logic and integration test scaffold can be built in parallel.

## Phase 1: Setup

<!--
  Project initialization and structural setup.
-->

- [x] T001 Create `cmd/finfocus-plugin-aws-public/config.go` for `parseWebConfig` stub

## Phase 2: Foundational

<!--
  Blocking prerequisites for all user stories.
-->

- [x] T002 Implement `parseWebConfig` function signature and return type in `cmd/finfocus-plugin-aws-public/config.go`
- [x] T003 Create `cmd/finfocus-plugin-aws-public/config_test.go` with initial test table structure

## Phase 3: User Story 1 - Configure Cross-Origin Access (P1)

<!--
  Goal: Allow specific origins to access the plugin from a browser.
  Independent Test: Curl with Origin header gets Access-Control-Allow-Origin response.
-->

- [x] T004 [US1] Add test cases for `FINFOCUS_CORS_ALLOWED_ORIGINS` parsing (valid, empty, wildcard) to `cmd/finfocus-plugin-aws-public/config_test.go`
- [x] T005 [US1] Implement parsing logic for allowed origins in `cmd/finfocus-plugin-aws-public/config.go`
- [x] T006 [US1] Implement wildcard warning logging in `cmd/finfocus-plugin-aws-public/config.go`
- [x] T007 [US1] Implement `FR-007` & `FR-008` (Max Age parsing/defaults) in `cmd/finfocus-plugin-aws-public/config.go`
- [x] T008 [US1] Update `cmd/finfocus-plugin-aws-public/main.go` to use `parseWebConfig` and set `pluginsdk.WebConfig`
- [x] T022 [US1] Implement `FR-009` (Log applied CORS config at debug level) in `cmd/finfocus-plugin-aws-public/config.go`
- [x] T009 [US1] Create integration test `test/integration/cors_test.go` verifying Basic CORS and Max Age

## Phase 4: User Story 2 - Enable Credentials (P2)

<!--
  Goal: Support authenticated requests with credentials.
  Independent Test: Start with creds=true, verify Access-Control-Allow-Credentials header.
-->

- [x] T010 [US2] Add test cases for `FINFOCUS_CORS_ALLOW_CREDENTIALS` (true/false, case-insensitivity) to `cmd/finfocus-plugin-aws-public/config_test.go`
- [x] T011 [US2] Add test case for Fatal Error on Wildcard + Credentials to `cmd/finfocus-plugin-aws-public/config_test.go`
- [x] T012 [US2] Implement credentials parsing logic in `cmd/finfocus-plugin-aws-public/config.go`
- [x] T013 [US2] Implement fatal error validation (wildcard + creds) in `cmd/finfocus-plugin-aws-public/config.go`
- [x] T014 [US2] Implement fatal error handling (exit process) in `cmd/finfocus-plugin-aws-public/main.go`
- [x] T015 [US2] Update `test/integration/cors_test.go` to verify Credentials header and Fatal Error exit

## Phase 5: User Story 3 - Health Endpoint (P2)

<!--
  Goal: Expose /healthz for orchestration probes.
  Independent Test: Curl /healthz returns 200 OK.
-->

- [x] T016 [US3] Add test cases for `FINFOCUS_PLUGIN_HEALTH_ENDPOINT` parsing to `cmd/finfocus-plugin-aws-public/config_test.go`
- [x] T017 [US3] Implement health endpoint parsing logic in `cmd/finfocus-plugin-aws-public/config.go`
- [x] T018 [US3] Update `test/integration/cors_test.go` to verify `/healthz` endpoint availability

## Phase 6: Polish & Cross-Cutting

<!--
  Documentation, cleanup, and final verification.
-->

- [x] T019 Update `CLAUDE.md` with new environment variables documentation
- [x] T020 Run `make lint` and fix any new linting issues
- [x] T021 Run `make test` to ensure no regressions in existing suites

## Dependencies

- **US1** must complete before **US2** (credentials build on basic CORS config structure)
- **US3** is independent of US1/US2 logic but shares the same config struct
- **T008** (Main integration) blocks all integration tests (T009, T015, T018)

## Parallel Execution Opportunities

- **T009**, **T015**, **T018** (Integration Tests) can be written in parallel with Implementation tasks, though they require the binary to pass.
- **US3** (Health Endpoint) logic can be implemented in parallel with **US1**/**US2** if multiple developers were involved (editing same file `config.go` but different struct fields).
