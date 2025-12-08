# AGENTS.md - Development Guidelines for pulumicost-plugin-aws-public

## Build/Lint/Test Commands

- **Lint**: `make lint` (golangci-lint)
- **Test all**: `make test` (go test -v ./...)
- **Test single**: `go test ./internal/plugin -v -run TestName`
- **Build**: `make build` or `make build-region REGION=us-east-1`

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
- Embedded JSON files in Go binaries (no runtime storage) (011-s3-cost-estimation)

- Go 1.25.4 + gRPC, zerolog, embedded JSON pricing data (010-eks-cost-estimation)
- Embedded JSON pricing data (no runtime storage) (010-eks-cost-estimation)
- Go 1.25.4 + GoReleaser, gRPC, build tags (006-region-build-matrix)
- Embedded JSON files in Go binaries (006-region-build-matrix)

## Recent Changes

- 006-region-build-matrix: Added Go 1.25.4 + GoReleaser, gRPC, build tags

## Custom Commands

### /code-review

Performs automated code review on specified target files or directories.

Usage:

```bash
/code-review [OPTIONS] [TARGET]
```

Options:

- `-h, --help`: Show help message
- `-f, --format`: Output format (text, json, markdown)
- `-s, --severity`: Minimum severity level (low, medium, high)
- `-l, --language`: Language override
- `--fix`: Suggest fixes for issues when possible

Examples:

```bash
/code-review
/code-review src/
/code-review --severity high src/main.go
/code-review --format json --fix src/
```

Supported languages: Go, JavaScript/TypeScript, Python

### speckit.research

Specialized research agent for technical decision-making and technology evaluation.

Usage:

- Called by speckit.plan for research tasks
- Can be invoked directly for specific research needs
- Provides structured research findings with recommendations

### speckit.code-review

Automated code review agent integrated with speckit workflow.

Usage:

- Automatically called during speckit.implement for quality assurance
- Can be invoked manually for code review needs
- Provides detailed findings with severity levels and fix suggestions
