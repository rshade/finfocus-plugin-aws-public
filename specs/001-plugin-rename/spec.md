# Feature Specification: Plugin Rename to FinFocus

**Feature Branch**: `001-plugin-rename`  
**Created**: 2026-01-11  
**Status**: Draft  
**Input**: User description: "🔄 Rename Plugin: finfocus-plugin-aws-public → finfocus-plugin-aws-public

## 🔄 Project Rename: FinFocus Migration

### Overview
Complete the migration from `finfocus-plugin-aws-public` to `finfocus-plugin-aws-public` as outlined in [RENAME-PLAN.md](./RENAME-PLAN.md) Phase 3. The foundational `finfocus-spec@v0.5.0` is now available with the updated `finfocus.v1` proto package.

### Context
- **Source**: `finfocus-plugin-aws-public` 
- **Target**: `finfocus-plugin-aws-public`
- **Spec Version**: `finfocus-spec@v0.5.0` (replaces `finfocus-spec@v0.4.14`)
- **Proto Package**: `finfocus.v1` (replaces `finfocus.v1`)
- **Breaking Change**: Yes, requires v0.2.0 release per RENAME-PLAN.md

### Implementation Tasks

#### Phase 1: Core Dependencies ✅
- [ ] **Update go.mod module name** from `finfocus-plugin-aws-public` to `finfocus-plugin-aws-public`
- [ ] **Update spec dependency** from `github.com/rshade/finfocus-spec v0.4.14` to `github.com/rshade/finfocus-spec v0.5.0`
- [ ] **Update all imports** from `finfocus-spec` to `finfocus-spec` (40+ files affected)
- [ ] **Update proto package imports** from `finfocus.v1` to `finfocus.v1`

#### Phase 2: Build & Binary Changes
- [ ] **Rename command directory** `cmd/finfocus-plugin-aws-public/` → `cmd/finfocus-plugin-aws-public/`
- [ ] **Update main.go** and plugin registration to use `finfocus` naming
- [ ] **Update Makefile** to build `finfocus-plugin-aws-public*` binaries
- [ ] **Update .goreleaser.yaml** for new binary names and paths

#### Phase 3: Code & Documentation
- [ ] **Update logging prefixes** from `[finfocus-plugin-aws-public]` to `[finfocus-plugin-aws-public]`
- [ ] **Update README.md, docs, and all references** from `finfocus` to `finfocus`
- [ ] **Update .gitignore and test files** with new paths

#### Phase 4: Verification
- [ ] **Run make lint** and ensure no linting errors
- [ ] **Run make test** and ensure all tests pass
- [ ] **Verify builds work** with new naming
- [ ] **Test plugin installation** and functionality

### Files Requiring Changes
**High Priority (40+ files):**
- `go.mod` - Module name and dependencies
- `cmd/finfocus-plugin-aws-public/main.go` - Entry point
- `internal/plugin/*.go` - All plugin implementation files
- `Makefile` - Build configuration
- `.goreleaser.yaml` - Release configuration

**Documentation (10+ files):**
- `README.md` - Main documentation
- `docs/` - All documentation files
- `*.md` files - References in markdown

### Acceptance Criteria
- [ ] **Module builds successfully** with `make build`
- [ ] **All tests pass** with `make test` 
- [ ] **Linting passes** with `make lint`
- [ ] **Plugin registers correctly** as `finfocus-plugin-aws-public`
- [ ] **Binary outputs** match new naming convention
- [ ] **Documentation updated** throughout codebase
- [ ] **No remaining references** to `finfocus` in code
- [ ] **Ready for v0.2.0 release** as breaking change

### Dependencies
- ✅ `finfocus-spec@v0.5.0` - Available and tested
- ⏳ `finfocus-core` - Phase 2 of RENAME-PLAN.md (not required for this plugin)

### Testing Strategy
- Run full test suite after each phase
- Verify plugin discovery works with new naming
- Test end-to-end with `finfocus-core` once available
- Validate against finfocus-spec conformance tests

### Risk Assessment
- **High Impact**: Breaking change affecting users
- **Migration Path**: Clear upgrade path documented
- **Rollback Plan**: Can revert to previous commit if issues found

### Related Issues
- Part of [RENAME-PLAN.md](./RENAME-PLAN.md) Phase 3
- Depends on finfocus-spec#273 (completed)
- Follows finfocus-core Phase 2 (when available)"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Migrate Core Dependencies (Priority: P1)

As a developer, I want to update the module name, spec dependency, and all imports from finfocus to finfocus so that the plugin uses the new naming and dependencies.

**Why this priority**: This is the foundational change required for the rename, enabling all other phases.

**Independent Test**: Can be tested by verifying go.mod has correct module name and dependencies, and imports compile without errors.

**Acceptance Scenarios**:

1. **Given** the current codebase, **When** go.mod is updated to finfocus-plugin-aws-public and dependencies are changed, **Then** go mod tidy succeeds and all imports resolve.
2. **Given** updated dependencies, **When** code with old imports is updated to finfocus, **Then** the build succeeds without import errors.

---

### User Story 2 - Update Build and Binary Configuration (Priority: P2)

As a developer, I want to rename command directories, update main.go, Makefile, and goreleaser config so that binaries are built with the new finfocus naming.

**Why this priority**: This ensures the plugin can be built and distributed under the new name.

**Independent Test**: Can be tested by running make build and verifying output binaries have finfocus in the name.

**Acceptance Scenarios**:

1. **Given** renamed command directory and updated main.go, **When** make build is run, **Then** binaries are created with finfocus-plugin-aws-public prefix.
2. **Given** updated .goreleaser.yaml, **When** release process runs, **Then** artifacts use correct finfocus naming.

---

### User Story 3 - Update Code and Documentation (Priority: P3)

As a developer, I want to update logging prefixes, documentation, and all references from finfocus to finfocus so that the codebase is consistently renamed.

**Why this priority**: This completes the rename in user-facing and internal documentation.

**Independent Test**: Can be tested by searching for any remaining finfocus references and verifying they are all updated.

**Acceptance Scenarios**:

1. **Given** updated logging prefixes, **When** plugin runs, **Then** logs show [finfocus-plugin-aws-public] prefix.
2. **Given** updated documentation, **When** README and docs are reviewed, **Then** all references use finfocus naming.

---

### Edge Cases

- What happens when build fails due to missing dependencies during transition?
- How does system handle partial updates if some files are missed?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST update go.mod module name from `finfocus-plugin-aws-public` to `finfocus-plugin-aws-public`
- **FR-002**: System MUST update spec dependency from `github.com/rshade/finfocus-spec v0.4.14` to `github.com/rshade/finfocus-spec v0.5.0`
- **FR-003**: System MUST update all imports from `finfocus-spec` to `finfocus-spec` across 40+ files
- **FR-004**: System MUST update proto package imports from `finfocus.v1` to `finfocus.v1`
- **FR-005**: System MUST rename command directory `cmd/finfocus-plugin-aws-public/` to `cmd/finfocus-plugin-aws-public/`
- **FR-006**: System MUST update main.go and plugin registration to use `finfocus` naming
- **FR-007**: System MUST update Makefile to build `finfocus-plugin-aws-public*` binaries
- **FR-008**: System MUST update .goreleaser.yaml for new binary names and paths
- **FR-009**: System MUST update logging prefixes from `[finfocus-plugin-aws-public]` to `[finfocus-plugin-aws-public]`
- **FR-010**: System MUST update README.md, docs, and all references from `finfocus` to `finfocus`
- **FR-011**: System MUST update .gitignore and test files with new paths
- **FR-012**: System MUST ensure make lint passes with no errors
- **FR-013**: System MUST ensure make test passes all tests
- **FR-014**: System MUST verify builds work with new naming
- **FR-015**: System MUST test plugin installation and functionality

### Key Entities *(include if feature involves data)*

- **Plugin Binary**: The compiled plugin executable with new naming
- **Dependencies**: Updated go modules and spec packages
- **Documentation**: Updated files and references

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Module builds successfully with `make build`
- **SC-002**: All tests pass with `make test`
- **SC-003**: Linting passes with `make lint`
- **SC-004**: Plugin registers correctly as `finfocus-plugin-aws-public`
- **SC-005**: Binary outputs match new naming convention
- **SC-006**: Documentation updated throughout codebase
- **SC-007**: No remaining references to `finfocus` in code
- **SC-008**: Ready for v0.2.0 release as breaking change
