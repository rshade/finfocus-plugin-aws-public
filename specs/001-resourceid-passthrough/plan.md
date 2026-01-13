# Implementation Plan: Resource ID Passthrough in GetRecommendations

**Branch**: `001-resourceid-passthrough` | **Date**: 2025-12-26 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-resourceid-passthrough/spec.md`

## Summary

Pass through the native `Id` field from `ResourceDescriptor` to `Recommendation.Resource.Id` for proper resource correlation in batch requests. This requires updating finfocus-spec to v0.4.11+ and modifying the `GetRecommendations` handler to prioritize the native ID over the existing tag-based correlation.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: finfocus-spec v0.4.11+ (provides `Id` field on `ResourceDescriptor`), gRPC, zerolog
**Storage**: N/A (stateless gRPC service)
**Testing**: Go testing with table-driven tests, existing test patterns in `recommendations_test.go`
**Target Platform**: Linux server (gRPC plugin binary)
**Project Type**: Single (gRPC service plugin)
**Performance Goals**: No impact - simple field copy operation (<1μs overhead)
**Constraints**: Must maintain backward compatibility with tag-based correlation
**Scale/Scope**: Single file modification (`recommendations.go`), ~10-15 lines of code change

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. Code Quality & Simplicity | ✅ PASS | Simple field copy, no new abstractions |
| II. Testing Discipline | ✅ PASS | Table-driven unit tests for ID priority logic |
| III. Protocol & Interface Consistency | ✅ PASS | Uses proto-defined types only (ResourceDescriptor.Id, Recommendation.Resource.Id) |
| IV. Performance & Reliability | ✅ PASS | No performance impact, maintains thread safety |
| V. Build & Release Quality | ✅ PASS | Standard Go build, no new dependencies beyond spec version bump |

**Gate Result**: PASS - No constitution violations.

## Project Structure

### Documentation (this feature)

```text
specs/001-resourceid-passthrough/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output (minimal - no unknowns)
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── contracts/           # Phase 1 output (gRPC contract updates)
```

### Source Code (repository root)

```text
internal/
└── plugin/
    ├── recommendations.go       # MODIFY: Add native Id passthrough
    └── recommendations_test.go  # MODIFY: Add tests for Id priority
go.mod                           # MODIFY: Update finfocus-spec to v0.4.11+
```

**Structure Decision**: Existing plugin structure. No new files needed - only modifications to existing `recommendations.go` and its test file, plus dependency version bump.

## Complexity Tracking

> No constitution violations. Table not needed.

## Implementation Approach

### Change Summary

1. **Dependency Update** (`go.mod`):
   - Update `github.com/rshade/finfocus-spec` from current version to `v0.4.11`

2. **Code Change** (`internal/plugin/recommendations.go`):
   - Modify the correlation logic in `GetRecommendations` (around lines 112-123)
   - Add priority check: use `resource.Id` if non-empty, else fall back to `tags["resource_id"]`
   - Trim whitespace before checking emptiness

3. **Test Update** (`internal/plugin/recommendations_test.go`):
   - Add table-driven tests for ID passthrough scenarios:
     - Native ID populated → use native ID
     - Native ID empty, tag present → use tag
     - Both present → use native ID (priority)
     - Neither present → empty ID on recommendation
     - Whitespace-only native ID → use tag

### Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Breaking existing tag-based correlation | Low | High | Maintain fallback to tags when native ID empty |
| finfocus-spec v0.4.11 API incompatibility | Very Low | Medium | SDK follows semver; Id field is additive |
| Test coverage gap | Low | Low | Explicit test cases for all scenarios |

### Dependencies

- **External**: finfocus-spec v0.4.11 (available)
- **Internal**: None - self-contained change
