# Implementation Plan - Rename Plugin to FinFocus

**Feature:** Rename Plugin to FinFocus  
**Feature Branch:** `001-rename-plugin-finfocus`  
**Specification:** [specs/001-rename-plugin-finfocus/spec.md](specs/001-rename-plugin-finfocus/spec.md)  
**Status:** Phase 2: Implementation

## Technical Context

This feature involves a comprehensive renaming of the Go module, binaries, and documentation from `finfocus-plugin-aws-public` to `finfocus-plugin-aws-public`. This is a breaking change that aligns with the project's rebranding to FinFocus.

**Key Technical components:**

- **Go Module:** `go.mod` rename and import path updates.
- **Dependencies:** Upgrade to `github.com/rshade/finfocus-spec v0.5.0` and usage of `finfocus.v1` proto package.
- **Build System:** `Makefile` and `.goreleaser.yaml` updates for binary naming.
- **Plugin Logic:** Update `internal/plugin` to register as `finfocus-plugin-aws-public` and use new logging prefixes.
- **Documentation:** Global search-and-replace in `README.md`, `docs/`, and `specs/`.

**Architecture:**
The plugin architecture remains unchanged (gRPC server), but the identity it presents and the protocol package it uses are updated.

**Constraints:**
- Must maintain backward compatibility for `FINFOCUS_` environment variables (with deprecation warnings).
- Must explicitly NOT provide binary aliases (symlinks) to encourage migration.

**Unknowns & Risks:**
- **Risk:** Missed internal imports or string references could break the build or runtime behavior.
- **Risk:** CI/CD pipelines (GitHub Actions) might need updates if they rely on hardcoded paths/names not covered by the repo content.
- **Unknown:** Are there any hardcoded references in the `tools/` directory that need special handling? [RESOLVED: None found]

## Constitution Check

### Principle I: Code Quality & Simplicity
- **Compliance:** The change is primarily a rename and refactor, simplifying the codebase by aligning with the new project identity.
- **Violation Risk:** None identified.

### Principle II: Testing Discipline
- **Compliance:** Existing tests will be updated to use the new names. No logic changes, so existing coverage should hold.
- **Violation Risk:** None identified.

### Principle III: Protocol & Interface Consistency
- **Compliance:** Updates to use `finfocus.v1` proto package, which is the new standard.
- **Violation Risk:** Must ensure `Name()` returns the new plugin name `finfocus-plugin-aws-public`.

### Principle IV: Performance & Reliability
- **Compliance:** Naming changes do not impact performance.
- **Violation Risk:** None identified.

### Principle V: Build & Release Quality
- **Compliance:** `Makefile` and `.goreleaser.yaml` updates ensure consistent build artifacts.
- **Violation Risk:** Breaking changes to binary names must be communicated (handled by the feature nature).

### Security Requirements
- **Compliance:** No changes to security posture.
- **Violation Risk:** None identified.

## Complexity Tracking

| Component | Logic Complexity (Low/Med/High) | Interaction Complexity (Low/Med/High) | Justification |
|-----------|---------------------------------|---------------------------------------|---------------|
| Build System | Low | Low | Simple rename of artifacts. |
| Plugin Core | Low | Low | Renaming imports and constants. |

## Proposed Gates

1. **Research Gate:** Confirm all occurrences of legacy names and identify any tricky edge cases (e.g., in tools). [COMPLETED]
2. **Design Gate:** Verify data model changes (none expected beyond naming) and contract updates. [COMPLETED]
3. **Implementation Gate:** All tests pass, build succeeds, and linting is clean.

## Phase 0: Outline & Research

### Goals
1. Identify all file paths and content requiring updates.
2. Confirm the scope of changes in `tools/`.
3. Verify the exact version of `finfocus-spec` to use.

### Tasks
- [x] Scan codebase for "finfocus" to build a definitive replacement list.
- [x] Check `tools/` for hidden dependencies on the old name.
- [x] Confirm `finfocus-spec v0.5.0` availability.

## Phase 1: Design & Contracts

### Goals
1. Update `go.mod` and dependencies.
2. Define the exact mapping of old-to-new import paths.

### Tasks
- [x] Create `data-model.md` (minimal/renamed).
- [x] Update `contracts/` (created).
- [x] Update `quickstart.md` with new commands.

## Phase 2: Implementation (Skeleton)

### Goals
1. Execute the rename across all historical documentation (`specs/`).
2. Verify the codebase is clean of legacy references (excluding intentional ones).
3. Ensure the build and tests pass.

### Tasks
- [ ] T001: Execute global find-and-replace for `finfocus-plugin-aws-public` -> `finfocus-plugin-aws-public` in `specs/`.
- [ ] T002: Execute global find-and-replace for `[finfocus-plugin-aws-public]` -> `[finfocus-plugin-aws-public]` in `specs/`.
- [ ] T003: Update `internal/plugin/testmode.go` to support `FINFOCUS_TEST_MODE` with explicit deprecated warning if not already correct.
- [ ] T004: Ensure `Makefile` and `.goreleaser.yaml` are strictly using the new names (verify).
- [ ] T005: Run `go mod tidy` and verify dependencies.
- [ ] T006: Run `make build` and verify binary output name.
- [ ] T007: Run `make test` and ensure all pass.
- [ ] T008: Verify `README.md` and `docs/` are clean of legacy references.
