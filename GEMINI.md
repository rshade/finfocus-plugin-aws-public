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
- `make build-region REGION=<region>` - Build for a specific region
- `make build-all-regions` - Build for all regions
- `make test` - Run all tests
- `go test ./internal/plugin -v` - Run tests for specific package
- `make lint` - Run `golangci-lint`
- `make generate-pricing` - Fetch pricing from AWS API
- `make clean` - Remove build artifacts

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
- Go 1.25+ + gRPC (pulumicost.v1 protocol), internal/pricing (embedded data), zerolog (014-lambda-cost-estimation)
- Embedded JSON pricing data (using `//go:embed`) (014-lambda-cost-estimation)
- Go 1.25+ + gRPC, RS/Zerolog, Pluginsdk (016-dynamodb-cost)
- N/A (Embedded pricing data) (016-dynamodb-cost)

## Recent Changes
- 016-dynamodb-cost: Added Go 1.25+ + gRPC, RS/Zerolog, Pluginsdk
