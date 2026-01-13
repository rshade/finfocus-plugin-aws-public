# Contracts: Embed Raw AWS Pricing JSON Per Service

**Feature**: 018-raw-pricing-embed
**Date**: 2025-12-20

## No API Changes

This refactor is internal to the pricing data pipeline. The gRPC CostSourceService
interface remains unchanged:

- `Name()` - unchanged
- `Supports()` - unchanged
- `GetProjectedCost()` - unchanged
- `GetActualCost()` - unchanged
- `GetPricingSpec()` - unchanged

## Proto Definitions

The plugin continues to use proto definitions from:

```text
github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1
```

No changes to:

- `ResourceDescriptor`
- `GetProjectedCostRequest/Response`
- `SupportsRequest/Response`
- `ErrorCode` enum

## Internal Interface

The `PricingClient` interface in `internal/pricing/client.go` is also unchanged.
The refactor only affects how embedded data is organized and parsed, not the
public contract.
