# PulumiCost AWS Public Plugin

A gRPC-based cost estimation plugin for
[PulumiCost](https://github.com/rshade/pulumicost-core) that estimates AWS
infrastructure costs using publicly available AWS on-demand pricing data.

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/rshade/pulumicost-plugin-aws-public.git
cd pulumicost-plugin-aws-public

# Build for your region (example: us-east-1)
make build-region REGION=us-east-1

# Start the plugin
./pulumicost-plugin-aws-public-us-east-1
```

### Basic Usage

```bash
# The plugin starts and announces its port
PORT=50051

# Use grpcurl to test (example EC2 instance)
grpcurl -plaintext localhost:$PORT \
  pulumicost.v1.CostSourceService/GetProjectedCost \
  -d '{
    "resource": {
      "provider": "aws",
      "resource_type": "ec2",
      "sku": "t3.micro",
      "region": "us-east-1"
    }
  }'
```

## Overview

This plugin provides monthly cost estimates for AWS resources without requiring
access to AWS Cost Explorer, CUR data, or third-party services. It embeds AWS
public pricing data at build time and serves cost estimates via gRPC.

### Supported Resources

**Fully Supported (with accurate pricing):**

- **EC2 Instances**: On-demand Linux instances with shared tenancy
- **EBS Volumes**: All volume types (gp2, gp3, io1, io2, etc.)
- **Lambda Functions**: Request-based and compute-duration pricing
- **S3 Storage**: Storage cost estimation by storage class and size
- **DynamoDB**: On-demand and provisioned capacity modes with storage
- **ELB Load Balancers**: ALB and NLB pricing with LCU/NLCU billing

**Stub Support (returns $0 with explanation):**

- RDS

## Features

- **gRPC Protocol**: Implements `CostSourceService` from `pulumicost.v1` proto
- **Region-Specific Binaries**: One binary per AWS region with embedded pricing
- **Carbon Footprint Estimation**: EC2 instances include gCO2e metrics using CCF methodology
- **Thread-Safe**: Concurrent RPC calls are handled safely
- **Graceful Errors**: Proto-defined error codes with detailed error information
- **No AWS Credentials Required**: Uses embedded public pricing data
- **Build Tags**: Optimized binaries with only relevant region pricing data

## Architecture

### Binary Distribution

Each region has its own binary to minimize size and ensure accurate pricing:

**US Regions:**

- `pulumicost-plugin-aws-public-us-east-1` (US East - N. Virginia)
- `pulumicost-plugin-aws-public-us-west-2` (US West - Oregon)

**Europe Regions:**

- `pulumicost-plugin-aws-public-eu-west-1` (EU - Ireland)

**Asia Pacific Regions:**

- `pulumicost-plugin-aws-public-ap-southeast-1` (Asia Pacific - Singapore)
- `pulumicost-plugin-aws-public-ap-southeast-2` (Asia Pacific - Sydney)
- `pulumicost-plugin-aws-public-ap-northeast-1` (Asia Pacific - Tokyo)
- `pulumicost-plugin-aws-public-ap-south-1` (Asia Pacific - Mumbai)

**Canada Regions:**

- `pulumicost-plugin-aws-public-ca-central-1` (Canada Central - Montreal)

**South America Regions:**

- `pulumicost-plugin-aws-public-sa-east-1` (South America - São Paulo)

### Cost Estimation

**EC2 Instances:**

- Pricing lookup: `instance_type + operating_system + tenancy`
- Monthly cost: `hourly_rate × 730 hours`
- Assumptions: Linux, Shared tenancy, 24×7 on-demand

**EBS Volumes:**

- Pricing lookup: `volume_type`
- Monthly cost: `rate_per_gb_month × volume_size_gb`
- Size extraction: From `tags["size"]` or `tags["volume_size"]`
- Default size: 8 GB if not specified

**Lambda Functions:**

- Pricing lookup: Requests and Compute Duration (GB-seconds)
- Monthly cost: `(requests × price_per_request) + (gb_seconds × price_per_gb_second)`
- GB-seconds: `(memory_mb / 1024) × (avg_duration_ms / 1000) × requests`
- Tag requirements: `requests_per_month`, `avg_duration_ms`
- Defaults: 128MB memory, 0 requests, 100ms duration if tags missing

**S3 Storage:**

- Pricing lookup: `storage_class`
- Monthly cost: `rate_per_gb_month × storage_size_gb`
- Size extraction: From `tags["size"]`
- Default size: 1 GB if not specified

**DynamoDB:**

- **On-Demand Mode**: `(read_requests × price_per_read) + (write_requests × price_per_write) + (storage_gb × price_per_gb_month)`
- **Provisioned Mode**: `(rcu × 730 × price_per_rcu_hour) + (wcu × 730 × price_per_wcu_hour) + (storage_gb × price_per_gb_month)`
- Tag requirements: `read_capacity_units`/`read_requests_per_month`, `write_capacity_units`/`write_requests_per_month`, `storage_gb`
- SKU specifies capacity mode: "provisioned" or defaults to "on-demand"

**ELB Load Balancers:**

- **ALB Pricing**: `(730 × hourly_rate) + (730 × lcu_per_hour × price_per_lcu)`
- **NLB Pricing**: `(730 × hourly_rate) + (730 × nlcu_per_hour × price_per_nlcu)`
- Load balancer type auto-detected from SKU (contains "alb"/"nlb") or defaults to ALB
- Tag requirements: `lcu_per_hour` (ALB) or `nlcu_per_hour` (NLB), or generic `capacity_units`

### Carbon Estimation (EC2 Only)

EC2 instances include carbon footprint estimation using the
[Cloud Carbon Footprint](https://www.cloudcarbonfootprint.org/) methodology.

**Formula:**

```text
avgWatts = minWatts + (utilization × (maxWatts - minWatts))
energyKWh = (avgWatts × vCPUs × hours) / 1000
energyWithPUE = energyKWh × 1.135  (AWS PUE)
carbonGrams = energyWithPUE × gridIntensity × 1,000,000
```

**Features:**

- Returns `METRIC_KIND_CARBON_FOOTPRINT` in `ImpactMetrics` (unit: gCO2e)
- Supports 500+ EC2 instance types from CCF coefficients
- Region-specific grid emission factors for 12 AWS regions
- Utilization override: per-resource > request-level > 50% default

**Utilization Override:**

```json
{
  "utilization_percentage": 0.8,
  "resource": {
    "utilization_percentage": 0.9
  }
}
```

Priority: `resource.utilization_percentage` > `request.utilization_percentage` > 0.5

## Installation & Setup

### ⚠️ IMPORTANT: Build Tags Required

**The plugin requires Go build tags to embed region-specific pricing data.**

The v0.0.10 release was built without build tags, resulting in all costs returning $0.
Always use one of the methods below to build with the correct `-tags` flag.

### From Source

**For development/testing (fallback pricing):**

```bash
# Clone the repository
git clone https://github.com/rshade/pulumicost-plugin-aws-public.git
cd pulumicost-plugin-aws-public

# Build with fallback pricing (development only - NOT for production)
make build
```

**For production (real AWS pricing - RECOMMENDED):**

```bash
# Clone the repository
git clone https://github.com/rshade/pulumicost-plugin-aws-public.git
cd pulumicost-plugin-aws-public

# Build for default region (us-east-1 with real pricing)
make build-default-region

# OR build for any region with real pricing
make build-region REGION=us-east-1

# OR use go build directly with region tags
go build -tags region_use1 -o pulumicost-plugin-aws-public-us-east-1 \
  ./cmd/pulumicost-plugin-aws-public
```

### Using GoReleaser

```bash
# Generate pricing data for all supported regions (9 regions)
go run ./tools/generate-pricing \
  --regions us-east-1,us-west-2,eu-west-1,\
ap-southeast-1,ap-southeast-2,ap-northeast-1,\
ap-south-1,ca-central-1,sa-east-1 \
  --out-dir ./internal/pricing/data

# Build all region binaries (9 regions × 3 OS × 2 architectures)
goreleaser build --snapshot --clean
```

### Building Individual Region Binaries

```bash
# Singapore (ap-southeast-1)
go build -tags region_apse1 -o pulumicost-plugin-aws-public-ap-southeast-1 ./cmd/pulumicost-plugin-aws-public

# Sydney (ap-southeast-2)
go build -tags region_apse2 -o pulumicost-plugin-aws-public-ap-southeast-2 ./cmd/pulumicost-plugin-aws-public

# Tokyo (ap-northeast-1)
go build -tags region_apne1 -o pulumicost-plugin-aws-public-ap-northeast-1 \
  ./cmd/pulumicost-plugin-aws-public

# Mumbai (ap-south-1)
go build -tags region_aps1 -o pulumicost-plugin-aws-public-ap-south-1 ./cmd/pulumicost-plugin-aws-public

# Canada (ca-central-1)
go build -tags region_cac1 -o pulumicost-plugin-aws-public-ca-central-1 ./cmd/pulumicost-plugin-aws-public

# South America (sa-east-1)
go build -tags region_sae1 -o pulumicost-plugin-aws-public-sa-east-1 ./cmd/pulumicost-plugin-aws-public
```

## Usage

### Starting the Plugin

The plugin is designed to be started by PulumiCost core, but can be run
standalone for testing:

```bash
# Start the plugin (announces PORT on stdout)
./pulumicost-plugin-aws-public-us-east-1
```

Output:

```text
PORT=50051
```

### Integration with PulumiCost Core

PulumiCost core discovers and communicates with the plugin via:

1. **Startup**: Core starts plugin binary as subprocess
2. **Port Discovery**: Core reads `PORT=<port>` from stdout
3. **gRPC Communication**: Core connects to `127.0.0.1:<port>`
4. **Lifecycle**: Core cancels context to trigger graceful shutdown

### Trace ID Propagation

The plugin supports distributed tracing through trace ID propagation for request
correlation across the PulumiCost ecosystem.

#### How It Works

- **Request Correlation**: Each gRPC request can include a `trace_id` for tracking
  requests across multiple services
- **Automatic Generation**: If no `trace_id` is provided, the plugin automatically
  generates a UUID v4
- **Log Correlation**: All log entries include the `trace_id` for debugging and
  monitoring
- **Error Correlation**: Error responses include `trace_id` in the error details

#### Sending Trace ID

Include `trace_id` in gRPC metadata using the key `pulumicost-trace-id`:

```go
import "google.golang.org/grpc/metadata"

// Create metadata with trace_id
md := metadata.Pairs("pulumicost-trace-id", "your-custom-trace-id")
ctx := metadata.NewOutgoingContext(context.Background(), md)

// Use ctx for gRPC calls
```

#### Log Output

All structured log entries include the trace_id:

```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "info",
  "message": "Processing GetProjectedCost request",
  "plugin_name": "aws-public",
  "plugin_version": "1.0.0",
  "trace_id": "your-custom-trace-id",
  "operation": "GetProjectedCost",
  "resource_type": "ec2",
  "duration_ms": 5
}
```

#### Error Responses

Error responses include trace_id in the details map:

```json
{
  "code": 9,
  "message": "region mismatch",
  "details": {
    "pluginRegion": "us-east-1",
    "requiredRegion": "eu-west-1",
    "trace_id": "your-custom-trace-id"
  }
}
```

### ResourceDescriptor Format

Resources are described using the `ResourceDescriptor` proto message:

```protobuf
message ResourceDescriptor {
  string provider = 1;       // "aws"
  string resource_type = 2;  // "ec2", "ebs", "s3", etc.
  string sku = 3;            // instance_type (EC2) or volume_type (EBS)
  string region = 4;         // "us-east-1", "us-west-2", etc.
  map<string, string> tags = 5;  // Additional metadata (e.g., "size": "100")
}
```

### Example: EC2 Instance

```json
{
  "provider": "aws",
  "resource_type": "ec2",
  "sku": "t3.micro",
  "region": "us-east-1"
}
```

**Response:**

```json
{
  "cost_per_month": 7.592,
  "unit_price": 0.0104,
  "currency": "USD",
  "billing_detail": "On-demand Linux, Shared tenancy, 730 hrs/month",
  "impact_metrics": [
    {
      "kind": "METRIC_KIND_CARBON_FOOTPRINT",
      "value": 3507.6,
      "unit": "gCO2e"
    }
  ]
}
```

### Example: EBS Volume

```json
{
  "provider": "aws",
  "resource_type": "ebs",
  "sku": "gp3",
  "region": "us-east-1",
  "tags": {
    "size": "100"
  }
}
```

**Response:**

```json
{
  "cost_per_month": 8.0,
  "unit_price": 0.08,
  "currency": "USD",
  "billing_detail": "gp3 volume, 100 GB, $0.0800/GB-month"
}
```

### Example: AP Region (Singapore)

```json
{
  "provider": "aws",
  "resource_type": "ec2",
  "sku": "t3.micro",
  "region": "ap-southeast-1"
}
```

**Response:**

```json
{
  "cost_per_month": 8.468,
  "unit_price": 0.0116,
  "currency": "USD",
  "billing_detail": "On-demand Linux, Shared tenancy, 730 hrs/month"
}
```

### Example: S3 Storage

```json
{
  "provider": "aws",
  "resource_type": "s3",
  "sku": "STANDARD",
  "region": "us-east-1",
  "tags": {
    "size": "100"
  }
}
```

**Response:**

```json
{
  "cost_per_month": 2.3,
  "unit_price": 0.023,
  "currency": "USD",
  "billing_detail": "S3 STANDARD storage, 100 GB, $0.0230/GB-month"
}
```

### Example: DynamoDB On-Demand

```json
{
  "provider": "aws",
  "resource_type": "dynamodb",
  "sku": "on-demand",
  "region": "us-east-1",
  "tags": {
    "read_requests_per_month": "1000000",
    "write_requests_per_month": "500000",
    "storage_gb": "50"
  }
}
```

**Response:**

```json
{
  "cost_per_month": 137.5,
  "unit_price": 0.023,
  "currency": "USD",
  "billing_detail": "DynamoDB on-demand, 1000000 reads, 500000 writes, 50GB storage"
}
```

### Example: DynamoDB Provisioned

```json
{
  "provider": "aws",
  "resource_type": "dynamodb",
  "sku": "provisioned",
  "region": "us-east-1",
  "tags": {
    "read_capacity_units": "100",
    "write_capacity_units": "50",
    "storage_gb": "50"
  }
}
```

**Response:**

```json
{
  "cost_per_month": 178.45,
  "unit_price": 0.00013,
  "currency": "USD",
  "billing_detail": "DynamoDB provisioned, 100 RCUs, 50 WCUs, 730 hrs/month, 50GB storage"
}
```

### Example: ALB Load Balancer

```json
{
  "provider": "aws",
  "resource_type": "elb",
  "sku": "alb",
  "region": "us-east-1",
  "tags": {
    "lcu_per_hour": "10"
  }
}
```

**Response:**

```json
{
  "cost_per_month": 219.0,
  "unit_price": 0.0225,
  "currency": "USD",
  "billing_detail": "ALB, 730 hrs/month, 10.0 LCU avg/hr"
}
```

## gRPC Service API

### Name()

Returns the plugin identifier.

```protobuf
rpc Name(NameRequest) returns (NameResponse);
```

**Response:** `name: "pulumicost-plugin-aws-public"`

### Supports()

Checks if the plugin can estimate costs for a given resource.

```protobuf
rpc Supports(SupportsRequest) returns (SupportsResponse);
```

**Returns:**

- `supported: true` - For EC2/EBS in plugin's region
- `supported: true` with reason - For stub services (S3, Lambda, etc.)
- `supported: false` with reason - For region mismatch or unknown types
- `supported_metrics` - For EC2: includes `METRIC_KIND_CARBON_FOOTPRINT`

### GetProjectedCost()

Estimates monthly cost for a resource.

```protobuf
rpc GetProjectedCost(GetProjectedCostRequest) returns (GetProjectedCostResponse);
```

**Returns:**

- `cost_per_month` - Estimated monthly cost
- `unit_price` - Hourly rate (EC2) or per-GB-month rate (EBS)
- `currency` - Always "USD"
- `billing_detail` - Human-readable explanation of calculation
- `impact_metrics` - Array of environmental metrics (EC2 only: carbon footprint in gCO2e)

## Error Handling

### ERROR_CODE_UNSUPPORTED_REGION

Returned when resource region doesn't match plugin region.

**Error Details:**

```json
{
  "pluginRegion": "us-east-1",
  "requiredRegion": "eu-west-1"
}
```

**gRPC Code:** `FailedPrecondition`

### ERROR_CODE_INVALID_RESOURCE

Returned when ResourceDescriptor is missing required fields.

**gRPC Code:** `InvalidArgument`

## E2E Test Support

The plugin includes support for E2E testing with expected cost ranges and test
mode features.

### Enabling Test Mode

Set the environment variable before starting the plugin:

```bash
export PULUMICOST_TEST_MODE=true
./pulumicost-plugin-aws-public-us-east-1
```

**Valid Values:**

- `true` - Enable test mode (enhanced logging, validation support)
- `false` or unset - Production mode (standard behavior)
- Other values - Treated as disabled with warning logged

### Expected Cost Ranges

Reference values for E2E test validation (as of 2025-12-01):

| Resource | SKU       | Region    | Monthly Cost | Tolerance |
|----------|-----------|-----------|--------------|-----------|
| EC2      | t3.micro  | us-east-1 | $7.592       | ±1%       |
| EBS      | gp2 (8GB) | us-east-1 | $0.80        | ±5%       |

### Enhanced Logging

When test mode is enabled, additional debug logs include calculation details:

```bash
LOG_LEVEL=debug PULUMICOST_TEST_MODE=true ./pulumicost-plugin-aws-public-us-east-1
```

## Development

### Prerequisites

- Go 1.25+
- golangci-lint
- goreleaser (optional, for releases)

### Building

```bash
# Standard build (uses fallback pricing data)
make build

# Region-specific build
go build -tags region_use1 -o pulumicost-plugin-aws-public-us-east-1 ./cmd/pulumicost-plugin-aws-public
```

### Adding New AWS Regions

To add support for a new AWS region:

1. **Update regions.yaml**: Add the new region to `internal/pricing/regions.yaml`

   ```yaml
   regions:
     - id: euw3      # Short code from scripts/region-tag.sh
       name: eu-west-3  # Full AWS region name
       tag: region_euw3 # Build tag
   ```

2. **Generate configs**: Run the generation scripts

   ```bash
   make generate-embeds    # Creates embed_euw3.go
   make generate-goreleaser # Updates .goreleaser.yaml
   make verify-regions     # Validates all configurations
   ```

3. **Test the region**: Build and test the new region

   ```bash
   make build-region REGION=eu-west-3
   ```

The automated system ensures consistency across region configurations.

### Testing

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/plugin/... -v
go test ./internal/pricing/... -v
```

### Linting

```bash
make lint
```

### Project Structure

```text
.
├── cmd/
│   └── pulumicost-plugin-aws-public/    # CLI entrypoint
│       └── main.go
├── internal/
│   ├── plugin/
│   │   ├── plugin.go          # Plugin interface implementation
│   │   ├── supports.go        # Supports() RPC
│   │   ├── projected.go       # GetProjectedCost() RPC
│   │   └── *_test.go         # Unit tests
│   └── pricing/
│       ├── client.go          # Pricing lookup client
│       ├── types.go           # Pricing data structures
│       ├── embed_*.go         # Region-specific embedded data
│       └── data/              # Generated pricing JSON files
├── tools/
│   └── generate-pricing/      # Pricing data generator
│       └── main.go
├── .goreleaser.yaml           # Release configuration
└── Makefile                   # Build automation
```

## Edge Cases & Limitations

### EBS Volume Size

If `tags["size"]` or `tags["volume_size"]` is not provided, defaults to 8 GB.

**Response includes assumption:**

```text
"gp2 volume, 8 GB (defaulted), $0.1000/GB-month"
```

### Unknown Instance Types

If instance type is not found in pricing data, returns $0 with explanation.

### Stub Services

S3, RDS, DynamoDB return $0 with:

```text
"s3 cost estimation not fully implemented - returns $0 estimate"
```

### Region Boundaries

Each binary only serves estimates for its embedded region. Requests for other
regions return `ERROR_CODE_UNSUPPORTED_REGION`.

## Assumptions

- **EC2**: Linux operating system, Shared tenancy
- **Hours per Month**: 730 (24×7 on-demand)
- **Currency**: USD only
- **Pricing**: Public on-demand rates (no Reserved Instances, Spot, or Savings Plans)

## Troubleshooting

### Common Issues

#### "Region not supported by this binary"

- Ensure you're using the correct regional binary (e.g., `pulumicost-plugin-aws-public-us-east-1` for `us-east-1` resources)

#### "EC2 instance type not found in pricing data"

- Verify the instance type is valid AWS instance type
- Check if the instance type is available in your region
- Regenerate pricing data if it's a new instance type: `make generate-pricing`

#### "failed to initialize pricing client"

- Ensure the binary was built with proper region tags
- Run `make generate-pricing` before building if pricing data is missing

#### Plugin not starting"

- Check that the binary has execute permissions: `chmod +x ./pulumicost-plugin-aws-public-*`
- Verify you're in the correct directory
- Check stderr for detailed error messages

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make lint && make test`
5. Submit a pull request

## Release Checklist (v0.0.11+)

**Before releasing a new version, ensure pricing data is embedded in all binaries.**

### Pre-Release (Before Creating Tag)

- [ ] Verify pricing data files exist: `ls -lh internal/pricing/data/aws_pricing_*.json`
- [ ] Run unit tests with region tag: `go test -tags=region_use1 -run TestEmbeddedPricing ./internal/pricing/...`
- [ ] Run functional pricing test: `go test -tags=integration -run TestIntegration_VerifyPricingEmbedded ./internal/plugin/... -v`
- [ ] Run full test suite: `make test`
- [ ] Run linter: `make lint`

### During Release

- [ ] Use automated release workflow: GitHub Actions will build all regions
- [ ] Do NOT manually build with `make build` (uses fallback pricing)
- [ ] Do NOT build outside the CI/CD pipeline
- [ ] Verify workflow completes successfully

### Post-Release Verification

- [ ] Download released binary for primary region (us-east-1)
- [ ] Check binary size: `stat -c%s pulumicost-plugin-aws-public-us-east-1` → should be > 10MB
- [ ] Verify binary signature/checksum matches release

### Testing Real Pricing

```bash
# Download released binary
wget https://github.com/rshade/pulumicost-plugin-aws-public/releases/download/v0.x.x/pulumicost-plugin-aws-public_v0.x.x_Linux_x86_64

# Extract and test against real Pulumi plan
./pulumicost-plugin-aws-public-us-east-1 &
# Plugin starts and announces PORT

# Use gRPC client to verify costs for real instance types:
# - t3.micro should cost ~$7.59/month
# - m5.large should cost ~$96/month
# - NOT $0 for all instance types
```

**v0.0.10 Issue (DO NOT REPEAT):**

The v0.0.10 release shipped with fallback pricing (all costs = $0) because the binary
was built without region tags. This checklist prevents recurrence by:

1. Running verification tests in CI before merge
2. Running functional tests that query the binary for real costs
3. Verifying binary sizes after building all regions
4. Automating the release workflow (reduces manual errors)

## License

See [LICENSE](LICENSE) file for details.

## Attribution

### Cloud Carbon Footprint

This project uses instance specification data from
[Cloud Carbon Footprint](https://www.cloudcarbonfootprint.org/) for carbon
emission estimation.

> Copyright 2021 Thoughtworks, Inc.
> Licensed under the Apache License, Version 2.0

## Links

- [PulumiCost Core](https://github.com/rshade/pulumicost-core)
- [PulumiCost Spec](https://github.com/rshade/pulumicost-spec)
- [AWS Pricing Documentation](https://aws.amazon.com/pricing/)
- [API Documentation](docs/api.md)
- [Code Documentation](https://pkg.go.dev/github.com/rshade/pulumicost-plugin-aws-public)
