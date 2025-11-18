# PulumiCost AWS Public Plugin – Prompt Pack Overview

You are implementing the `pulumicost-plugin-aws-public` Go plugin according to the design below.

## Context

- Repo: `https://github.com/rshade/pulumicost-plugin-aws-public`
- Language: Go (version 1.21 or later)
- Purpose: A **fallback cost plugin** for PulumiCost that estimates AWS resource costs using **public AWS on-demand pricing**, without needing CUR/Cost Explorer/Vantage data.
- Architecture:
  - The PulumiCost **core** calls external plugins via **gRPC**.
  - This plugin implements the **CostSourceService** gRPC interface from `pulumicost.v1` proto.
  - Plugin announces its PORT to stdout, then serves gRPC on 127.0.0.1.
  - Core calls methods like `GetProjectedCost()`, `Supports()`, `Name()` via gRPC.
  - **One resource per RPC call** (not batch processing).
- Key design choices (already agreed):
  1. Use **embedded, trimmed JSON** pricing per region via `//go:embed` (Option A).
  2. Use **GoReleaser** to build **one binary per region**, each with only that region's pricing.
  3. Binary naming convention:
     - `pulumicost-plugin-aws-public-<region>`
     - e.g. `pulumicost-plugin-aws-public-us-east-1`
  4. For now, **single-region** per binary; multi-region stacks are handled by having core call multiple region binaries.
  5. gRPC error protocol:
     - Use proto-defined `ErrorCode` enum (not custom codes).
     - On mismatched region, return `ERROR_CODE_UNSUPPORTED_REGION` with `ErrorDetail.details` map containing `pluginRegion` and `requiredRegion`.
     - Core is responsible for **downloading/using** the correct region binary based on error metadata.

## Services to support in v1

Start small but useful:

- **Required in v1**
  - EC2 instances
  - EBS volumes
- Nice to scaffold / stub (but can be partially unimplemented):
  - S3 buckets
  - Lambda
  - RDS
  - DynamoDB

For unimplemented services:
- `Supports()` RPC returns `supported=true` with reason "Limited support - returns $0 estimate"
- `GetProjectedCost()` RPC returns `cost_per_month=0` with billing_detail explaining not implemented

## Pricing data

- Source: AWS public price list JSON endpoints.
- We **do not** hit these at runtime.
- Instead:
  - A small Go tool in `tools/generate-pricing` calls AWS pricing APIs at build/release time.
  - It **trims** the data to only the fields/services we need.
  - Writes per-region JSON files into `data/aws_pricing_<region>.json`.
- The plugin embeds one such JSON per region using `//go:embed`.
- Pricing client must be **thread-safe** for concurrent gRPC calls.

## Build tags per region

- Each region has its own file under `internal/pricing`:

  ```go
  // internal/pricing/embed_use1.go
  //go:build region_use1

  package pricing

  import _ "embed"

  //go:embed ../../data/aws_pricing_us-east-1.json
  var rawPricingJSON []byte

  const Region = "us-east-1"
  ```

- Example mapping:
  - `us-east-1` → build tag `region_use1`
  - `us-west-2` → build tag `region_usw2`
  - `eu-west-1` → build tag `region_euw1`

## gRPC Protocol

The plugin implements `CostSourceService` from `pulumicost.v1`:

### Required RPCs

1. **Name()** → `NameResponse{name: "aws-public"}`
2. **Supports(ResourceDescriptor)** → `SupportsResponse{supported: bool, reason: string}`
3. **GetProjectedCost(ResourceDescriptor)** → `GetProjectedCostResponse{unit_price, currency, cost_per_month, billing_detail}`

### Optional RPCs
4. **GetPricingSpec(ResourceDescriptor)** → `PricingSpec` (detailed pricing info)

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
  string billing_detail = 4;   // Human-readable assumptions
}
```

### Error Handling

Use `ErrorCode` enum from proto (see `pulumicost-spec/proto/pulumicost/v1/costsource.proto`):

- `ERROR_CODE_UNSUPPORTED_REGION` (9): Resource region doesn't match plugin binary region
  - Include `ErrorDetail.details` map with `pluginRegion` and `requiredRegion`
- `ERROR_CODE_INVALID_RESOURCE` (6): Missing required ResourceDescriptor fields
- `ERROR_CODE_DATA_CORRUPTION` (11): Embedded pricing data is corrupt

**DO NOT** define custom error codes outside the proto enum.

## Plugin Lifecycle

1. **Startup**: Plugin writes `PORT=<port>` to stdout
2. **Serve**: Plugin serves gRPC on 127.0.0.1:<port> using `pluginsdk.Serve()`
3. **Shutdown**: Graceful shutdown on context cancellation

Use `pluginsdk.Serve()` from `pulumicost-core/pkg/pluginsdk` for lifecycle management.

## Configuration & usage assumptions

- The plugin will eventually support configuration (profiles, discounts, etc) via env or flags, but for v1, keep it minimal:
  - Currency: default `USD`
  - Account discount factor: default `1.0` (no discount)
  - EC2: assume 730 hours/month (24x7 on-demand), Linux OS, Shared tenancy
  - EBS: cost = size (GB) × `rate_per_gb_month`
- Leave clear TODO markers where more configuration will be added later.

## Files in this prompt pack

- `10-scaffold-and-layout.md` – scaffold module, directories, basic main with gRPC setup
- `20-pricing-embed-and-client.md` – implement `tools/generate-pricing` and the embedded pricing client (thread-safe)
- `30-estimation-logic-ec2-ebs.md` – implement v1 cost logic for EC2 and EBS resources
- `40-plugin-main-and-protocol.md` – implement the gRPC service entrypoint with pluginsdk.Serve()
- `50-goreleaser-setup.md` – configure GoReleaser to build region-specific binaries with embedded JSON
- `60-readme-and-docs.md` – update README and basic docs for usage & installation

Apply these prompts **one at a time**, committing between major steps if you're using git.

## Proto Dependencies

This plugin depends on proto definitions from `pulumicost-spec`:
- `github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1`
  - `CostSourceService` gRPC service
  - `ResourceDescriptor`, `GetProjectedCostRequest/Response`
  - `SupportsRequest/Response`, `PricingSpec`
  - `ErrorCode` enum, `ErrorDetail` message

Always refer to `../pulumicost-spec/proto/` for the authoritative API contract.

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
