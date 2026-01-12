# Distribution and Release Configuration

This document describes the configuration files used for building, packaging, and distributing the FinFocus AWS Public Plugin.

## GoReleaser Configuration

The `.goreleaser.yaml` file defines the build and release process for the plugin. It is automatically generated from the region configuration in `internal/pricing/regions.yaml`.

### Build Configuration

Each AWS region has a dedicated build configuration:

```yaml
builds:
  - id: us-east-1
    main: ./cmd/finfocus-plugin-aws-public
    binary: finfocus-plugin-aws-public-us-east-1
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    tags:
      - region_use1
    ldflags:
      - -s -w -X main.version={{ .Version }}
```

### Key Settings

- **Binary Naming**: `finfocus-plugin-aws-public-{region}` format
- **Cross-Platform**: Linux, macOS (Darwin), and Windows builds
- **Architecture**: AMD64 and ARM64 support
- **Build Tags**: Region-specific tags for embedding pricing data
- **Optimization**: Stripped binaries with version information

### Release Process

The release process generates 18 binaries (3 regions × 3 OS × 2 architectures) and creates:

1. **GitHub Release**: With changelog and release notes
2. **Archives**: Tar.gz for Linux/Darwin, ZIP for Windows
3. **Checksums**: SHA256 verification files
4. **Metadata**: Release assets with proper naming

## Region Configuration

The `internal/pricing/regions.yaml` file defines all supported AWS regions:

```yaml
regions:
  - id: use1
    name: us-east-1
    tag: region_use1
  - id: usw2
    name: us-west-2
    tag: region_usw2
```

### Configuration Fields

- **id**: Short identifier used in build tags and file names
- **name**: Full AWS region name
- **tag**: Go build tag for region-specific compilation

### Automated Generation

The following files are automatically generated from `regions.yaml`:

- `internal/pricing/embed_*.go`: Region-specific pricing data embeds
- `.goreleaser.yaml`: Build configurations for each region
- Region validation scripts

## Build Tags

The plugin uses Go build tags to embed region-specific pricing data:

```bash
# Build for US East 1 with real pricing
go build -tags region_use1 -o finfocus-plugin-aws-public-us-east-1 ./cmd/finfocus-plugin-aws-public

# Build with fallback pricing (development only)
go build -o finfocus-plugin-aws-public ./cmd/finfocus-plugin-aws-public
```

### Tag Format

- `region_{id}` where `id` is the short region identifier
- Example: `region_use1` for `us-east-1`

### Critical Build Requirement

**Always use region-specific build tags for production builds.** The v0.0.10 release was broken because it was built without proper region tags, resulting in all costs returning $0.

## CI/CD Integration

### GitHub Actions Workflows

- **test.yml**: Runs tests and validates region configurations
- **release.yml**: Builds and releases all region binaries
- **release-please.yml**: Manages versioning and changelog generation

### Validation Scripts

- `scripts/verify-regions.sh`: Validates region configurations
- `scripts/build-region.sh`: Builds individual regions
- `scripts/release-region.sh`: Releases individual regions

### Quality Gates

Before release:

1. All region configurations validated
2. Pricing data files exist and are valid
3. Build tags correctly applied
4. Binary sizes verified (>10MB indicates embedded pricing)

## Troubleshooting Distribution Issues

### Common Problems

#### "Binary size too small"

- Check that region-specific build tags were used
- Verify pricing data was generated: `make generate-pricing`

#### "Region not supported"

- Ensure the binary was built for the correct region
- Check that the region exists in `regions.yaml`

#### "Build tag mismatch"

- Verify region configuration is up to date
- Regenerate configuration files: `make generate-goreleaser`

### Manual Build Commands

```bash
# Generate pricing data for all regions
go run ./tools/generate-pricing \
  --regions us-east-1,us-west-2,eu-west-1 \
  --out-dir ./internal/pricing/data

# Build specific region
go build -tags region_use1 -o finfocus-plugin-aws-public-us-east-1 ./cmd/finfocus-plugin-aws-public

# Verify binary size (should be >10MB)
ls -lh finfocus-plugin-aws-public-us-east-1
```

## Adding New Regions

To add support for a new AWS region:

1. **Update regions.yaml**:

   ```yaml
   regions:
     - id: euw3
       name: eu-west-3
       tag: region_euw3
   ```

2. **Generate configurations**:

   ```bash
   make generate-embeds
   make generate-goreleaser
   make verify-regions
   ```

3. **Test the region**:

   ```bash
   make build-region REGION=eu-west-3
   ```

The new region will automatically be included in future releases.
