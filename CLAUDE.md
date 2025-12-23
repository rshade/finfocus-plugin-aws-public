# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `pulumicost-plugin-aws-public`, a fallback PulumiCost plugin that estimates AWS resource costs using public AWS on-demand pricing data, without requiring CUR/Cost Explorer/Vantage data access. The plugin implements the gRPC CostSourceService protocol and is invoked as a separate process by PulumiCost core.

## Code Style

Go 1.25+: Follow standard conventions
- **No Dummy Data:** Do not create dummy, fake, or hardcoded placeholder data for core functionality (especially pricing). Always implement fetchers for real authoritative data sources (e.g., AWS Price List API).

## Architecture

### Plugin Protocol (gRPC)
- The plugin implements **CostSourceService** from `pulumicost.v1` proto (see `../pulumicost-spec/proto/pulumicost/v1/costsource.proto`)
- **Not** stdin/stdout JSON - uses gRPC with PORT announcement
- On startup: plugin writes `PORT=<port>` to stdout, then serves gRPC on 127.0.0.1
- Core connects via gRPC and calls methods like `GetProjectedCost()`, `Supports()`, `Name()`
- **One resource per RPC call** (not batch processing)
- Uses **pluginsdk.Serve()** from `pulumicost-spec/sdk/go/pluginsdk` for lifecycle management
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

**Per-Service Architecture (v0.0.12+):**

Pricing data is embedded as separate per-service JSON files for maintainability:

| Service | File Pattern | Typical Size |
|---------|--------------|--------------|
| EC2 | `ec2_{region}.json` | ~154MB |
| RDS | `rds_{region}.json` | ~7MB |
| EKS | `eks_{region}.json` | ~772KB |
| Lambda | `lambda_{region}.json` | ~445KB |
| S3 | `s3_{region}.json` | ~306KB |
| DynamoDB | `dynamodb_{region}.json` | ~22KB |
| ELB | `elb_{region}.json` | ~13KB |

**Build Process:**

1. `tools/generate-pricing` fetches AWS public pricing per service
2. Filters out Reserved Instance and Savings Plans terms (see [Pricing Term Filtering](#pricing-term-filtering))
3. Output: `internal/pricing/data/{service}_{region}.json` files
4. Files embedded via `//go:embed` in region-specific files (`embed_use1.go`, etc.)

**Parallel Initialization:**

The pricing client uses parallel goroutines for fast initialization:

```go
// In client.go init()
var wg sync.WaitGroup
wg.Add(7)  // One goroutine per service

go func() { defer wg.Done(); c.parseEC2Pricing(rawEC2JSON) }()
go func() { defer wg.Done(); c.parseS3Pricing(rawS3JSON) }()
// ... other services ...

wg.Wait()  // Wait for all parsing to complete
```

Each parser writes to its own dedicated index, so no locking is needed. Region is
captured from EC2 data (largest/most reliable) after all parsing completes.

**Performance Tracking:**

Run benchmarks to detect parsing regressions:

```bash
go test -tags=region_use1 -bench=BenchmarkNewClient -benchmem ./internal/pricing/...
```

**Thread Safety:**

- `sync.Once` ensures parsing happens exactly once
- Lookup methods are read-only after initialization
- Safe for concurrent gRPC calls

### Pricing Term Filtering

The pricing generator (`tools/generate-pricing`) filters AWS pricing data to keep only
On-Demand terms, reducing binary size significantly while maintaining full functionality:

| Service | With RI/SP | OnDemand Only | Reduction |
|---------|------------|---------------|-----------|
| EC2     | ~400MB     | ~154MB        | 61%       |
| Other services | Varies | Minimal savings | <5% |

**What is filtered:**

The AWS Price List API returns multiple term types in the `terms` object:

- **OnDemand** (KEPT): Pay-as-you-go pricing with no commitment
- **Reserved** (FILTERED): Reserved Instance pricing (1yr, 3yr upfront commitments)
  - Typically 30-75% discount vs OnDemand, but requires commitment
  - ~14,000 SKUs for EC2 in us-east-1 alone
- **savingsPlan** (FILTERED): Savings Plans pricing (flexible discount program)

**Why filter:**

1. **Binary size**: Reduces EC2 data from ~400MB to ~154MB (61% reduction)
2. **Scope alignment**: Plugin only supports On-Demand pricing for v1
3. **User expectation**: On-demand is the default/fallback pricing model

**Implications for users:**

- Plugin estimates **On-Demand costs only**
- Reserved Instance and Savings Plans discounts are NOT reflected
- For RI/SP pricing, users need different data sources (CUR, Cost Explorer, Vantage)

**Adding RI/SP support (future consideration):**

If Reserved/Savings Plans cost estimation is needed, consider:

1. **Separate binary**: RI/SP data is too large to embed with OnDemand (would exceed reasonable binary size)
2. **External data source**: API call rather than embedded data
3. **User-provided pricing**: Import from their AWS account (CUR export)
4. **Hybrid approach**: On-demand embedded + optional RI/SP overlay from external source

**Code location:** `tools/generate-pricing/main.go` in `fetchServicePricingRaw()` function, lines 188-211.

### ⚠️ CRITICAL: No Pricing Data Filtering

**DO NOT filter, trim, or strip pricing data in `tools/generate-pricing`.**

The v0.0.10 and v0.0.11 releases were broken because aggressive filtering was
added to `tools/generate-pricing/main.go` that stripped 85% of pricing data:

- EC2 products reduced from ~90,000 to ~12,000
- EBS volume pricing was missing
- Many instance types returned $0

**Rules for pricing data handling:**

1. **Merge ALL products** - The `generateCombinedPricingData()` function must
   merge all products without filtering
2. **Keep ALL attributes** - Do not strip product attributes to "required"
   fields
3. **Keep ALL terms** - Merge all OnDemand terms without filtering by
   ProductFamily
4. **No "optimization"** - Do not add filtering to "reduce binary size" - the
   full data is required

**Immutable tests prevent regression:**

- `TestEmbeddedPricingDataSize` - Fails if data < 100MB (v0.0.10 had ~5MB)
- `TestEmbeddedPricingProductCount` - Fails if < 50,000 (v0.0.10 had ~16,000)

**If you need to change `tools/generate-pricing/main.go`:**

1. Verify the generated JSON is still ~150MB for us-east-1
2. Run `go test -tags=region_use1 ./internal/pricing/...` to verify thresholds
3. Check that product count is ~98,000 for us-east-1

### Service Support (v1)

**Fully implemented:**

- EC2 instances
- EBS volumes
- EKS clusters
- DynamoDB (On-Demand and Provisioned modes)
- Elastic Load Balancing (ALB/NLB) - Application Load Balancers and Network Load Balancers

**Stubbed/Partial:**

- S3
- Lambda
- RDS

Stubbed services behavior:

- Supports() returns `supported=true` with reason "Limited support - returns $0 estimate"
- GetProjectedCost() returns `cost_per_month=0` with billing_detail explaining not implemented

## Directory Structure

```
cmd/
  pulumicost-plugin-aws-public/     # gRPC service entrypoint
    main.go                          # Calls pluginsdk.Serve()
internal/
  carbon/
    constants.go     # PUE, default utilization, hours per month
    estimator.go     # CarbonEstimator interface and CCF formula
    grid_factors.go  # AWS region grid emission factors
    instance_specs.go  # go:embed CSV parsing with sync.Once
    utilization.go   # Utilization priority logic
    data/
      ccf_instance_specs.csv  # Embedded CCF instance power data
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
  parse-regions/
    main.go          # CLI tool to parse regions.yaml (replaces fragile sed/awk)
data/
  aws_pricing_*.json  # Generated pricing files (not in git)
```

## Build Tag System (CRITICAL)

**The plugin uses Go build tags to embed region-specific AWS pricing data. This is CRITICAL for production.**

### How It Works

The plugin has 12 region-specific embed files in `internal/pricing/`:
- `embed_use1.go` (us-east-1) → requires `-tags=region_use1`
- `embed_usw2.go` (us-west-2) → requires `-tags=region_usw2`
- `embed_euw1.go` (eu-west-1) → requires `-tags=region_euw1`
- `embed_apse1.go` (ap-southeast-1) → requires `-tags=region_apse1`
- `embed_apse2.go` (ap-southeast-2) → requires `-tags=region_apse2`
- `embed_apne1.go` (ap-northeast-1) → requires `-tags=region_apne1`
- `embed_aps1.go` (ap-south-1) → requires `-tags=region_aps1`
- `embed_cac1.go` (ca-central-1) → requires `-tags=region_cac1`
- `embed_sae1.go` (sa-east-1) → requires `-tags=region_sae1`
- `embed_govw1.go` (us-gov-west-1) → requires `-tags=region_govw1`
- `embed_gove1.go` (us-gov-east-1) → requires `-tags=region_gove1`
- `embed_usw1.go` (us-west-1) → requires `-tags=region_usw1`
- `embed_fallback.go` → Used when NO region tag (dummy pricing for testing)

### ⚠️ CRITICAL v0.0.10 Issue (FIXED in v0.0.11+)

The released v0.0.10 binary was built **WITHOUT region tags**, resulting in:
- All EC2 prices returned $0
- Only test SKUs (t3.micro, t3.small) had prices
- Real instance types (m5.large, c5.xlarge, etc.) were unsupported
- Silent failure - no error message, just $0 costs

**Root cause:** Binary was built with `make build` or `go build` instead of through the automated release workflow.

**Prevention in v0.0.11+:**
- Build verification tests fail if pricing data < 1MB
- Functional integration test queries binary to verify real costs are returned
- CI workflow runs pricing verification before merge
- Release workflow verifies all binaries have pricing embedded
- Clear Makefile warnings when building without region tags

### Building Correctly

**For development/testing (fallback pricing):**

```bash
make build  # ⚠️  Only for testing plugin structure, NOT for releases
```

**For production or real cost testing (REQUIRED for releases):**

```bash
make build-default-region  # Build us-east-1 with real pricing (RECOMMENDED)
# OR
make build-region REGION=us-east-1  # Build any region with real pricing
# OR
make build-all-regions  # Build all 12 regions with real pricing
```

**Verification:**

```bash
# Unit test - fails if pricing data < 1MB
go test -tags=region_use1 -run TestEmbeddedPricing ./internal/pricing/...

# Functional test - actually queries binary for real costs
go test -tags=integration -run TestIntegration_VerifyPricingEmbedded ./internal/plugin/... -v
```

## Common Commands

> **⚠️ IMPORTANT:** Before building, run `make generate-carbon-data` to
> generate the CCF instance specs CSV. This file is in `.gitignore` and
> required for carbon estimation. The build will panic at startup if missing.
> Use `make develop` to set up the complete development environment.

### Building
```bash
# Standard build (no region tags, uses default fallback)
go build ./...

# Build with specific region tag
go build -tags region_use1 -o pulumicost-plugin-aws-public-us-east-1 ./cmd/pulumicost-plugin-aws-public

# Build all region binaries using GoReleaser
goreleaser build --snapshot --clean
```

**Build Verification Tip:** When verifying that code compiles correctly with
region build tags, building a single region is sufficient. Don't wait for all
54 builds (9 regions × 6 architectures). Use a single-region build instead:

```bash
go build -tags region_use1 ./cmd/pulumicost-plugin-aws-public
```

### Testing
```bash
# Run all unit tests (preferred)
make test

# Run tests for specific package
go test ./internal/plugin
go test ./internal/pricing

# Run integration tests (requires building binaries)
go test -tags=integration ./internal/plugin/...

# Run specific integration test
go test -tags=integration ./internal/plugin/... -run TestIntegration_TraceIDPropagation

# Test gRPC service manually (requires grpcurl or similar)
# 1. Start plugin: ./pulumicost-plugin-aws-public-us-east-1
# 2. Capture PORT from stdout
# 3. Call RPCs: grpcurl -plaintext -d '{"resource": {...}}' localhost:<port> pulumicost.v1.CostSourceService/GetProjectedCost
```

### Test Cleanup
- **Cleanup Temporary Files:** Ensure all tests and build scripts remove any temporary files (e.g., `sample_ec2.json`, generated binaries, temporary output logs) created during execution. Do not leave artifacts cluttering the workspace.

### Generating Pricing Data

**For Development/Testing (Recommended):**
```bash
# Generate pricing data for single region only (saves space, tests all logic)
go run ./tools/generate-pricing --regions us-east-1 --out-dir ./data

# Clean up old region data first if switching regions
rm -f ./data/aws_pricing_*.json
```

**For Regional Verification (Before Release):**
```bash
# Generate pricing data for all supported regions
# This is ONLY needed to verify regional binary embedding works correctly
rm -f ./data/aws_pricing_*.json  # Clean old data first
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1,ca-central-1,sa-east-1,ap-southeast-1,ap-southeast-2,ap-northeast-1,ap-south-1 --out-dir ./data

# Then build all regional binaries
goreleaser build --snapshot --clean
```

**Why Single-Region Testing is Sufficient:**
- Pricing parser logic is region-agnostic; testing with us-east-1 verifies all parsing logic works
- ELB estimation code is independent of region (uses pricing client interface)
- Build-tag system is verified by compiling one region successfully
- SC-003 (regional support) is validated by pricing client tests, not by building all regions

### Generating Carbon Data
```bash
# Fetch CCF instance specs for carbon estimation (from cloud-carbon-coefficients repo)
go run ./tools/generate-carbon-data --out-dir ./internal/carbon/data

# Or use make target
make generate-carbon-data
```

The tool downloads AWS instance power specifications from the
[cloud-carbon-coefficients](https://github.com/cloud-carbon-footprint/cloud-carbon-coefficients)
repository (Apache 2.0 license). The CSV is embedded at build time via `//go:embed`.

**Note:** This file is in `.gitignore` and must be generated before building.
Run `make develop` to set up the complete development environment.

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

### Supported Resource Type Formats

The plugin accepts multiple resource type formats:

1. **Simple identifiers** (legacy): `ec2`, `ebs`, `rds`
2. **Pulumi format** (preferred): `aws:ec2/instance:Instance`, `aws:ebs/volume:Volume`
3. **Short Pulumi format**: `aws:ec2:Instance`

All formats are normalized internally via the `detectService()` function in `internal/plugin/projected.go`.

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

### EKS Clusters
- `resource_type`: "eks" or "aws:eks/cluster:Cluster"
- `sku`: Not used (cluster type determined by tags)
- Support tier: Read from `tags["support_type"]`, defaults to "standard"
  - "standard": $0.10/hour
  - "extended": $0.50/hour
- Assumptions (hardcoded for v1):
  - `hoursPerMonth = 730` (24×7 on-demand)
- `unit_price`: Hourly cluster management fee from pricing data
- `cost_per_month`: unit_price × 730
- `billing_detail`: "EKS cluster (<support_type> support), 730 hrs/month (control plane only, excludes worker nodes)"

### Elastic Load Balancing (ALB/NLB)

- `resource_type`: "elb", "alb", or "nlb"
- `sku`: Load balancer type: "alb" (Application) or "nlb" (Network), defaults to ALB if unspecified
- Capacity Units: Read from `tags["lcu_per_hour"]` (ALB) or `tags["nlcu_per_hour"]` (NLB), fallback to `tags["capacity_units"]`
- Pricing: Fixed hourly rate + capacity unit charges
  - ALB: Fixed hourly (e.g., $0.0225) + LCU rate (e.g., $0.008/LCU-hr)
  - NLB: Fixed hourly (e.g., $0.0225) + NLCU rate (e.g., $0.006/NLCU-hr)
- Assumptions (hardcoded for v1):
  - `hoursPerMonth = 730` (24×7 on-demand)
- `unit_price`: Fixed hourly rate from pricing data
- `cost_per_month`: (fixed_rate × 730) + (capacity_units × cu_rate × 730)
- `billing_detail`: "<ALB|NLB>, 730 hrs/month, <capacity_units> <LCU|NLCU> avg/hr"

### Stub Services (S3, Lambda, RDS)

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

## Cost Estimation Scope

Each service estimate covers specific cost components. Understanding what is included
and excluded helps users accurately estimate total infrastructure costs.

| Service | Included | Excluded | Carbon |
|---------|----------|----------|--------|
| EC2 | On-demand instance hours | Spot, Reserved, data transfer, EBS | ✅ gCO2e |
| EBS | Storage GB-month | IOPS, throughput, snapshots | ❌ [#135](https://github.com/rshade/pulumicost-plugin-aws-public/issues/135) |
| EKS | Control plane hours | Worker nodes, add-ons, data transfer | ❌ [#136](https://github.com/rshade/pulumicost-plugin-aws-public/issues/136) |
| ELB (ALB/NLB) | Fixed hourly + capacity unit charges | Data transfer, SSL/TLS termination | N/A |
| NAT Gateway | Hourly rate + data processing (per GB) | Data transfer OUT to internet, VPC peering transfer | N/A |
| RDS | Not implemented | - | ❌ [#137](https://github.com/rshade/pulumicost-plugin-aws-public/issues/137) |
| S3 | Not implemented | - | ❌ [#137](https://github.com/rshade/pulumicost-plugin-aws-public/issues/137) |
| Lambda | Not implemented | - | ❌ [#137](https://github.com/rshade/pulumicost-plugin-aws-public/issues/137) |
| DynamoDB | On-Demand/Provisioned throughput, storage | Global tables, streams, DAX, backups | ❌ [#137](https://github.com/rshade/pulumicost-plugin-aws-public/issues/137) |

### EKS Clusters

EKS cost estimation covers **control plane only**:

- **Included:** Hourly cluster management fee ($0.10/hr standard, $0.50/hr extended support)
- **Excluded:**
  - Worker node EC2 instances (estimate separately as EC2)
  - Data transfer costs
  - EKS add-ons (EBS CSI driver, CoreDNS, kube-proxy, etc.)
  - Load balancer costs (ALB/NLB)
  - Fargate pod costs

To estimate total EKS cluster cost, sum:

1. EKS control plane (this estimate)
2. EC2 instances for worker nodes (estimate each as EC2)
3. EBS volumes for persistent storage (estimate each as EBS)

### EC2 Instances

- **Included:** On-demand hourly instance cost for Linux, shared tenancy
- **Excluded:**
  - Spot instance pricing
  - Reserved instance pricing
  - Savings Plans pricing
  - Data transfer costs
  - EBS volumes (estimate separately)
  - Elastic IP costs

### EBS Volumes

- **Included:** Storage cost per GB-month
- **Excluded:**
  - Provisioned IOPS (io1/io2)
  - Provisioned throughput (gp3)
  - Snapshot storage costs
  - Data transfer costs

### NAT Gateway

- `resource_type`: "natgw", "nat_gateway", "nat-gateway", or "aws:ec2/natGateway:NatGateway"
- `sku`: Not used
- Data processing: Read from `tags["data_processed_gb"]`, defaults to 0 if missing
- Pricing: Fixed hourly rate + data processing per GB
  - Hourly: ~$0.045/hour (varies by region)
  - Data processing: ~$0.045/GB (varies by region)
- Assumptions (hardcoded for v1):
  - `hoursPerMonth = 730` (24×7 on-demand)
- `unit_price`: Hourly rate from pricing data
- `cost_per_month`: (hourly_rate × 730) + (data_gb × data_rate)
- `billing_detail`: "NAT Gateway, 730 hrs/month ($X.XXX/hr) + Y.YY GB data processed ($X.XXX/GB)"

**Excluded:**
- Data transfer OUT to the internet (charged separately by AWS)
- Data transfer via VPC peering or Transit Gateway (charged separately)
- Cross-AZ data transfer costs

### DynamoDB Tables

DynamoDB cost estimation supports both capacity modes:

**On-Demand Mode:**
- **Included:**
  - Read request units ($0.25 per million in us-east-1)
  - Write request units ($1.25 per million in us-east-1)
  - Storage per GB-month ($0.25/GB)
- **Required tags:** `read_requests_per_month`, `write_requests_per_month`, `storage_gb`

**Provisioned Mode:**
- **Included:**
  - Read Capacity Units (RCU) per hour
  - Write Capacity Units (WCU) per hour
  - Storage per GB-month
- **Required tags:** `read_capacity_units`, `write_capacity_units`, `storage_gb`

**Excluded (both modes):**
- Global tables replication costs
- DynamoDB Streams
- DynamoDB Accelerator (DAX)
- On-demand backup and restore
- Point-in-time recovery
- Data transfer costs

**SKU behavior:**
- `sku: "on-demand"` - Use on-demand pricing (case-insensitive)
- `sku: "provisioned"` - Use provisioned capacity pricing (case-insensitive)
- SKU is required by the SDK validation

### Carbon Estimation (EC2 Only)

Carbon footprint estimation uses the Cloud Carbon Footprint (CCF) methodology.

**Formula:**
```text
avgWatts = minWatts + (utilization × (maxWatts - minWatts))
energyKWh = (avgWatts × vCPUs × hours) / 1000
energyWithPUE = energyKWh × 1.135  (AWS PUE)
carbonGrams = energyWithPUE × gridIntensity × 1,000,000
```

**Data Sources:**
- Instance power specs: [cloud-carbon-coefficients](https://github.com/cloud-carbon-footprint/cloud-carbon-coefficients) (Apache 2.0)
- Grid emission factors: 12 AWS regions (metric tons CO2eq/kWh)

**Supported Metrics:**
- `METRIC_KIND_CARBON_FOOTPRINT` in `ImpactMetrics`
- Unit: gCO2e (grams CO2 equivalent)

**Utilization Priority:**
1. Per-resource: `ResourceDescriptor.UtilizationPercentage`
2. Request-level: `GetProjectedCostRequest.UtilizationPercentage`
3. Default: 50%

**Limitations (v1):**
- GPU power consumption not included
- Only EC2 instances (not EBS, EKS, etc.)
- Embodied carbon not calculated

**Future Enhancements:**
- [#135](https://github.com/rshade/pulumicost-plugin-aws-public/issues/135) - EBS storage carbon estimation
- [#136](https://github.com/rshade/pulumicost-plugin-aws-public/issues/136) - EKS cluster carbon estimation
- [#137](https://github.com/rshade/pulumicost-plugin-aws-public/issues/137) - S3, Lambda, RDS, DynamoDB carbon
- [#138](https://github.com/rshade/pulumicost-plugin-aws-public/issues/138) - GPU power consumption coefficients
- [#139](https://github.com/rshade/pulumicost-plugin-aws-public/issues/139) - Embodied carbon calculation
- [#140](https://github.com/rshade/pulumicost-plugin-aws-public/issues/140) - Annual grid factor update process

**Files:**
- `internal/carbon/` - Carbon estimation module
- `internal/carbon/data/ccf_instance_specs.csv` - Embedded instance specs (via `make generate-carbon-data`)

## Code Style Guidelines

### Comprehensive Docstrings (CodeRabbit Requirement)

**IMPORTANT**: Always write comprehensive Go doc comments for all exported functions and test
functions. CodeRabbit will flag functions with minimal or missing documentation.

**Good docstring pattern for test functions:**
```go
// TestIntegration_TraceIDPropagation verifies end-to-end trace_id propagation through the gRPC server.
//
// This test validates that when a client sends a request with a trace_id in gRPC metadata
// (using pluginsdk.TraceIDMetadataKey), the server extracts and includes that trace_id
// in all structured log entries. This is critical for distributed tracing and request
// correlation in production environments.
//
// Test workflow:
//  1. Builds the ap-southeast-1 binary with region_apse1 tag
//  2. Starts the binary, capturing stderr (where JSON logs are written)
//  3. Connects via gRPC and sends a request with trace_id in outgoing metadata
//  4. Parses the captured stderr and verifies trace_id appears in log JSON
//
// Prerequisites:
//   - Go toolchain available for building
//   - Port available for gRPC server (uses ephemeral port)
//
// Run with: go test -tags=integration ./internal/plugin/... -run TestIntegration_TraceIDPropagation
func TestIntegration_TraceIDPropagation(t *testing.T) {
```

**Docstring checklist:**
- First line: One-sentence summary starting with function name (Go convention)
- Purpose: What the function does and why it matters
- For tests: Test workflow with numbered steps
- Prerequisites or requirements
- Run command for integration tests
- Use `//` comments with proper spacing per Go conventions

### gRPC Metadata for Trace ID Propagation

When testing trace_id propagation through gRPC:

**Client side (sending):**
```go
md := metadata.New(map[string]string{
    pluginsdk.TraceIDMetadataKey: "my-trace-id",
})
ctx := metadata.NewOutgoingContext(context.Background(), md)
// Use ctx for gRPC call
```

**Server side (receiving):**
```go
if md, ok := metadata.FromIncomingContext(ctx); ok {
    if values := md.Get(pluginsdk.TraceIDMetadataKey); len(values) > 0 {
        traceID = values[0]
    }
}
```

### Integration Test Pattern for Log Verification

To verify log output in integration tests, capture stderr from the binary:
```go
var stderrBuf bytes.Buffer
cmd.Stderr = &stderrBuf
// ... run binary and make gRPC calls ...
logOutput := stderrBuf.String()
// Parse JSON log lines and verify fields
```

## Development Notes

### Implementing the Plugin Interface
From `pulumicost-spec/sdk/go/pluginsdk`:
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
    "github.com/rshade/pulumicost-spec/sdk/go/pluginsdk"
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
6. **Research carbon estimation data** for the new service:
   - Check [Cloud Carbon Footprint](https://www.cloudcarbonfootprint.org/docs/methodology) for applicable coefficients
   - If data exists, add carbon estimation to `internal/carbon/` and return `ImpactMetrics`
   - If no data, document in the service's billing_detail that carbon is not available
   - Update `getSupportedMetrics()` in `supports.go` to advertise carbon capability
   - Related issues: #135 (EBS), #136 (EKS), #137 (stub services)

### Working with Build Tags
- Region-specific files use build tags like `//go:build region_use1`
- The fallback file uses negation: `//go:build !region_use1 && !region_usw2 && !region_euw1`
- Always ensure exactly one embed file is selected at build time

### Pricing Data Generation
- The `tools/generate-pricing` tool fetches real pricing data from AWS Price List API
- No AWS credentials required - uses public pricing endpoint
- Data includes all instance types and volume types available in each region

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
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1 --out-dir ./data
```

This fetches real AWS pricing data and ensures pricing files exist before embedding.

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
- Go 1.25+ + pulumicost-spec v0.4.8 (pluginsdk, mapping packages), (013-sdk-migration)
- N/A (embedded pricing data via go:embed) (013-sdk-migration)
- Go 1.25+ + pulumicost-spec v0.4.10+ (MetricKind, ImpactMetric), zerolog, gRPC (015-carbon-estimation)
- Embedded data via `//go:embed` (CSV for instance specs, constants for grid factors) (015-carbon-estimation)
- Go 1.25+ + encoding/json, sync.Once, go:embed, zerolog, gRPC (pulumicost-spec) (018-raw-pricing-embed)
- Embedded JSON files via `//go:embed` (no external storage) (018-raw-pricing-embed)
- Go 1.25+ + pulumicost-spec v0.4.11+ (provides `Id` field on `ResourceDescriptor`), gRPC, zerolog (001-resourceid-passthrough)
- N/A (stateless gRPC service) (001-resourceid-passthrough)

- **Go 1.25+** with gRPC via pulumicost-spec/sdk/go/pluginsdk
- **pulumicost-spec** protos for CostSourceService API
- **zerolog** for structured JSON logging (stderr only)
- **Embedded JSON** pricing data via `//go:embed` (no external storage)

## Recent Changes

| Issue | Summary |
|-------|---------|
| #91 | EKS cost estimation scope documentation |
| #76 | EKS cluster cost estimation (control plane only) |
| 008 | E2E test mode, expected cost validation (t3.micro, gp2) |
| 005 | zerolog logging, trace_id propagation, LOG_LEVEL |
| 004 | GetActualCost fallback: `projected × (hours/730)` |
| 003 | Added ca-central-1, sa-east-1 (9 regions total) |
| 002 | Added 4 AP regions (Singapore, Sydney, Tokyo, Mumbai) |
| 001 | Initial plugin: EC2/EBS, us-east-1/us-west-2/eu-west-1 |

**Performance:** ~16MB binary, <0.1ms region mismatch, <15μs logging
