# AGENTS.md - Development Guidelines for pulumicost-plugin-aws-public

## Build/Lint/Test Commands
- **Lint**: `make lint` (golangci-lint)
- **Test all**: `make test` (go test -v ./...)
- **Test single**: `go test ./internal/plugin -v -run TestName`
- **Build**: `make build` (standard build) or `make build-region REGION=us-east-1` (region-specific)

## Code Style Guidelines

### Go Standards
- **Version**: Go 1.25.4 minimum
- **Formatting**: Standard gofmt/gofmt -w
- **Imports**: Group by standard library, third-party, local packages
- **Naming**: PascalCase for exported, camelCase for unexported
- **Documentation**: Godoc comments for exported functions/types

### Error Handling
- Use proto-defined ErrorCode enum (not custom codes)
- gRPC status codes with proper error details
- Thread-safe operations (sync.Once, sync.RWMutex)

### Logging
- Use zerolog with structured logging
- Never log to stdout (except PORT announcement)
- Prefix stderr logs: `[pulumicost-plugin-aws-public]`

### Architecture
- One resource per RPC call (no batching)
- Region-specific binaries with build tags
- Thread-safe pricing lookups for concurrent gRPC calls

## Active Technologies
- Go 1.25.4 minimum + gRPC (pulumicost.v1), pluginsdk, embedded pricing data (006-add-us-regions)
- Embedded pricing data (no external storage) (006-add-us-regions)

## Recent Changes
- 006-add-us-regions: Added Go 1.25.4 minimum + gRPC (pulumicost.v1), pluginsdk, embedded pricing data
