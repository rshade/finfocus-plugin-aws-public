# Implementation Plan - Expose CORS configuration via environment variables

**Feature Branch**: `031-expose-cors-config`
**Feature Spec**: `specs/031-expose-cors-config/spec.md`
**Status**: Draft
**Phase**: 1 - Design

## Technical Context

<!--
  Analyze the codebase and document technical choices/constraints.
  Mark unknowns as [NEEDS CLARIFICATION].
-->

### Architecture & Patterns

- **Application Type**: gRPC Service (Plugin)
- **Framework**: connect-go + pluginsdk (v0.5.0 verified)
- **Configuration Pattern**: Environment variables -> `main.go` -> `pluginsdk.WebConfig`
- **Existing CORS Support**: `pluginsdk` provides `WebConfig` struct. We need to populate it.
- **Entry Point**: `cmd/finfocus-plugin-aws-public/main.go` uses `pluginsdk.Serve`.

### Dependencies

- **pluginsdk**: `github.com/rshade/finfocus-spec/sdk/go/pluginsdk` (Local path in current `go.mod` context). Verified via `main.go` import.
- **standard library**: `os`, `strings`, `strconv` for env parsing.

### Code Locations

- **Entry Point**: `cmd/finfocus-plugin-aws-public/main.go`
- **Config Handling**: Inside `run()` function in `main.go`. Will extract to private `parseWebConfig` in `main.go` to keep `run()` clean.
- **Tests**: `cmd/finfocus-plugin-aws-public/main_test.go` (needs creation/expansion for unit tests).
- **Integration**: `test/integration/cors_test.go` (new file, modeled after `test/integration/web_server_test.go`).

## Constitution Check

<!--
  Validate the plan against the project constitution (.specify/memory/constitution.md).
-->

| Principle | Check | Notes |
|-----------|-------|-------|
| **I. Code Quality & Simplicity** | [x] | Keep parsing logic simple, single function `parseWebConfig`. |
| **II. Testing Discipline** | [x] | Unit tests for parser, integration for headers. |
| **III. Protocol Consistency** | [x] | No changes to gRPC protocol, just HTTP wrapper config. |
| **IV. Performance** | [x] | Env parsing only at startup (<1ms), no runtime impact. |
| **V. Build Quality** | [x] | Linting required, no new deps. |
| **Security** | [x] | Fail fast on insecure config (wildcard + creds). Warn on wildcard. |

## Gates & Checks

- [x] **Constitution Compliance**: Does this plan violate any core principles? (No)
- [x] **Spec Coverage**: Does the plan cover all FRs and SCs? (Yes)
- [x] **Unknowns Resolution**: Are all [NEEDS CLARIFICATION] items resolved? (Yes)

## Phase 0: Research & Decisions

<!--
  Document decisions made during research.
-->

### Decisions

- **Environment Parsing Strategy**:
  - **Decision**: Implement `parseWebConfig(enabled bool, logger zerolog.Logger) (pluginsdk.WebConfig, error)` in `main.go`.
  - **Rationale**: Keeps logic co-located with usage but testable. `run()` remains clean.
  
- **Error Handling**:
  - `parseWebConfig` returns error on fatal config (wildcard + creds).
  - `main` logs fatal and exits.
  - Warnings logged directly via zerolog within parser.

- **Testing Strategy**:
  - **Unit**: New `cmd/finfocus-plugin-aws-public/config_test.go` (or `main_test.go` if appropriate) testing `parseWebConfig` with table-driven tests.
  - **Integration**: New `test/integration/cors_test.go` following `web_server_test.go` pattern: build binary, start with env vars, verify headers via `http.Client`.

## Phase 1: Design & Contracts

<!--
  Define data models and API contracts.
-->

### Data Model

*No persistent data model changes.*

### Configuration Model (Runtime)

```go
type WebConfig struct {
    Enabled              bool
    AllowedOrigins       []string
    AllowCredentials     bool
    MaxAge               *int
    EnableHealthEndpoint bool
}
```

### API Contracts

- **Health Endpoint**:
  - `GET /healthz` -> 200 OK
  - Other methods -> 405/404 (handled by SDK/Connect-Go)

## Phase 2: Implementation Plan

<!--
  Break down implementation into phases/steps.
-->

### Step 1: Configuration Parsing Logic
- Create `cmd/finfocus-plugin-aws-public/config.go` to hold `parseWebConfig` logic.
- Implement `FR-001` to `FR-008`.
- Add unit tests in `cmd/finfocus-plugin-aws-public/config_test.go`.

### Step 2: Main Entry Integration
- Update `main.go` to call `parseWebConfig` instead of hardcoded struct.
- Wire up `pluginsdk.Serve`.
- Implement fatal error handling for `FR-005` (exit on error).

### Step 3: Integration Testing
- Create `test/integration/cors_test.go`.
- Test cases:
  - `TestCORS_AllowedOrigins`: Verify `Access-Control-Allow-Origin`.
  - `TestCORS_NoHeaders`: Verify no headers for non-matching origin.
  - `TestCORS_Credentials`: Verify `Access-Control-Allow-Credentials`.
  - `TestHealthEndpoint`: Verify `GET /healthz`.
  - `TestFatalConfig`: Verify process exits with error code when config is invalid (wildcard + creds).

### Step 4: Documentation
- Update `CLAUDE.md` with new env vars.
