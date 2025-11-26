# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `pulumicost-plugin-aws-public`, a fallback PulumiCost plugin that estimates AWS resource costs using public AWS on-demand pricing data, without requiring CUR/Cost Explorer/Vantage data access. The plugin implements the gRPC CostSourceService protocol and is invoked as a separate process by PulumiCost core.

## Architecture

### Plugin Protocol (gRPC)
- The plugin implements **CostSourceService** from `pulumicost.v1` proto (see `../pulumicost-spec/proto/pulumicost/v1/costsource.proto`)
- **Not** stdin/stdout JSON - uses gRPC with PORT announcement
- On startup: plugin writes `PORT=<port>` to stdout, then serves gRPC on 127.0.0.1
- Core connects via gRPC and calls methods like `GetProjectedCost()`, `Supports()`, `Name()`
- **One resource per RPC call** (not batch processing)
- Uses **pluginsdk.Serve()** from `pulumicost-core/pkg/pluginsdk` for lifecycle management
- Graceful shutdown on context cancellation

### Required gRPC Methods
- **Name()**: Returns plugin name "aws-public"
- **Supports(ResourceDescriptor)**: Checks if plugin supports a resource type/region
- **GetProjectedCost(ResourceDescriptor)**: Returns cost estimate for **one** resource
- **GetPricingSpec(ResourceDescriptor)** (optional): Returns detailed pricing info

### ResourceDescriptor (Proto Input)
```protobuf
message ResourceDescriptor {
  string provider = 1;       // "aws"
  string resource_type = 2;  // "ec2", "ebs", "s3", etc.
  string sku = 3;            // instance type (e.g., "t3.micro") or volume type (e.g., "gp3")
  string region = 4;         // "us-east-1", "us-west-2", etc.
  map<string, string> tags = 5;  // For EBS, may contain "size" or "volume_size"
}
```

### GetProjectedCostResponse (Proto Output)
```protobuf
message GetProjectedCostResponse {
  double unit_price = 1;       // Hourly rate for EC2, GB-month rate for EBS
  string currency = 2;         // "USD"
  double cost_per_month = 3;   // Total monthly cost (730 hours for EC2)
  string billing_detail = 4;   // Human-readable assumptions (e.g., "On-demand Linux, shared tenancy, 730 hrs/month")
}
```

### Error Handling
- Uses **ErrorCode enum** from proto (not custom codes)
- Key error codes:
  - `ERROR_CODE_UNSUPPORTED_REGION`: Resource region doesn't match plugin binary region
  - `ERROR_CODE_INVALID_RESOURCE`: Missing required ResourceDescriptor fields
  - `ERROR_CODE_DATA_CORRUPTION`: Embedded pricing data is corrupt
- ERROR_CODE_UNSUPPORTED_REGION includes `ErrorDetail.details` map with:
  - `pluginRegion`: The region this binary supports
  - `requiredRegion`: The region the resource needs

### Region-Specific Binaries
- **One binary per AWS region** using GoReleaser with build tags
- Binary naming: `pulumicost-plugin-aws-public-<region>` (e.g., `pulumicost-plugin-aws-public-us-east-1`)
- Each binary embeds only its region's pricing data via `//go:embed`
- Build tag mapping:
  - `us-east-1` → `region_use1`
  - `us-west-2` → `region_usw2`
  - `eu-west-1` → `region_euw1`
  - `ap-southeast-1` → `region_apse1` (Singapore)
  - `ap-southeast-2` → `region_apse2` (Sydney)
  - `ap-northeast-1` → `region_apne1` (Tokyo)
  - `ap-south-1` → `region_aps1` (Mumbai)

### Embedded Pricing Data
- At build time: `tools/generate-pricing` fetches/trims AWS public pricing
- Output: `data/aws_pricing_<region>.json` files
- These files are embedded into binaries using `//go:embed` in region-specific files under `internal/pricing/`
- The pricing client parses embedded JSON once using `sync.Once` and builds lookup indexes
- Must be thread-safe for concurrent gRPC calls

### Service Support (v1)
- **Fully implemented**: EC2 instances, EBS volumes
- **Stubbed/Partial**: S3, Lambda, RDS, DynamoDB
  - Supports() returns `supported=true` with reason "Limited support - returns $0 estimate"
  - GetProjectedCost() returns `cost_per_month=0` with billing_detail explaining not implemented

## Directory Structure

```
cmd/
  pulumicost-plugin-aws-public/     # gRPC service entrypoint
    main.go                          # Calls pluginsdk.Serve()
internal/
  plugin/
    plugin.go        # Implements Plugin interface from pluginsdk
    supports.go      # Supports() logic for resource type + region checks
    projected.go     # GetProjectedCost() logic for EC2/EBS/stubs
    pricingspec.go   # Optional GetPricingSpec() logic
  pricing/
    client.go        # Pricing client with thread-safe lookup methods
    embed_*.go       # Region-specific embedded pricing (build-tagged)
  config/
    config.go        # Configuration (currency, discount factor)
tools/
  generate-pricing/
    main.go          # Build-time tool to fetch/trim AWS pricing
data/
  aws_pricing_*.json  # Generated pricing files (not in git)
```

## Common Commands

### Building
```bash
# Standard build (no region tags, uses default fallback)
go build ./...

# Build with specific region tag
go build -tags region_use1 -o pulumicost-plugin-aws-public-us-east-1 ./cmd/pulumicost-plugin-aws-public

# Build all region binaries using GoReleaser
goreleaser build --snapshot --clean
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./internal/plugin
go test ./internal/pricing

# Test gRPC service manually (requires grpcurl or similar)
# 1. Start plugin: ./pulumicost-plugin-aws-public-us-east-1
# 2. Capture PORT from stdout
# 3. Call RPCs: grpcurl -plaintext -d '{"resource": {...}}' localhost:<port> pulumicost.v1.CostSourceService/GetProjectedCost
```

### Generating Pricing Data
```bash
# Generate dummy pricing data for development
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data --dummy

# (Future) Real AWS pricing fetch
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data
```

### Running the Plugin
```bash
# Start the plugin (it will announce its PORT)
./pulumicost-plugin-aws-public-us-east-1
# Output: PORT=12345
# Then serves gRPC on 127.0.0.1:12345

# With PORT env variable
PORT=9000 ./pulumicost-plugin-aws-public-us-east-1
# Output: PORT=9000
```

## Key Proto Types

### ResourceDescriptor
From `pulumicost.v1.ResourceDescriptor`:
- `provider`: "aws"
- `resource_type`: "ec2", "ebs", "s3", "lambda", "rds", "dynamodb"
- `sku`: Instance type for EC2 (e.g., "t3.micro"), volume type for EBS (e.g., "gp3")
- `region`: AWS region (e.g., "us-east-1")
- `tags`: Key-value pairs; for EBS, may contain "size" or "volume_size"

### ErrorCode Enum
From `pulumicost.v1.ErrorCode`:
- `ERROR_CODE_UNSUPPORTED_REGION` (9): Region not supported by this binary
- `ERROR_CODE_INVALID_RESOURCE` (6): Missing required fields in ResourceDescriptor
- `ERROR_CODE_DATA_CORRUPTION` (11): Embedded pricing data is corrupt
- See full list in `../pulumicost-spec/proto/pulumicost/v1/costsource.proto`

## Estimation Logic

### EC2 Instances
- `resource_type`: "ec2"
- `sku`: Instance type (e.g., "t3.micro", "m5.large")
- Assumptions (hardcoded for v1):
  - `operatingSystem = "Linux"`
  - `tenancy = "Shared"`
  - `hoursPerMonth = 730` (24×7 on-demand)
- `unit_price`: Hourly rate from pricing data
- `cost_per_month`: unit_price × 730
- `billing_detail`: "On-demand Linux, shared tenancy, 730 hrs/month"

### EBS Volumes
- `resource_type`: "ebs"
- `sku`: Volume type (e.g., "gp2", "gp3", "io1", "io2")
- Size: Read from `tags["size"]` or `tags["volume_size"]`, default to 8 GB if missing
- `unit_price`: Rate per GB-month from pricing data
- `cost_per_month`: unit_price × size_GB
- `billing_detail`: "EBS <sku> storage, <size>GB" (+ ", defaulted to 8GB" if size not specified)

### Stub Services (S3, Lambda, RDS, DynamoDB)
- `resource_type`: "s3", "lambda", "rds", "dynamodb"
- Supports() returns `supported=true` with `reason="Limited support - returns $0 estimate"`
- GetProjectedCost() returns:
  - `unit_price=0`
  - `cost_per_month=0`
  - `currency="USD"`
  - `billing_detail="<Service> cost estimation not implemented - returning $0"`

### Region Mismatch Handling
- Supports() checks if ResourceDescriptor.region matches plugin's embedded region
- If mismatch: returns `supported=false` with `reason="Region not supported by this binary"`
- GetProjectedCost() for mismatched region: returns gRPC error with ERROR_CODE_UNSUPPORTED_REGION and details map

## Development Notes

### Implementing the Plugin Interface
From `pulumicost-core/pkg/pluginsdk`:
```go
type Plugin interface {
    Name() string
    GetProjectedCost(ctx context.Context, req *pbc.GetProjectedCostRequest) (*pbc.GetProjectedCostResponse, error)
    GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error)
}
```

Your plugin struct should implement this interface. For aws-public:
- `Name()` returns "aws-public"
- `GetProjectedCost()` implements EC2/EBS/stub logic
- `GetActualCost()` returns an error (not applicable for public pricing)

Additionally, implement Supports() via the gRPC server (not in Plugin interface but in the service):
```go
func (s *Server) Supports(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error)
```

### Using pluginsdk.Serve()
In `cmd/pulumicost-plugin-aws-public/main.go`:
```go
import (
    "context"
    "github.com/rshade/pulumicost-core/pkg/pluginsdk"
    "github.com/rshade/pulumicost-plugin-aws-public/internal/plugin"
)

func main() {
    ctx := context.Background()
    p := plugin.NewAWSPublicPlugin()  // Your implementation
    err := pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
        Plugin: p,
        Port:   0,  // 0 = use PORT env or ephemeral
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### Adding New AWS Services
1. Update `internal/plugin/supports.go` to include the new resource_type
2. Add estimation logic in `internal/plugin/projected.go` with a helper function
3. Extend `tools/generate-pricing` to fetch pricing for the new service
4. Update `internal/pricing/client.go` with thread-safe lookup methods for the new service
5. Add tests for the new resource type

### Working with Build Tags
- Region-specific files use build tags like `//go:build region_use1`
- The fallback file uses negation: `//go:build !region_use1 && !region_usw2 && !region_euw1`
- Always ensure exactly one embed file is selected at build time

### Testing with Dummy Pricing
- Use `--dummy` flag with `tools/generate-pricing` to create minimal test pricing data
- This allows development/testing without AWS API access or credentials
- Dummy data includes only essential instance types (e.g., `t3.micro`) and volume types

### Logging
- **Never** log to stdout except for the PORT announcement
- Stdout is used **only** for `PORT=<port>` announcement
- Use stderr with prefix `[pulumicost-plugin-aws-public]` for debug/diagnostic messages
- Keep logging minimal by default

### Thread Safety
- Pricing lookups must be thread-safe (concurrent RPC calls)
- Use sync.RWMutex or sync.Once for initialization
- Avoid global mutable state

## Release Process

### Using GoReleaser
```bash
# Test release build locally
goreleaser build --snapshot --clean

# Create actual release
goreleaser release --clean
```

### Before Hooks
The `.goreleaser.yaml` runs:
```bash
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data --dummy
```

This ensures pricing files exist before embedding. For production, remove `--dummy` and implement real AWS pricing fetch.

## Configuration (v1)

Current configuration is minimal:
- Currency: `USD` (hardcoded default)
- Account discount factor: `1.0` (no discount)
- PORT: From environment variable or ephemeral (managed by pluginsdk)

Future versions will support:
- Environment variables or flags for additional configuration
- Custom discount rates
- Different EC2 tenancy models
- Spot/Reserved instance pricing

## Proto Dependencies

This plugin depends on proto definitions from:
- `github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1`
  - `CostSourceService` gRPC service
  - `ResourceDescriptor`, `GetProjectedCostRequest/Response`
  - `SupportsRequest/Response`, `PricingSpec`
  - `ErrorCode` enum, `ErrorDetail` message

Always refer to the proto files in `../pulumicost-spec/proto/` for the authoritative API contract.

## Critical Protocol Notes

**DO NOT:**
- Read from stdin or write JSON to stdout (except PORT announcement)
- Implement custom error codes outside the proto ErrorCode enum
- Process multiple resources in a single RPC call (one resource = one call)
- Assume batch processing semantics

**DO:**
- Use pluginsdk.Serve() for lifecycle management
- Announce PORT=<port> to stdout once on startup
- Serve gRPC on 127.0.0.1 loopback only
- Return proto-defined error codes via gRPC status
- Handle context cancellation for graceful shutdown
- Make pricing lookups thread-safe for concurrent RPCs

## PR and Commit Workflow

**IMPORTANT**: When completing a feature implementation, always:

1. **Generate PR_MESSAGE.md** instead of running `git commit` directly
2. **Validate PR_MESSAGE.md** passes both linters:
   - `npx markdownlint-cli PR_MESSAGE.md`
   - Extract commit message and test with `echo "..." | npx commitlint`
3. **Include in PR_MESSAGE.md**:
   - Summary of changes
   - Implementation details (new/modified files)
   - Test plan with checkboxes
   - Known limitations
   - Breaking changes section
   - Commit message in a code block (conventional commits format)
   - Closes #issue-number

This allows the user to review and make the commit themselves, and ensures
the commit message follows conventional commits format.

## Active Technologies
- Go 1.25+ + pulumicost-core/pkg/pluginsdk, pulumicost-spec/sdk/go/proto (003-ca-sa-region-support)
- Embedded JSON files (go:embed) - no external storage (003-ca-sa-region-support)

- Go 1.25+ (001-pulumicost-aws-plugin, 002-ap-region-support)
- Embedded JSON files (go:embed) - No external storage required
- Embedded JSON pricing files (go:embed) - no external storage

## Recent Changes
- 001-pulumicost-aws-plugin: Added Go 1.25+
- 002-ap-region-support: Added 4 Asia Pacific regions (Singapore, Sydney, Tokyo, Mumbai)
  - Created region-specific embed files (embed_apse1.go, embed_apse2.go, embed_apne1.go, embed_aps1.go)
  - Updated GoReleaser config with 4 new build targets
  - Extended test suites with AP region test cases
  - Added success criteria validation tests (concurrent calls, latency, cross-region pricing, region rejection)
  - All binaries are 16MB (< 20MB requirement)
  - Region mismatch latency: 0.01ms (< 100ms requirement)
  - 100% region rejection rate validated
- 003-ca-sa-region-support: Added 2 Americas regions (Canada, South America)
  - Created region-specific embed files (embed_cac1.go, embed_sae1.go)
  - Build tags: region_cac1 (ca-central-1), region_sae1 (sa-east-1)
  - Updated GoReleaser config with 2 new build targets (total 9 regions)
  - Extended test suites with CA/SA region test cases
  - All binaries are 16MB (< 20MB requirement)
  - Region mismatch latency: 0.02ms (< 100ms requirement)
  - All tests pass including concurrent RPC handling
