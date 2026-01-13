# Research: SDK Migration and Code Consolidation

**Feature**: 013-sdk-migration
**Date**: 2025-12-16

## 1. SDK v0.4.8 API Analysis

### Environment Variable Helpers

**Decision**: Use `pluginsdk.GetPort()`, `pluginsdk.GetLogLevel()`, and related
helpers for all FinFocus environment variables.

**Rationale**: SDK provides type-safe accessors with consistent naming
conventions (`FINFOCUS_*` prefix) and fallback logic where appropriate.

**Alternatives considered**:

- Direct `os.Getenv()` calls - Rejected: inconsistent naming, no type safety
- Custom wrapper functions - Rejected: duplicates SDK functionality

**Key API Signatures**:

```go
func GetPort() int              // FINFOCUS_PLUGIN_PORT only
func GetLogLevel() string       // FINFOCUS_LOG_LEVEL, falls back to LOG_LEVEL
func GetLogFormat() string      // FINFOCUS_LOG_FORMAT
func GetTraceID() string        // FINFOCUS_TRACE_ID
func GetTestMode() bool         // FINFOCUS_TEST_MODE="true"
func IsTestMode() bool          // Silent version (no warnings)
```

**Important**: `GetPort()` does NOT fall back to `PORT` env var. The plugin's
current behavior using `PORT` fallback must be preserved locally if needed.

### Request Validation Helpers

**Decision**: Use `pluginsdk.ValidateProjectedCostRequest()` and
`pluginsdk.ValidateActualCostRequest()` for standard validation, wrap with
custom region check.

**Rationale**: SDK validators check nil requests, required fields, and time
ranges. Plugin-specific region validation must be added after SDK validation.

**Alternatives considered**:

- Keep existing inline validation - Rejected: 4x duplication, inconsistent
- SDK-only validation - Rejected: missing plugin-specific region check

**Key API Signatures**:

```go
func ValidateProjectedCostRequest(req *pbc.GetProjectedCostRequest) error
func ValidateActualCostRequest(req *pbc.GetActualCostRequest) error
```

**Error Types** (not gRPC status, plain errors):

```go
var ErrProjectedCostRequestNil = errors.New("request is required")
var ErrProjectedCostResourceNil = errors.New("resource is required")
var ErrProjectedCostProviderEmpty = errors.New("resource.provider is required")
var ErrProjectedCostResourceTypeEmpty = errors.New("resource.resource_type is required")
var ErrProjectedCostSkuEmpty = errors.New("resource.sku is required (use mapping helpers)")
var ErrProjectedCostRegionEmpty = errors.New("resource.region is required (use mapping helpers)")
```

**Usage Pattern**:

```go
if err := pluginsdk.ValidateProjectedCostRequest(req); err != nil {
    return nil, status.Error(codes.InvalidArgument, err.Error())
}
// Then add custom region check
```

### Property Mapping Package

**Decision**: Use `mapping.ExtractAWSSKU()` and `mapping.ExtractAWSRegion()` for
tag-based extraction where applicable.

**Rationale**: SDK mapping is optimized (<50ns/op), never panics, handles all
edge cases consistently.

**Alternatives considered**:

- Keep inline tag extraction - Rejected: verbose, inconsistent key handling
- Custom wrapper - Rejected: SDK already handles AWS-specific key priorities

**Key API Signatures**:

```go
func ExtractAWSSKU(properties map[string]string) string
// Priority: instanceType > instanceClass > type > volumeType

func ExtractAWSRegion(properties map[string]string) string
// Priority: region > availabilityZone (with AZ parsing)

func ExtractAWSRegionFromAZ(availabilityZone string) string
// "us-east-1a" -> "us-east-1"
```

**Limitation**: SDK mapping doesn't track "defaulted" state. Plugin must track
separately for billing_detail annotations.

### ARN Field Support

**Decision**: Implement ARN parsing in `internal/plugin/arn.go` using the new
`req.Arn` field (proto field 5).

**Rationale**: ARN contains partition, service, region, account, and resource
info - extractable without AWS API calls.

**Alternatives considered**:

- Use existing AWS SDK ARN parser - Rejected: adds large dependency
- Ignore ARN field - Rejected: loses valuable integration capability

**ARN Format**:

```text
arn:partition:service:region:account-id:resource-type/resource-id
arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0
```

**Service-to-ResourceType Mapping**:

| ARN Service | Pulumi ResourceType |
|-------------|---------------------|
| ec2 (instance) | aws:ec2/instance:Instance |
| ec2 (volume) | aws:ebs/volume:Volume |
| rds | aws:rds/instance:Instance |
| s3 | aws:s3/bucket:Bucket |
| lambda | aws:lambda/function:Function |
| dynamodb | aws:dynamodb/table:Table |
| eks | aws:eks/cluster:Cluster |

## 2. Duplicate Code Analysis

### EC2 Attribute Extraction (52 lines duplicated)

**Decision**: Create `internal/plugin/ec2_attrs.go` with unified extractors.

**Rationale**: Logic is identical except data source (protobuf Struct vs map).
Abstract the source, share the normalization.

**Design**:

```go
type EC2Attributes struct {
    OS      string // "Linux" or "Windows"
    Tenancy string // "Shared", "Dedicated", or "Host"
}

func DefaultEC2Attributes() EC2Attributes
func ExtractEC2AttributesFromTags(tags map[string]string) EC2Attributes
func ExtractEC2AttributesFromStruct(attrs *structpb.Struct) EC2Attributes
```

### RegionConfig Consolidation (5 lines + 37 validation)

**Decision**: Create `internal/regionsconfig/` package with Load() and
Validate().

**Rationale**: Struct is identical; validation only exists in generate-goreleaser.
Both tools should validate consistently.

**Design**:

```go
type RegionConfig struct {
    ID   string `yaml:"id"`
    Name string `yaml:"name"`
    Tag  string `yaml:"tag"`
}

type Config struct {
    Regions []RegionConfig `yaml:"regions"`
}

func Load(filename string) (*Config, error)
func LoadAndValidate(filename string) (*Config, error)
func Validate(regions []RegionConfig) error
```

### Validation Consolidation (100+ lines duplicated)

**Decision**: Create `internal/plugin/validation.go` with shared validation
helpers.

**Rationale**: Region mismatch handling (76 lines x 3 files) and required fields
check (24 lines x 4 files) are nearly identical.

**Design**:

```go
// ValidateRequest wraps SDK validation + custom region check
func (p *AWSPublicPlugin) ValidateProjectedCostRequest(
    ctx context.Context, req *pbc.GetProjectedCostRequest,
) (*pbc.ResourceDescriptor, error)

func (p *AWSPublicPlugin) ValidateActualCostRequest(
    ctx context.Context, req *pbc.GetActualCostRequest,
) (*pbc.ResourceDescriptor, error)

// RegionMismatchError creates standardized error with details
func (p *AWSPublicPlugin) RegionMismatchError(
    traceID, resourceRegion string,
) error
```

## 3. Implementation Decisions

### Port Fallback Handling

**Decision**: Keep local `PORT` fallback alongside `FINFOCUS_PLUGIN_PORT`.

**Rationale**: SDK's `GetPort()` only reads `FINFOCUS_PLUGIN_PORT`. For
backward compatibility, plugin should check both.

**Implementation**:

```go
port := pluginsdk.GetPort()
if port == 0 {
    if portStr := os.Getenv("PORT"); portStr != "" {
        port, _ = strconv.Atoi(portStr)
    }
}
```

### Default Tracking for billing_detail

**Decision**: Track defaults manually in estimation functions.

**Rationale**: SDK mapping doesn't track "defaulted" state. Plugin needs this
for billing_detail annotations like "(defaulted)".

**Implementation**: Continue using `sizeAssumed`, `engineDefaulted` bool flags
in estimation functions.

### Error Response Format

**Decision**: Convert SDK validation errors to gRPC status with ErrorDetail.

**Rationale**: Constitution requires ErrorCode enum from proto. SDK returns
plain errors; plugin must wrap them.

**Implementation**:

```go
if err := pluginsdk.ValidateProjectedCostRequest(req); err != nil {
    errDetail := &pbc.ErrorDetail{
        Code:    pbc.ErrorCode_ERROR_CODE_INVALID_RESOURCE,
        Message: err.Error(),
        Details: map[string]string{"trace_id": traceID},
    }
    st := status.New(codes.InvalidArgument, err.Error())
    stWithDetails, _ := st.WithDetails(errDetail)
    return nil, stWithDetails.Err()
}
```

## 4. Testing Strategy

### Unit Tests for New Helpers

| File | Test Focus |
|------|------------|
| ec2_attrs_test.go | Platform/tenancy normalization, defaults, edge cases |
| validation_test.go | SDK integration, region mismatch, error formatting |
| arn_test.go | All 7 service ARN formats, global services, invalid ARNs |
| regionsconfig/config_test.go | Load, validate, validation errors |

### Existing Tests

All existing tests in `internal/plugin/*_test.go` should pass without
modification - validation behavior is backward compatible.

### Integration Tests

Verify gRPC error format unchanged for finfocus-core compatibility:

- ERROR_CODE_INVALID_RESOURCE with trace_id
- ERROR_CODE_UNSUPPORTED_REGION with pluginRegion/requiredRegion

## 5. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| SDK validation differs from current | Low | Medium | Compare error messages in tests |
| Port fallback breaks E2E | Medium | High | Keep local PORT fallback |
| ARN parsing edge cases | Medium | Low | Comprehensive test coverage |
| RegionConfig validation too strict | Low | Medium | Match existing generate-goreleaser logic |
