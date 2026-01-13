# Quickstart: Asia Pacific Region Support

**Feature**: 002-ap-region-support
**Audience**: Developers implementing or testing AP region support
**Estimated Time**: 30 minutes

## Prerequisites

- Go 1.25+ installed
- GoReleaser installed (for multi-platform builds)
- Repository cloned: `finfocus-plugin-aws-public`
- Working directory: Repository root

## Quick Build (Single Region)

### Build Singapore Binary (ap-southeast-1)

```bash
# Generate pricing data for Singapore
go run ./tools/generate-pricing \
  --regions ap-southeast-1 \
  --out-dir ./internal/pricing/data \
  --dummy

# Build binary with region tag
go build \
  -tags region_apse1 \
  -o finfocus-plugin-aws-public-ap-southeast-1 \
  ./cmd/finfocus-plugin-aws-public

# Run and verify
./finfocus-plugin-aws-public-ap-southeast-1
# Output: PORT=<port>
# (Binary serves gRPC on localhost:<port>)
```

### Build Other AP Regions

**Sydney (ap-southeast-2)**:

```bash
go run ./tools/generate-pricing --regions ap-southeast-2 --out-dir ./internal/pricing/data --dummy
go build -tags region_apse2 -o finfocus-plugin-aws-public-ap-southeast-2 ./cmd/finfocus-plugin-aws-public
```

**Tokyo (ap-northeast-1)**:

```bash
go run ./tools/generate-pricing --regions ap-northeast-1 --out-dir ./internal/pricing/data --dummy
go build -tags region_apne1 -o finfocus-plugin-aws-public-ap-northeast-1 ./cmd/finfocus-plugin-aws-public
```

**Mumbai (ap-south-1)**:

```bash
go run ./tools/generate-pricing --regions ap-south-1 --out-dir ./internal/pricing/data --dummy
go build -tags region_aps1 -o finfocus-plugin-aws-public-ap-south-1 ./cmd/finfocus-plugin-aws-public
```

## Build All AP Regions (GoReleaser)

### Generate Pricing Data for All AP Regions

```bash
go run ./tools/generate-pricing \
  --regions ap-southeast-1,ap-southeast-2,ap-northeast-1,ap-south-1 \
  --out-dir ./internal/pricing/data \
  --dummy
```

### Build All Binaries (Snapshot Mode)

```bash
# Build all regions for all platforms
goreleaser build --snapshot --clean

# Build only AP regions
goreleaser build --snapshot --clean --id ap-southeast-1,ap-southeast-2,ap-northeast-1,ap-south-1

# Check output
ls -lh dist/
```

**Output Structure**:

```
dist/
├── finfocus-plugin-aws-public-ap-southeast-1_linux_amd64/
│   └── finfocus-plugin-aws-public-ap-southeast-1
├── finfocus-plugin-aws-public-ap-southeast-1_darwin_arm64/
│   └── finfocus-plugin-aws-public-ap-southeast-1
├── finfocus-plugin-aws-public-ap-southeast-2_linux_amd64/
... (24 total AP artifacts: 4 regions × 6 platforms)
```

## Testing a Binary

### Manual Test with grpcurl

1. **Start the binary**:

```bash
./finfocus-plugin-aws-public-ap-southeast-1
# Note the PORT output, e.g., PORT=54321
```

2. **Test Name() RPC**:

```bash
grpcurl -plaintext \
  -d '{}' \
  localhost:54321 \
  finfocus.v1.CostSourceService/Name

# Expected output:
# {
#   "name": "aws-public"
# }
```

3. **Test Supports() RPC** (matching region):

```bash
grpcurl -plaintext \
  -d '{
    "resource": {
      "provider": "aws",
      "resource_type": "ec2",
      "sku": "t3.micro",
      "region": "ap-southeast-1"
    }
  }' \
  localhost:54321 \
  finfocus.v1.CostSourceService/Supports

# Expected output:
# {
#   "supported": true,
#   "reason": "EC2 and EBS resources supported"
# }
```

4. **Test Supports() RPC** (wrong region):

```bash
grpcurl -plaintext \
  -d '{
    "resource": {
      "provider": "aws",
      "resource_type": "ec2",
      "sku": "t3.micro",
      "region": "us-east-1"
    }
  }' \
  localhost:54321 \
  finfocus.v1.CostSourceService/Supports

# Expected output:
# {
#   "supported": false,
#   "reason": "Region not supported by this binary"
# }
```

5. **Test GetProjectedCost() RPC**:

```bash
grpcurl -plaintext \
  -d '{
    "resource": {
      "provider": "aws",
      "resource_type": "ec2",
      "sku": "t3.micro",
      "region": "ap-southeast-1"
    }
  }' \
  localhost:54321 \
  finfocus.v1.CostSourceService/GetProjectedCost

# Expected output (dummy data):
# {
#   "unitPrice": 0.0116,
#   "currency": "USD",
#   "costPerMonth": 8.468,
#   "billingDetail": "On-demand Linux, shared tenancy, 730 hrs/month"
# }
```

### Automated Testing

```bash
# Run all tests (includes AP region test cases)
make test

# Run specific test packages
go test ./internal/plugin -v
go test ./internal/pricing -v

# Run tests with coverage
make test-coverage
```

## Development Workflow

### Step 1: Create Embed Files

For each AP region, create an embed file in `internal/pricing/`:

**Example** (`embed_apse1.go`):

```go
//go:build region_apse1

package pricing

import _ "embed"

//go:embed data/aws_pricing_ap-southeast-1.json
var rawPricingJSON []byte
```

Repeat for `embed_apse2.go`, `embed_apne1.go`, `embed_aps1.go`.

### Step 2: Update Fallback Embed

Edit `internal/pricing/embed_fallback.go`:

```go
//go:build !region_use1 && !region_usw2 && !region_euw1 && !region_apse1 && !region_apse2 && !region_apne1 && !region_aps1

package pricing

// ... rest of fallback file
```

### Step 3: Update GoReleaser

Add to `.goreleaser.yaml` under `builds:`:

```yaml
# ap-southeast-1 binary
- id: ap-southeast-1
  main: ./cmd/finfocus-plugin-aws-public
  binary: finfocus-plugin-aws-public-ap-southeast-1
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
    - region_apse1
  ldflags:
    - -s -w -X main.version={{.Version}}

# Repeat for other AP regions...
```

### Step 4: Update Tests

Extend table-driven tests with AP region cases:

**Example** (`internal/plugin/supports_test.go`):

```go
func TestSupports(t *testing.T) {
    tests := []struct {
        name         string
        region       string
        resourceType string
        want         bool
    }{
        // Existing tests...
        {
            name:         "supports ap-southeast-1",
            region:       "ap-southeast-1",
            resourceType: "ec2",
            want:         true,
        },
        {
            name:         "supports ap-southeast-2",
            region:       "ap-southeast-2",
            resourceType: "ec2",
            want:         true,
        },
        // Add ap-northeast-1 and ap-south-1...
    }
    // ... test execution
}
```

### Step 5: Update Documentation

Update `README.md`:

```markdown
## Supported Regions

### North America
- us-east-1 (N. Virginia)
- us-west-2 (Oregon)

### Europe
- eu-west-1 (Ireland)

### Asia Pacific
- ap-southeast-1 (Singapore)
- ap-southeast-2 (Sydney)
- ap-northeast-1 (Tokyo)
- ap-south-1 (Mumbai)

## Building Region-Specific Binaries

```bash
# Build Singapore binary
go build -tags region_apse1 -o finfocus-plugin-aws-public-ap-southeast-1 ./cmd/finfocus-plugin-aws-public

# Build all binaries with GoReleaser
goreleaser build --snapshot --clean
```

```

## Verification Checklist

After implementing AP region support, verify:

- [ ] All 4 pricing data files exist in `internal/pricing/data/`
- [ ] All 4 embed files exist in `internal/pricing/`
- [ ] Fallback embed excludes all 4 new region tags
- [ ] `.goreleaser.yaml` has 4 new build configurations
- [ ] `make test` passes with AP region test cases
- [ ] Each AP binary builds without errors
- [ ] Each AP binary serves gRPC and announces PORT
- [ ] Each AP binary returns correct region in pricing data
- [ ] Each AP binary rejects wrong-region requests
- [ ] README.md documents AP region support

## Common Issues & Solutions

### Issue: Build fails with "pattern internal/pricing/data/*.json: no matching files found"

**Solution**: Generate pricing data first:
```bash
go run ./tools/generate-pricing --regions ap-southeast-1,ap-southeast-2,ap-northeast-1,ap-south-1 --out-dir ./internal/pricing/data --dummy
```

### Issue: Multiple embed files selected at build time

**Solution**: Check fallback embed build constraint includes all region tags:

```bash
grep -A1 "//go:build" internal/pricing/embed_fallback.go
# Should show: !region_use1 && !region_usw2 && !region_euw1 && !region_apse1 && !region_apse2 && !region_apne1 && !region_aps1
```

### Issue: Binary returns wrong region in pricing data

**Solution**: Verify build tag matches expected region:

```bash
# Build with verbose output
go build -v -tags region_apse1 -o test-binary ./cmd/finfocus-plugin-aws-public 2>&1 | grep embed

# Should show: internal/pricing/embed_apse1.go (not embed_use1.go or embed_fallback.go)
```

### Issue: GoReleaser builds 42 artifacts but some fail

**Solution**: Check that all pricing data files exist before running GoReleaser:

```bash
ls -1 internal/pricing/data/
# Should show:
# aws_pricing_us-east-1.json
# aws_pricing_us-west-2.json
# aws_pricing_eu-west-1.json
# aws_pricing_ap-southeast-1.json
# aws_pricing_ap-southeast-2.json
# aws_pricing_ap-northeast-1.json
# aws_pricing_ap-south-1.json
```

## Next Steps

After completing quickstart:

1. Run `/speckit.tasks` to generate detailed implementation tasks
2. Follow task breakdown for systematic implementation
3. Create PR when all tasks complete and tests pass
4. Update `CLAUDE.md` if new patterns discovered

## Reference Commands

### Makefile Commands

```bash
make lint          # Run golangci-lint
make test          # Run all tests
make test-coverage # Run tests with coverage report
make build         # Build default binary (fallback)
```

### One-Liner: Full Build & Test

```bash
go run ./tools/generate-pricing --regions ap-southeast-1,ap-southeast-2,ap-northeast-1,ap-south-1 --out-dir ./internal/pricing/data --dummy && \
goreleaser build --snapshot --clean --id ap-southeast-1,ap-southeast-2,ap-northeast-1,ap-south-1 && \
make test
```

## Additional Resources

- [Region Mappings Contract](./contracts/region-mappings.md) - Authoritative region mapping table
- [Data Model](./data-model.md) - Entity relationships and data flow
- [Research](./research.md) - Decision rationale and alternatives considered
- [CLAUDE.md](../../../CLAUDE.md) - Project conventions and guidance
