# Quickstart: Cache Normalized Service Type

**Feature Branch**: `001-cache-service-type`
**Date**: 2026-01-22

## Overview

This feature is a pure internal optimization with no external API changes. The quickstart focuses on the implementation pattern for the new `serviceResolver` type.

## Usage Pattern

### Before (Current Code)

```go
func (p *AWSPublicPlugin) GetProjectedCost(ctx context.Context, req *pbc.GetProjectedCostRequest) (*pbc.GetProjectedCostResponse, error) {
    // ... validation calls detectService() 1-2 times ...

    resource := req.Resource

    // Third call to normalize and detect
    normalizedType := normalizeResourceType(resource.ResourceType)
    serviceType := detectService(normalizedType)

    switch serviceType {
    case "ec2":
        return p.estimateEC2(traceID, resource, req)
    // ...
    }
}
```

### After (Optimized Code)

```go
func (p *AWSPublicPlugin) GetProjectedCost(ctx context.Context, req *pbc.GetProjectedCostRequest) (*pbc.GetProjectedCostResponse, error) {
    resource := req.Resource

    // Create resolver once at request entry
    resolver := newServiceResolver(resource.ResourceType)

    // Validation uses resolver (caches on first access)
    if _, err := p.validateProjectedCostRequestWithResolver(ctx, req, resolver); err != nil {
        return nil, err
    }

    // Routing uses cached value (no recomputation)
    switch resolver.ServiceType() {
    case "ec2":
        return p.estimateEC2(traceID, resource, req)
    // ...
    }
}
```

## API Reference

### newServiceResolver

Creates a new service resolver for a resource type string.

```go
func newServiceResolver(resourceType string) *serviceResolver
```

**Parameters**:
- `resourceType`: The original resource_type from ResourceDescriptor (e.g., "ec2", "aws:eks/cluster:Cluster")

**Returns**: A resolver instance that caches normalization results

### serviceResolver.NormalizedType

Returns the normalized resource type (result of normalizeResourceType).

```go
func (r *serviceResolver) NormalizedType() string
```

**Behavior**:
- First call: Computes and caches the normalized type
- Subsequent calls: Returns cached value

### serviceResolver.ServiceType

Returns the detected service type (result of detectService).

```go
func (r *serviceResolver) ServiceType() string
```

**Behavior**:
- First call: Computes NormalizedType if needed, then detectService, caches result
- Subsequent calls: Returns cached value

## Testing

### Unit Tests

```bash
# Run service resolver unit tests
go test -v ./internal/plugin/... -run TestServiceResolver
```

### Benchmark Tests

```bash
# Baseline (before implementation)
go test -tags=region_use1 -bench=BenchmarkGetProjectedCost -benchmem ./internal/plugin/...

# After implementation - compare results
go test -tags=region_use1 -bench=BenchmarkGetProjectedCost -benchmem ./internal/plugin/...
```

### Race Detection

```bash
# Verify no data races
go test -tags=region_use1 -race ./internal/plugin/...
```

## Verification Checklist

- [ ] All existing tests pass: `make test`
- [ ] Linting passes: `make lint`
- [ ] No race conditions: `go test -race ./internal/plugin/...`
- [ ] Benchmark shows reduced detectService calls
- [ ] Memory overhead < 100 bytes per resource

## Notes

- **No gRPC API changes**: This is purely internal optimization
- **No contracts directory**: No new external interfaces introduced
- **Backward compatible**: All existing behavior preserved
