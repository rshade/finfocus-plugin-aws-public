# Quick Start: Plugin Rename to FinFocus

**Feature**: 001-plugin-rename
**Date**: 2026-01-11
**Status**: Draft

## Overview

This guide helps developers understand the rename from `finfocus-plugin-aws-public` to `finfocus-plugin-aws-public`. This is a breaking change (v0.2.0) that updates naming across the entire codebase while maintaining functional compatibility.

## What Changed

| Component | Before | After |
|-----------|--------|-------|
| Module Name | `finfocus-plugin-aws-public` | `finfocus-plugin-aws-public` |
| Binary Names | `finfocus-plugin-aws-public-<region>` | `finfocus-plugin-aws-public-<region>` |
| Proto Package | `finfocus.v1` | `finfocus.v1` |
| Spec Dependency | `github.com/rshade/finfocus-spec v0.4.14` | `github.com/rshade/finfocus-spec v0.5.0` |
| Logging Prefix | `[finfocus-plugin-aws-public]` | `[finfocus-plugin-aws-public]` |
| Command Directory | `cmd/finfocus-plugin-aws-public/` | `cmd/finfocus-plugin-aws-public/` |

## What Didn't Change

- gRPC protocol interface (proto message structure is identical)
- Functional behavior and cost estimation logic
- Performance characteristics
- Build tags (region_use1, region_usw2, region_euw1)
- Pricing data format and content

## Getting Started

### Prerequisites

- Go 1.25.5 or later
- Git
- Make
- golangci-lint (for linting)

### Installation

**Before (finfocus):**
```bash
git clone git@github.com:rshade/finfocus-plugin-aws-public.git
cd finfocus-plugin-aws-public
make build
```

**After (finfocus):**
```bash
git clone git@github.com:rshade/finfocus-plugin-aws-public.git
cd finfocus-plugin-aws-public
make build
```

**Note**: The repository URL will change from `finfocus-plugin-aws-public` to `finfocus-plugin-aws-public` after migration.

### Building the Plugin

The build process is identical, only output binary names have changed:

```bash
# Build all region binaries
make build

# Or build specific region
make build-region REGION=us-east-1  # Builds: bin/finfocus-plugin-aws-public-use1
make build-region REGION=us-west-2  # Builds: bin/finfocus-plugin-aws-public-usw2
make build-region REGION=eu-west-1  # Builds: bin/finfocus-plugin-aws-public-ew1
```

**Output Binaries**:
- `bin/finfocus-plugin-aws-public-use1` (us-east-1)
- `bin/finfocus-plugin-aws-public-usw2` (us-west-2)
- `bin/finfocus-plugin-aws-public-ew1` (eu-west-1)

### Running Tests

The test commands remain unchanged:

```bash
# Run all tests
make test

# Run tests for specific package
go test ./internal/pricing -v

# Run specific test
go test ./internal/plugin -v -run TestGetProjectedCost
```

### Linting

The linting command remains unchanged:

```bash
# Run linter
make lint
```

## Development Workflow

### 1. Clone Repository

```bash
git clone git@github.com:rshade/finfocus-plugin-aws-public.git
cd finfocus-plugin-aws-public
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Make Changes

Edit source files as needed. The code structure is unchanged:

```text
cmd/finfocus-plugin-aws-public/    # Entry point
└── main.go

internal/
├── pricing/                        # Pricing lookup logic
│   ├── loader.go
│   └── lookup.go
└── plugin/                        # gRPC service implementation
    ├── server.go
    └── costs.go
```

### 4. Build and Test

```bash
# Build
make build

# Test
make test

# Lint
make lint
```

### 5. Test gRPC Functionality

**Start the plugin:**
```bash
./bin/finfocus-plugin-aws-public-use1
# Output: PORT=50051
```

**Test with grpcurl:**
```bash
# List services
grpcurl -plaintext 127.0.0.1:50051 list

# Get plugin name
grpcurl -plaintext 127.0.0.1:50051 \
  finfocus.v1.CostSourceService/Name

# Test Supports method
grpcurl -plaintext -d '{
  "region": "us-east-1",
  "resource_type": "aws:ec2/instance:Instance"
}' 127.0.0.1:50051 \
  finfocus.v1.CostSourceService/Supports

# Test GetProjectedCost method
grpcurl -plaintext -d '{
  "region": "us-east-1",
  "resource_type": "aws:ec2/instance:Instance",
  "properties": {
    "instanceType": "t3.micro"
  }
}' 127.0.0.1:50051 \
  finfocus.v1.CostSourceService/GetProjectedCost
```

## Code Examples

### Importing the Plugin

**Before (finfocus):**
```go
import (
    finfocusv1 "github.com/rshade/finfocus-spec/v1"
)
```

**After (finfocus):**
```go
import (
    finfocusv1 "github.com/rshade/finfocus-spec/v1"
)
```

### Logging

**Before (finfocus):**
```go
log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
    With().
    Str("component", "[finfocus-plugin-aws-public]").
    Logger()
```

**After (finfocus):**
```go
log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
    With().
    Str("component", "[finfocus-plugin-aws-public]").
    Logger()
```

### Proto Package Usage

**Before (finfocus):**
```go
import finfocusv1 "github.com/rshade/finfocus-spec/proto/v1"

// Use proto types
desc := &finfocusv1.ResourceDescriptor{
    Region: "us-east-1",
    ResourceType: "aws:ec2/instance:Instance",
}
```

**After (finfocus):**
```go
import finfocusv1 "github.com/rshade/finfocus-spec/proto/v1"

// Use proto types (structure unchanged)
desc := &finfocusv1.ResourceDescriptor{
    Region: "us-east-1",
    ResourceType: "aws:ec2/instance:Instance",
}
```

## Migration Guide for Users

If you're upgrading from `finfocus-plugin-aws-public` to `finfocus-plugin-aws-public`:

### 1. Update Your Repository

```bash
# Clone new repository
git clone git@github.com:rshade/finfocus-plugin-aws-public.git

# Or update remote if you have the old one
git remote set-url origin git@github.com:rshade/finfocus-plugin-aws-public.git
git fetch --all
```

### 2. Update Dependencies

If you're importing the plugin as a dependency:

```bash
# Update go.mod to use new module name
go get github.com/rshade/finfocus-plugin-aws-public@v0.2.0

# Or if you're embedding the plugin binary, just download the new binaries
```

### 3. Update Configuration

Update any configuration files that reference the plugin:

**Before:**
```yaml
plugin_binary: "/path/to/finfocus-plugin-aws-public-use1"
```

**After:**
```yaml
plugin_binary: "/path/to/finfocus-plugin-aws-public-use1"
```

### 4. Update Logging Configuration

If you're filtering logs by component name:

**Before:**
```bash
# Filter for finfocus logs
grep "\[finfocus-plugin-aws-public\]" app.log
```

**After:**
```bash
# Filter for finfocus logs
grep "\[finfocus-plugin-aws-public\]" app.log
```

## Troubleshooting

### Build Fails with "module not found"

**Problem**: Go can't find the module after rename.

**Solution**:
```bash
# Clean module cache
go clean -modcache

# Redownload dependencies
go mod download
go mod tidy
```

### Tests Fail with Import Errors

**Problem**: Tests still reference old import paths.

**Solution**: Run `go mod tidy` to update imports automatically, then run tests again.

### Plugin Doesn't Start

**Problem**: Plugin exits immediately without output.

**Solution**:
```bash
# Check for errors
./bin/finfocus-plugin-aws-public-use1

# Try with debug logging
LOG_LEVEL=debug ./bin/finfocus-plugin-aws-public-use1
```

### gRPC Connection Refused

**Problem**: Can't connect to plugin with grpcurl.

**Solution**:
```bash
# Check if plugin is running
./bin/finfocus-plugin-aws-public-use1 &
PORT=50051  # Check output

# Verify PORT is announced
# Should see: PORT=50051
```

## Version Compatibility

| Version | Status | Notes |
|---------|--------|-------|
| v0.1.x | Deprecated | Old `finfocus-plugin-aws-public` versions |
| v0.2.0 | Current | First `finfocus-plugin-aws-public` release |

**Breaking Changes in v0.2.0**:
- Module name changed
- Binary names changed
- Proto package name changed (finfocus.v1 → finfocus.v1)

**No Breaking Changes**:
- gRPC protocol interface unchanged
- Proto message structure unchanged
- Functional behavior unchanged

## Additional Resources

- **Constitution**: `.specify/memory/constitution.md` - Core principles and guidelines
- **Data Model**: `specs/001-plugin-rename/data-model.md` - Entity and data flow documentation
- **Research**: `specs/001-plugin-rename/research.md` - Technical decisions and rationale
- **RENAME-PLAN.md**: Project rename roadmap and migration strategy

## Getting Help

- Check the GitHub repository for issues and discussions
- Review the constitution for development guidelines
- See AGENTS.md for project-specific commands and conventions
- Use `--help` flag with make commands: `make help`

## Next Steps

After reviewing this guide:
1. Read the constitution to understand development principles
2. Review the data model to understand the plugin architecture
3. Run `make build`, `make test`, and `make lint` to verify your environment
4. Test the gRPC interface with grpcurl
5. Start contributing!