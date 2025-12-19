# Research: Lambda Cost Estimation

**Feature**: 014-lambda-cost-estimation
**Date**: 2025-12-18

## Technical Decisions

### 1. Pricing Client Interface Extension

- **Decision**: Extend `PricingClient` interface with `LambdaPricePerRequest()`
  and `LambdaPricePerGBSecond()` methods.
- **Rationale**: Follows the existing pattern used for EC2, EBS, and other
  services. Keeps the client interface strongly typed and service-specific.
- **Alternatives Considered**:
  - Generic `GetPrice(service, type)` method: Rejected as it loses type safety
    and discoverability.
  - Separate `LambdaPricingClient`: Rejected as it adds unnecessary complexity;
    the single client is designed to handle all supported services for a region.

### 2. Pricing Data Structure

- **Decision**: Add `lambdaPricing *lambdaPrice` field to the main `Client`
  struct, populated at initialization.
- **Rationale**: Lambda pricing is uniform per region (unlike EC2 which depends
  on instance type). A single struct lookup is O(1) and extremely efficient.
- **Alternatives Considered**:
  - Map-based lookup: Unnecessary since there are no variable keys (like
    instance size) for the base rates.

### 3. Estimator Logic Location

- **Decision**: Implement `estimateLambda` as a private method on
  `AWSPublicPlugin` in `internal/plugin/projected.go`.
- **Rationale**: Consistent with `estimateEC2` and `estimateEBS`. Keeps all
  estimation logic for the plugin in one place.

### 4. Memory Parsing Strategy

- **Decision**: Parse `resource.Sku` as a raw numeric string (e.g., "128",
  "1024").
- **Rationale**: Pulumi's AWS provider typically exposes memory size as an
  integer. Parsing this directly is robust. Default to 128MB on failure to
  provide a "safe" lower bound estimate (or $0 if preferred, but 128MB is the
  minimum allocatable).

### 5. Provisioned Concurrency

- **Decision**: Explicitly out of scope for this iteration.
- **Rationale**: Adds significant complexity (different pricing dimensions).
  Core value is in standard Request + Duration pricing. Can be added later as a
  separate feature.

## Open Questions Resolved

- **Memory SKU Format**: Confirmed as raw numeric string (MB).
- **Unsupported Region**: Confirmed to return $0 with "Region not supported"
  message.
- **Architecture**: Confirmed to use x86 default if not specified, or use
  resource tags/properties.
