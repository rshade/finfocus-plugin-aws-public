# PulumiCost AWS Public Plugin

A gRPC-based cost estimation plugin for
[PulumiCost](https://github.com/rshade/pulumicost-core) that estimates AWS
infrastructure costs using publicly available AWS on-demand pricing data.

## Overview

This plugin provides monthly cost estimates for AWS resources without requiring
access to AWS Cost Explorer, CUR data, or third-party services. It embeds AWS
public pricing data at build time and serves cost estimates via gRPC.

### Supported Resources

**Fully Supported (with accurate pricing):**

- **EC2 Instances**: On-demand Linux instances with shared tenancy
- **EBS Volumes**: All volume types (gp2, gp3, io1, io2, etc.)

**Stub Support (returns $0 with explanation):**

- S3, Lambda, RDS, DynamoDB

## Features

- **gRPC Protocol**: Implements `CostSourceService` from `pulumicost.v1` proto
- **Region-Specific Binaries**: One binary per AWS region with embedded pricing
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

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/rshade/pulumicost-plugin-aws-public.git
cd pulumicost-plugin-aws-public

# Build for specific region
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
  "billing_detail": "On-demand Linux, Shared tenancy, 730 hrs/month"
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

S3, Lambda, RDS, DynamoDB return $0 with:

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

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make lint && make test`
5. Submit a pull request

## License

See [LICENSE](LICENSE) file for details.

## Links

- [PulumiCost Core](https://github.com/rshade/pulumicost-core)
- [PulumiCost Spec](https://github.com/rshade/pulumicost-spec)
- [AWS Pricing Documentation](https://aws.amazon.com/pricing/)
