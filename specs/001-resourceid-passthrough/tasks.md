# Tasks: Resource ID Passthrough in GetRecommendations

**Feature**: 001-resourceid-passthrough
**Branch**: `001-resourceid-passthrough`
**Generated**: 2025-12-26
**Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

## Summary

| Metric | Value |
|--------|-------|
| Total Tasks | 9 |
| Setup Phase | 2 tasks |
| US1 Tasks | 3 tasks |
| US2 Tasks | 3 tasks |
| Polish Phase | 1 task |
| Parallel Opportunities | 2 |
| Estimated Time | 30-45 minutes |

## User Story Mapping

| Story | Priority | Description | Tasks |
|-------|----------|-------------|-------|
| US1 | P1 | Resource Correlation in Batch Requests | T003, T004, T005 |
| US2 | P2 | Backward Compatibility with Tag-Based Correlation | T006, T007, T008 |

## Phase 1: Setup

**Goal**: Update dependencies and verify build environment.

- [x] T001 Update finfocus-spec to v0.4.11 in go.mod
- [x] T002 Run `go mod tidy` and verify build compiles with `go build -tags region_use1 ./cmd/finfocus-plugin-aws-public`

## Phase 2: User Story 1 - Resource Correlation (P1)

**Goal**: Pass through native `ResourceDescriptor.Id` to `Recommendation.Resource.Id`.

**Independent Test**: Send batch of resources with unique native IDs, verify each recommendation includes correct ID.

**Acceptance Criteria**:

- Resources with `Id` field populated have that ID in `Recommendation.Resource.Id`
- Multiple recommendations from same resource have identical `Resource.Id`

### Tasks

- [x] T003 [US1] Add `strings` import to internal/plugin/recommendations.go if not present
- [x] T004 [US1] Modify ID correlation logic in GetRecommendations (lines 112-123) to check `resource.Id` first with `strings.TrimSpace()` in internal/plugin/recommendations.go
- [x] T005 [US1] Add table-driven test `TestGetRecommendations_NativeIDPassthrough` with cases for native ID populated and multiple recommendations in internal/plugin/recommendations_test.go

## Phase 3: User Story 2 - Backward Compatibility (P2)

**Goal**: Maintain fallback to `tags["resource_id"]` when native ID is empty.

**Independent Test**: Send resources without native ID but with `resource_id` tag, verify tag value is used.

**Acceptance Criteria**:

- Empty native ID falls back to `tags["resource_id"]`
- Whitespace-only native ID falls back to tag
- Native ID takes priority when both present
- No ID available leaves `Resource.Id` empty

### Tasks

- [x] T006 [US2] Ensure fallback logic preserves existing `tags["resource_id"]` behavior when native ID is empty, and verify `tags["name"]` correlation is unchanged (FR-004) in internal/plugin/recommendations.go
- [x] T007 [P] [US2] Add table-driven test cases for tag fallback scenarios (empty ID, whitespace ID, neither present) in internal/plugin/recommendations_test.go
- [x] T008 [P] [US2] Add table-driven test case for priority (both native and tag present) in internal/plugin/recommendations_test.go

## Phase 4: Polish & Validation

**Goal**: Verify all tests pass and code meets quality standards.

- [x] T009 Run `make lint` and `make test` to verify all checks pass

## Dependencies

```text
T001 ──► T002 ──► T003 ──► T004 ──► T005
                    │
                    └──► T006 ──► T007 [P]
                              └──► T008 [P]
                                    │
                              T009 ◄┘
```

**Notes**:

- T001-T002 (Setup) must complete before any implementation
- T003-T004 must complete before T005 (tests need implementation)
- T006 can start after T004 (extends same code)
- T007 and T008 are parallelizable (independent test cases)
- T009 runs after all implementation and tests complete

## Parallel Execution Examples

### Phase 2-3 Parallelization

After T004 completes:

```bash
# Terminal 1: US1 test
# T005: Add native ID passthrough tests

# Terminal 2: US2 implementation
# T006: Add fallback logic (may already be present, just verify)
```

### Phase 3 Parallelization

After T006 completes:

```bash
# Terminal 1
# T007: Add tag fallback test cases

# Terminal 2
# T008: Add priority test cases
```

## Implementation Strategy

### MVP Scope (User Story 1 only)

For minimal viable implementation, complete:

- T001, T002 (Setup)
- T003, T004, T005 (US1)
- T009 (Validation)

This delivers core native ID passthrough. US2 (backward compatibility) can be deferred but is low-risk since existing tag logic is preserved.

### Recommended Order

1. **Setup** (T001-T002): 5 minutes
2. **US1 Implementation** (T003-T004): 10 minutes
3. **US1 Tests** (T005): 5 minutes
4. **US2 Verification** (T006): 5 minutes (likely already done by T004)
5. **US2 Tests** (T007-T008): 10 minutes (parallel)
6. **Validation** (T009): 5 minutes

### Code Change Reference

**File**: `internal/plugin/recommendations.go` (lines 112-123)

**Before**:

```go
// Populate correlation info: Use tags for correlation
for _, rec := range recs {
    if rec.Resource != nil {
        // Use resource_id tag if available for correlation
        if resourceID := resource.Tags["resource_id"]; resourceID != "" {
            rec.Resource.Id = resourceID
        }
```

**After**:

```go
// Populate correlation info: Native Id takes priority over tag
for _, rec := range recs {
    if rec.Resource != nil {
        // Priority 1: Use native Id field (FR-001, FR-002)
        if id := strings.TrimSpace(resource.Id); id != "" {
            rec.Resource.Id = id
        } else if resourceID := resource.Tags["resource_id"]; resourceID != "" {
            // Priority 2: Fall back to resource_id tag (FR-003)
            rec.Resource.Id = resourceID
        }
```

## Success Criteria Validation

| Criteria | Validated By |
|----------|--------------|
| SC-001: 100% ID passthrough | T005 tests |
| SC-002: Zero breaking changes | T007, T008 tests |
| SC-003: Existing tests pass | T009 validation |
| SC-004: 100% coverage of ID logic | T005, T007, T008 combined |
