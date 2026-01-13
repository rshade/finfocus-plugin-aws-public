# Feature Specification: Rename Plugin to FinFocus

**Feature Branch**: `001-rename-plugin-finfocus`  
**Created**: 2026-01-12  
**Status**: Draft  
**Input**: User description: "Rename Plugin: pulumicost-aws-plugin -> finfocus-plugin-aws-public (Issue #239)"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Core Dependency Migration (Priority: P1)

As a developer, I want to update the project's module name and core dependencies from pulumicost to finfocus so that the plugin aligns

**Why this priority**: This is the critical first step. Without updating the module name and dependencies, no other work can proceed as the code will not compile or link correctly with the new `finfocus` ecosystem.

**Independent Test**: Can be tested by verifying `go.mod` contains the new module name and `finfocus-spec v0.5.0` dependency, followed by a successful `go mod tidy` and compilation of a single package.

**Acceptance Scenarios**:

1. **Given** the current `finfocus` module, **When** the module name in `go.mod` is changed to `finfocus-plugin-aws-public`, **Then** all internal package imports can be successfully updated to reflect the new module path.
2. **Given** the new `finfocus-spec` dependency, **When** proto package imports are changed from `finfocus.v1` to `finfocus.v1`, **Then** the code correctly references types and methods from the new spec.

---

### User Story 2 - Build and Distribution Update (Priority: P2)

As a developer, I want to rename the command directory and update build configurations (Makefile, GoReleaser) so that the generated binaries and release artifacts use the new `finfocus` naming convention.

**Why this priority**: This ensures that the output of the project (the binaries) matches the new branding, which is essential for deployment and user consumption.

**Independent Test**: Can be tested by running `make build-region REGION=us-east-1` and verifying the resulting binary is named `finfocus-plugin-aws-public-us-east-1` instead of the old name.

**Acceptance Scenarios**:

1. **Given** the renamed command directory `cmd/finfocus-plugin-aws-public/`, **When** the `Makefile` is updated to point to this directory, **Then** the build process produces correctly named binaries.
2. **Given** an updated `.goreleaser.yaml`, **When** a dry-run release is performed, **Then** all generated archive files and binaries follow the `finfocus-plugin-aws-public` pattern.

---

### User Story 3 - Branding and Documentation Completion (Priority: P3)

As a user/developer, I want all logs, documentation, and non-code references to be updated to `finfocus` so that there is no confusion about the plugin's identity.

**Why this priority**: While not breaking the build, inconsistent naming in logs and docs creates significant technical debt and user confusion.

**Independent Test**: Can be tested by running a global search for the string "finfocus" (case-insensitive) and verifying zero occurrences in non-legacy contexts.

**Acceptance Scenarios**:

1. **Given** the plugin is running, **When** it emits logs to stderr, **Then** every log line is prefixed with `[finfocus-plugin-aws-public]` instead of `[pulumicost-aws-plugin]`.
2. **Given** the documentation files (README.md, docs/), **When** a user reads them, **Then** all descriptions, examples, and installation instructions refer only to `finfocus`.

---

### Edge Cases

- **Legacy Environment Variables**: The plugin will support both `PULUMICOST_` and `FINFOCUS_` prefixes for environment variables. `FINFOCUS_` variables will take precedence if both are set.
- **Plugin Discovery**: How does the system handle a situation where both old and new binaries exist in the same path?
- **Dependency Conflicts**: What happens if a sub-dependency still refers to `finfocus-spec`?

## Clarifications

### Session 2026-01-12

- Q: Should occurrences of `finfocus` in the `specs/` directory (historical specifications) be updated to `finfocus` as part of this rename? → A: Update all occurrences to ensure consistency across reference material.
- Q: What is the lifecycle for the legacy `FINFOCUS_` environment variable support? → A: Support as deprecated in v0.2.x with log warnings, targeted for removal in v0.3.0.
- Q: Should we provide legacy-named binary aliases (e.g., symlinks) for backward compatibility? → A: No. Only produce `finfocus-plugin-aws-public` binaries to keep the build process simple and encourage migration.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST use `github.com/rshade/finfocus-plugin-aws-public` as its Go module name.
- **FR-002**: The system MUST depend on `github.com/rshade/finfocus-spec v0.5.0` or higher.
- **FR-003**: All source code files MUST use `finfocus.v1` for gRPC/protobuf package imports.
- **FR-004**: The main entry point directory MUST be renamed to `cmd/finfocus-plugin-aws-public/`.
- **FR-005**: The plugin MUST register itself with the name `finfocus-plugin-aws-public` in its gRPC handshake/announcement.
- **FR-006**: The `Makefile` MUST be updated to support building binaries with the new name.
- **FR-007**: The `.goreleaser.yaml` MUST be updated to produce artifacts with the new name.
- **FR-008**: All structured logging MUST use the prefix `[finfocus-plugin-aws-public]`.
- **FR-009**: The `README.md`, all files in `docs/`, and all historical files in `specs/` MUST be updated to replace `finfocus` with `finfocus`.
- **FR-010**: All test fixtures and integration tests MUST be updated to use the new paths and names.
- **FR-011**: The system MUST support both `PULUMICOST_` and `FINFOCUS_` environment variable prefixes. `FINFOCUS_` takes precedence. Usage of `PULUMICOST_` MUST emit a deprecation warning log. This support is transitional for v0.2.x.

### Key Entities *(include if feature involves data)*

- **Plugin Identity**: The string "finfocus-plugin-aws-public" used for registration, logging, and filenames.
- **Protocol Spec**: The `finfocus.v1` protobuf definitions which define the interface between the core and the plugin.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of internal imports updated to the new module path, verified by successful compilation.
- **SC-002**: Zero occurrences of the string "finfocus" in the `internal/` and `cmd/` directories (excluding explicit legacy comments if any).
- **SC-003**: `make build` completes in under 2 minutes (baseline performance) and produces `finfocus-*` binaries.
- **SC-004**: `make test` reports 100% pass rate for all unit and integration tests.
- **SC-005**: `make lint` reports zero issues related to naming or module resolution.
- **SC-006**: The plugin is successfully discovered and queried by a `finfocus-spec` compliant client.