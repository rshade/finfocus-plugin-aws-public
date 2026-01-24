# Data Model: Cache Normalized Service Type

**Feature Branch**: `001-cache-service-type`
**Date**: 2026-01-22

## Entities

### serviceResolver (NEW)

A lightweight memoization wrapper that caches the results of `normalizeResourceType()` and `detectService()` for a single resource type string.

| Field | Type | Description | Validation |
|-------|------|-------------|------------|
| original | string | The original resource_type string from ResourceDescriptor | Immutable after construction |
| normalizedType | string | Result of normalizeResourceType(original) | Computed lazily on first access |
| serviceType | string | Result of detectService(normalizedType) | Computed lazily on first access |
| initialized | bool | Whether computation has occurred | Internal state flag |

**Lifecycle**:

```text
┌─────────────────┐    newServiceResolver()    ┌─────────────────┐
│   Uninitialized │ ──────────────────────────▶│   Created       │
│   (no instance) │                            │   initialized=F │
└─────────────────┘                            └────────┬────────┘
                                                        │
                                               First access to
                                               NormalizedType() or
                                               ServiceType()
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │   Initialized   │
                                               │   initialized=T │
                                               └─────────────────┘
```

**Relationships**:
- Created by: gRPC handler methods (GetProjectedCost, GetActualCost, Supports, etc.)
- Uses: normalizeResourceType(), detectService() (existing functions)
- Used by: Validation functions, cost routing logic

### Existing Entities (No Changes)

| Entity | Role in Feature | Changes |
|--------|-----------------|---------|
| ResourceDescriptor | Source of resource_type string | None |
| AWSPublicPlugin | Creates serviceResolver per request | Minor: instantiate resolver at request start |

## State Transitions

The serviceResolver has a simple two-state lifecycle:

| State | Trigger | Next State | Actions |
|-------|---------|------------|---------|
| Uninitialized | First call to NormalizedType() or ServiceType() | Initialized | Compute normalizedType, serviceType; set initialized=true |
| Initialized | Any subsequent access | Initialized | Return cached values (no recomputation) |

**Invariants**:
- Once `initialized=true`, values are immutable
- No state can transition back to Uninitialized
- Thread-safe for concurrent reads (but designed for single-goroutine use per request)

## Validation Rules

| Rule | Scope | Description |
|------|-------|-------------|
| V-001 | serviceResolver | Original string is stored as-is (no validation at construction) |
| V-002 | serviceResolver | Empty original produces empty normalizedType and serviceType (matches existing behavior) |
| V-003 | serviceResolver | Malformed resource types produce predictable outputs (no panic) |

## Memory Profile

| Scenario | Memory per Resource | Notes |
|----------|---------------------|-------|
| serviceResolver struct | ~64 bytes | 3 strings (16 bytes each header) + bool + padding |
| Single request (1 resource) | ~64 bytes | One resolver instance |
| Batch request (100 resources) | ~6.4 KB | 100 resolver instances (acceptable per SC-005: <100 bytes each) |

## Package Placement

```text
internal/plugin/
├── service_cache.go      # NEW: serviceResolver type and methods
├── service_cache_test.go # NEW: Unit tests for serviceResolver
└── ... (existing files)
```

**Rationale**: Placed in `internal/plugin` alongside the functions it wraps (normalizeResourceType, detectService). No new packages needed.
