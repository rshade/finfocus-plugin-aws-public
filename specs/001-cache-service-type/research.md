# Research: Cache Normalized Service Type

**Feature Branch**: `001-cache-service-type`
**Date**: 2026-01-22

## Research Tasks

### 1. Caching Strategy Selection

**Question**: What is the best approach for caching the normalized service type within a request lifecycle?

**Options Evaluated**:

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| A. Context-based caching | Store cached value in Go context.Context | Zero allocation after first call; automatic cleanup | Complex key management; requires context threading through all call sites |
| B. Memoized wrapper struct | Struct that caches normalizeResourceType() + detectService() results | Simple, explicit; no context required; clear ownership | ~32 bytes allocation per resource |
| C. Request-scoped map | Map[string]string passed through call chain | Can cache multiple resource types efficiently | Requires explicit passing; risk of cross-request leakage |
| D. Sync.Map global cache | Global thread-safe map with resource_type as key | Zero allocation for repeated types across requests | Memory growth over time; requires eviction policy |

**Decision**: Option B - Memoized wrapper struct

**Rationale**:
1. **Simplicity**: A struct with two fields (normalizedType, serviceType) is trivial to understand and maintain
2. **No Context Threading**: Doesn't require modifying function signatures to accept context
3. **Request Isolation**: Each request creates its own wrapper; no cross-request contamination possible
4. **Memory Bounded**: Memory is proportional to active requests, not all historical resource types
5. **Fits Constitution**: Aligns with "Explicit is better than implicit" principle

**Alternatives Rejected**:
- Context-based (A): Adds complexity without significant benefit for single-resource requests
- Request-scoped map (C): Over-engineered for the common case (1 resource per request)
- Global cache (D): Risk of memory leaks; requires additional eviction logic

### 2. Thread Safety Analysis

**Question**: Does the memoized wrapper require synchronization?

**Analysis**:
- Each gRPC request creates a new wrapper instance (not shared)
- Wrapper is populated once, then read-only
- No concurrent writes to the same wrapper instance

**Decision**: No synchronization required

**Rationale**: The wrapper is request-scoped. Each gRPC handler creates its own instance at the start of the request, populates it once (either lazily or eagerly), and uses it throughout the request lifecycle. Since gRPC handlers are concurrent but each request has its own wrapper, there's no data race.

### 3. Call Site Inventory

**Question**: Which call sites need modification?

**Analysis from grep across internal/plugin/**:

| File | Line | Context | Action |
|------|------|---------|--------|
| projected.go:160 | `serviceType := detectService(normalizedType)` | GetProjectedCost routing | Replace with wrapper |
| supports.go:36 | `normalizedType := detectService(normalizedResourceType)` | Supports() check | Replace with wrapper |
| validation.go:39 | `service := detectService(normalized)` | isZeroCostResource | Replace with wrapper |
| validation.go:99 | `service := detectService(normalizedResourceType)` | ValidateProjectedCostRequest | Replace with wrapper |
| validation.go:183 | `service := detectService(normalizedResourceType)` | ValidateActualCostRequest ARN | Replace with wrapper |
| validation.go:215 | `service := detectService(normalizedResourceType)` | ValidateActualCostRequest tags | Replace with wrapper |
| actual.go:236 | `serviceType := detectService(normalizedType)` | GetActualCost routing | Replace with wrapper |
| pricingspec.go:35 | `serviceType := detectService(normalizedResourceType)` | GetPricingSpec routing | Replace with wrapper |
| recommendations.go:142 | `service := detectService(resource.ResourceType)` | Batch processing | Replace with wrapper per resource |
| plugin.go:325 | `serviceType := detectService(normalizedType)` | Metadata extraction | Replace with wrapper |

**Decision**: Create wrapper at request entry point (e.g., GetProjectedCost, GetActualCost, Supports) and pass to internal functions. For batch requests (GetRecommendations), create wrapper per resource in the loop.

### 4. Benchmark Baseline

**Question**: What is the current performance baseline?

**Approach**: Before implementation, run benchmarks to establish baseline.

```bash
# Proposed benchmark commands
go test -tags=region_use1 -bench=BenchmarkGetProjectedCost -benchmem ./internal/plugin/...
go test -tags=region_use1 -bench=BenchmarkGetRecommendations -benchmem ./internal/plugin/...
```

**Expected Metrics**:
- Current: 2-3 detectService calls per single request
- After: 1 detectService call per single request
- Expected improvement: ~50-66% reduction in string parsing overhead

**Decision**: Add new benchmark tests as part of implementation to measure improvement.

### 5. API Design

**Question**: What should the wrapper interface look like?

**Proposed Design**:

```go
// serviceResolver caches the normalized resource type and detected service
// for a single ResourceDescriptor within one request lifecycle.
//
// Usage:
//   resolver := newServiceResolver(resource.ResourceType)
//   normalized := resolver.NormalizedType()  // cached
//   service := resolver.ServiceType()         // cached
type serviceResolver struct {
    original       string
    normalizedType string  // result of normalizeResourceType()
    serviceType    string  // result of detectService()
    initialized    bool
}

func newServiceResolver(resourceType string) *serviceResolver
func (r *serviceResolver) NormalizedType() string
func (r *serviceResolver) ServiceType() string
```

**Decision**: Simple struct with lazy initialization. The `initialized` flag ensures computation happens exactly once.

**Alternative Considered**: Eager initialization in constructor. Rejected because some code paths (e.g., early validation failures) may not need the service type.

## Summary

| Topic | Decision | Confidence |
|-------|----------|------------|
| Caching strategy | Memoized wrapper struct | High |
| Thread safety | No sync required (request-scoped) | High |
| Call site count | ~10 sites across 7 files | High |
| API design | serviceResolver struct with lazy init | High |
| Performance measurement | Add benchmarks before/after | High |

## Next Steps

1. Create `service_cache.go` with `serviceResolver` type
2. Add benchmark tests for baseline measurement
3. Refactor call sites to use resolver
4. Verify all tests pass with `go test -race`
5. Compare benchmark results
