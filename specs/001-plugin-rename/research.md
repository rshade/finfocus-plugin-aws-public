# Research: Plugin Rename to FinFocus

**Feature**: 001-plugin-rename
**Date**: 2026-01-11
**Status**: Complete

## Overview

This document consolidates research findings for the plugin rename from `pulumicost-plugin-aws-public` to `finfocus-plugin-aws-public`. Since this is a systematic rename operation with no architecture changes, no external research was required. All technical decisions are established from the existing codebase and RENAME-PLAN.md Phase 3 requirements.

## Research Tasks

### Task 1: Module Rename Best Practices for Go Projects

**Decision**: Use `go mod edit -module` command and update all import statements

**Rationale**:
- Standard Go tooling provides module rename support
- `go mod edit -module new-name` updates go.mod
- Import statements must be manually updated or using automated refactoring tools
- `go mod tidy` resolves dependencies after rename

**Alternatives Considered**:
- Manual editing of go.mod only - rejected because imports still reference old module name
- Creating new module from scratch - rejected because this would lose history and Git metadata

**Implementation**:
```bash
go mod edit -module github.com/rshade/finfocus-plugin-aws-public
# Update all imports from pulumicost-spec to finfocus-spec
go mod tidy
```

### Task 2: Proto Package Migration

**Decision**: Update proto imports from `pulumicost.v1` to `finfocus.v1`

**Rationale**:
- Proto package names changed in finfocus-spec v0.5.0
- Message structure is identical between pulumicost.v1 and finfocus.v1
- Only package name changes, no breaking changes to message fields
- gRPC service interface remains compatible

**Alternatives Considered**:
- Keep using pulumicost-spec v0.4.14 - rejected because finfocus-spec v0.5.0 is the new standard
- Shim layer to translate between packages - rejected as unnecessary complexity

**Implementation**:
- Find all `import "pulumicost.v1"` statements
- Replace with `import "finfocus.v1"`
- Update proto code generation if custom proto files exist

### Task 3: Build System Update Strategy

**Decision**: Update Makefile and .goreleaser.yaml to use finfocus naming

**Rationale**:
- Makefile contains build targets referencing pulumicost-plugin-aws-public
- .goreleaser.yaml defines binary names and release artifact naming
- Binary naming is part of the user-facing contract
- Build tags (region_use1, region_usw2, region_euw1) remain unchanged

**Alternatives Considered**:
- Alias binaries (create symlinks) - rejected because this maintains legacy naming
- Environment variable-based naming - rejected as unnecessary complexity

**Implementation**:
- Update `BINARY_NAME` variable in Makefile
- Update binary name templates in .goreleaser.yaml
- Update reference to cmd/pulumicost-plugin-aws-public in build paths
- Keep region build tags unchanged

### Task 4: Directory Renaming Strategy

**Decision**: Rename `cmd/pulumicost-plugin-aws-public/` to `cmd/finfocus-plugin-aws-public/`

**Rationale**:
- Command directory follows module naming convention
- Go import paths must match directory structure for local packages
- Maintains consistency with new module name

**Alternatives Considered**:
- Keep old directory name - rejected because import paths would be inconsistent
- Git mv vs cp+rm - use `git mv` to preserve file history

**Implementation**:
```bash
git mv cmd/pulumicost-plugin-aws-public cmd/finfocus-plugin-aws-public
# Update internal imports if any
```

### Task 5: Logging Prefix Update

**Decision**: Update logging prefixes from `[pulumicost-plugin-aws-public]` to `[finfocus-plugin-aws-public]`

**Rationale**:
- Constitution requirement III: "Log entries MUST include [pulumicost-plugin-aws-public] component identifier"
- Update to maintain consistency with new naming
- Zerolog logger initialization typically includes component name

**Alternatives Considered**:
- Keep old logging prefix - rejected because it violates updated naming convention
- Remove prefix entirely - rejected because constitution requires component identifier

**Implementation**:
- Find all `[pulumicost-plugin-aws-public]` strings in logging code
- Replace with `[finfocus-plugin-aws-public]`
- Ensure log level configuration (LOG_LEVEL env var) remains unchanged

### Task 6: Documentation Update Strategy

**Decision**: Update all references in README.md, docs/, and markdown files

**Rationale**:
- Documentation is user-facing and must reflect new naming
- Code examples must use correct import paths
- README typically contains installation instructions with binary names

**Alternatives Considered**:
- Aliases or redirect pages - rejected as unnecessary complexity
- Leave legacy references - rejected because it confuses users

**Implementation**:
- Use grep to find all occurrences of "pulumicost" in markdown files
- Replace with "finfocus" where appropriate (package names, binary names)
- Update import path examples in code blocks
- Update links to other repositories if they reference pulumicost-spec

## Verification Strategy

### Build Verification
```bash
make build
# Verify binaries named: finfocus-plugin-aws-public-<region>
```

### Test Verification
```bash
make test
# Verify all tests pass
```

### Lint Verification
```bash
make lint
# Verify golangci-lint passes
```

### gRPC Functionality Verification
```bash
# Build and start plugin
./bin/finfocus-plugin-aws-public-use1
# Use grpcurl to test gRPC methods
grpcurl -plaintext 127.0.0.1:<PORT> list
```

### Reference Check
```bash
# Verify no remaining pulumicost references
grep -r "pulumicost" --exclude-dir=specs --exclude-dir=.git .
# Should return only results in RENAME-PLAN.md, AGENTS.md, etc. (documentation of migration)
```

## Risks and Mitigations

### Risk 1: Missed Import Statements
**Mitigation**: Use automated refactoring tools (gopls) to update all imports, then verify with `go mod tidy`

### Risk 2: Documentation Inconsistencies
**Mitigation**: Comprehensive grep search for "pulumicost" in markdown files, manual review of code examples

### Risk 3: Build System Errors
**Mitigation**: Run full build for all three regions (us-east-1, us-west-2, eu-west-1) before committing

### Risk 4: gRPC Protocol Incompatibility
**Mitigation**: Verify proto message structure is identical between pulumicost.v1 and finfocus.v1, manual gRPC testing

## Open Questions

None. All technical decisions are established and documented in RENAME-PLAN.md Phase 3.

## Next Steps

1. Execute Phase 1: Generate design artifacts (data-model.md, quickstart.md)
2. Proceed to Phase 2: Generate implementation tasks (tasks.md)
3. Execute implementation following the 4-phase approach outlined in spec.md

## References

- RENAME-PLAN.md Phase 3
- finfocus-spec v0.5.0 release notes
- Constitution v2.2.0
- AGENTS.md build/test/lint commands