# Research: Pre-allocate Map Capacity for Pricing Indexes

**Branch**: `021-map-prealloc` | **Date**: 2026-01-04

## Status

No NEEDS CLARIFICATION items identified. This is a well-understood Go optimization pattern.

## Decisions

### 1. Map Capacity Values

**Decision**: Use static capacity hints based on observed AWS pricing data volumes.

| Map | Capacity | Rationale |
|-----|----------|-----------|
| `ec2Index` | 100,000 | ~90k EC2 products in us-east-1; 10% buffer |
| `ebsIndex` | 50 | ~20-30 volume types; 2x buffer |
| `s3Index` | 100 | ~50-100 storage classes |
| `rdsInstanceIndex` | 5,000 | Instance type × engine combinations |
| `rdsStorageIndex` | 100 | Storage types across engines |
| `elasticacheIndex` | 1,000 | Node type × engine (Redis/Memcached/Valkey) |

**Alternatives Considered**:

1. **Dynamic sizing via data inspection**: Parse JSON first to count entries, then allocate.
   - Rejected: Adds complexity and parsing overhead; minimal benefit over static estimates.

2. **Two-pass parsing**: First pass counts products, second pass populates.
   - Rejected: Doubles initialization time; contradicts performance goal.

3. **Grow-only with large initial capacity**: Use maximum observed capacity across all regions.
   - Rejected: Over-allocates for smaller regions; us-east-1 values are sufficient.

### 2. Benchmark Validation Approach

**Decision**: Use existing `BenchmarkNewClient` with `benchstat` for statistical comparison.

**Rationale**: The benchmark already tracks `ns/op`, `B/op`, and `allocs/op`. Adding `benchstat` provides statistical significance testing without new test code.

**Validation workflow**:
1. Capture baseline on `main` branch (10 runs)
2. Capture results on feature branch (10 runs)
3. Use `benchstat` to compare with p-value significance

### 3. CI Integration for Baseline Comparison

**Decision**: Capture baseline in CI workflow and compare against PR branch.

**Implementation approach** (from clarification):
- Run benchmarks on `main` branch as baseline
- Run benchmarks on PR branch
- Report results in PR comments for objective validation

**Alternatives Considered**:

1. **Manual local comparison**: Developer runs and reports results.
   - Rejected: "Works on my machine" risk; not reproducible.

2. **Hardcoded threshold tests**: Fail if allocs exceed X.
   - Rejected: Brittle; thresholds need updating as codebase evolves.

## Go Map Internals Reference

For context on why pre-allocation helps:

- Go maps use hash buckets that grow by doubling when load factor exceeds ~6.5 items/bucket
- Without capacity hint: 0 → 1 → 2 → 4 → 8 → ... → 131072 buckets (for 90k items)
- With capacity hint: Single allocation to ~100k buckets
- Each growth operation allocates new bucket array and rehashes all entries

This is a standard Go optimization pattern documented in:
- [Effective Go - Maps](https://go.dev/doc/effective_go#maps)
- Go runtime source: `runtime/map.go`
