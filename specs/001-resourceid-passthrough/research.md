# Research: Resource ID Passthrough

**Feature**: 001-resourceid-passthrough
**Date**: 2025-12-26
**Status**: Complete (no unknowns)

## Summary

This feature has no technical unknowns requiring research. All decisions are straightforward based on:

1. The existing implementation in `recommendations.go`
2. The proto contract from finfocus-spec
3. Standard Go patterns already used in the codebase

## Decisions

### D-001: ID Priority Order

**Decision**: Native `ResourceDescriptor.Id` takes priority over `tags["resource_id"]`

**Rationale**:
- Native proto field is the canonical source
- Tag-based correlation was a workaround when proto lacked Id field
- Aligns with how finfocus-core will use the field

**Alternatives Considered**:
- Tag takes priority: Rejected - would break callers migrating to native ID
- Merge both: Rejected - unnecessary complexity, unclear semantics

### D-002: Whitespace Handling

**Decision**: Treat whitespace-only `Id` as empty (fall back to tag)

**Rationale**:
- Standard defensive programming
- Prevents subtle bugs from trailing spaces
- Consistent with Go idiom (`strings.TrimSpace`)

**Alternatives Considered**:
- Accept whitespace as valid ID: Rejected - almost certainly a bug
- Error on whitespace: Rejected - too strict, breaks existing callers

### D-003: Empty ID Behavior

**Decision**: When no ID available (neither native nor tag), leave `Resource.Id` empty

**Rationale**:
- Recommendations are still valid without correlation
- Caller can still use other fields (SKU, Region) for identification
- Existing behavior - no breaking change

**Alternatives Considered**:
- Generate synthetic ID: Rejected - misleading, not a real resource ID
- Error: Rejected - ID is optional, not required

## Dependencies Verified

| Dependency | Version | Status | Notes |
|------------|---------|--------|-------|
| finfocus-spec | v0.4.11 | ✅ Available | Contains `Id` field on `ResourceDescriptor` |

## No Further Research Needed

All technical decisions are clear. Proceed to Phase 1 design artifacts.
