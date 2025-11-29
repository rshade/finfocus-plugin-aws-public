# Gemini Context for pulumicost-plugin-aws-public

This file provides context for Gemini when working with the `pulumicost-plugin-aws-public` repository.

## Project Overview

**pulumicost-plugin-aws-public** is a gRPC-based plugin for PulumiCost that estimates AWS infrastructure costs using publicly available on-demand pricing data.

*   **Purpose:** Provides cost estimates without requiring AWS credentials, Cost Explorer access, or CUR data.
*   **Architecture:**
    *   **Protocol:** Implements `CostSourceService` from `pulumicost.v1` via gRPC.
    *   **Deployment:** Runs as a standalone subprocess started by PulumiCost core.
    *   **Regionality:** Uses **region-specific binaries** (e.g., `pulumicost-plugin-aws-public-us-east-1`) to minimize size.
    *   **Data:** Embeds pricing data at build time using `//go:embed`.
    *   **Concurrency:** Thread-safe handling of concurrent gRPC calls.
*   **Language:** Go 1.25+

## Building and Running

### Prerequisites
*   Go 1.25+
*   `golangci-lint` (for linting)
*   `goreleaser` (optional, for releases)

### Key Commands

| Action | Command | Description |
| :--- | :--- | :--- |
| **Build (Default)** | `make build` | Builds the plugin with fallback pricing data. |
| **Build (Region)** | `make build-region REGION=<region>` | Builds a binary for a specific region (e.g., `us-east-1`, `eu-west-1`). |
| **Build (All)** | `make build-all-regions` | Builds binaries for all supported regions. |
| **Test (All)** | `make test` | Runs all unit and integration tests. |
| **Test (Specific)** | `go test ./internal/plugin -v` | Runs tests for a specific package. |
| **Lint** | `make lint` | Runs `golangci-lint`. |
| **Generate Pricing**| `make generate-pricing` | Fetches/generates pricing data (uses `--dummy` for dev). |
| **Clean** | `make clean` | Removes build artifacts. |

### Running Locally
1.  **Build:** `make build-region REGION=us-east-1`
2.  **Run:** `./pulumicost-plugin-aws-public-us-east-1`
3.  **Output:** The plugin will print `PORT=<number>` to stdout and start the gRPC server on that port.

## Development Conventions

### Code Style
*   **Formatting:** Standard `gofmt`.
*   **Linting:** Strict adherence to `golangci-lint`.
*   **Documentation:** Comprehensive Go doc comments for all exported functions and tests (CodeRabbit requirement).

### Architecture Patterns
*   **One Resource per RPC:** The plugin handles one resource cost estimation per gRPC call.
*   **No Batching:** Do not assume batch processing.
*   **Region Tags:** Use Go build tags (e.g., `//go:build region_use1`) to include only relevant pricing data in each binary.
*   **Error Handling:** Use proto-defined `ErrorCode` enum (e.g., `ERROR_CODE_UNSUPPORTED_REGION`, `ERROR_CODE_INVALID_RESOURCE`). Do not use custom error codes.

### Logging
*   **Library:** `rs/zerolog` with structured logging.
*   **Stdout:** **RESERVED** for `PORT=<port>` announcement only.
*   **Stderr:** Use for all other logs (debug, info, error).
*   **Prefix:** Logs should identify the plugin (e.g., `[pulumicost-plugin-aws-public]`).

### Testing
*   **Unit Tests:** comprehensive tests for pricing logic and plugin handlers.
*   **Integration Tests:** Use `-tags=integration` to run tests that build and execute the binary.
*   **Trace ID:** Verify `trace_id` propagation from gRPC metadata to logs in integration tests.

## Directory Overview

*   `cmd/pulumicost-plugin-aws-public/`: Main entry point (`main.go`).
*   `internal/plugin/`: Core plugin logic (`plugin.go`, `supports.go`, `projected.go`, `actual.go`).
*   `internal/pricing/`: Pricing data handling (`client.go`) and embedded data (`embed_*.go`).
*   `tools/generate-pricing/`: Tool to fetch and generate `data/aws_pricing_*.json`.
*   `data/`: Generated JSON pricing files (not committed to Git).
*   `specs/`: Project specifications and planning documents.

## Key Files

*   `internal/plugin/plugin.go`: Implements the `Plugin` interface.
*   `internal/pricing/client.go`: Thread-safe client for looking up pricing data.
*   `internal/plugin/supports.go`: Logic to determine if a resource is supported.
*   `internal/plugin/projected.go`: Logic to calculate projected costs.
*   `CLAUDE.md`: Detailed reference for architecture and protocol (highly recommended reading).
