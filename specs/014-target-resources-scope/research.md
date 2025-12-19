# Research: Support target_resources Scope

## Dependency Verification

**Status**: `github.com/rshade/pulumicost-spec` is already at `v0.4.9` in `go.mod`.
**Action**: No upgrade required. Will verify that the generated proto code in the vendor/module cache matches the expected fields (`TargetResources`).

## Implementation Approach

### Iteration vs Recursion
**Decision**: Iteration.
**Rationale**: The depth is flat (list of resources). Recursion adds unnecessary stack overhead.
**Logic**:
1. Normalize input:
   - If `TargetResources` populated -> use it.
   - Else if `Filter.Sku` + `Filter.ResourceType` -> create temp slice with 1 item.
   - Else -> return empty.
2. Iterate slice.
3. Apply `matchesFilter(resource, filter)` -> bool.
4. If match -> call internal cost estimator.
5. Collect results.

### Filtering Logic
**Decision**: "AND" logic.
**Rationale**: Explicit requirement. A resource must match ALL populated fields in `Filter` to be included.
**Fields to check**:
- `Region`
- `ResourceType`
- `Provider` (Implicitly must be "aws")
- `Sku` (If present in filter, though less likely in batch)

### Error Handling & Validation
**Decision**: Fail-fast on batch size > 100.
**Partial Failures**: Individual resources with unsupported types or missing pricing data are logged as warnings but do not fail the entire batch request (best effort).

### Correlation Strategy
**Decision**: Explicit mapping in `ResourceRecommendationInfo`.
**Rationale**: Essential for client to map async/batch results back to input resources.
**Priority**: `ResourceId` > `Arn` > `Name`.
- The plugin will inspect the input `ResourceDescriptor` and populate the output `ResourceRecommendationInfo` accordingly.

### Logging Strategy
**Decision**: Summary + Exception-only detail.
**Rationale**: Prevent log flooding with 100-item batches.
**Pattern**:
- Log 1 summary line at INFO level: "Processed batch: N resources, M recommendations, $X total savings".
- Log individual resource details at WARN/ERROR level only (e.g., "Failed to price resource X").

## Alternatives Considered

- **Parallel Processing**: Spawning goroutines for each item in batch.
  - **Rejected**: Batch size is small (max 100). Overhead of goroutine scheduling likely exceeds cost of sequential map lookup in embedded data. Simplicity (KISS) prevails.
- **Strict 1:1 Response Mapping**: Returning `nil` or error placeholders for filtered resources.
  - **Rejected**: Proto response is a list of valid recommendations. Correlation ID handles mapping; filtered resources are simply omitted (standard filter behavior).