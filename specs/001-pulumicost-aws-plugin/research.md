# Research: PulumiCost AWS Public Plugin

**Phase**: 0 - Technical Research
**Date**: 2025-11-16
**Status**: Complete - All Technical Context resolved

## Purpose

This document captures technical research for implementing the PulumiCost AWS Public Plugin. Since this is a greenfield project, research focuses on external dependencies, protocol requirements, and Go tooling patterns.

## External Dependencies Analysis

### 1. pulumicost-spec Proto Definitions

**Repository**: `github.com/rshade/pulumicost-spec`
**Package**: `github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1`

**Key Proto Messages** (from `proto/pulumicost/v1/costsource.proto`):

```protobuf
// CostSourceService - gRPC service interface
service CostSourceService {
  rpc Name(NameRequest) returns (NameResponse);
  rpc Supports(SupportsRequest) returns (SupportsResponse);
  rpc GetProjectedCost(GetProjectedCostRequest) returns (GetProjectedCostResponse);
  rpc GetActualCost(GetActualCostRequest) returns (GetActualCostResponse);
  rpc GetPricingSpec(GetPricingSpecRequest) returns (PricingSpec);
}

// ResourceDescriptor - Input for all resource-specific RPCs
message ResourceDescriptor {
  string provider = 1;       // "aws"
  string resource_type = 2;  // "ec2", "ebs", "s3", etc.
  string sku = 3;            // Instance type (t3.micro) or volume type (gp3)
  string region = 4;         // "us-east-1", "us-west-2", etc.
  map<string, string> tags = 5;  // Additional metadata (e.g., "size": "100")
}

// GetProjectedCostResponse - Primary cost estimate output
message GetProjectedCostResponse {
  double unit_price = 1;      // Per-unit rate (e.g., $/hour for EC2, $/GB-month for EBS)
  string currency = 2;        // "USD"
  double cost_per_month = 3;  // Estimated monthly cost
  string billing_detail = 4;  // Human-readable explanation of assumptions
}

// SupportsResponse - Capability advertising
message SupportsResponse {
  bool supported = 1;
  string reason = 2;  // Explanation (e.g., "Fully supported" or "Region mismatch")
}

// ErrorCode enum - Standard error codes
enum ErrorCode {
  ERROR_CODE_UNSPECIFIED = 0;
  ERROR_CODE_INVALID_RESOURCE = 6;
  ERROR_CODE_UNSUPPORTED_REGION = 9;
  ERROR_CODE_DATA_CORRUPTION = 11;
}
```

**Implementation Requirements**:
- Plugin MUST implement CostSourceService interface
- Use ResourceDescriptor for all inputs (no custom JSON types)
- Return proto-defined response messages only
- Use ErrorCode enum values (no custom error codes)

**Import Path**: `pbc "github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1"`

---

### 2. pulumicost-core/pkg/pluginsdk

**Repository**: `github.com/rshade/pulumicost-core`
**Package**: `github.com/rshade/pulumicost-core/pkg/pluginsdk`

**Key Functions** (from `pkg/pluginsdk/sdk.go`):

```go
// Serve starts the plugin gRPC server with lifecycle management
func Serve(ctx context.Context, config ServeConfig) error

type ServeConfig struct {
    Plugin Plugin  // Implementation of CostSourceService
    Port   int     // 0 = use PORT env or ephemeral
}

type Plugin interface {
    // Implements pulumicost.v1.CostSourceServiceServer
}
```

**Lifecycle Behavior**:
1. `Serve()` selects port (from PORT env or ephemeral)
2. Writes `PORT=<port>` to stdout (ONLY stdout output allowed)
3. Registers gRPC server on `127.0.0.1:<port>`
4. Blocks until context cancelled
5. Graceful shutdown on cancellation

**Integration Pattern**:
```go
func main() {
    // Initialize plugin
    pricingClient, _ := pricing.NewClient()
    plugin := plugin.NewAWSPublicPlugin(region, pricingClient)

    // Serve via pluginsdk
    ctx := context.Background()
    pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
        Plugin: plugin,
        Port:   0,
    })
}
```

---

### 3. gRPC and Protobuf Dependencies

**Packages**:
- `google.golang.org/grpc` - gRPC runtime
- `google.golang.org/protobuf` - Protobuf runtime
- `google.golang.org/grpc/status` - gRPC status errors
- `google.golang.org/grpc/codes` - gRPC error codes

**Error Handling Pattern**:
```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// Return gRPC error
return nil, status.Error(codes.InvalidArgument, "missing sku")

// Return gRPC error with details
st := status.New(codes.FailedPrecondition, "region mismatch")
// Add ErrorDetail with proto ErrorCode enum
return nil, st.Err()
```

---

## Go Language Patterns

### 1. Build Tags for Region-Specific Binaries

**Pattern**: Use build tags to select exactly one embedded pricing file at compile time.

**File Structure**:
```go
// internal/pricing/embed_use1.go
//go:build region_use1

package pricing

import _ "embed"

//go:embed ../../data/aws_pricing_us-east-1.json
var rawPricingJSON []byte
```

```go
// internal/pricing/embed_fallback.go
//go:build !region_use1 && !region_usw2 && !region_euw1

package pricing

// Dummy data for development
var rawPricingJSON = []byte(`{"region": "unknown", "ec2": {}, "ebs": {}}`)
```

**Build Command**:
```bash
go build -tags region_use1 -o pulumicost-plugin-aws-public-us-east-1 ./cmd/pulumicost-plugin-aws-public
```

---

### 2. Thread-Safe Initialization with sync.Once

**Pattern**: Parse embedded pricing data exactly once, safely for concurrent access.

```go
type Client struct {
    region   string
    currency string

    once sync.Once
    err  error

    ec2Index map[string]ec2OnDemandPrice
    ebsIndex map[string]ebsVolumePrice
}

func (c *Client) init() error {
    c.once.Do(func() {
        // Parse rawPricingJSON
        var data pricingData
        if err := json.Unmarshal(rawPricingJSON, &data); err != nil {
            c.err = err
            return
        }

        // Build indexes
        c.ec2Index = buildEC2Index(data.EC2)
        c.ebsIndex = buildEBSIndex(data.EBS)
    })
    return c.err
}

func (c *Client) EC2OnDemandPricePerHour(instanceType, os, tenancy string) (float64, bool) {
    if err := c.init(); err != nil {
        return 0, false
    }
    key := fmt.Sprintf("%s/%s/%s", instanceType, os, tenancy)
    price, found := c.ec2Index[key]
    return price.HourlyRate, found
}
```

**Why**: gRPC handlers may be called concurrently. `sync.Once` ensures initialization happens exactly once, even under concurrent load.

---

## Build Tooling

### 1. GoReleaser Configuration

**Purpose**: Build multiple region-specific binaries in a single release.

**Key Configuration** (`.goreleaser.yaml`):
```yaml
before:
  hooks:
    - go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data --dummy

builds:
  - id: us-east-1
    main: ./cmd/pulumicost-plugin-aws-public
    binary: pulumicost-plugin-aws-public-us-east-1
    tags:
      - region_use1
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]

  - id: us-west-2
    main: ./cmd/pulumicost-plugin-aws-public
    binary: pulumicost-plugin-aws-public-us-west-2
    tags:
      - region_usw2
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]

  - id: eu-west-1
    main: ./cmd/pulumicost-plugin-aws-public
    binary: pulumicost-plugin-aws-public-eu-west-1
    tags:
      - region_euw1
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]

archives:
  - id: archives
    format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}"
```

**Workflow**:
1. Before hook generates pricing data files
2. Each build ID targets one region with specific build tag
3. Produces 3 binaries × (linux/darwin/windows) × (amd64/arm64) = 18 artifacts per release

---

### 2. Pricing Data Generation Tool

**Purpose**: Fetch and trim AWS public pricing data at build time.

**Tool Location**: `tools/generate-pricing/main.go`

**CLI Interface**:
```bash
# Development mode with dummy data
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data --dummy

# Production mode (future - fetch real AWS pricing)
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data
```

**Output Format** (`data/aws_pricing_us-east-1.json`):
```json
{
  "region": "us-east-1",
  "currency": "USD",
  "ec2": {
    "t3.micro": {
      "instance_type": "t3.micro",
      "operating_system": "Linux",
      "tenancy": "Shared",
      "hourly_rate": 0.0104
    }
  },
  "ebs": {
    "gp3": {
      "volume_type": "gp3",
      "rate_per_gb_month": 0.08
    },
    "gp2": {
      "volume_type": "gp2",
      "rate_per_gb_month": 0.10
    }
  }
}
```

**Requirements**:
- Support `--dummy` flag for development without AWS API access
- Output one JSON file per region
- Keep file sizes small (<500KB per region)
- Include only on-demand pricing for EC2 (Linux/Shared) and EBS (standard volume types)

---

## Protocol Requirements Summary

### PORT Announcement

**Requirement**: Plugin MUST write `PORT=<port>` to stdout exactly once, then serve gRPC.

**Enforcement**: `pluginsdk.Serve()` handles this automatically. Plugin code MUST NOT write to stdout.

**Verification**:
```bash
./pulumicost-plugin-aws-public-us-east-1
# Expected output: PORT=12345
# Then plugin serves gRPC on 127.0.0.1:12345
```

---

### gRPC Error Responses

**Region Mismatch Example**:
```go
// When ResourceDescriptor.region != plugin.region
st := status.New(codes.FailedPrecondition, fmt.Sprintf(
    "Resource in region %s but plugin compiled for %s",
    rd.Region, p.region,
))
// TODO: Add ErrorDetail with proto ErrorCode enum and details map
return nil, st.Err()
```

**Invalid Input Example**:
```go
// When ResourceDescriptor lacks required fields
return nil, status.Error(codes.InvalidArgument, "missing sku (instance type)")
```

---

## Testing Strategy

### 1. Unit Tests

**Focus**: Pure pricing lookup logic and cost calculations.

**Pattern**: Mock `PricingClient` interface for isolation.

```go
type PricingClient interface {
    Region() string
    Currency() string
    EC2OnDemandPricePerHour(instanceType, os, tenancy string) (float64, bool)
    EBSPricePerGBMonth(volumeType string) (float64, bool)
}

// In tests
type mockPricingClient struct {
    region    string
    currency  string
    ec2Prices map[string]float64
    ebsPrices map[string]float64
}
```

**Test Coverage**:
- `internal/plugin/projected_test.go`: GetProjectedCost for EC2, EBS, stubs
- `internal/plugin/supports_test.go`: Supports logic for all resource types
- `internal/pricing/client_test.go`: Pricing data parsing and lookups

---

### 2. Integration Tests

**Focus**: gRPC service lifecycle and RPC calls.

**Approach**: Start plugin subprocess, read PORT, connect gRPC client.

**Example** (manual testing with grpcurl):
```bash
# Start plugin
./pulumicost-plugin-aws-public-us-east-1
# Output: PORT=12345

# Call GetProjectedCost
grpcurl -plaintext \
  -d '{
    "resource": {
      "provider": "aws",
      "resource_type": "ec2",
      "sku": "t3.micro",
      "region": "us-east-1"
    }
  }' \
  localhost:12345 \
  pulumicost.v1.CostSourceService/GetProjectedCost
```

---

## Open Questions / Decisions

### 1. AWS Pricing API Integration (Future)

**Decision Deferred**: v1 uses `--dummy` flag for development. Real AWS pricing API integration planned for v2.

**Implications**:
- Build-time tool currently generates minimal dummy data
- Production releases will need AWS API access in CI environment
- Need to document AWS pricing API endpoints and data structure

---

### 2. ErrorDetail Proto Message Usage

**Question**: How to attach ErrorDetail with ErrorCode enum to gRPC status errors?

**Research Needed**: Check pulumicost-spec for ErrorDetail proto definition and usage examples in pulumicost-core.

**Workaround for v1**: Use gRPC status.New() with standard codes, document proto ErrorCode in error message text.

---

## Conclusion

All Technical Context requirements from plan.md are resolved:
- ✅ Language/Version: Go 1.21+
- ✅ Primary Dependencies: pulumicost-spec (proto), pluginsdk, gRPC
- ✅ Testing: go test with table-driven tests, mockPricingClient
- ✅ Build Tooling: GoReleaser with build tags
- ✅ Protocol: gRPC CostSourceService with PORT announcement

**Next Phase**: Phase 1 - Design artifacts (data-model.md, contracts/, quickstart.md)
