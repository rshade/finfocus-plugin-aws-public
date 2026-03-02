# Quickstart: Multi-Region Router Plugin

**Branch**: `036-multi-region-router` | **Date**: 2026-02-12

## Prerequisites

- Go 1.25+
- `make develop` has been run (pricing data generated)
- Repository cloned with `finfocus-spec` dependency resolved

## Build the Router

```bash
# Build router binary (no region tag needed)
go build -o dist/finfocus-plugin-aws-public ./cmd/finfocus-plugin-aws-public-router

# Build a region binary for testing
make build-default-region  # Builds us-east-1
```

## Run Locally (Connected Mode)

```bash
# Start the router (it will auto-download region binaries on demand)
./dist/finfocus-plugin-aws-public
# Output: PORT=<port>

# In another terminal, test with grpcurl or the SDK client
grpcurl -plaintext localhost:<port> finfocus.v1.CostSourceService/Name
```

## Run Locally (Offline Mode)

```bash
# Place region binaries alongside the router
cp dist/finfocus-plugin-aws-public-us-east-1 dist/

# Start in offline mode
FINFOCUS_PLUGIN_OFFLINE=true ./dist/finfocus-plugin-aws-public
# Output: PORT=<port>

# Requests for us-east-1 work, others return helpful error
```

## Run Tests

```bash
# Unit tests (no build tags needed for router)
go test ./internal/router/...

# All tests
make test

# Integration test with real child binary
go test -tags=integration ./internal/router/... -run TestIntegration -v
```

## Key Files

| File | Purpose |
|------|---------|
| `cmd/finfocus-plugin-aws-public-router/main.go` | Router entry point |
| `internal/router/router.go` | RouterPlugin (implements pluginsdk.Plugin) |
| `internal/router/child.go` | Child process lifecycle management |
| `internal/router/registry.go` | Thread-safe child process registry |
| `internal/router/downloader.go` | Binary download + SHA256 verification |
| `internal/router/discovery.go` | Local binary discovery |

## Architecture Overview

```text
┌─────────────────────────────────────────────────┐
│              FinFocus Core                       │
│         (gRPC client on PORT)                    │
└──────────────────┬──────────────────────────────┘
                   │ gRPC / Connect
                   ▼
┌─────────────────────────────────────────────────┐
│           Router Plugin (this binary)            │
│  - Announces PORT to stdout                      │
│  - Implements CostSourceService                  │
│  - Extracts region from ResourceDescriptor       │
│  - Manages child processes per region            │
│  - Downloads missing binaries (connected mode)   │
└──┬──────────────┬──────────────┬────────────────┘
   │ Connect      │ Connect      │ Connect
   ▼              ▼              ▼
┌────────┐  ┌────────┐    ┌────────┐
│us-east-1│  │us-west-2│    │eu-west-1│
│ child   │  │ child   │    │ child   │
│ (PORT X)│  │ (PORT Y)│    │ (PORT Z)│
└────────┘  └────────┘    └────────┘
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `FINFOCUS_PLUGIN_OFFLINE` | `false` | Disable auto-download of region binaries |
| `FINFOCUS_PLUGIN_WEB_ENABLED` | `false` | Enable multi-protocol support on router |
| `FINFOCUS_LOG_LEVEL` | `info` | Log level for router (debug, info, warn, error) |
| `PORT` | `0` (ephemeral) | Fixed port for router (0 = auto-assign) |
