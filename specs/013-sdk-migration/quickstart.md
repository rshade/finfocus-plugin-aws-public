# Quickstart: SDK Migration and Code Consolidation

**Feature**: 013-sdk-migration
**Date**: 2025-12-16

## Prerequisites

- Go 1.25+
- finfocus-spec v0.4.8 in go.mod
- Existing tests passing (`make test`)

## Implementation Order

### Phase 1: Internal Consolidation (No SDK Dependencies)

1. **EC2 Attribute Helper** (`internal/plugin/ec2_attrs.go`)
2. **RegionConfig Package** (`internal/regionsconfig/`)
3. Update callers in estimate.go, projected.go
4. Update tools to use shared package

### Phase 2: SDK Integration

5. **Validation Helper** (`internal/plugin/validation.go`)
6. **Environment Variable Migration** (plugin.go, main.go)
7. Update GetProjectedCost, GetActualCost, GetPricingSpec

### Phase 3: ARN Support

8. **ARN Parser** (`internal/plugin/arn.go`)
9. Integrate into GetActualCost flow

## Code Patterns

### EC2 Attributes Usage

```go
// Before (in projected.go)
os := "Linux"
tenancy := "Shared"
if resource.Tags != nil {
    if platform, ok := resource.Tags["platform"]; ok {
        // ... 15 lines of normalization
    }
}

// After
attrs := ExtractEC2AttributesFromTags(resource.Tags)
// attrs.OS is "Linux" or "Windows"
// attrs.Tenancy is "Shared", "Dedicated", or "Host"
```

### Validation Usage

```go
// Before (76 lines repeated 3x)
if resource.Region != p.region {
    details := map[string]string{...}
    errDetail := &pbc.ErrorDetail{...}
    st := status.New(codes.FailedPrecondition, ...)
    // ... 20+ lines
}

// After
resource, err := p.validateProjectedCostRequest(ctx, req)
if err != nil {
    return nil, err  // Already formatted with trace_id
}
```

### ARN Parsing Usage

```go
// In GetActualCost
if req.Arn != "" {
    components, err := ParseARN(req.Arn)
    if err == nil {
        resource := &pbc.ResourceDescriptor{
            Provider:     "aws",
            ResourceType: components.ToPulumiResourceType(),
            Region:       components.Region,
            Sku:          req.Tags["sku"],  // ARN doesn't have SKU
            Tags:         req.Tags,
        }
        // Use resource for estimation
    }
    // Fall through to JSON/Tags parsing on error
}
```

### Environment Variables Usage

```go
// Before
port := os.Getenv("PORT")

// After
port := pluginsdk.GetPort()
if port == 0 {
    // Backward compatibility: check PORT
    if portStr := os.Getenv("PORT"); portStr != "" {
        port, _ = strconv.Atoi(portStr)
    }
}
```

## Testing Checklist

- [ ] `go test ./internal/plugin/... -run TestEC2Attributes`
- [ ] `go test ./internal/regionsconfig/...`
- [ ] `go test ./internal/plugin/... -run TestValidation`
- [ ] `go test ./internal/plugin/... -run TestARN`
- [ ] `make test` (all existing tests pass)
- [ ] `make lint` (no new warnings)

## Verification Steps

1. **EC2 Attributes**:

   ```bash
   go test ./internal/plugin -run TestExtractEC2 -v
   ```

2. **RegionConfig**:

   ```bash
   go test ./internal/regionsconfig -v
   make generate-embeds     # Should work
   make generate-goreleaser # Should work
   ```

3. **Validation**:

   ```bash
   go test ./internal/plugin -run TestValidate -v
   # Verify error format matches existing behavior
   ```

4. **ARN Parsing**:

   ```bash
   go test ./internal/plugin -run TestARN -v
   go test ./internal/plugin -run TestParseARN -v
   ```

5. **Full Integration**:

   ```bash
   make test
   make lint
   goreleaser build --snapshot --clean
   ```

## Common Issues

### Issue: SDK validation error messages differ

**Solution**: Wrap SDK errors to preserve existing message format:

```go
if err := pluginsdk.ValidateProjectedCostRequest(req); err != nil {
    // Map to our existing error messages if needed
    msg := mapSDKErrorToLegacyMessage(err)
    return nil, p.newErrorWithID(traceID, codes.InvalidArgument, msg, ...)
}
```

### Issue: PORT env var not recognized

**Solution**: Keep local fallback (SDK only reads FINFOCUS_PLUGIN_PORT):

```go
port := pluginsdk.GetPort()
if port == 0 {
    port, _ = strconv.Atoi(os.Getenv("PORT"))
}
```

### Issue: ARN region empty for S3

**Solution**: Fall back to tags or default:

```go
region := components.Region
if region == "" && components.Service == "s3" {
    region = req.Tags["region"]
    if region == "" {
        region = "us-east-1"  // Default for global services
    }
}
```

## Files Changed Summary

| File | Change Type | Lines Changed (est.) |
|------|-------------|---------------------|
| internal/plugin/ec2_attrs.go | NEW | +80 |
| internal/plugin/ec2_attrs_test.go | NEW | +100 |
| internal/plugin/validation.go | NEW | +120 |
| internal/plugin/validation_test.go | NEW | +150 |
| internal/plugin/arn.go | NEW | +150 |
| internal/plugin/arn_test.go | NEW | +200 |
| internal/regionsconfig/config.go | NEW | +80 |
| internal/regionsconfig/config_test.go | NEW | +100 |
| internal/plugin/estimate.go | MODIFY | -30 |
| internal/plugin/projected.go | MODIFY | -50 |
| internal/plugin/actual.go | MODIFY | -30, +20 |
| internal/plugin/pricingspec.go | MODIFY | -20 |
| internal/plugin/plugin.go | MODIFY | -10, +5 |
| tools/generate-embeds/main.go | MODIFY | -10, +5 |
| tools/generate-goreleaser/main.go | MODIFY | -40, +5 |
| cmd/.../main.go | MODIFY | ~0 (pluginsdk.Serve handles) |

**Net Change**: ~+700 new lines, ~-190 removed = ~+510 lines

**Duplication Reduction**: ~210 lines eliminated through consolidation
