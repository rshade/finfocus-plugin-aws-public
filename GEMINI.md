# Gemini Context for pulumicost-plugin-aws-public

This file provides context for Gemini when working with this repository.

## Project Overview

**pulumicost-plugin-aws-public** is a gRPC-based plugin for PulumiCost that
estimates AWS infrastructure costs using publicly available on-demand pricing.

- **Purpose:** Provides cost estimates without requiring AWS credentials,
  Cost Explorer access, or CUR data.
- **Architecture:**
  - **Protocol:** Implements `CostSourceService` from `pulumicost.v1` via gRPC.
  - **Deployment:** Runs as a standalone subprocess started by PulumiCost core.
  - **Regionality:** Uses region-specific binaries to minimize size.
  - **Data:** Embeds pricing data at build time using `//go:embed`.
  - **Concurrency:** Thread-safe handling of concurrent gRPC calls.
- **Language:** Go 1.25+

## Building and Running

### Prerequisites

- Go 1.25+
- `golangci-lint` (for linting)
- `goreleaser` (optional, for releases)

### Key Commands

- `make build` - Build with fallback pricing
- `make build-region REGION=<region>` - Build for a specific region (recommended: us-east-1 for testing)
- `make build-all-regions` - Build for all regions (only before release verification)
- `make test` - Run all tests
- `go test ./internal/plugin -v` - Run tests for specific package
- `make lint` - Run `golangci-lint`
- `make generate-pricing` - Fetch pricing from AWS API (default: all 9 regions)
- `make clean` - Remove build artifacts

### Development Workflow: Pricing Data Management

**For Daily Development/Testing:**

```bash
# Generate only us-east-1 pricing data (fast, space-efficient)
rm -f ./data/aws_pricing_*.json  # Clean old data if switching
go run ./tools/generate-pricing --regions us-east-1 --out-dir ./data

# Build single region binary
make build-region REGION=us-east-1

# Run tests (all tests use us-east-1 for ELB, EC2, DynamoDB, etc.)
make test
```

**Before Release/Regional Verification:**

```bash
# Clean old data FIRST
rm -f ./data/aws_pricing_*.json

# Generate all regions
go run ./tools/generate-pricing --regions us-east-1,us-west-2,eu-west-1,ca-central-1,sa-east-1,ap-southeast-1,ap-southeast-2,ap-northeast-1,ap-south-1 --out-dir ./data

# Build all regional binaries
make build-all-regions
```

**Why Single-Region Testing is Sufficient for Features Like ELB:**

- Pricing parser logic is region-agnostic; us-east-1 tests all parsing code paths
- Estimation logic (ALB/NLB cost calculations) is independent of region
- Build-tag verification works with one region
- Functional correctness validated through unit/integration tests
- Regional verification needed only for release artifact verification

### Running Locally

1. **Build:** `make build-region REGION=us-east-1`
2. **Run:** `./pulumicost-plugin-aws-public-us-east-1`
3. **Output:** Plugin prints `PORT=<number>` to stdout and starts gRPC server.

## Development Conventions

### Code Style

- **Formatting:** Standard `gofmt`.
- **Linting:** Strict adherence to `golangci-lint`.
- **Documentation:** Comprehensive Go doc comments for exported functions.
- **No Dummy Data:** Always implement fetchers for real data sources.

### ⚠️ CRITICAL: No Pricing Data Filtering

**DO NOT filter, trim, or strip pricing data in `tools/generate-pricing`.**

The v0.0.10/v0.0.11 releases were broken because filtering stripped 85% of data:

- EC2 products: ~90,000 → ~12,000 (filtered)
- Many instance types returned $0

**Rules:**

1. Merge ALL products without filtering by ProductFamily
2. Keep ALL attributes (do not strip to "required" fields)
3. Keep ALL OnDemand terms
4. No "optimization" - full ~150MB data is required per region

**Immutable tests prevent regression:**

- `TestEmbeddedPricingDataSize` - Fails if < 100MB
- `TestEmbeddedPricingProductCount` - Fails if < 50,000 products

### Architecture Patterns

- **Batch Processing:** `GetRecommendations` supports batching via `TargetResources` (max 100 items).
- **One Resource per RPC (Legacy):** Other RPCs still handle single resources.
- **Region Tags:** Use Go build tags (e.g., `//go:build region_use1`).
- **Error Handling:** Use proto-defined `ErrorCode` enum.

### Logging

- **Library:** `rs/zerolog` with structured logging.
- **Stdout:** **RESERVED** for `PORT=<port>` announcement only.
- **Stderr:** Use for all other logs (debug, info, error).
- **Prefix:** Logs should identify the plugin.

### Testing

- **Unit Tests:** Comprehensive tests for pricing logic and handlers.
- **Integration Tests:** Use `-tags=integration` for binary tests.
- **Trace ID:** Verify `trace_id` propagation in integration tests.
- **Cleanup:** Ensure tests clean up temporary files after execution.

## Directory Overview

- `cmd/pulumicost-plugin-aws-public/`: Main entry point (`main.go`).
- `internal/plugin/`: Core plugin logic (plugin, supports, projected, actual).
- `internal/pricing/`: Pricing data handling and embedded data.
- `tools/generate-pricing/`: Tool to fetch and generate pricing JSON.
- `data/`: Generated JSON pricing files (not committed to Git).
- `specs/`: Project specifications and planning documents.

## Key Files

- `internal/plugin/plugin.go`: Implements the `Plugin` interface.
- `internal/pricing/client.go`: Thread-safe client for looking up pricing data.
- `internal/plugin/supports.go`: Logic to determine if a resource is supported.
- `internal/plugin/projected.go`: Logic to calculate projected costs.
- `CLAUDE.md`: Detailed reference for architecture and protocol.

## Active Technologies

- Go 1.25+ + gRPC (pulumicost.v1), rs/zerolog, pluginsdk; embedded JSON pricing via `//go:embed` parsed into indexed maps

## Recent Changes

- 017-elb-cost-estimation: Added ELB (ALB/NLB) cost estimation with pricing data
- 016-dynamodb-cost: Added DynamoDB cost estimation with On-Demand and Provisioned modes
- 014-lambda-cost-estimation: Added Lambda cost estimation framework
